package testhelpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/lib/pq"
	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDatabase creates a test database instance using a containerized PostgreSQL with pgvector.
func SetupTestDatabase(t *testing.T) *gorm.DB {
	ctx := context.Background()

	// Set CI environment and required secrets
	os.Setenv("CI", "true")
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_NAME", "alchemorsel")
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("TEST_DB_PASSWORD", "postpass")
	os.Setenv("TEST_JWT_SECRET", "test-jwt-secret")
	os.Setenv("TEST_REDIS_PASSWORD", "test-redis-pass")
	os.Setenv("TEST_REDIS_URL", "redis://localhost:6379")

	defer func() {
		os.Unsetenv("CI")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSL_MODE")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("TEST_DB_PASSWORD")
		os.Unsetenv("TEST_JWT_SECRET")
		os.Unsetenv("TEST_REDIS_PASSWORD")
		os.Unsetenv("TEST_REDIS_URL")
	}()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Debug logging
	t.Logf("[DEBUG] DB_HOST: %s", cfg.DBHost)
	t.Logf("[DEBUG] DB_PORT: %s", cfg.DBPort)
	t.Logf("[DEBUG] DB_USER: %s", cfg.DBUser)
	t.Logf("[DEBUG] DB_PASSWORD: %s", cfg.DBPassword)
	t.Logf("[DEBUG] DB_NAME: %s", cfg.DBName)
	t.Logf("[DEBUG] DB_SSL_MODE: %s", cfg.DBSSLMode)

	// Create PostgreSQL container with enhanced health checks
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     cfg.DBUser,
				"POSTGRES_PASSWORD": cfg.DBPassword,
				"POSTGRES_DB":       cfg.DBName,
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections"),
				wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
					return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
						cfg.DBUser,
						cfg.DBPassword,
						host,
						port.Port(),
						cfg.DBName,
						cfg.DBSSLMode)
				}),
				wait.ForExec([]string{"pg_isready", "-U", cfg.DBUser}),
			).WithStartupTimeout(120 * time.Second),
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

	// Connect to database with retries and exponential backoff
	var db *gorm.DB
	maxRetries := 10
	baseDelay := 1 * time.Second
	for i := 0; i < maxRetries; i++ {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host,
			mappedPort.Port(),
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode)

		// Log connection attempt (without sensitive data)
		t.Logf("Attempting to connect to database at %s:%s as user %s (attempt %d/%d)",
			host,
			mappedPort.Port(),
			cfg.DBUser,
			i+1,
			maxRetries)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			// Verify connection is working
			sqlDB, err := db.DB()
			if err == nil {
				err = sqlDB.Ping()
				if err == nil {
					break
				}
			}
		}

		if i < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(i))
			t.Logf("Connection failed, retrying in %v...", delay)
			time.Sleep(delay)
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to database after %d attempts: %v", maxRetries, err)
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}

		// Wait for container to be fully terminated
		<-ctx.Done()
	})

	return db
}
