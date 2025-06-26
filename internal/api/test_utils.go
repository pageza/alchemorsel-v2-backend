package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/mock"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MockTokenValidator implements a mock token validator for testing
type MockTokenValidator struct {
	mock.Mock
}

func (v *MockTokenValidator) ValidateToken(token string) (*types.TokenClaims, error) {
	// LINT-FIX-2025: Use unconditional TrimPrefix instead of conditional check
	// gosimple suggests this pattern is more idiomatic and handles edge cases better
	token = strings.TrimPrefix(token, "Bearer ")
	
	args := v.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

// Helper for test middleware to set 'user_id' in context
func SetUserIDInContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
}

// TestDB holds test database and services
type TestDB struct {
	DB          *gorm.DB
	AuthService *service.AuthService
	Container   testcontainers.Container
}

// SetupTestDB creates a test database and services
func SetupTestDB(t *testing.T) *TestDB {
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
			// LINT-FIX-2025: Replace deprecated WithStartupTimeout with WithDeadline
			// WithStartupTimeout is deprecated in testcontainers, use WithDeadline instead
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			).WithDeadline(120 * time.Second),
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

	// Connect to database with retry logic
	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, mappedPort.Port())

	var db *gorm.DB
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
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

	// Get underlying sql.DB to ensure proper cleanup
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
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

	// Create auth service
	authService := service.NewAuthService(db, "test-secret")

	// Register cleanup
	t.Cleanup(func() {
		// Close database connection
		if err := sqlDB.Close(); err != nil {
			t.Errorf("failed to close database connection: %v", err)
		}
		// Terminate container
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	return &TestDB{
		DB:          db,
		AuthService: authService,
		Container:   container,
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
		ID:            userID,
		Name:          "Test User",
		Email:         fmt.Sprintf("testuser+%s@example.com", userID.String()),
		PasswordHash:  string(hashedPassword),
		EmailVerified: true, // Enable email verification for tests
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

// SetupTestRouter creates a new router with test configuration
func SetupTestRouter(t *testing.T) *gin.Engine {
	// Setup test database and services
	testDB := SetupTestDB(t)

	// Create handlers
	emailService := service.NewEmailService()
	authHandler := NewAuthHandler(testDB.AuthService, emailService, testDB.DB)
	recipeHandler := NewRecipeHandler(service.NewRecipeService(testDB.DB, nil), testDB.AuthService, nil, nil, testDB.DB)
	// Use a mock LLM handler instead of the real one
	llmHandler := NewMockLLMHandler()

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// Register routes
	v1 := router.Group("/api/v1")
	authHandler.RegisterRoutes(v1)
	llmHandler.RegisterRoutes(v1)

	// Add auth middleware to protected routes
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(testDB.AuthService))
	recipeHandler.RegisterRoutes(protected)

	// Register cleanup for router
	t.Cleanup(func() {
		// Force close any remaining connections
		if sqlDB, err := testDB.DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	return router
}

// PerformRequest is a helper function to make HTTP requests in tests
func PerformRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// Add Authorization header with Bearer prefix
	req.Header.Set("Authorization", "Bearer test-token")

	router.ServeHTTP(w, req)
	return w
}

// MockAuthService is a mock implementation of the AuthService interface
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateToken(token string) (*types.TokenClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

// MockLLMService implements a mock LLM service for testing
// Copied from api_test.go for use in test router and all tests
type MockLLMService struct {
	drafts map[string]*service.RecipeDraft
}

func NewMockLLMService() *MockLLMService {
	return &MockLLMService{
		drafts: make(map[string]*service.RecipeDraft),
	}
}

func (m *MockLLMService) GenerateRecipe(query string, dietaryPrefs, allergens []string, originalRecipe *service.RecipeDraft) (string, error) {
	return `{"name":"Test Recipe","description":"Desc","category":"Cat","ingredients":["i1"],"instructions":["s1"],"calories":100,"protein":10,"carbs":20,"fat":5}`, nil
}

func (m *MockLLMService) SaveDraft(ctx context.Context, draft *service.RecipeDraft) error {
	draft.ID = "test-draft-id"
	m.drafts[draft.ID] = draft
	return nil
}

func (m *MockLLMService) GetDraft(ctx context.Context, id string) (*service.RecipeDraft, error) {
	if draft, exists := m.drafts[id]; exists {
		return draft, nil
	}
	// Return a default draft - in real tests this should be set up properly
	// For now, return an error to indicate the draft doesn't exist
	return nil, fmt.Errorf("draft not found")
}

func (m *MockLLMService) UpdateDraft(ctx context.Context, draft *service.RecipeDraft) error {
	m.drafts[draft.ID] = draft
	return nil
}

func (m *MockLLMService) DeleteDraft(ctx context.Context, id string) error {
	delete(m.drafts, id)
	return nil
}

func (m *MockLLMService) CalculateMacros(ingredients []string) (*service.Macros, error) {
	return &service.Macros{
		Calories: 100,
		Protein:  10,
		Carbs:    20,
		Fat:      5,
	}, nil
}

func (m *MockLLMService) GenerateRecipesBatch(prompts []string) ([]string, error) {
	return []string{`{"name":"Test Recipe","description":"Desc","category":"Cat","ingredients":["i1"],"instructions":["s1"],"calories":100,"protein":10,"carbs":20,"fat":5}`}, nil
}

// Multi-call recipe generation methods
func (m *MockLLMService) GenerateBasicRecipe(ctx context.Context, query string, dietaryPrefs []string, allergens []string, userID string) (*service.RecipeDraft, error) {
	draft := &service.RecipeDraft{
		ID:          "test-basic-draft-id",
		Name:        "Test Basic Recipe",
		Description: "A test basic recipe",
		Category:    "Main Course",
		Cuisine:     "American",
		Ingredients: []string{"1 cup flour", "2 eggs"},
		Instructions: []string{"Mix ingredients", "Cook"},
		PrepTime:    "15 minutes",
		CookTime:    "30 minutes",
		Servings:    service.ServingsType{Value: "4"},
		Difficulty:  "Easy",
		UserID:      userID,
		Calories:    0, // Basic recipe doesn't have nutrition
		Protein:     0,
		Carbs:       0,
		Fat:         0,
	}
	
	// Save the draft
	m.drafts[draft.ID] = draft
	return draft, nil
}

func (m *MockLLMService) CalculateRecipeNutrition(ctx context.Context, draftID string) (*service.Macros, error) {
	// Simulate nutrition calculation
	macros := &service.Macros{
		Calories: 350,
		Protein:  15,
		Carbs:    45,
		Fat:      12,
	}
	
	// Update the draft with nutrition data
	if draft, exists := m.drafts[draftID]; exists {
		draft.Calories = macros.Calories
		draft.Protein = macros.Protein
		draft.Carbs = macros.Carbs
		draft.Fat = macros.Fat
	}
	
	return macros, nil
}

func (m *MockLLMService) FinalizeRecipe(ctx context.Context, draftID string) (*service.RecipeDraft, error) {
	if draft, exists := m.drafts[draftID]; exists {
		// Simulate embedding generation during finalization
		if draft.Embedding.Slice() == nil || len(draft.Embedding.Slice()) == 0 {
			// Create a mock embedding (1536 dimensions like OpenAI's ada-002)
			mockEmbedding := make([]float32, 1536)
			for i := range mockEmbedding {
				mockEmbedding[i] = 0.1 // Simple mock values
			}
			draft.Embedding = pgvector.NewVector(mockEmbedding)
		}
		
		// Return the finalized draft with embedding
		return draft, nil
	}
	return nil, fmt.Errorf("draft not found")
}

// NewMockLLMHandler creates a mock LLM handler for testing
func NewMockLLMHandler() *LLMHandler {
	return &LLMHandler{
		// Mock implementation or nil if not needed for tests
	}
}
