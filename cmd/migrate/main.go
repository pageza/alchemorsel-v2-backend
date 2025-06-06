package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	migrationsDir := "migrations"
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("failed to read migrations directory: %v", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	// Sort files to apply in order
	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		path := filepath.Join(migrationsDir, file)
		fmt.Printf("Applying migration: %s\n", path)
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("failed to read migration %s: %v", file, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			log.Fatalf("failed to apply migration %s: %v", file, err)
		}
	}

	fmt.Println("All migrations applied successfully.")
}
