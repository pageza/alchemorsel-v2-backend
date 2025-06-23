package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// TestLLMService with dietary tracking
type TestLLMService struct {
	*MockLLMService
	lastDietaryPrefs []string
	lastAllergens    []string
}

func NewTestLLMService() *TestLLMService {
	return &TestLLMService{
		MockLLMService: NewMockLLMService(),
	}
}

func (m *TestLLMService) GenerateRecipe(query string, dietaryPrefs, allergens []string, originalRecipe *service.RecipeDraft) (string, error) {
	// Store the dietary preferences and allergens for verification
	m.lastDietaryPrefs = dietaryPrefs
	m.lastAllergens = allergens
	
	// Generate a recipe that respects dietary preferences
	if len(dietaryPrefs) > 0 && contains(dietaryPrefs, "vegan") {
		return `{
			"name":"Vegan Chickpea Curry",
			"description":"A delicious plant-based curry",
			"category":"Main Course",
			"cuisine":"Indian",
			"ingredients":["chickpeas","coconut milk","curry spices","vegetables"],
			"instructions":["Cook chickpeas","Add spices and vegetables","Simmer with coconut milk"],
			"prep_time":"15 minutes",
			"cook_time":"30 minutes",
			"servings":"4",
			"difficulty":"Easy",
			"calories":300,
			"protein":12,
			"carbs":40,
			"fat":10
		}`, nil
	}
	
	return m.MockLLMService.GenerateRecipe(query, dietaryPrefs, allergens, originalRecipe)
}

func (m *TestLLMService) GetDraft(ctx context.Context, id string) (*service.RecipeDraft, error) {
	// First check our parent's drafts
	if draft, err := m.MockLLMService.GetDraft(ctx, id); err == nil {
		return draft, nil
	}
	// Return not found
	return nil, fmt.Errorf("draft not found")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func setupTestDBWithSchema() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Create tables with simplified schema for SQLite
	sql := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME, 
		deleted_at DATETIME,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		is_email_verified BOOLEAN DEFAULT false,
		email_verified_at DATETIME,
		verification_token TEXT,
		verification_token_expires_at DATETIME
	);
	
	CREATE TABLE dietary_preferences (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		preference_type TEXT NOT NULL,
		custom_name TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);
	
	CREATE TABLE allergens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		allergen_name TEXT NOT NULL,
		severity_level INTEGER NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`
	
	err = db.Exec(sql).Error
	if err != nil {
		panic("failed to create test tables: " + err.Error())
	}

	return db
}

func TestLLMHandler_GenerateRecipeWithDietaryRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithSchema()

	// Create test user with dietary preferences
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Name:          "Test User",
		Email:         "test@example.com",
		PasswordHash:  "hash",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	db.Create(user)

	// Add vegan dietary preference
	dietaryPref := &models.DietaryPreference{
		ID:             uuid.New(),
		UserID:         userID,
		PreferenceType: "vegan",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	db.Create(dietaryPref)

	// Add dairy allergen
	allergen := &models.Allergen{
		ID:            uuid.New(),
		UserID:        userID,
		AllergenName:  "dairy",
		SeverityLevel: 5,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	db.Create(allergen)

	// Setup test LLM service
	testLLMService := NewTestLLMService()
	handler := NewLLMHandler(db, nil, testLLMService, nil)

	// Create test request - user asks for "chicken dinner" but has vegan preference
	reqBody := QueryRequest{
		Query:  "chicken dinner",
		Intent: "generate",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/llm/query", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID)

	// Call handler
	handler.Query(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify recipe was generated
	assert.Contains(t, response, "recipe")
	assert.Contains(t, response, "draft_id")

	// CRITICAL: Verify dietary preferences were passed to LLM service
	assert.Equal(t, []string{"vegan"}, testLLMService.lastDietaryPrefs, "Dietary preferences should be passed to LLM service")
	assert.Equal(t, []string{"dairy"}, testLLMService.lastAllergens, "Allergens should be passed to LLM service")

	// Verify the generated recipe respects vegan dietary preference
	recipe := response["recipe"].(map[string]interface{})
	assert.Equal(t, "Vegan Chickpea Curry", recipe["name"])
	assert.NotContains(t, recipe["name"], "Chicken", "Vegan recipe should not contain chicken")
}

func TestLLMHandler_NoDietaryRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithSchema()

	// Create test user WITHOUT dietary preferences
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Name:          "Test User",
		Email:         "test@example.com",
		PasswordHash:  "hash",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	db.Create(user)

	// Setup test LLM service
	testLLMService := NewTestLLMService()
	handler := NewLLMHandler(db, nil, testLLMService, nil)

	// Create test request
	reqBody := QueryRequest{
		Query:  "chocolate cake",
		Intent: "generate",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/llm/query", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID)

	// Call handler
	handler.Query(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify empty dietary preferences were passed
	assert.Empty(t, testLLMService.lastDietaryPrefs, "No dietary preferences should be passed")
	assert.Empty(t, testLLMService.lastAllergens, "No allergens should be passed")
}

func TestLLMHandler_ModifyRecipeWithDietaryRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithSchema()

	// Create test user with dietary preferences
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Name:          "Test User",
		Email:         "test@example.com",
		PasswordHash:  "hash",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	db.Create(user)

	// Add gluten-free dietary preference
	dietaryPref := &models.DietaryPreference{
		ID:             uuid.New(),
		UserID:         userID,
		PreferenceType: "gluten-free",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	db.Create(dietaryPref)

	// Setup test LLM service with existing draft
	testLLMService := NewTestLLMService()
	
	// Create a draft to modify
	draft := &service.RecipeDraft{
		ID:           "test-draft-123",
		Name:         "Pasta Carbonara",
		Description:  "Classic Italian pasta",
		Category:     "Main Course",
		Ingredients:  []string{"pasta", "eggs", "bacon", "parmesan"},
		Instructions: []string{"Cook pasta", "Mix eggs and cheese", "Combine"},
		UserID:       userID.String(),
	}
	testLLMService.SaveDraft(context.Background(), draft)
	
	handler := NewLLMHandler(db, nil, testLLMService, nil)

	// Create test request to modify the draft
	reqBody := QueryRequest{
		Query:   "make it healthier",
		Intent:  "modify",
		DraftID: "test-draft-123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/llm/query", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID)

	// Call handler
	handler.Query(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// CRITICAL: Verify dietary preferences were passed even for modifications
	assert.Equal(t, []string{"gluten-free"}, testLLMService.lastDietaryPrefs, "Dietary preferences should be passed for modifications")
}