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
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	"github.com/spf13/cobra"
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
		fmt.Println("init called")

		currentpath, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		fmt.Println(currentpath)

		// Remove existing .loupebox directory

		err = os.RemoveAll(".loupebox")
		if err != nil {
			log.Fatal(err)
		}

		// Create new loupebox directory

		err = os.Mkdir(".loupebox", 0755)
		if err != nil {
			log.Fatal(err)
		}

		// initilaize loupebox database

		dbPath := ".loupebox/loupebox.db"

		initializeDatabase(dbPath)

		// Create photos table

		sqliteDatabase, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			panic(err)
		}
		defer sqliteDatabase.Close()

		createPhotosTable(sqliteDatabase)

		// Create repos table

		sqliteDatabase, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			panic(err)
		}
		defer sqliteDatabase.Close()

		createReposTable(sqliteDatabase)
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

func initializeDatabase(dbPath string) {
	os.Remove(dbPath)

	log.Println("Creating loupebox.db...")

	file, err := os.Create(dbPath) // Create SQLite file
	if err != nil {
		log.Fatal(err.Error())
	}

	file.Close()

	log.Println("loupebox.db created")
}

func createPhotosTable(db *sql.DB) {

	sql := `CREATE TABLE photos (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"inserted_at" DATETIME,
		"updated_at" DATETIME,
		"sha_hash" TEXT,
		"source_path" TEXT,
		"path" TEXT,
		"dir" TEXT,
		"source_filename" TEXT,
		"date_taken" TEXT,
		"status" TEXT
	  );`

	log.Println("Creating photos table...")

	statement, err := db.Prepare(sql)
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec()

	log.Println("photos table created")
}

// createReposTable creates a table to keep track of the source repos added into the loupebox repo
func createReposTable(db *sql.DB) {

	sql := `CREATE TABLE repos (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"inserted_at" DATETIME,
		"updated_at" DATETIME,
		"path" TEXT,
		"status" TEXT
	  );`

	log.Println("Creating repos table...")

	statement, err := db.Prepare(sql)
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec()

	log.Println("Created repos table")
}
