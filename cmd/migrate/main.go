package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	// Parse command line flags
	rollback := flag.Bool("rollback", false, "Rollback the last migration")
	flag.Parse()

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

	if *rollback {
		// Get the last applied migration
		var lastMigration struct {
			Version string
			Name    string
		}
		err := db.QueryRow(`
			SELECT version, name 
			FROM schema_migrations 
			ORDER BY applied_at DESC 
			LIMIT 1
		`).Scan(&lastMigration.Version, &lastMigration.Name)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Fatal("No migrations to rollback")
			}
			log.Fatalf("failed to get last migration: %v", err)
		}

		// Find the rollback file
		rollbackFile := fmt.Sprintf("%s_rollback.sql", strings.TrimSuffix(lastMigration.Name, ".sql"))
		rollbackPath := filepath.Join(migrationsDir, rollbackFile)

		// Check if rollback file exists
		if _, err := os.Stat(rollbackPath); os.IsNotExist(err) {
			log.Fatalf("rollback file not found: %s", rollbackPath)
		}

		// Read and execute rollback
		content, err := ioutil.ReadFile(rollbackPath)
		if err != nil {
			log.Fatalf("failed to read rollback file: %v", err)
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("failed to start transaction: %v", err)
		}

		// Execute rollback
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			log.Fatalf("failed to execute rollback: %v", err)
		}

		// Remove migration record
		if _, err := tx.Exec("SELECT remove_migration($1)", lastMigration.Version); err != nil {
			tx.Rollback()
			log.Fatalf("failed to remove migration record: %v", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Fatalf("failed to commit rollback: %v", err)
		}

		fmt.Printf("Successfully rolled back migration: %s\n", lastMigration.Name)
		return
	}

	// Apply migrations
	for _, file := range migrationFiles {
		// Extract version from filename (assuming format: VERSION_NAME.sql)
		version := strings.Split(file, "_")[0]

		// Check if migration has already been applied
		var applied bool
		err := db.QueryRow("SELECT migration_applied($1)", version).Scan(&applied)
		if err != nil {
			log.Fatalf("failed to check migration status: %v", err)
		}

		if applied {
			fmt.Printf("Migration already applied: %s\n", file)
			continue
		}

		path := filepath.Join(migrationsDir, file)
		fmt.Printf("Applying migration: %s\n", path)

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("failed to start transaction: %v", err)
		}

		// Read and execute migration
		content, err := ioutil.ReadFile(path)
		if err != nil {
			tx.Rollback()
			log.Fatalf("failed to read migration %s: %v", file, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			log.Fatalf("failed to apply migration %s: %v", file, err)
		}

		// Record migration
		if _, err := tx.Exec("SELECT record_migration($1, $2)", version, file); err != nil {
			tx.Rollback()
			log.Fatalf("failed to record migration: %v", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Fatalf("failed to commit migration: %v", err)
		}

		fmt.Printf("Successfully applied migration: %s\n", file)
	}

	fmt.Println("All migrations applied successfully.")
}
