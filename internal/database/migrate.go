package database

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"gorm.io/gorm"
)

// RunMigrations executes all SQL migration files in the migrations directory
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	if db.Dialector.Name() == "sqlite" {
		log.Printf("Using GORM auto-migration for SQLite")
		return db.AutoMigrate(
			&models.User{},
			&models.UserProfile{},
			&models.DietaryPreference{},
			&models.Allergen{},
			&models.UserAppliance{},
		)
	}

	// Get all migration files
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name to ensure correct order
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Create migrations table if it doesn't exist (PostgreSQL)
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
		content, err := ioutil.ReadFile(filepath.Join(migrationsDir, file.Name()))
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
