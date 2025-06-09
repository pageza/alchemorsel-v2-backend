package testhelpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDatabase creates a test database instance using a containerized PostgreSQL with pgvector.
func SetupTestDatabase(t *testing.T) *gorm.DB {
	// Check if we're in CI environment
	if os.Getenv("CI") == "true" {
		// Debug logging
		t.Logf("CI environment detected")
		t.Logf("[DEBUG] DB_HOST: %s", os.Getenv("DB_HOST"))
		t.Logf("[DEBUG] DB_PORT: %s", os.Getenv("DB_PORT"))
		t.Logf("[DEBUG] DB_USER: %s", os.Getenv("DB_USER"))
		t.Logf("[DEBUG] DB_PASSWORD: %s", os.Getenv("DB_PASSWORD"))
		t.Logf("[DEBUG] DB_NAME: %s", os.Getenv("DB_NAME"))
		t.Logf("[DEBUG] DB_SSL_MODE: %s", os.Getenv("DB_SSL_MODE"))

		// Validate required environment variables
		requiredEnvVars := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
		for _, envVar := range requiredEnvVars {
			if os.Getenv(envVar) == "" {
				t.Fatalf("required environment variable %s is not set", envVar)
			}
		}

		// Use environment variables for database connection
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_SSL_MODE"),
		)
		t.Logf("DSN: %s", dsn)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}

		// Install pgvector extension
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector;").Error; err != nil {
			t.Fatalf("failed to install pgvector extension: %v", err)
		}

		// Create dietary preference type enum
		if err := db.Exec(`
			DO $$ BEGIN
				CREATE TYPE dietary_preference_type AS ENUM (
					'vegetarian',
					'vegan',
					'pescatarian',
					'gluten-free',
					'dairy-free',
					'nut-free',
					'soy-free',
					'egg-free',
					'shellfish-free',
					'custom'
				);
			EXCEPTION
				WHEN duplicate_object THEN null;
			END $$;
		`).Error; err != nil {
			t.Fatalf("failed to create dietary preference type: %v", err)
		}

		// Auto-migrate the schema
		err = db.AutoMigrate(
			&models.User{},
			&models.UserProfile{},
			&models.Recipe{},
			&models.RecipeFavorite{},
			&models.DietaryPreference{},
			&models.Allergen{},
		)
		if err != nil {
			t.Fatalf("failed to migrate test database: %v", err)
		}

		return db
	}

	// Local development environment - use test container
	ctx := context.Background()

	// Create PostgreSQL container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections"),
			).WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Get container host and port
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, mappedPort.Port())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Install pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector;").Error; err != nil {
		t.Fatalf("failed to install pgvector extension: %v", err)
	}

	// Create dietary preference type enum
	if err := db.Exec(`
		DO $$ BEGIN
			CREATE TYPE dietary_preference_type AS ENUM (
				'vegetarian',
				'vegan',
				'pescatarian',
				'gluten-free',
				'dairy-free',
				'nut-free',
				'soy-free',
				'egg-free',
				'shellfish-free',
				'custom'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
	`).Error; err != nil {
		t.Fatalf("failed to create dietary preference type: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Recipe{},
		&models.RecipeFavorite{},
		&models.DietaryPreference{},
		&models.Allergen{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	return db
}
