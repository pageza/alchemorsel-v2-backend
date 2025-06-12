package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/mocks"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	testmocks "github.com/pageza/alchemorsel-v2/backend/internal/testhelpers/mocks"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockLLMService implements the LLM service interface for testing

func TestRecipeGenerationEndpoint(t *testing.T) {
	authService := &mocks.MockAuthService{}
	profileService := &mocks.MockProfileService{}
	recipeService := &mocks.MockRecipeService{}
	llmService := testmocks.NewMockLLMService()

	testUserID := uuid.New()
	testToken := "test-token"
	authService.On("ValidateToken", testToken).Return(&types.TokenClaims{
		UserID:   testUserID,
		Username: "testuser",
	}, nil)

	router := setupTestRouter(authService, profileService, recipeService)

	recipes := router.Group("/api/v1/recipes")
	recipes.POST("/generate", middleware.AuthMiddleware(authService), func(c *gin.Context) {
		var req struct {
			Ingredients        []string `json:"ingredients"`
			DietaryPreferences []string `json:"dietary_preferences"`
			Allergens          []string `json:"allergens"`
			Cuisine            string   `json:"cuisine"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		recipeJSON, err := llmService.GenerateRecipe("", req.DietaryPreferences, req.Allergens, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var recipe models.Recipe
		if err := json.Unmarshal([]byte(recipeJSON), &recipe); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse generated recipe"})
			return
		}
		c.JSON(http.StatusOK, recipe)
	})

	t.Run("Generate recipe from ingredients", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"ingredients":         []string{"chicken", "rice", "broccoli"},
			"dietary_preferences": []string{"high-protein"},
			"allergens":           []string{},
		})
		req := httptest.NewRequest("POST", "/api/v1/recipes/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testToken)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		var result map[string]interface{}
		_ = json.Unmarshal(resp.Body.Bytes(), &result)
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "ingredients")
		assert.Contains(t, result, "instructions")
	})
}

// Define a local struct for validation result
type recipeValidationResult struct {
	QualityScore       float64  `json:"quality_score"`
	Suggestions        []string `json:"suggestions"`
	NutritionalBalance float64  `json:"nutritional_balance"`
}

func TestRecipeModification(t *testing.T) {
	// Setup mock services
	authService := &mocks.MockAuthService{}
	profileService := &mocks.MockProfileService{}
	recipeService := &mocks.MockRecipeService{}

	// Setup test user and recipe
	testUserID := uuid.New()
	testRecipeID := uuid.New()
	testToken := "test-token"

	// Mock token validation
	authService.On("ValidateToken", testToken).Return(&types.TokenClaims{
		UserID:   testUserID,
		Username: "testuser",
	}, nil)

	// Mock recipe operations
	recipeService.On("GetRecipe", mock.Anything, testRecipeID).Return(&models.Recipe{
		ID:           testRecipeID,
		UserID:       testUserID,
		Name:         "Original Recipe",
		Description:  "Original Description",
		Ingredients:  []string{"chicken", "rice", "broccoli"},
		Instructions: []string{"Step 1", "Step 2"},
		Calories:     500,
		Protein:      30,
		Carbs:        40,
		Fat:          20,
	}, nil)

	// Mock GetRecipe for non-existent recipe - only for specific UUIDs
	recipeService.On("GetRecipe", mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool {
		return id != testRecipeID
	})).Return(nil, gorm.ErrRecordNotFound)

	// Mock UpdateRecipe for all test cases
	recipeService.On("UpdateRecipe", mock.Anything, testRecipeID, mock.AnythingOfType("*models.Recipe")).Return(&models.Recipe{
		ID:           testRecipeID,
		UserID:       testUserID,
		Name:         "Original Recipe",
		Description:  "Original Description",
		Ingredients:  []string{"tofu", "rice", "broccoli"},
		Instructions: []string{"Step 1", "Step 2"},
		Calories:     1000,
		Protein:      60,
		Carbs:        80,
		Fat:          40,
	}, nil)

	// Setup test router
	router := setupTestRouter(authService, profileService, recipeService)

	// Test cases for recipe modification
	tests := []struct {
		name           string
		recipeID       string
		requestBody    map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, response *httptest.ResponseRecorder)
	}{
		{
			name:     "Scale recipe portions",
			recipeID: testRecipeID.String(),
			requestBody: map[string]interface{}{
				"scale_factor": 2.0,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var result map[string]interface{}
				err := json.Unmarshal(response.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.Contains(t, result, "ingredients")
				assert.Contains(t, result, "calories")
				assert.Contains(t, result, "protein")
				assert.Contains(t, result, "carbs")
				assert.Contains(t, result, "fat")
			},
		},
		{
			name:     "Substitute ingredients",
			recipeID: testRecipeID.String(),
			requestBody: map[string]interface{}{
				"substitutions": map[string]string{
					"chicken": "tofu",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var result map[string]interface{}
				err := json.Unmarshal(response.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.Contains(t, result, "ingredients")
				assert.Contains(t, result, "instructions")
				assert.Contains(t, result, "calories")
			},
		},
		{
			name:     "Adapt for dietary preferences",
			recipeID: testRecipeID.String(),
			requestBody: map[string]interface{}{
				"dietary_preferences": []string{"vegetarian"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var result map[string]interface{}
				err := json.Unmarshal(response.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.Contains(t, result, "ingredients")
				assert.Contains(t, result, "instructions")
				assert.Contains(t, result, "dietary_preferences")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			body, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)

			// Create request
			req := httptest.NewRequest("POST", "/api/v1/recipes/"+tt.recipeID+"/modify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+testToken)

			// Record response
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.Code)

			// Check response content
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestRecipeQualityValidation(t *testing.T) {
	// Setup mock services
	authService := &mocks.MockAuthService{}
	profileService := &mocks.MockProfileService{}
	recipeService := &mocks.MockRecipeService{}

	// Setup test user and recipe
	testUserID := uuid.New()
	testRecipeID := uuid.New()
	testToken := "test-token"

	// Mock token validation
	authService.On("ValidateToken", testToken).Return(&types.TokenClaims{
		UserID:   testUserID,
		Username: "testuser",
	}, nil)

	// Mock recipe operations
	recipeService.On("GetRecipe", mock.Anything, testRecipeID).Return(&models.Recipe{
		ID:           testRecipeID,
		UserID:       testUserID,
		Name:         "Test Recipe",
		Description:  "Test Description",
		Ingredients:  []string{"chicken", "rice", "broccoli"},
		Instructions: []string{"Step 1", "Step 2"},
		Calories:     500,
		Protein:      30,
		Carbs:        40,
		Fat:          20,
	}, nil)

	// Mock GetRecipe for non-existent recipe - only for specific UUIDs
	recipeService.On("GetRecipe", mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool {
		return id != testRecipeID
	})).Return(nil, gorm.ErrRecordNotFound)

	// Mock UpdateRecipe for all test cases
	recipeService.On("UpdateRecipe", mock.Anything, testRecipeID, mock.AnythingOfType("*models.Recipe")).Return(&models.Recipe{
		ID:           testRecipeID,
		UserID:       testUserID,
		Name:         "Original Recipe",
		Description:  "Original Description",
		Ingredients:  []string{"tofu", "rice", "broccoli"},
		Instructions: []string{"Step 1", "Step 2"},
		Calories:     1000,
		Protein:      60,
		Carbs:        80,
		Fat:          40,
	}, nil)

	// Setup test router
	router := setupTestRouter(authService, profileService, recipeService)

	// Test cases for recipe quality validation
	tests := []struct {
		name           string
		recipeID       string
		expectedStatus int
		checkResponse  func(t *testing.T, response *httptest.ResponseRecorder)
	}{
		{
			name:           "Validate recipe quality",
			recipeID:       testRecipeID.String(),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var result recipeValidationResult
				err := json.Unmarshal(response.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, result.QualityScore, 0.0)
				assert.LessOrEqual(t, result.QualityScore, 1.0)
				assert.NotNil(t, result.Suggestions)
				assert.GreaterOrEqual(t, result.NutritionalBalance, 0.0)
				assert.LessOrEqual(t, result.NutritionalBalance, 1.0)
			},
		},
		{
			name:           "Validate non-existent recipe",
			recipeID:       uuid.New().String(),
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var result map[string]interface{}
				err := json.Unmarshal(response.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.Contains(t, result, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("POST", "/api/v1/recipes/"+tt.recipeID+"/validate", nil)
			req.Header.Set("Authorization", "Bearer "+testToken)

			// Record response
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.Code)

			// Check response content
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}
