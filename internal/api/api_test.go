package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
)

func TestLLMQueryValidatesInput(t *testing.T) {
	router := setupLLMTestRouter(t)

	// Test empty query
	w := PerformRequest(router, "POST", "/api/v1/llm/query", map[string]interface{}{
		"query":  "",
		"intent": "generate",
	})
	assert.Equal(t, 400, w.Code)

	// Test missing query
	w = PerformRequest(router, "POST", "/api/v1/llm/query", map[string]interface{}{
		"intent": "generate",
	})
	assert.Equal(t, 400, w.Code)
}

func TestLLMQueryModifyRecipe(t *testing.T) {
	router := setupLLMTestRouter(t)

	// Test recipe modification
	w := PerformRequest(router, "POST", "/api/v1/llm/query", map[string]interface{}{
		"query":    "Make this recipe vegetarian",
		"intent":   "modify",
		"draft_id": "test-draft-id",
		"recipe": map[string]interface{}{
			"name":        "Test Recipe",
			"ingredients": []string{"beef", "salt", "pepper"},
		},
	})
	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "recipe")
}

func setupLLMTestRouter(t *testing.T) *gin.Engine {
	println("[DEBUG] setupLLMTestRouter called")
	testDB := SetupTestDB(t)
	llmHandler := NewLLMHandler(testDB.DB, testDB.AuthService, NewMockLLMService())

	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")

	// Create mock token validator
	mockValidator := &MockTokenValidator{}
	testUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	mockValidator.On("ValidateToken", "test-token").Return(&types.TokenClaims{
		UserID:   testUserID,
		Username: "testuser",
	}, nil)

	// Add auth middleware with mock validator
	v1.Use(middleware.AuthMiddleware(mockValidator))

	// Add test-only middleware AFTER auth middleware to ensure user_id is set
	v1.Use(func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		println("[DEBUG] Test-only middleware Authorization header:", token)
		if token == "Bearer test-token" {
			println("[DEBUG] Setting user_id in context!")
			c.Set("user_id", testUserID)
		}
		c.Next()
	})

	llmHandler.RegisterRoutes(v1)

	return router
}
