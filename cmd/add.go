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
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

			log.Println("Not working for now. Doing dry run")

			dry(filenames)

		} else {

			// Add current repo to repo database
			// This is used to track recent adds
			yfile, err := ioutil.ReadFile(".loupebox/config.yaml")

			if err != nil {

				log.Fatal(err)
			}

			var config Config
			err = yaml.Unmarshal(yfile, &config)

			if err != nil {

				log.Fatal(err)
			}

			// Log current import

			conn := DbConnect()

			lastInsertedId, err := InsertImport(conn, path, config.Repo.ID)
			if err != nil {
				log.Fatal(err)
			}

			conn.Close()

			// Add photos

			log.Printf("Adding photos from %s\n", path)

			addfiles(filenames, config.Repo.ID)

			// Update repo with completed status

			conn = DbConnect()

			UpdateImportWithCompleteStatus(conn, lastInsertedId)

			conn.Close()

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

// addfiles uses sync.WaitGroup to create multiple import jobs
func addfiles(filenames []string, repoID uuid.UUID) {

	throttle := make(chan int, 4)
	var wg sync.WaitGroup

	for _, f := range filenames {
		throttle <- 1 // whatever number
		wg.Add(1)
		go addPhoto(f, repoID, &wg, throttle)
	}

	wg.Wait()
}

func addPhoto(f string, repoID uuid.UUID, wg *sync.WaitGroup, throttle chan int) {
	defer wg.Done() // Need this here clean up once the wait group close

	var tm time.Time

	// Check if path already exists in the database

	conn := DbConnect()

	pathExists := CheckIfPathExists(conn, f)

	conn.Close()

	if pathExists {
		fmt.Print(".")
		<-throttle
		return
	}

	// Check if directory and file name exists
	// This checks for the intance when the source repo path might be different but the
	// local directory and photo name are the same as one that was already imported

	extension := strings.ToLower(filepath.Ext(f))
	filename := filepath.Base(f)
	dir := filepath.Base(filepath.Dir(f))

	conn = DbConnect()

	dirFilenameExists := CheckIfDirFilenameExists(conn, filename, dir)

	conn.Close()

	if dirFilenameExists {
		fmt.Print(".")
		<-throttle
		return
	}

	// Skips extensions with json

	ext := strings.ToLower(filepath.Ext(f))

	if ext == ".json" {
		<-throttle
		return
	}

	// Checks is path is s directory
	// Skip to next path if it is a direcotry

	info, err := os.Stat(f)
	if os.IsNotExist(err) {
		// This throws an error is the source path doesn't exist
		log.Fatal("File does not exist.")
	}

	if info.IsDir() {
		<-throttle
		return
	}

	// Skip files that includes the suffix -edited in its filename
	// Handles edited files and duplicates from google photos
	// Sha won't catch thes files as different

	if strings.Contains(f, "-edited") {
		<-throttle
		return
	}

	// Open file, detect content type and read exif

	file, err := os.Open(f)
	if err != nil {
		log.Printf("An error ocurred while trying to open: %s\n", f)
		log.Println(err)
	}

	content, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	file.Close()

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
			<-throttle
			return

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
		RepoID:     repoID,
		SourcePath: f,
		Path:       newPhotoPath,
		DateTaken:  tm,
	}

	// Check if photo already exists in database
	conn = DbConnect()

	photoExists := CheckIfPhotoExists(conn, photo)

	conn.Close()

	// Add photo to repo

	if photoExists {
		fmt.Println("Photo already exists, skipping copy")

		conn = DbConnect()

		sourcePathExists := CheckShaAndSourceRepo(conn, photo)

		conn.Close()

		if sourcePathExists {
			log.Println("Source path already exists in database, skipping...")
			<-throttle
			return
		}

		log.Println("Logging new source path")

		photo.Status = "skipped"
		photo.Path = ""

		insertPhotoIntoDb(photo)

		<-throttle
		return
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

	// if contentType == "video/avi" {

	// 	// TODO: Add thumbnail generator for avi

	// 	log.Printf("No thumbnail generated for  %s", path)

	// } else if contentType == "application/octet-stream" && extension == ".mov" {

	// 	// TODO: Add thumbnail generator for mov

	// 	log.Printf("No thumbnail generated for  %s", path)

	// } else if extension == ".thm" {

	// 	// .thm extension is an old sony ericsson phone extension

	// 	log.Printf("No thumbnail generated for %s", path)

	// } else {

	// 	thumbFilename := fmt.Sprintf("%s/.loupebox/cache/%s.jpeg", currentPath, sha)
	// 	cmd := exec.Command("darktable-cli", path, thumbFilename, "--height", "300")
	// 	cmd.Stdout = os.Stdout
	// 	cmd.Stderr = os.Stderr
	// 	err = cmd.Run()
	// 	if err != nil {
	// 		log.Fatalf("cmd.Run() failed with %s\n", err)
	// 	}

	// }

	log.Printf("Copied %s to %s\n", f, newPhotoPath)

	// Add to database

	insertPhotoIntoDb(photo)
	<-throttle
}

func dry(filenames []string) {
	for _, p := range filenames {
		filename := filepath.Base(p)
		dir := filepath.Base(filepath.Dir(p))

		// Check if path already exists in the database
		conn := DbConnect()

		dirFilenameExists := CheckIfDirFilenameExists(conn, filename, dir)

		conn.Close()

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

func CheckIfPathExists(conn *pgxpool.Pool, path string) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos WHERE source_path = $1);`

	var exists bool

	err := conn.QueryRow(context.Background(), sql, path).Scan(&exists)
	// fmt.Printf("row: %#v\n", row.Scan())
	if err != nil {
		log.Fatal(err)
	}

	return exists
}

func CheckIfPhotoExists(conn *pgxpool.Pool, photo Photo) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos where sha_hash = $1);`

	var exists bool

	err := conn.QueryRow(context.Background(), sql, photo.ShaHash).Scan(&exists)
	if err != nil {
		panic(err)
	}

	return exists
}

func CheckIfDirFilenameExists(conn *pgxpool.Pool, filename string, dir string) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos WHERE dir = $1 AND source_filename = $2);`

	var exists bool

	err := conn.QueryRow(context.Background(), sql, dir, filename).Scan(&exists)
	if err != nil {
		panic(err)
	}

	return exists
}

func CheckShaAndSourceRepo(conn *pgxpool.Pool, photo Photo) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM photos WHERE sha_hash = $1 AND source_path = $2);`

	var exists bool

	err := conn.QueryRow(context.Background(), sql, photo.ShaHash, photo.SourcePath).Scan(&exists)
	if err != nil {
		panic(err)
	}

	return exists
}

