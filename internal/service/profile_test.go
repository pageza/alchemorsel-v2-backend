package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	ctx := context.Background()

	// Create PostgreSQL container with improved configuration
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":        "testuser",
				"POSTGRES_PASSWORD":    "testpass",
				"POSTGRES_DB":          "testdb",
				"POSTGRES_INITDB_ARGS": "--auth-host=scram-sha-256",
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			).WithStartupTimeout(120 * time.Second), //nolint:staticcheck // testcontainers API limitation
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

	// Retry connection with exponential backoff
	var db *gorm.DB
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
			host, mappedPort.Port())

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			// Test the connection
			sqlDB, pingErr := db.DB()
			if pingErr == nil && sqlDB.Ping() == nil {
				break
			}
		}

		// Wait before retrying
		time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)

		if i == maxRetries-1 {
			t.Fatalf("failed to connect to database after %d attempts: %v", maxRetries, err)
		}
	}

	// Install pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector;").Error; err != nil {
		t.Fatalf("failed to install pgvector extension: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Recipe{},
		&models.RecipeFavorite{},
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

func TestGetProfile(t *testing.T) {
	// Create a test DB instance
	db := setupTestDB(t)
	if db == nil {
		t.Fatal("failed to setup test database")
	}

	// Ensure database connection is active
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get database connection: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("database connection is not active: %v", err)
	}

	svc := NewProfileService(db)
	userID := uuid.New()

	// Create a test user and profile first with transaction for atomicity
	err = db.Transaction(func(tx *gorm.DB) error {
		user := &models.User{
			ID:           userID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		profile := &models.UserProfile{
			UserID:   userID,
			Username: "testuser",
		}
		if err := tx.Create(profile).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	// Test getting the profile
	result, err := svc.GetProfile(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	if result != nil {
		assert.Equal(t, "testuser", result.Username)
	}
}

func TestUpdateProfile(t *testing.T) {
	// Create a test DB instance
	db := setupTestDB(t)
	if db == nil {
		t.Fatal("failed to setup test database")
	}

	// Ensure database connection is active
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get database connection: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("database connection is not active: %v", err)
	}

	svc := NewProfileService(db)
	userID := uuid.New()

	// Create a test user and profile first with transaction for atomicity
	err = db.Transaction(func(tx *gorm.DB) error {
		user := &models.User{
			ID:           userID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		profile := &models.UserProfile{
			UserID:   userID,
			Username: "testuser",
		}
		if err := tx.Create(profile).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	// Test updating the profile
	bio := "Updated bio"
	update := &types.UpdateProfileRequest{
		Username: "updateduser",
		Bio:      &bio,
	}
	result, err := svc.UpdateProfile(context.Background(), userID, update)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	if result != nil {
		assert.Equal(t, "updateduser", result.Username)
		assert.Equal(t, bio, result.Bio)
	}
}
