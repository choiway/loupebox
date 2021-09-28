/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

loupebox add /source/dir

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("add called")

		for _, value := range args {
			fmt.Println(value)
		}

		// Get path argument
		// Will throw an error if it iesn't a valid path but should
		path := args[0]

		fmt.Println("hello, from lightbox")
		fmt.Printf("Processing photos from %s\n", path)

		filenames, err := walkdirectory(path)
		if err != nil {
			log.Fatalln("error reading path")
		}

		addfiles(filenames)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func walkdirectory(path string) ([]string, error) {

	paths := []string{}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		paths = append(paths, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paths, nil
}

func addfiles(filenames []string) {
	for _, path := range filenames {

		log.Printf("Opening: %s\n", path)

		f, err := os.Open(path)
		if err != nil {
			log.Printf("An error ocurred while trying to open: %s\n", path)
			log.Println(err)
		}

		content, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Println(err)
		}

		// Optionally register camera makenote data parsing - currently Nikon and
		// Canon are supported.
		exif.RegisterParsers(mknote.All...)
		exifData, err := exif.Decode(bytes.NewReader(content))

		if err != nil {
			log.Print("EXIST decode error")
			log.Println(err)
			fmt.Print("\n\n")
		} else {

			sha := hashContent(content)
			// fmt.Printf("SHA Hash: %s\n", sha)

			currentPath, err := os.Getwd()
			if err != nil {
				log.Println(err)
			}

			filename := filepath.Base(path)

			// camMake, _ := x.Get(exif.Make)
			// fmt.Println(camMake.StringVal())

			// camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
			// fmt.Println(camModel.StringVal())

			tm, _ := exifData.DateTime()
			newPath := buildContentPath(tm, currentPath)
			newFilename := generateFileName(filename, sha)
			newPhotoPath := filepath.Join(newPath, newFilename)

			photo := Photo{
				ShaHash:    sha,
				SourcePath: path,
				Path:       newPhotoPath,
				DateTaken:  tm,
			}

			// Check if in database
			db, err := openDatabase()
			if err != nil {
				panic(err)
			}

			photoExists := CheckIfPhotoExists(db, photo)

			db.Close()

			if photoExists {
				fmt.Println("Photo already exists, skipping copy")

				db, err := openDatabase()
				if err != nil {
					panic(err)
				}

				sourcePathExists := CheckShaAndSourceRepo(db, photo)

				if sourcePathExists {
					log.Println("Source path already exists in database, skipping...")
				} else {
					log.Println("Logging new source path")

					photo.Status = "skipped"

					db, err := openDatabase()
					if err != nil {
						panic(err)
					}

					err = InsertPhoto(db, photo)
					if err != nil {
						panic(err)
					}
				}

			} else {

				err = os.MkdirAll(newPath, 0755)
				if err != nil {
					panic(err)
				}

				err = ioutil.WriteFile(newPhotoPath, content, 0755)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Copied %s to %s\n", path, newPhotoPath)

				// Add to database
				db, err := openDatabase()
				if err != nil {
					panic(err)
				}

				err = InsertPhoto(db, photo)
				if err != nil {
					panic(err)
				}

				db.Close()
			}

			fmt.Print("\n\n")
		}

		f.Close()
	}
}

func hashContent(content []byte) string {
	h := sha256.New()

	_, err := io.Copy(h, bytes.NewReader(content))
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func buildContentPath(tm time.Time, workingDir string) string {
	year := strconv.Itoa(tm.Year())
	month := doubleDigitMonth(tm.Month())
	day := doubleDigitDay(tm.Day())

	return filepath.Join(workingDir, year, month, day)
}

func doubleDigitMonth(month time.Month) string {
	i := int(month)

	if i < 10 {
		return fmt.Sprintf("0%s", strconv.Itoa(i))
	}

	return strconv.Itoa(i)
}

func doubleDigitDay(day int) string {
	if day < 10 {
		return fmt.Sprintf("0%s", strconv.Itoa(day))
	}

	return strconv.Itoa(day)
}

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)

	if err != nil {
		panic(err)
	}

	if db == nil {
		panic("db nil")
	}

	return db
}

func CheckIfPhotoExists(db *sql.DB, photo Photo) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos where sha_hash = ?);`

	var exists int

	err := db.QueryRow(sql, photo.ShaHash).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if exists == 1 {
		return true
	}

	return false
}

func CheckShaAndSourceRepo(db *sql.DB, photo Photo) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos WHERE sha_hash = ? AND source_path = ?);`

	var exists int

	err := db.QueryRow(sql, photo.ShaHash, photo.SourcePath).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if exists == 1 {
		return true
	}

	return false
}

func InsertPhoto(db *sql.DB, photo Photo) error {
	sql := `
	INSERT OR REPLACE INTO photos(
		inserted_at,
		sha_hash,
		source_path,
		path,
		date_taken,
		status
	) values(CURRENT_TIMESTAMP, ?, ?, ?, ?, ?)
	`
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err2 := stmt.Exec(photo.ShaHash, photo.SourcePath, photo.Path, photo.DateTaken, photo.Status)
	if err2 != nil {
		return err2
	}
	log.Println("Added photo to database")

	return nil
}

func openDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ".loupebox/loupebox.db")

	if err != nil {
		return nil, err
	}

	return db, nil
}

func generateFileName(filename string, sha string) string {

	ext := filepath.Ext(filename)
	n := strings.TrimSuffix(filename, ext)
	shortSha := sha[0:6]

	return fmt.Sprintf("%s_%s%s", n, shortSha, ext)
}