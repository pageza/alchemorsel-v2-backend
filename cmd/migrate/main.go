package main

import (
	"database/sql"
	"flag"
	"fmt"
	// LINT-FIX-2025: Removed unused io import after replacing ioutil usage with os package
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
	// LINT-FIX-2025: Handle database close error properly
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Warning: failed to close database connection: %v", err)
		}
	}()

	migrationsDir := "migrations"
	// LINT-FIX-2025: Use os.ReadDir instead of deprecated ioutil.ReadDir
	files, err := os.ReadDir(migrationsDir)
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

		// LINT-FIX-2025: Use os.ReadFile instead of deprecated ioutil.ReadFile
		// Read and execute rollback
		content, err := os.ReadFile(rollbackPath)
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
			// LINT-FIX-2025: Handle rollback error properly with error checking
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("failed to rollback transaction: %v", rollbackErr)
			}
			log.Fatalf("failed to execute rollback: %v", err)
		}

		// Remove migration record
		if _, err := tx.Exec("SELECT remove_migration($1)", lastMigration.Version); err != nil {
			// LINT-FIX-2025: Handle rollback error properly with error checking
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("failed to rollback transaction: %v", rollbackErr)
			}
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
		// Handle the bootstrap case where migration functions don't exist yet
		var applied bool
		err := db.QueryRow("SELECT migration_applied($1)", version).Scan(&applied)
		if err != nil {
			// If the function doesn't exist, this is likely the first migration
			// Check if this is the migration tracking setup migration
			if strings.Contains(err.Error(), "function migration_applied") && version == "000000" {
				fmt.Printf("Setting up migration tracking system: %s\n", file)
				applied = false // Force execution of the migration tracking setup
			} else {
				log.Fatalf("failed to check migration status: %v", err)
			}
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

		// LINT-FIX-2025: Use os.ReadFile instead of deprecated ioutil.ReadFile
		// Read and execute migration
		content, err := os.ReadFile(path)
		if err != nil {
			// LINT-FIX-2025: Handle rollback error properly with error checking
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("failed to rollback transaction: %v", rollbackErr)
			}
			log.Fatalf("failed to read migration %s: %v", file, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			// LINT-FIX-2025: Handle rollback error properly with error checking
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("failed to rollback transaction: %v", rollbackErr)
			}
			log.Fatalf("failed to apply migration %s: %v", file, err)
		}

		// Record migration (only if the migration didn't record itself)
		// E2E-FIX-2025-B: Handle migrations that record themselves to prevent duplicate key errors
		if _, err := tx.Exec("SELECT record_migration($1, $2)", version, file); err != nil {
			// Check if this is a duplicate key error (migration already recorded itself)
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				fmt.Printf("Migration %s recorded itself successfully\n", file)
			} else {
				// LINT-FIX-2025: Handle rollback error properly with error checking
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Printf("failed to rollback transaction: %v", rollbackErr)
				}
				log.Fatalf("failed to record migration: %v", err)
			}
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Fatalf("failed to commit migration: %v", err)
		}

		fmt.Printf("Successfully applied migration: %s\n", file)
	}

	fmt.Println("All migrations applied successfully.")
}
