package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

// TestDB wraps a test database instance
type TestDB struct {
	DB        *gorm.DB
	Config    *config.Config
	Container testcontainers.Container
}

// Close cleans up the test database
func (td *TestDB) Close() error {
	if td.Container != nil {
		return td.Container.Terminate(context.Background())
	}
	return nil
}

// SetupTestDB creates a new test database instance
func SetupTestDB(t *testing.T) *TestDB {
	// Set test environment
	_ = os.Setenv("ENV", "test")

	// Set up environment variables for local testing
	_ = os.Setenv("POSTGRES_USER", "test")
	_ = os.Setenv("POSTGRES_PASSWORD", "test")
	_ = os.Setenv("POSTGRES_DB", "test")
	_ = os.Setenv("POSTGRES_HOST", "localhost")
	_ = os.Setenv("POSTGRES_PORT", "5432")
	_ = os.Setenv("JWT_SECRET", "test-secret")

	// Create test container
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get container host and port
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Update environment variables with container info
	os.Setenv("POSTGRES_HOST", host)
	os.Setenv("POSTGRES_PORT", port.Port())

	// Create config
	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	// Connect to database
	db, err := database.New(cfg)
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Recipe{},
		&models.RecipeFavorite{},
		&models.DietaryPreference{},
		&models.Allergen{},
		&models.ProfileHistory{},
	)
	require.NoError(t, err)

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

	// Create test database instance
	testDB := &TestDB{
		DB:        db,
		Config:    cfg,
		Container: container,
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Logf("Error cleaning up test database: %v", err)
		}
	})

	return testDB
}
