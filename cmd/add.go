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
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [path/to/source]",
	Short: "Add photos to your repo",
	Long: `Add photos to an existing loupebox repo. Run the following command from a directory that 
was already initialized:

loupebox add /media/external/keepsakes

You can also pass the --dryrun flag which will check if the photos in your source directory already 
exist in the loupebox repo without copying. This is a quick way to check if there are new photos in
in the source directory.
`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Print("Please enter the full path to the source directory")
			return
		}

		dryrun, _ := cmd.Flags().GetBool("dryrun")

		// Get path argument
		// Will throw an error if it iesn't a valid path but should
		path := args[0]

		filenames, err := walkdirectory(path)
		if err != nil {
			log.Fatalln("There was an error reading the path. It may not exist or you may have entered it incorrectly. Please check and try again.")
		}

		if dryrun {

			log.Println("Starting dry run")

			dry(filenames)

		} else {

			// Add current repo to repo database
			// This is used to track recent adds
			yfile, err := ioutil.ReadFile(".loupebox/config.yaml")

			if err != nil {

				log.Fatal("Can't find the config file. Make sure you're in the correct directory or initialize the repo before trying to add photos.")
			}

			var config Config
			err = yaml.Unmarshal(yfile, &config)

			if err != nil {

				log.Fatal(err)
			}

			// Add photos

			log.Printf("Adding photos from %s\n", path)

			addfilesWithHash(filenames)

			fmt.Print("\n")
			log.Printf("Finished adding %s", path)
		}

	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("dir", "", "Path to the source directory")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	addCmd.Flags().BoolP("dryrun", "r", false, "Will check if photos already exist in the repo without copying.")
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

func addfilesWithHash(filenames []string) {

	photoMap := PhotoMap{
		photos: map[string]Photo{},
	}

	// Hydrate photomap from existing photos cache

	csvFile, err := os.Open(".loupebox/cache/photos")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)

	rows, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	for _, row := range rows {

		t, _ := time.Parse("2006-01-02", row[3])

		p := Photo{
			ShaHash:    row[0],
			SourcePath: row[1],
			Path:       row[2],
			DateTaken:  t,
		}

		photoMap.Insert(p)
	}

	// Start processing photos
	// Sets the number os workers to the number of cpus.
	// TODO: Consider whether this should be passed in as a flag
	throttle := make(chan int, runtime.NumCPU())
	var wg sync.WaitGroup

	for _, f := range filenames {
		throttle <- 1 // whatever number
		wg.Add(1)
		go addPhotosUsingMap(f, &wg, throttle, &photoMap)
	}

	wg.Wait()

	// Persists photomap to disk

	var pps [][]string
	for _, p := range photoMap.photos {
		t := p.DateTaken.Format("2006-02-01")
		pps = append(pps, []string{p.ShaHash, p.SourcePath, p.Path, t})
	}

	f, e := os.Create(".loupebox/cache/photos")
	if e != nil {
		fmt.Println(e)
	}

	writer := csv.NewWriter(f)

	e = writer.WriteAll(pps)
	if e != nil {
		fmt.Println(e)
	}

}

