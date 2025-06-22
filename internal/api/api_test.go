package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/mocks"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
)

func TestLLMQueryValidatesInput(t *testing.T) {
	testDB := SetupTestDB(t)
	router, _ := setupLLMTestRouter(t, testDB)

	// Create test user and token
	_, token := CreateTestUserAndToken(t, testDB)

	// Test empty query
	w := PerformRequestWithToken(router, "POST", "/api/v1/llm/query", map[string]interface{}{
		"query":  "",
		"intent": "generate",
	}, token)
	assert.Equal(t, 400, w.Code)

	// Test missing query
	w = PerformRequestWithToken(router, "POST", "/api/v1/llm/query", map[string]interface{}{
		"intent": "generate",
	}, token)
	assert.Equal(t, 400, w.Code)
}

func TestLLMQueryModifyRecipe(t *testing.T) {
	testDB := SetupTestDB(t)
	router, mockLLMService := setupLLMTestRouter(t, testDB)

	// Create test user and token
	userID, token := CreateTestUserAndToken(t, testDB)

	// Create a draft for the user first using the shared mock LLM service
	draft := &service.RecipeDraft{
		ID:           "test-draft-id",
		Name:         "Test Recipe",
		Description:  "Desc",
		Category:     "Cat",
		Ingredients:  []string{"beef", "salt", "pepper"},
		Instructions: []string{"s1"},
		Calories:     100,
		Protein:      10,
		Carbs:        20,
		Fat:          5,
		UserID:       userID.String(), // Use the actual test user ID
	}
	_ = mockLLMService.SaveDraft(context.Background(), draft)

	// Test recipe modification
	w := PerformRequestWithToken(router, "POST", "/api/v1/llm/query", map[string]interface{}{
		"query":    "Make this recipe vegetarian",
		"intent":   "modify",
		"draft_id": "test-draft-id",
	}, token)
	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "recipe")
}

func setupLLMTestRouter(t *testing.T, testDB *TestDB) (*gin.Engine, *MockLLMService) {
	println("[DEBUG] setupLLMTestRouter called")
	mockRecipeService := &mocks.MockRecipeService{}
	mockLLMService := NewMockLLMService()
	llmHandler := NewLLMHandler(testDB.DB, testDB.AuthService, mockLLMService, mockRecipeService)

	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")

	// Use the real auth service for both middleware and handler consistency
	v1.Use(middleware.AuthMiddleware(testDB.AuthService))

	llmHandler.RegisterRoutes(v1)

	return router, mockLLMService
}

// createTestJWT creates a valid JWT token for testing
func createTestJWT(authService *service.AuthService) (string, error) {
	testUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	return authService.GenerateToken(&types.TokenClaims{
		UserID:   testUserID,
		Username: "testuser",
	})
}

// PerformRequestWithToken performs an HTTP request with a JWT token
func PerformRequestWithToken(router *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
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
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)
	return w
}
