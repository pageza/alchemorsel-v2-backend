package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDatabase represents a test database instance with its configuration
type TestDatabase struct {
	db          *gorm.DB
	config      *config.Config
	authService *service.AuthService
}

// DB returns the underlying *gorm.DB instance
func (db *TestDatabase) DB() *gorm.DB {
	return db.db
}

// SetupTestDB creates a new test database using PostgreSQL testcontainer
func SetupTestDB(t *testing.T) *TestDatabase {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Check if we're in CI environment
	if config.IsCI() {
		// In CI, use the service container with environment variables
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode)

		// Log connection attempt (without sensitive data)
		t.Logf("Attempting to connect to database at %s:%s as user %s",
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBUser)

		// Try to connect with retries
		var db *gorm.DB
		for i := 0; i < 5; i++ {
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Info),
			})
			if err == nil {
				break
			}
			t.Logf("Connection attempt %d failed: %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			t.Fatalf("failed to connect to database after 5 attempts: %v", err)
		}

		return setupDatabase(t, db, cfg)
	}

	// Local development: use testcontainers
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
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

	// Create the test database instance
	testDB := &TestDatabase{
		db:          db,
		config:      cfg,
		authService: service.NewAuthService(db, cfg.JWTSecret),
	}

	return testDB
}

// setupDatabase performs common database setup tasks
func setupDatabase(t *testing.T, db *gorm.DB, cfg *config.Config) *TestDatabase {
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
	err := db.AutoMigrate(
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

	return &TestDatabase{
		db:          db,
		config:      cfg,
		authService: service.NewAuthService(db, cfg.JWTSecret),
	}
}

// CreateTestUserAndToken creates a test user and returns their ID and a valid JWT token
func CreateTestUserAndToken(t *testing.T, db *TestDatabase) (uuid.UUID, string) {
	// Create a test user with configured password
	userID := uuid.New()
	password := os.Getenv("TEST_USER_PASSWORD")
	if password == "" {
		password = "testpassword123" // Fallback for local development
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		ID:           userID,
		Name:         "Test User",
		Email:        fmt.Sprintf("testuser+%s@example.com", userID.String()),
		PasswordHash: string(hashedPassword),
	}
	if err := db.db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create user profile
	profile := models.UserProfile{
		ID:       uuid.New(),
		UserID:   userID,
		Username: fmt.Sprintf("testuser_%s", userID.String()),
	}
	if err := db.db.Create(&profile).Error; err != nil {
		t.Fatalf("failed to create test user profile: %v", err)
	}

	// Generate token
	token, err := db.authService.GenerateToken(&types.TokenClaims{
		UserID:   user.ID,
		Username: profile.Username,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return userID, token
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)
	return user
}

// CreateTestProfile creates a test user profile in the database
func CreateTestProfile(t *testing.T, db *gorm.DB, userID uuid.UUID) *models.UserProfile {
	profile := &models.UserProfile{
		UserID:   userID,
		Username: "testuser",
	}
	err := db.Create(profile).Error
	assert.NoError(t, err)
	return profile
}

// CreateTestRecipe creates a test recipe in the database
func CreateTestRecipe(t *testing.T, db *gorm.DB, userID uuid.UUID) *models.Recipe {
	recipe := &models.Recipe{
		UserID:       userID,
		Name:         "Test Recipe",
		Description:  "A test recipe",
		Ingredients:  models.JSONBStringArray{"ingredient1", "ingredient2"},
		Instructions: models.JSONBStringArray{"step1", "step2"},
	}
	err := db.Create(recipe).Error
	assert.NoError(t, err)
	return recipe
}

// CreateTestRecipeFavorite creates a test recipe favorite
func CreateTestRecipeFavorite(t *testing.T, userID, recipeID uuid.UUID) *models.RecipeFavorite {
	return &models.RecipeFavorite{
		ID:       uuid.New(),
		UserID:   userID,
		RecipeID: recipeID,
	}
}

// MockTokenValidator is a mock token validator for testing
type MockTokenValidator struct {
	Claims *types.TokenClaims
	Error  error
}

// ValidateToken validates a token and returns claims
func (m *MockTokenValidator) ValidateToken(token string) (*types.TokenClaims, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Claims, nil
}

// JSONMarshal is a helper function to marshal JSON for testing
func JSONMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return data
}
