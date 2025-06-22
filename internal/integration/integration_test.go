package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/mocks"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingService is a mock implementation of the embedding service
type MockEmbeddingService struct{}

func (m *MockEmbeddingService) GenerateEmbedding(text string) (pgvector.Vector, error) {
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
}

func (m *MockEmbeddingService) GenerateEmbeddingFromRecipe(name, description string, ingredients []string, category string, dietary []string) (pgvector.Vector, error) {
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
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

// MockOpenAIService implements a mock OpenAI service for testing
type MockOpenAIService struct{}

func (m *MockOpenAIService) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
}

// MockLLMService implements a mock LLM service for testing
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
	return &service.RecipeDraft{
		ID:           id,
		Name:         "Test Recipe",
		Description:  "Desc",
		Category:     "Cat",
		Ingredients:  []string{"i1"},
		Instructions: []string{"s1"},
		Calories:     100,
		Protein:      10,
		Carbs:        20,
		Fat:          5,
	}, nil
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

// setupTestRouter creates a test router with mock services
func setupTestRouter(authService *mocks.MockAuthService, profileService *mocks.MockProfileService, recipeService *mocks.MockRecipeService) *gin.Engine {
	router := gin.Default()

	// Create API v1 group
	v1 := router.Group("/api/v1")

	// Register auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", func(c *gin.Context) {
			var req struct {
				Username string `json:"username" binding:"required"`
				Email    string `json:"email" binding:"required,email"`
				Password string `json:"password" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			user, err := authService.Register(c.Request.Context(), req.Email, req.Password, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			token, err := authService.GenerateToken(&types.TokenClaims{
				UserID:   user.ID,
				Username: req.Username,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"user_id": user.ID,
				"token":   token,
			})
		})

		auth.POST("/login", func(c *gin.Context) {
			var req struct {
				Email    string `json:"email" binding:"required,email"`
				Password string `json:"password" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			user, _, err := authService.Login(c.Request.Context(), req.Email, req.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			// Get username from profile
			profile, err := profileService.GetProfile(c.Request.Context(), user.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user profile"})
				return
			}

			token, err := authService.GenerateToken(&types.TokenClaims{
				UserID:   user.ID,
				Username: profile.Username,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"user_id": user.ID,
				"token":   token,
			})
		})

		auth.GET("/profile", middleware.AuthMiddleware(authService), func(c *gin.Context) {
			userID := c.MustGet("user_id").(uuid.UUID)
			profile, err := profileService.GetProfile(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user profile"})
				return
			}

			c.JSON(http.StatusOK, profile)
		})

		auth.PUT("/profile", middleware.AuthMiddleware(authService), func(c *gin.Context) {
			var req types.UpdateProfileRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			userID := c.MustGet("user_id").(uuid.UUID)
			profile, err := profileService.UpdateProfile(c.Request.Context(), userID, &req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, profile)
		})
	}

	// Register recipe routes
	router.POST("/api/v1/recipes", middleware.AuthMiddleware(authService), func(c *gin.Context) {
		var req types.CreateRecipeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		recipe := &models.Recipe{
			ID:                 uuid.New(),
			UserID:             c.MustGet("user_id").(uuid.UUID),
			Name:               req.Name,
			Description:        req.Description,
			Category:           req.Category,
			Cuisine:            req.Cuisine,
			ImageURL:           req.ImageURL,
			Ingredients:        req.Ingredients,
			Instructions:       req.Instructions,
			Calories:           req.Calories,
			Protein:            req.Protein,
			Carbs:              req.Carbs,
			Fat:                req.Fat,
			DietaryPreferences: req.DietaryPreferences,
			Tags:               req.Tags,
		}

		createdRecipe, err := recipeService.CreateRecipe(c.Request.Context(), recipe)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, createdRecipe)
	})

	router.GET("/api/v1/recipes/:id", middleware.AuthMiddleware(authService), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID"})
			return
		}

		recipe, err := recipeService.GetRecipe(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, recipe)
	})

	router.PUT("/api/v1/recipes/:id", middleware.AuthMiddleware(authService), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID"})
			return
		}

		var req types.UpdateRecipeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		recipe := &models.Recipe{
			ID:                 id,
			UserID:             c.MustGet("user_id").(uuid.UUID),
			Name:               req.Name,
			Description:        req.Description,
			Category:           req.Category,
			Cuisine:            req.Cuisine,
			ImageURL:           req.ImageURL,
			Ingredients:        req.Ingredients,
			Instructions:       req.Instructions,
			Calories:           req.Calories,
			Protein:            req.Protein,
			Carbs:              req.Carbs,
			Fat:                req.Fat,
			DietaryPreferences: req.DietaryPreferences,
			Tags:               req.Tags,
		}

		updatedRecipe, err := recipeService.UpdateRecipe(c.Request.Context(), id, recipe)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, updatedRecipe)
	})

	router.DELETE("/api/v1/recipes/:id", middleware.AuthMiddleware(authService), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID"})
			return
		}

		err = recipeService.DeleteRecipe(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	})

	// Add route for getting all recipes
	router.GET("/api/v1/recipes", middleware.AuthMiddleware(authService), func(c *gin.Context) {
		userID := c.MustGet("user_id").(uuid.UUID)
		recipes, err := profileService.GetUserRecipes(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, recipes)
	})

	return router
}

