package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
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

// TestDatabase represents a test database with all necessary dependencies
type TestDatabase struct {
	*gorm.DB
	AuthService *service.AuthService
}

// SetupTestDB creates a new test database using PostgreSQL testcontainer
func SetupTestDB(t *testing.T) *gorm.DB {
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

// CreateTestUserAndToken creates a test user and returns their ID and a valid JWT token
func CreateTestUserAndToken(t *testing.T, db *TestDatabase) (uuid.UUID, string) {
	// Create a test user with a known password
	userID := uuid.New()
	password := "testpassword123"
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
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create user profile
	profile := models.UserProfile{
		ID:       uuid.New(),
		UserID:   userID,
		Username: fmt.Sprintf("testuser_%s", userID.String()),
	}
	if err := db.DB.Create(&profile).Error; err != nil {
		t.Fatalf("failed to create test user profile: %v", err)
	}

	// DEBUG: Print input values for Login
	fmt.Printf("[DEBUG] Calling Login with email: %s, password: %s\n", user.Email, password)
	loggedInUser, _, err := db.AuthService.Login(context.Background(), user.Email, password)
	// DEBUG: Print output values for Login
	fmt.Printf("[DEBUG] Login returned user: %+v, err: %v\n", loggedInUser, err)

	// Generate token
	token, err := db.AuthService.GenerateToken(&types.TokenClaims{
		UserID:   loggedInUser.ID,
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
