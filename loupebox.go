package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

func main() {

	fmt.Println("hello, from lightbox")

	filenames, err := walkdirectory("/mnt/f/test_photos")
	// filenames, err := walkdirectory("/mnt/e/Pictures")
	if err != nil {
		log.Fatalln("error reading path")
	}

	addfiles(filenames)
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

		// TODO: Skip directories

		f, err := os.Open(path)
		if err != nil {
			log.Println("Error")
			log.Println(err)
		}

		byteArray, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Println(err)
		}

		h := sha256.New()
		_, err = io.Copy(h, bytes.NewReader(byteArray))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("SHA Hash: %x\n", h.Sum(nil))

		// Optionally register camera makenote data parsing - currently Nikon and
		// Canon are supported.
		exif.RegisterParsers(mknote.All...)
		x, err := exif.Decode(bytes.NewReader(byteArray))
		// info, err := os.Stat(path)

		if err != nil {
			log.Print("EXIST decode error")
			log.Println(err)
			fmt.Print("\n\n")
		} else {
			json, err := x.MarshalJSON()
			if err != nil {
				log.Fatal("couldn't unmarshel")
			}
			fmt.Print(string(json))

			// camMake, _ := x.Get(exif.Make)
			// fmt.Println(camMake.StringVal())

			// camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
			// fmt.Println(camModel.StringVal())

			// focal, _ := x.Get(exif.FocalLength)
			// numer, denom, _ := focal.Rat2(0) // retrieve first (only) rat. value
			// fmt.Printf("%v/%v\n", numer, denom)

			// // // Two convenience functions exist for date/time taken and GPS coords:
			// tm, _ := x.DateTime()
			// fmt.Println("Taken: ", tm)

			// lat, long, _ := x.LatLong()
			// fmt.Println("lat, long: ", lat, ", ", long)

			fmt.Print("\n\n")
		}

		f.Close()

	}
}
