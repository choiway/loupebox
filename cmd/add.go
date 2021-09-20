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

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("add called")

		for _, value := range args {
			fmt.Println(value)
		}

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

		fmt.Println(path)

		f, err := os.Open(path)
		if err != nil {
			log.Println("Error")
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
			fmt.Printf("SHA Hash: %s\n", sha)

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

			photo := Photo{
				ShaHash:   sha,
				Path:      filepath.Join(newPath, filename),
				DateTaken: tm,
			}

			// Check if in database
			db, err := openDatabase()

			if err != nil {
				panic(err)
			}
			photoExists := CheckIfPhotoExists(db, photo)

			db.Close()

			if photoExists {

				fmt.Print("not copying...")

			} else {

				err = os.MkdirAll(newPath, 0755)
				if err != nil {
					panic(err)
				}

				err = ioutil.WriteFile(filepath.Join(newPath, filename), content, 0755)
				if err != nil {
					log.Fatal(err)
				}

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

	// fmt.Printf("exists: %s\n", strconv.Itoa(exists))

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
		path,
		date_taken
	) values(CURRENT_TIMESTAMP, ?, ?, ?)
	`
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	result, err2 := stmt.Exec(photo.ShaHash, photo.Path, photo.DateTaken)
	if err2 != nil {
		return err2
	}
	fmt.Printf("Result: %s", result)

	return nil
}

func openDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ".loupebox/loupebox.db")

	if err != nil {
		return nil, err
	}

	return db, nil
}