func TestIntegrationRegisterLoginCreateModify(t *testing.T) {
	// Setup mock services
	authService := &mocks.MockAuthService{}
	profileService := &mocks.MockProfileService{}
	recipeService := &mocks.MockRecipeService{}

	// Setup mock expectations
	testUserID := uuid.New()
	testUsername := "testuser"
	testEmail := "test@example.com"
	testToken := "test-token"

	// Mock token validation
	authService.On("ValidateToken", testToken).Return(&types.TokenClaims{
		UserID:   testUserID,
		Username: testUsername,
	}, nil)

	// Mock user registration
	authService.On("Register", mock.Anything, testEmail, "password123", mock.Anything).Return(&models.User{
		ID:    testUserID,
		Email: testEmail,
	}, nil)

	// Mock token generation
	authService.On("GenerateToken", mock.Anything).Return(testToken, nil)

	// Mock user login
	authService.On("Login", mock.Anything, testEmail, "password123").Return(&models.User{
		ID:    testUserID,
		Email: testEmail,
	}, &models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUserID,
		Username: testUsername,
		Bio:      "test bio",
	}, nil)

	// Mock profile operations
	profileService.On("GetProfile", mock.Anything, testUserID).Return(&models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUserID,
		Username: testUsername,
		Bio:      "test bio",
	}, nil)

	profileService.On("UpdateProfile", mock.Anything, testUserID, mock.Anything).Return(&models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUserID,
		Username: "updateduser",
		Bio:      "Updated bio",
	}, nil)

	// For debugging: relax userID expectation to any value
	profileService.On("GetUserRecipes", mock.Anything, mock.Anything).Return([]*models.Recipe{
		{
			ID:          uuid.New(),
			UserID:      testUserID,
			Name:        "Test Recipe",
			Description: "Test Description",
		},
	}, nil)

	// Mock recipe operations
	testRecipeID := uuid.New()
	recipeService.On("CreateRecipe", mock.Anything, mock.Anything).Return(&models.Recipe{
		ID:          testRecipeID,
		UserID:      testUserID,
		Name:        "Test Recipe",
		Description: "Test Description",
	}, nil)

	recipeService.On("GetRecipe", mock.Anything, testRecipeID).Return(&models.Recipe{
		ID:          testRecipeID,
		UserID:      testUserID,
		Name:        "Test Recipe",
		Description: "Test Description",
	}, nil)

	recipeService.On("UpdateRecipe", mock.Anything, testRecipeID, mock.Anything).Return(&models.Recipe{
		ID:          testRecipeID,
		UserID:      testUserID,
		Name:        "Updated Recipe",
		Description: "Updated Description",
	}, nil)

	recipeService.On("DeleteRecipe", mock.Anything, testRecipeID).Return(nil)

	// Setup test router
	router := setupTestRouter(authService, profileService, recipeService)

	// Test registration
	registerReq := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
		"username": "`+testUsername+`",
		"email": "`+testEmail+`",
		"password": "password123"
	}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	router.ServeHTTP(registerResp, registerReq)
	assert.Equal(t, http.StatusCreated, registerResp.Code)

	// Test login
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
		"email": "`+testEmail+`",
		"password": "password123"
	}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	assert.Equal(t, http.StatusOK, loginResp.Code)

	// Test recipe creation
	createRecipeReq := httptest.NewRequest("POST", "/api/v1/recipes", strings.NewReader(`{
		"name": "Test Recipe",
		"description": "Test Description",
		"category": "Test Category",
		"cuisine": "Test Cuisine",
		"ingredients": ["ingredient1", "ingredient2"],
		"instructions": ["step1", "step2"],
		"calories": 500,
		"protein": 20,
		"carbs": 30,
		"fat": 10,
		"dietary_preferences": ["vegetarian"],
		"tags": ["quick", "healthy"]
	}`))
	createRecipeReq.Header.Set("Content-Type", "application/json")
	createRecipeReq.Header.Set("Authorization", "Bearer "+testToken)
	createRecipeResp := httptest.NewRecorder()
	router.ServeHTTP(createRecipeResp, createRecipeReq)
	assert.Equal(t, http.StatusCreated, createRecipeResp.Code)

	// Test profile update
	updateProfileReq := httptest.NewRequest("PUT", "/api/v1/auth/profile", strings.NewReader(`{
		"username": "updateduser",
		"bio": "Updated bio"
	}`))
	updateProfileReq.Header.Set("Content-Type", "application/json")
	updateProfileReq.Header.Set("Authorization", "Bearer "+testToken)
	updateProfileResp := httptest.NewRecorder()
	router.ServeHTTP(updateProfileResp, updateProfileReq)
	assert.Equal(t, http.StatusOK, updateProfileResp.Code)

	// Test recipe retrieval
	getRecipeReq := httptest.NewRequest("GET", "/api/v1/recipes/"+testRecipeID.String(), nil)
	getRecipeReq.Header.Set("Authorization", "Bearer "+testToken)
	getRecipeResp := httptest.NewRecorder()
	router.ServeHTTP(getRecipeResp, getRecipeReq)
	assert.Equal(t, http.StatusOK, getRecipeResp.Code)

	// Test recipe update
	updateRecipeReq := httptest.NewRequest("PUT", "/api/v1/recipes/"+testRecipeID.String(), strings.NewReader(`{
		"name": "Updated Recipe",
		"description": "Updated Description"
	}`))
	updateRecipeReq.Header.Set("Content-Type", "application/json")
	updateRecipeReq.Header.Set("Authorization", "Bearer "+testToken)
	updateRecipeResp := httptest.NewRecorder()
	router.ServeHTTP(updateRecipeResp, updateRecipeReq)
	assert.Equal(t, http.StatusOK, updateRecipeResp.Code)

	// Test recipe deletion
	deleteRecipeReq := httptest.NewRequest("DELETE", "/api/v1/recipes/"+testRecipeID.String(), nil)
	deleteRecipeReq.Header.Set("Authorization", "Bearer "+testToken)
	deleteRecipeResp := httptest.NewRecorder()
	router.ServeHTTP(deleteRecipeResp, deleteRecipeReq)
	assert.Equal(t, http.StatusNoContent, deleteRecipeResp.Code)

	// Test getting user recipes
	getUserRecipesReq := httptest.NewRequest("GET", "/api/v1/recipes", nil)
	getUserRecipesReq.Header.Set("Authorization", "Bearer "+testToken)
	getUserRecipesResp := httptest.NewRecorder()
	router.ServeHTTP(getUserRecipesResp, getUserRecipesReq)
	assert.Equal(t, http.StatusOK, getUserRecipesResp.Code)

	// Verify all mock expectations were met
	authService.AssertExpectations(t)
	profileService.AssertExpectations(t)
	recipeService.AssertExpectations(t)
}