func dry(filenames []string) {
	photos := make(map[string]Photo)

	// Hydrate photos hash map from photos cache

	csvFile, err := os.Open(".loupebox/cache/photos")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)

	rows, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	for _, row := range rows {

		t, _ := time.Parse("2006-01-02", row[3])

		p := Photo{
			ShaHash:    row[0],
			SourcePath: row[1],
			Path:       row[2],
			DateTaken:  t,
		}

		truncatedPath := TruncatePath(p.SourcePath)

		photos[truncatedPath] = p
	}

	for _, p := range filenames {

		filename := filepath.Base(p)
		dir := filepath.Base(filepath.Dir(p))

		// Check is directory and path have already been imported
		// TODO: Currently this doesn't catch a file that may have been added from a different
		// source path since we don't keep track of the source of duplicates and only add the first
		// photo entered into the hash map.
		// Consider creating another map with the truncated directory as the index
		// Have think through whethere it's worth the additional space and complexity
		_, exists := photos[TruncatePath(p)]
		if exists {
			continue
		}

		// Exclude extensions
		ext := strings.ToLower(filepath.Ext(p))

		if ext == ".json" {
			continue
		}

		if ext == ".ini" {
			continue
		}

		if ext == ".db" {
			continue
		}

		if ext == ".url" {
			continue
		}

		if ext == ".rss" {
			continue
		}

		if ext == ".ofa" {
			continue
		}

		// Exclude filenames

		if filename == ".DS_Store" {
			continue
		}

		if filename == ".BridgeSort" {
			continue
		}

		// Stat checks

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

// TruncatePath takes a full path and returns the most filename and one directory above
// For example, if you pass in /home/monkey/stash/banana/peel.txt this function will return
// /banana/peel.txt
func TruncatePath(path string) string {
	filename := filepath.Base(path)
	dir := filepath.Base(filepath.Dir(path))
	return filepath.Join(dir, filename)
}

func addPhotosUsingMap(p string, wg *sync.WaitGroup, throttle chan int, photoMap *PhotoMap) {
	defer wg.Done()

	var tm time.Time

	filename := filepath.Base(p)
	ext := strings.ToLower(filepath.Ext(p))

	// Exclude extensions

	if ext == ".json" {
		<-throttle
		return
	}

	if ext == ".ini" {
		<-throttle
		return
	}

	if ext == ".db" {
		<-throttle
		return
	}

	if ext == ".url" {
		<-throttle
		return
	}

	if ext == ".rss" {
		<-throttle
		return
	}

	if ext == ".ofa" {
		<-throttle
		return
	}

	// Exclude certain files

	if filename == ".DS_Store" {
		<-throttle
		return
	}

	if filename == ".BridgeSort" {
		<-throttle
		return
	}

	// Stat checks

	info, err := os.Stat(p)
	if os.IsNotExist(err) {
		log.Fatal("File does not exist.")
	}

	if info.IsDir() {
		<-throttle
		return
	}

	if strings.Contains(p, "-edited") {
		<-throttle
		return
	}

	// Open file, detect content type and read exif

	file, err := os.Open(p)
	if err != nil {
		log.Printf("An error ocurred while trying to open: %s\n", p)
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
	// We hard a code a datetaken date for the movie files since Exif doesn't read it
	// The date are hardcoded by the movie type. TODO: Figure out how to read date taken from movie files
	if err != nil {
		if contentType == "video/avi" {

			tm, _ = time.Parse("2006-01-02", "1971-08-11")

		} else if contentType == "application/octet-stream" && ext == ".mov" {

			tm, _ = time.Parse("2006-01-02", "1971-01-19")

		} else if contentType == "application/octet-stream" && ext == ".mp4" {

			tm, _ = time.Parse("2006-01-02", "1971-07-30")

		} else if contentType == "video/mp4" && ext == ".mp4" {

			tm, _ = time.Parse("2006-01-02", "1971-07-30")

		} else if contentType == "image/jpeg" {

			tm, _ = time.Parse("2006-01-02", "1971-09-19")

		} else if contentType == "image/png" {

			tm, _ = time.Parse("2006-01-02", "1971-09-17")

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

	// Create new path and filename
	newPath := buildContentPath(tm, currentPath)
	newFilename := generateFileName(filename, sha)
	newPhotoPath := filepath.Join(newPath, newFilename)

	photo := Photo{
		ShaHash:    sha,
		SourcePath: p,
		Path:       newPhotoPath,
		DateTaken:  tm,
	}

	_, exists := photoMap.Get(photo.ShaHash)
	if exists {
		fmt.Print("*")
		<-throttle
		return
	}

	// Create new path for photo is it doesn't already exist
	err = os.MkdirAll(newPath, 0755)
	if err != nil {
		panic(err)
	}

	// Write content to repo
	err = ioutil.WriteFile(newPhotoPath, content, 0755)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\n")
	log.Printf("Copied %s to %s\n", p, newPhotoPath)

	// Add photo to photoMap
	photoMap.Insert(photo)
	<-throttle

}

func hashContent(content []byte) string {
	h := sha256.New()

	_, err := io.Copy(h, bytes.NewReader(content))
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// buildContentPath generates a path from the time data
// The path is generated by the year, month, day where if the day is
// 2004-01-02, the path will be 2004/01/02 in the working directory
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

func generateFileName(filename string, sha string) string {

	ext := filepath.Ext(filename)
	n := strings.TrimSuffix(filename, ext)
	shortSha := sha[0:6]

	return fmt.Sprintf("%s_%s%s", n, shortSha, ext)
}
