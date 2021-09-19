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
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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

		// Optionally register camera makenote data parsing - currently Nikon and
		// Canon are supported.
		exif.RegisterParsers(mknote.All...)
		x, err := exif.Decode(f)

		if err != nil {
			log.Print("EXIST decode error")
			log.Println(err)
			fmt.Print("\n\n")
		} else {
			filename := filepath.Base(path)
			fmt.Println(filename)

			camMake, _ := x.Get(exif.Make)
			fmt.Println(camMake.StringVal())

			camModel, _ := x.Get(exif.Model) // normally, don't ignore errors!
			fmt.Println(camModel.StringVal())

			// focal, _ := x.Get(exif.FocalLength)
			// numer, denom, _ := focal.Rat2(0) // retrieve first (only) rat. value
			// fmt.Printf("%v/%v\n", numer, denom)

			// Two convenience functions exist for date/time taken and GPS coords:
			// tm, _ := x.DateTime()
			// fmt.Println("Taken: ", tm)

			// lat, long, _ := x.LatLong()
			// fmt.Println("lat, long: ", lat, ", ", long)

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("SHA Hash: %x\n", h.Sum(nil))
			fmt.Print("\n\n")
		}

	}
}