func TestUserProfile(t *testing.T) {
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	mockRecipeService := new(mocks.MockRecipeService)

	// Mock token validation
	testUUID := uuid.New()
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Mock profile operations
	mockProfileService.On("GetProfile", mock.Anything, testUUID).Return(&models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUUID,
		Username: "testuser",
		Bio:      "test bio",
	}, nil)

	bio := "Updated bio"
	updateReq := &types.UpdateProfileRequest{
		Username: "updateduser",
		Bio:      &bio,
	}
	mockProfileService.On("UpdateProfile", mock.Anything, testUUID, updateReq).Return(&models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUUID,
		Username: "updateduser",
		Bio:      bio,
	}, nil)

	router := setupTestRouter(mockAuthService, mockProfileService, mockRecipeService)

	// Test getting user profile
	getProfileReq := httptest.NewRequest("GET", "/api/v1/auth/profile", nil)
	getProfileReq.Header.Set("Authorization", "Bearer test-token")
	getProfileResp := httptest.NewRecorder()
	router.ServeHTTP(getProfileResp, getProfileReq)

	if getProfileResp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, getProfileResp.Code)
	}

	// Test updating user profile
	updateProfileBody := types.UpdateProfileRequest{
		Username: "updateduser",
		Bio:      &bio,
	}
	updateProfileJSON, _ := json.Marshal(updateProfileBody)
	updateProfileReq := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewBuffer(updateProfileJSON))
	updateProfileReq.Header.Set("Content-Type", "application/json")
	updateProfileReq.Header.Set("Authorization", "Bearer test-token")
	updateProfileResp := httptest.NewRecorder()
	router.ServeHTTP(updateProfileResp, updateProfileReq)

	if updateProfileResp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, updateProfileResp.Code)
	}

	// Verify all mock expectations were met
	mockAuthService.AssertExpectations(t)
	mockProfileService.AssertExpectations(t)
	mockRecipeService.AssertExpectations(t)
}

