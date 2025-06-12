package testhelpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed, skipping container-based test")
	}
	// Create a temporary directory for secrets
	secretsDir, err := os.MkdirTemp("", "alchemorsel-test-secrets-*")
	if err != nil {
		t.Fatalf("failed to create temporary secrets directory: %v", err)
	}

	// Set the secrets directory environment variable
	os.Setenv("SECRETS_DIR", secretsDir)
	defer os.Unsetenv("SECRETS_DIR")

	// Clean up the temporary directory after the test
	t.Cleanup(func() {
		if err := os.RemoveAll(secretsDir); err != nil {
			t.Errorf("failed to remove temporary secrets directory: %v", err)
		}
	})

	// Write test secrets
	secrets := map[string]string{
		"db_user":        "postgres",
		"db_password":    "postpass",
		"jwt_secret":     "test-jwt-secret",
		"redis_password": "test-redis-pass",
		"db_host":        "localhost",
		"db_port":        "5432",
		"db_name":        "alchemorsel",
		"db_ssl_mode":    "disable",
		"redis_host":     "localhost",
		"redis_port":     "6379",
		"redis_url":      "redis://localhost:6379",
		"server_port":    "8080",
		"server_host":    "localhost",
	}

	for name, value := range secrets {
		secretPath := filepath.Join(secretsDir, name)
		if err := os.WriteFile(secretPath, []byte(value), 0644); err != nil {
			t.Fatalf("failed to write secret %s: %v", name, err)
		}
	}

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

	// Use testcontainers for both CI and local environments
	ctx := context.Background()

	// Create PostgreSQL container
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
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		mappedPort.Port(),
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode)

	// Log connection attempt (without sensitive data)
	t.Logf("Attempting to connect to database at %s:%s as user %s",
		host,
		mappedPort.Port(),
		cfg.DBUser)

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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}

		// Wait for container to be fully terminated
		<-ctx.Done()
	})

	return db
}