func InsertPhoto(conn *pgxpool.Pool, p Photo) error {
	ogFilename := path.Base(p.SourcePath)
	ogDir := path.Base(path.Dir(p.SourcePath))

	sql := `
	INSERT INTO photos(
		inserted_at,
		updated_at, 
		repo_id, 
		sha_hash,
		source_path,
		path,
		dir,
		source_filename,
		date_taken,
		status
	) values(now(), now(), $1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := conn.Exec(context.Background(), sql, p.RepoID, p.ShaHash, p.SourcePath, p.Path, ogDir, ogFilename, p.DateTaken, p.Status)
	if err != nil {
		return err
	}

	log.Println("Successfully inserted photo record")

	return nil
}

func generateFileName(filename string, sha string) string {

	ext := filepath.Ext(filename)
	n := strings.TrimSuffix(filename, ext)
	shortSha := sha[0:6]

	return fmt.Sprintf("%s_%s%s", n, shortSha, ext)
}

func insertPhotoIntoDb(photo Photo) {
	pool := DbConnect()

	err := InsertPhoto(pool, photo)
	if err != nil {
		panic(err)
	}

	pool.Close()
}

func InsertImport(conn *pgxpool.Pool, path string, loupeboxID uuid.UUID) (int64, error) {
	s := `
	INSERT INTO imports(
		inserted_at,
		updated_at,
		repo_id, 
		path,
		status
	) values(now(), now(), $1, $2, 'started')
	RETURNING id
	`
	var lastInsertID int64

	err := conn.QueryRow(context.Background(), s, loupeboxID, path).Scan(&lastInsertID)
	if err != nil {
		return 0, err
	}

	log.Println("Successfully inserted import event")

	return lastInsertID, nil
}

func CheckIfImportExists(conn *pgxpool.Pool, path string, loupeboxID uuid.UUID) bool {
	sql := `SELECT EXISTS (SELECT 1 FROM imports WHERE path = $1);`

	var exists int

	err := conn.QueryRow(context.Background(), sql, path).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if exists == 1 {
		return true
	}

	return false
}

func UpdateImportWithCompleteStatus(conn *pgxpool.Pool, id int64) {
	s := `UPDATE imports 
		SET status = $1, 
			updated_at = now()
		WHERE id = $2`

	_, err := conn.Exec(context.Background(), s, "completed", id)
	if err != nil {
		log.Fatal(err)
	}
}