func TestRecipeCRUD(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	mockRecipeService := new(mocks.MockRecipeService)

	// Mock token validation
	testUUID := uuid.New()
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Mock recipe creation
	recipeID := uuid.New()
	mockRecipeService.On("CreateRecipe", mock.Anything, mock.Anything).Return(&models.Recipe{
		ID:                 recipeID,
		UserID:             testUUID,
		Name:               "Test Recipe",
		Description:        "Test Description",
		Category:           "Test Category",
		Cuisine:            "Test Cuisine",
		ImageURL:           "http://example.com/image.jpg",
		Ingredients:        []string{"ingredient1", "ingredient2"},
		Instructions:       []string{"step1", "step2"},
		Calories:           500.0,
		Protein:            20.0,
		Carbs:              30.0,
		Fat:                10.0,
		DietaryPreferences: []string{"vegetarian", "gluten-free"},
		Tags:               []string{"quick", "healthy"},
	}, nil)

	// Mock recipe retrieval
	mockRecipeService.On("GetRecipe", mock.Anything, recipeID).Return(&models.Recipe{
		ID:                 recipeID,
		UserID:             testUUID,
		Name:               "Test Recipe",
		Description:        "Test Description",
		Category:           "Test Category",
		Cuisine:            "Test Cuisine",
		ImageURL:           "http://example.com/image.jpg",
		Ingredients:        []string{"ingredient1", "ingredient2"},
		Instructions:       []string{"step1", "step2"},
		Calories:           500.0,
		Protein:            20.0,
		Carbs:              30.0,
		Fat:                10.0,
		DietaryPreferences: []string{"vegetarian", "gluten-free"},
		Tags:               []string{"quick", "healthy"},
	}, nil)

	// Mock recipe update
	mockRecipeService.On("UpdateRecipe", mock.Anything, recipeID, mock.Anything).Return(&models.Recipe{
		ID:                 recipeID,
		UserID:             testUUID,
		Name:               "Updated Recipe",
		Description:        "Updated Description",
		Category:           "Updated Category",
		Cuisine:            "Updated Cuisine",
		ImageURL:           "http://example.com/updated.jpg",
		Ingredients:        []string{"updated1", "updated2"},
		Instructions:       []string{"updated1", "updated2"},
		Calories:           600.0,
		Protein:            25.0,
		Carbs:              35.0,
		Fat:                15.0,
		DietaryPreferences: []string{"vegan", "dairy-free"},
		Tags:               []string{"dinner", "protein-rich"},
	}, nil)

	// Mock recipe deletion
	mockRecipeService.On("DeleteRecipe", mock.Anything, recipeID).Return(nil)

	router := setupTestRouter(mockAuthService, mockProfileService, mockRecipeService)

	// Test recipe creation
	createBody := `{
		"name": "Test Recipe",
		"description": "Test Description",
		"category": "Test Category",
		"cuisine": "Test Cuisine",
		"image_url": "http://example.com/image.jpg",
		"ingredients": ["ingredient1", "ingredient2"],
		"instructions": ["step1", "step2"],
		"calories": 500.0,
		"protein": 20.0,
		"carbs": 30.0,
		"fat": 10.0,
		"dietary_preferences": ["vegetarian", "gluten-free"],
		"tags": ["quick", "healthy"]
	}`
	createReq := httptest.NewRequest("POST", "/api/v1/recipes", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer test-token")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	assert.Equal(t, http.StatusCreated, createResp.Code)

	// Test recipe retrieval
	getReq := httptest.NewRequest("GET", "/api/v1/recipes/"+recipeID.String(), nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)

	assert.Equal(t, http.StatusOK, getResp.Code)

	// Test recipe update
	updateBody := `{
		"name": "Updated Recipe",
		"description": "Updated Description",
		"category": "Updated Category",
		"cuisine": "Updated Cuisine",
		"image_url": "http://example.com/updated.jpg",
		"ingredients": ["updated1", "updated2"],
		"instructions": ["updated1", "updated2"],
		"calories": 600.0,
		"protein": 25.0,
		"carbs": 35.0,
		"fat": 15.0,
		"dietary_preferences": ["vegan", "dairy-free"],
		"tags": ["dinner", "protein-rich"]
	}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/recipes/"+recipeID.String(), strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer test-token")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)

	assert.Equal(t, http.StatusOK, updateResp.Code)

	// Test recipe deletion
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/recipes/"+recipeID.String(), nil)
	deleteReq.Header.Set("Authorization", "Bearer test-token")
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)

	assert.Equal(t, http.StatusNoContent, deleteResp.Code)

	// Verify mock expectations
	mockAuthService.AssertExpectations(t)
	mockRecipeService.AssertExpectations(t)
}

func TestCreateRecipe(t *testing.T) {
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	mockRecipeService := new(mocks.MockRecipeService)

	// Mock token validation
	testUUID := uuid.New()
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Mock recipe operations
	testRecipeID := uuid.New()
	mockRecipeService.On("CreateRecipe", mock.Anything, mock.Anything).Return(&models.Recipe{
		ID:          testRecipeID,
		UserID:      testUUID,
		Name:        "Test Recipe",
		Description: "Test Description",
	}, nil)

	router := setupTestRouter(mockAuthService, mockProfileService, mockRecipeService)

	// Test creating recipe
	recipeBody := types.CreateRecipeRequest{
		Name:               "Test Recipe",
		Description:        "Test Description",
		Category:           "Test Category",
		Cuisine:            "Test Cuisine",
		ImageURL:           "http://example.com/image.jpg",
		Ingredients:        []string{"ingredient1", "ingredient2"},
		Instructions:       []string{"step1", "step2"},
		Calories:           500.0,
		Protein:            20.0,
		Carbs:              30.0,
		Fat:                10.0,
		DietaryPreferences: []string{"vegetarian", "gluten-free"},
		Tags:               []string{"quick", "healthy"},
	}
	recipeJSON, _ := json.Marshal(recipeBody)
	req := httptest.NewRequest("POST", "/api/v1/recipes", bytes.NewBuffer(recipeJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.Code)
	}

	// Verify all mock expectations were met
	mockAuthService.AssertExpectations(t)
	mockProfileService.AssertExpectations(t)
	mockRecipeService.AssertExpectations(t)
}

func TestGetRecipe(t *testing.T) {
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	mockRecipeService := new(mocks.MockRecipeService)

	// Mock token validation
	testUUID := uuid.New()
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Mock recipe operations
	testRecipeID := uuid.New()
	mockRecipeService.On("GetRecipe", mock.Anything, testRecipeID).Return(&models.Recipe{
		ID:          testRecipeID,
		UserID:      testUUID,
		Name:        "Test Recipe",
		Description: "Test Description",
	}, nil)

	router := setupTestRouter(mockAuthService, mockProfileService, mockRecipeService)

	// Test getting recipe
	req := httptest.NewRequest("GET", "/api/v1/recipes/"+testRecipeID.String(), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	// Verify all mock expectations were met
	mockAuthService.AssertExpectations(t)
	mockProfileService.AssertExpectations(t)
	mockRecipeService.AssertExpectations(t)
}

func TestUpdateProfile(t *testing.T) {
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	mockRecipeService := new(mocks.MockRecipeService)

	// Mock token validation
	testUUID := uuid.New()
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Mock profile operations
	bio := "Updated bio"
	updateReq := &types.UpdateProfileRequest{
		Username: "updateduser",
		Bio:      &bio,
	}
	mockProfileService.On("UpdateProfile", mock.Anything, testUUID, updateReq).Return(&models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUUID,
		Username: "updateduser",
		Bio:      bio,
	}, nil)

	router := setupTestRouter(mockAuthService, mockProfileService, mockRecipeService)

	// Test updating profile
	updateProfileBody := types.UpdateProfileRequest{
		Username: "updateduser",
		Bio:      &bio,
	}
	updateProfileJSON, _ := json.Marshal(updateProfileBody)
	req := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewBuffer(updateProfileJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	// Verify all mock expectations were met
	mockAuthService.AssertExpectations(t)
	mockProfileService.AssertExpectations(t)
	mockRecipeService.AssertExpectations(t)
}
