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
	"net/http"
	"os"
	"os/exec"
	"path"
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

		if len(args) == 0 {
			fmt.Print("Please enter the full path to the source directory")
			return
		}

		dryrun, _ := cmd.Flags().GetBool("dryrun")

		for _, value := range args {
			fmt.Println(value)
		}

		// Get path argument
		// Will throw an error if it iesn't a valid path but should
		path := args[0]

		fmt.Println("hello, from lightbox")

		filenames, err := walkdirectory(path)
		if err != nil {
			log.Fatalln("There was an error reading the path. It may not exist or you may have entered it incorrectly. Please check and try again.")
		}

		if dryrun {

			log.Println("Doing dry run")

			dry(filenames)

		} else {

			// Add current repo to repo database
			// This is used to track recent adds

			db, err := openDatabase()
			if err != nil {
				panic(err)
			}

			result, err := InsertRepo(db, path)
			if err != nil {
				log.Fatal(err)
			}

			db.Close()

			lastInsertedId, err := result.LastInsertId()
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("Adding photos from %s\n", path)

			addfiles(filenames)

			// Update repo with completed status

			db, err = openDatabase()
			if err != nil {
				panic(err)
			}
			defer db.Close()

			UpdateRepoWithCompleteStatus(db, lastInsertedId)

			log.Printf("Finished adding %s", path)
		}

	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("dryrun", "", "Runs just checks if filename and directory already exists")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	addCmd.Flags().BoolP("dryrun", "d", false, "Help message for dryrun")
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

		var tm time.Time

		// Check if path already exists in the database
		db, err := openDatabase()
		if err != nil {
			panic(err)
		}

		pathExists := CheckIfPathExists(db, path)

		db.Close()

		if pathExists {
			fmt.Print(".")
			continue
		}

		// Check if directory and file name exists
		// This checks for the intance when the source repo path might be different but the
		// local directory and photo name are the same as one that was already imported

		extension := strings.ToLower(filepath.Ext(path))
		filename := filepath.Base(path)
		dir := filepath.Base(filepath.Dir(path))

		db, err = openDatabase()
		if err != nil {
			panic(err)
		}

		dirFilenameExists := CheckIfDirFilenameExists(db, filename, dir)

		db.Close()

		if dirFilenameExists {
			fmt.Print(".")
			continue
		}

		// Skips extensions with json

		ext := strings.ToLower(filepath.Ext(path))

		if ext == ".json" {
			continue
		}

		// Checks is path is s directory
		// Skip to next path if it is a direcotry

		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			// This throws an error is the source path doesn't exist
			log.Fatal("File does not exist.")
		}

		if info.IsDir() {
			continue
		}

		// Skip files that includes the suffix -edited in its filename
		// Handles edited files and duplicates from google photos
		// Sha won't catch thes files as different

		if strings.Contains(path, "-edited") {
			continue
		}

		// Open file, detect content type and read exif

		f, err := os.Open(path)
		if err != nil {
			log.Printf("An error ocurred while trying to open: %s\n", path)
			log.Println(err)
		}

		content, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Println(err)
		}

		f.Close()

		contentType := http.DetectContentType(content)
		exif.RegisterParsers(mknote.All...)
		exifData, err := exif.Decode(bytes.NewReader(content))

		// Handle movie files and set date taken timestamp

		if err != nil {
			if contentType == "video/avi" {

				tm, _ = time.Parse("2006-01-02", "1971-08-11")

			} else if contentType == "application/octet-stream" && extension == ".mov" {

				tm, _ = time.Parse("2006-01-02", "1971-01-19")

			} else {

				fmt.Print("x")
				continue

			}
		} else {

			tm, _ = exifData.DateTime()

		}

		// Generate sha and filename

		sha := hashContent(content)

		currentPath, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}

		// camMake, _ := x.Get(exif.Make)
		// fmt.Println(camMake.StringVal())

		// camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
		// fmt.Println(camModel.StringVal())
		// tm, _ := exifData.DateTime()
		newPath := buildContentPath(tm, currentPath)
		newFilename := generateFileName(filename, sha)
		newPhotoPath := filepath.Join(newPath, newFilename)

		photo := Photo{
			ShaHash:    sha,
			SourcePath: path,
			Path:       newPhotoPath,
			DateTaken:  tm,
		}

		// Check if photo already exists in database

		db, err = openDatabase()
		if err != nil {
			panic(err)
		}

		photoExists := CheckIfPhotoExists(db, photo)

		db.Close()

		// Add photo to repo

		if photoExists {
			fmt.Println("Photo already exists, skipping copy")

			db, err := openDatabase()
			if err != nil {
				panic(err)
			}

			sourcePathExists := CheckShaAndSourceRepo(db, photo)

			db.Close()

			if sourcePathExists {
				log.Println("Source path already exists in database, skipping...")
				continue
			}

			log.Println("Logging new source path")

			photo.Status = "skipped"
			photo.Path = ""

			insertPhotoIntoDb(photo)

			continue
		}

		// Create new directory

		err = os.MkdirAll(newPath, 0755)
		if err != nil {
			panic(err)
		}

		// Write content to repo

		err = ioutil.WriteFile(newPhotoPath, content, 0755)
		if err != nil {
			log.Fatal(err)
		}

		// Generate thumbnail
		if contentType == "video/avi" {

			// TODO: Add thumbnail generator for avi

			log.Printf("No thumbnail generated for  %s", path)

		} else if contentType == "application/octet-stream" && extension == ".mov" {

			// TODO: Add thumbnail generator for mov

			log.Printf("No thumbnail generated for  %s", path)

		} else {

			thumbFilename := fmt.Sprintf("%s/.loupebox/cache/%s.jpeg", currentPath, sha)
			cmd := exec.Command("darktable-cli", path, thumbFilename, "--height", "300")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatalf("cmd.Run() failed with %s\n", err)
			}

		}

		log.Printf("Copied %s to %s\n", path, newPhotoPath)

		// Add to database

		insertPhotoIntoDb(photo)
	}
}

