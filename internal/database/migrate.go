package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// RunMigrations executes all SQL migration files in the migrations directory
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	// Get all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name to ensure correct order
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Create migrations table if it doesn't exist
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Execute each migration file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Check if migration has already been applied
		var count int64
		if err := db.Table("migrations").Where("name = ?", file.Name()).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}
		if count > 0 {
			log.Printf("Skipping migration %s (already applied)", file.Name())
			continue
		}

		// Read migration file
		content, err := os.ReadFile(filepath.Join(migrationsDir, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		// Execute migration
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
		}

		// Record migration
		if err := db.Exec("INSERT INTO migrations (name) VALUES (?)", file.Name()).Error; err != nil {
			return fmt.Errorf("failed to record migration %s: %w", file.Name(), err)
		}

		log.Printf("Applied migration %s", file.Name())
	}

	return nil
}
