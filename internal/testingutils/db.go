package testingutils

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/pgvector/pgvector-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestUser represents a user in the test database
type TestUser struct {
	ID           string         `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"not null" json:"name"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
}

// TestUserPreference represents a user preference in the test database
type TestUserPreference struct {
	ID        string              `gorm:"primarykey" json:"id"`
	UserID    string              `gorm:"uniqueIndex;not null" json:"user_id"`
	Dietary   TestJSONStringArray `json:"dietary"`
	Allergies TestJSONStringArray `json:"allergies"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	DeletedAt gorm.DeletedAt      `gorm:"index" json:"-"`
}

// TestRecipe represents a recipe in the test database
type TestRecipe struct {
	ID           string              `gorm:"primarykey" json:"id"`
	UserID       string              `gorm:"not null" json:"user_id"`
	Name         string              `gorm:"not null" json:"name"`
	Description  string              `json:"description"`
	Category     string              `json:"category"`
	ImageURL     string              `json:"image_url"`
	Ingredients  TestJSONStringArray `json:"ingredients"`
	Instructions TestJSONStringArray `json:"instructions"`
	Calories     float64             `json:"calories"`
	Protein      float64             `json:"protein"`
	Carbs        float64             `json:"carbs"`
	Fat          float64             `json:"fat"`
	PrepTime     int                 `json:"prep_time"`
	CookTime     int                 `json:"cook_time"`
	Servings     int                 `json:"servings"`
	Difficulty   string              `json:"difficulty"`
	Embedding    []float32           `gorm:"type:text" json:"embedding"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	DeletedAt    gorm.DeletedAt      `gorm:"index" json:"-"`
}

// TestRecipeFavorite represents a recipe favorite in the test database
type TestRecipeFavorite struct {
	ID        string         `gorm:"primarykey" json:"id"`
	UserID    string         `gorm:"not null" json:"user_id"`
	RecipeID  string         `gorm:"not null" json:"recipe_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TestJSONStringArray is a custom type for handling string arrays in SQLite
type TestJSONStringArray []string

// Value implements the driver.Valuer interface
func (a TestJSONStringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *TestJSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*a = TestJSONStringArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, a)
}

// MockTokenValidator is a mock implementation of the token validator
type MockTokenValidator struct{}

func (v *MockTokenValidator) ValidateToken(token string) (*types.TokenClaims, error) {
	// For testing, accept any token that starts with "test-"
	if strings.HasPrefix(token, "test-") {
		return &types.TokenClaims{
			UserID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Username: "testuser",
		}, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// MockEmbeddingService is a mock implementation of the embedding service
type MockEmbeddingService struct{}

func (s *MockEmbeddingService) GenerateEmbedding(text string) (pgvector.Vector, error) {
	// Return a simple mock embedding for testing
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
}

func (s *MockEmbeddingService) GenerateEmbeddingFromRecipe(name, description string, ingredients []string, category string, dietary []string) (pgvector.Vector, error) {
	// Return a simple mock embedding for testing
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
}

// TestDB represents a test database instance
type TestDB struct {
	DB             *gorm.DB
	AuthService    *service.AuthService
	RecipeService  *service.RecipeService
	TokenValidator *MockTokenValidator
	Container      testcontainers.Container
}

// SetupTestDatabase creates a new test database using testcontainers
func SetupTestDatabase(t *testing.T) *TestDB {
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

	// Run migrations
	migrationsDir := "../../migrations"
	if err := database.RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create test services
	authService := service.NewAuthService(db, "test-secret-key")
	mockEmbeddingService := &MockEmbeddingService{}
	recipeService := service.NewRecipeService(db, mockEmbeddingService)
	mockTokenValidator := &MockTokenValidator{}

	// Register cleanup
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	return &TestDB{
		DB:             db,
		AuthService:    authService,
		RecipeService:  recipeService,
		TokenValidator: mockTokenValidator,
		Container:      container,
	}
}

// CreateTestUserAndToken creates a test user and returns their ID and a valid JWT token
func CreateTestUserAndToken(t *testing.T, db *TestDB) (uuid.UUID, string) {
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

	// Generate token
	token, err := db.AuthService.GenerateToken(&types.TokenClaims{
		UserID:   user.ID,
		Username: profile.Username,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return userID, token
}
