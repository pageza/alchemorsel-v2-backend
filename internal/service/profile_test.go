package service

import (
	"context"
	"fmt"
	"os/exec"
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
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed, skipping container-based test")
	}
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

	svc := NewProfileService(db)
	userID := uuid.New()

	// Create a test user and profile first
	user := &models.User{
		ID:           userID,
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	profile := &models.UserProfile{
		UserID:   userID,
		Username: "testuser",
	}
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}

	// Test getting the profile
	result, err := svc.GetProfile(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "testuser", result.Username)
}

func TestUpdateProfile(t *testing.T) {
	// Create a test DB instance
	db := setupTestDB(t)

	svc := NewProfileService(db)
	userID := uuid.New()

	// Create a test user and profile first
	user := &models.User{
		ID:           userID,
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	profile := &models.UserProfile{
		UserID:   userID,
		Username: "testuser",
	}
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("failed to create test profile: %v", err)
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
	assert.Equal(t, "updateduser", result.Username)
	assert.Equal(t, bio, result.Bio)
}
