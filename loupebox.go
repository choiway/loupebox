package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/choiway/loupebox/cmd"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

func main() {
	cmd.Execute()

	// fmt.Println("hello, from lightbox")

	// filenames, err := walkdirectory("/mnt/f/test_photos")
	// if err != nil {
	// 	log.Fatalln("error reading path")
	// }

	// addfiles(filenames)
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

		// Optionally register camera makenote data parsing - currently Nikon and
		// Canon are supported.
		exif.RegisterParsers(mknote.All...)
		x, err := exif.Decode(f)

		if err != nil {
			log.Print("EXIST decode error")
			log.Println(err)
			fmt.Print("\n\n")
		} else {
			camMake, _ := x.Get(exif.Make)
			fmt.Println(camMake.StringVal())

			camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
			fmt.Println(camModel.StringVal())

			focal, _ := x.Get(exif.FocalLength)
			numer, denom, _ := focal.Rat2(0) // retrieve first (only) rat. value
			fmt.Printf("%v/%v\n", numer, denom)

			// Two convenience functions exist for date/time taken and GPS coords:
			tm, _ := x.DateTime()
			fmt.Println("Taken: ", tm)

			lat, long, _ := x.LatLong()
			fmt.Println("lat, long: ", lat, ", ", long)

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("SHA Hash: %x\n", h.Sum(nil))
			fmt.Print("\n\n")
		}

	}
}