func dry(filenames []string) {
	for _, p := range filenames {
		filename := filepath.Base(p)
		dir := filepath.Base(filepath.Dir(p))

		// Check if path already exists in the database
		db, err := openDatabase()
		if err != nil {
			panic(err)
		}

		dirFilenameExists := CheckIfDirFilenameExists(db, filename, dir)

		db.Close()

		if dirFilenameExists {
			fmt.Print(".")
			continue
		}

		ext := strings.ToLower(filepath.Ext(p))

		if ext == ".json" {
			continue
		}

		info, err := os.Stat(p)
		if os.IsNotExist(err) {
			log.Fatal("File does not exist.")
		}

		if info.IsDir() {
			continue
		}

		if strings.Contains(p, "-edited") {
			continue
		}

		log.Printf("New: %s %s\n", dir, filename)
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

func CheckIfPathExists(db *sql.DB, path string) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos WHERE source_path = ?);`

	var exists int

	err := db.QueryRow(sql, path).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if exists == 1 {
		return true
	}

	return false
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

func CheckIfDirFilenameExists(db *sql.DB, filename string, dir string) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos WHERE dir = ? AND source_filename = ?);`

	var exists int

	err := db.QueryRow(sql, dir, filename).Scan(&exists)
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
	INSERT INTO photos(
		inserted_at,
		sha_hash,
		source_path,
		path,
		dir,
		source_filename,
		date_taken,
		status
	) values(CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	ogFilename := path.Base(photo.SourcePath)
	ogDir := path.Base(path.Dir(photo.SourcePath))

	_, err2 := stmt.Exec(photo.ShaHash, photo.SourcePath, photo.Path, ogDir, ogFilename, photo.DateTaken, photo.Status)
	if err2 != nil {
		return err2
	}

	log.Println("Successfully inserted photo record")

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

func insertPhotoIntoDb(photo Photo) {
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

func InsertRepo(db *sql.DB, path string) (sql.Result, error) {
	s := `
	INSERT INTO repos(
		inserted_at,
		updated_at,
		path,
		status
	) values(CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, "started")
	`
	stmt, err := db.Prepare(s)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var result sql.Result
	result, err = stmt.Exec(path)
	if err != nil {
		return nil, err
	}

	log.Println("Successfully inserted repo")

	return result, nil
}

func CheckIfRepoExists(db *sql.DB, path string) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM repos WHERE path = ?);`

	var exists int

	err := db.QueryRow(sql, path).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if exists == 1 {
		return true
	}

	return false
}

func UpdateRepoWithCompleteStatus(db *sql.DB, id int64) {
	s := `UPDATE repos 
		SET status = ?, 
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	stmt, err := db.Prepare(s)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec("completed", id)
	if err != nil {
		log.Fatal(err)
	}

}
