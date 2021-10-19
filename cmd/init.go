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
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes loupebox for the current directory",
	Long:  `Initializes the curre directory by creating a looupebox directory and cache.`,
	Run: func(cmd *cobra.Command, args []string) {

		currentpath, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("Working directory: %s\n", currentpath)

		// Initialize loupebox directory

		_, err = os.Stat(".loupebox")

		if os.IsNotExist(err) {

			initializeLoupebox()

		} else {

			scanner := bufio.NewScanner(os.Stdin)

			fmt.Print(`Loupebox has already been initialized in this foler. Do you want to erase
the existing info and reinitialize Loupebox? [Y/n]:`)
			scanner.Scan()
			userInput := scanner.Text()

			if userInput == "Y" {

				fmt.Println("Reinitializing...")
				initializeLoupebox()

			} else {

				fmt.Println("Aborting initialization")

			}
		}

	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type Config struct {
	Repo Repo `yaml:"repo"`
}

type Repo struct {
	ID uuid.UUID `yaml:"id"`
}

func initializeLoupebox() {

	// Remove existing .loupebox directory

	err := os.RemoveAll(".loupebox")
	if err != nil {
		log.Fatal(err)
	}

	// Create new loupebox directory

	err = os.Mkdir(".loupebox", 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Create .cache folder for thumbnails

	err = os.Mkdir(".loupebox/cache", 0755)
	if err != nil {
		log.Fatal(err)
	}

	// generate id for repo and save config to yaml

	config := Config{
		Repo: Repo{
			ID: uuid.New(),
		},
	}

	data, err := yaml.Marshal(&config)

	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(".loupebox/config.yaml", data, 0755)

	if err != nil {
		log.Fatal(err)
	}

	// Touch empty file to photos
	// We write the added photos to this file

	err = ioutil.WriteFile(".loupebox/cache/photos", []byte(""), 0755)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Created config file")

}
