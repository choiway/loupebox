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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes loupebox for the current directory",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		currentpath, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("Working directory: %s\n", currentpath)

		// Sqlite init
		_, err = os.Stat(".loupebox")
		if os.IsNotExist(err) {

			initializeLoupebox()
			createTables()
		} else {

			scanner := bufio.NewScanner(os.Stdin)

			fmt.Print(`Loupebox has already been initialized in this foler. Do you want to erase
		the existing info and reinitialize Loupebox? [Y/n]:`)
			scanner.Scan()
			text := scanner.Text()

			if text == "Y" {
				fmt.Println("Reinitializing...")
				initializeLoupebox()
				createTables()
			} else {
				fmt.Println("Aborting initialization")
			}
		}

		// Sqlite init
		// 		_, err = os.Stat(".loupebox")
		// 		if os.IsNotExist(err) {
		// 			initializeLoupebox()
		// 		} else {
		// 			scanner := bufio.NewScanner(os.Stdin)

		// 			fmt.Print(`Loupebox has already been initialized in this foler. Do you want to erase
		// the existing info and reinitialize Loupebox? [Y/n]:`)
		// 			scanner.Scan()
		// 			text := scanner.Text()

		// 			if text == "Y" {
		// 				fmt.Println("Reinitializing...")
		// 				initializeLoupebox()
		// 			} else {
		// 				fmt.Println("Aborting initialization")
		// 			}
		// 		}

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

	// TODO: should add check if .loupebox already exists
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

	log.Println("Create config file")

}

func createTables() {
	pool := DbConnect()

	defer pool.Close()
	// TODO: Add check if database exists

	// TODO: Add check if tables already exist and handle accordingly

	pqCreatePhotosTable(pool)
	pgCreateReposTable(pool)
}

// func initializeDatabase(dbPath string) {
// 	os.Remove(dbPath)

// 	log.Println("Creating loupebox.db...")

// 	file, err := os.Create(dbPath) // Create SQLite file
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// }

// func initializeLoupebox() {
// 	// Remove existing .loupebox directory

// 	// TODO: should add check if .loupebox already exists
// 	err := os.RemoveAll(".loupebox")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Create new loupebox directory

// 	err = os.Mkdir(".loupebox", 0755)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// initilaize loupebox database

// 	dbPath := ".loupebox/loupebox.db"

// 	initializeDatabase(dbPath)

// 	// Create .cache folder for thumbnails

// 	err = os.Mkdir(".loupebox/cache", 0755)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Create photos table

// 	sqliteDatabase, err := sql.Open("sqlite3", dbPath)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer sqliteDatabase.Close()

// 	createPhotosTable(sqliteDatabase)

// 	// Create repos table

// 	sqliteDatabase, err = sql.Open("sqlite3", dbPath)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer sqliteDatabase.Close()

// 	createReposTable(sqliteDatabase)
// }

// func initializeDatabase(dbPath string) {
// 	os.Remove(dbPath)

// 	log.Println("Creating loupebox.db...")

// 	file, err := os.Create(dbPath) // Create SQLite file
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}

// 	file.Close()

// 	log.Println("loupebox.db created")
// }

// func createPhotosTable(db *sql.DB) {

// 	sql := `CREATE TABLE photos (
// 		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
// 		"inserted_at" DATETIME,
// 		"updated_at" DATETIME,
// 		"sha_hash" TEXT,
// 		"source_path" TEXT,
// 		"path" TEXT,
// 		"dir" TEXT,
// 		"source_filename" TEXT,
// 		"date_taken" TEXT,
// 		"status" TEXT
// 	  );`

// 	log.Println("Creating photos table...")

// 	statement, err := db.Prepare(sql)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	statement.Exec()

// 	log.Println("photos table created")
// }

func pqCreatePhotosTable(conn *pgxpool.Pool) {
	sql := `CREATE TABLE photos (
		id SERIAL PRIMARY KEY,		
		inserted_at TIMESTAMP,
		updated_at TIMESTAMP,
		repo_id UUID,
		sha_hash TEXT,
		source_path TEXT,
		path TEXT,
		dir TEXT,
		source_filename TEXT,
		date_taken TEXT,
		status TEXT
	  );`

	log.Println("Creating photos table...")

	_, err := conn.Exec(context.Background(), sql)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Println("photos table created")
}

// createReposTable creates a table to keep track of the source repos added into the loupebox repo
// func createReposTable(db *sql.DB) {

// 	sql := `CREATE TABLE repos (
// 		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
// 		"inserted_at" DATETIME,
// 		"updated_at" DATETIME,
// 		"path" TEXT,
// 		"status" TEXT
// 	  );`

// 	log.Println("Creating repos table...")

// 	statement, err := db.Prepare(sql)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	statement.Exec()

// 	log.Println("Created repos table")
// }

func pgCreateReposTable(conn *pgxpool.Pool) {

	sql := `CREATE TABLE imports (
		id SERIAL PRIMARY KEY,		
		inserted_at TIMESTAMP,
		updated_at TIMESTAMP,
		path TEXT,
		status TEXT
	  );`

	log.Println("Creating imports table...")

	_, err := conn.Exec(context.Background(), sql)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Println("Created import table")
}
