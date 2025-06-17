package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

// Simple mock embedding service for tests
type mockEmbeddingService struct{}

func (m *mockEmbeddingService) GenerateEmbedding(text string) (pgvector.Vector, error) {
	// Return a simple dummy vector for testing
	vec := make([]float32, 1536)
	for i := range vec {
		vec[i] = 0.1
	}
	return pgvector.NewVector(vec), nil
}

func (m *mockEmbeddingService) GenerateEmbeddingFromRecipe(name, description string, ingredients []string, category string, dietary []string) (pgvector.Vector, error) {
	return m.GenerateEmbedding(name + " " + description)
}

func TestCreateRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	recipe := map[string]interface{}{
		"name":                "Test Recipe",
		"description":         "Test Description",
		"category":            "Test Category",
		"cuisine":             "Test Cuisine",
		"image_url":           "http://example.com/image.jpg",
		"ingredients":         []string{"ingredient1", "ingredient2"},
		"instructions":        []string{"step1", "step2"},
		"calories":            500,
		"protein":             20,
		"carbs":               30,
		"fat":                 10,
		"dietary_preferences": []string{"vegetarian", "gluten-free"},
		"tags":                []string{"quick", "healthy"},
		"embedding":           defaultEmbedding,
	}

	// Create request with auth token
	req := httptest.NewRequest("POST", "/api/v1/recipes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Marshal recipe to JSON
	jsonData, err := json.Marshal(recipe)
	if err != nil {
		t.Fatalf("Failed to marshal recipe: %v", err)
	}
	req.Body = io.NopCloser(bytes.NewBuffer(jsonData))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "recipe")
	recipeData := response["recipe"].(map[string]interface{})
	assert.Contains(t, recipeData, "id")
}

func TestGetRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	_, token := CreateTestUserAndToken(t, testDB)

	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	recipe := map[string]interface{}{
		"name":                "Test Recipe",
		"description":         "Test Description",
		"category":            "Test Category",
		"cuisine":             "Test Cuisine",
		"image_url":           "http://example.com/image.jpg",
		"ingredients":         []string{"ingredient1", "ingredient2"},
		"instructions":        []string{"step1", "step2"},
		"calories":            500,
		"protein":             20,
		"carbs":               30,
		"fat":                 10,
		"dietary_preferences": []string{"vegetarian", "gluten-free"},
		"tags":                []string{"quick", "healthy"},
		"embedding":           defaultEmbedding,
	}

	// Create recipe
	jsonData, err := json.Marshal(recipe)
	if err != nil {
		t.Fatalf("Failed to marshal recipe: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/v1/recipes", io.NopCloser(bytes.NewBuffer(jsonData)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipeData := response["recipe"].(map[string]interface{})
	recipeID := recipeData["id"].(string)

	// Get the recipe
	req = httptest.NewRequest("GET", "/api/v1/recipes/"+recipeID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestUpdateRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	_, token := CreateTestUserAndToken(t, testDB)

	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	recipe := map[string]interface{}{
		"name":                "Test Recipe",
		"description":         "Test Description",
		"category":            "Test Category",
		"cuisine":             "Test Cuisine",
		"image_url":           "http://example.com/image.jpg",
		"ingredients":         []string{"ingredient1", "ingredient2"},
		"instructions":        []string{"step1", "step2"},
		"calories":            500,
		"protein":             20,
		"carbs":               30,
		"fat":                 10,
		"dietary_preferences": []string{"vegetarian", "gluten-free"},
		"tags":                []string{"quick", "healthy"},
		"embedding":           defaultEmbedding,
	}

	// Create recipe
	jsonData, err := json.Marshal(recipe)
	if err != nil {
		t.Fatalf("Failed to marshal recipe: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/v1/recipes", io.NopCloser(bytes.NewBuffer(jsonData)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipeData := response["recipe"].(map[string]interface{})
	recipeID := recipeData["id"].(string)

	// Update recipe
	updatedRecipe := map[string]interface{}{
		"name":                "Updated Recipe",
		"description":         "Updated Description",
		"category":            "Updated Category",
		"cuisine":             "Updated Cuisine",
		"image_url":           "http://example.com/updated.jpg",
		"ingredients":         []string{"updated1", "updated2"},
		"instructions":        []string{"updated1", "updated2"},
		"calories":            600,
		"protein":             25,
		"carbs":               35,
		"fat":                 15,
		"dietary_preferences": []string{"vegan", "dairy-free"},
		"tags":                []string{"dinner", "protein-rich"},
		"embedding":           defaultEmbedding,
	}

	jsonData, err = json.Marshal(updatedRecipe)
	if err != nil {
		t.Fatalf("Failed to marshal updated recipe: %v", err)
	}
	req = httptest.NewRequest("PUT", "/api/v1/recipes/"+recipeID, io.NopCloser(bytes.NewBuffer(jsonData)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestDeleteRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	// Create a recipe first
	createRecipe := map[string]interface{}{
		"name":                "Test Recipe",
		"description":         "Test Description",
		"category":            "Test Category",
		"cuisine":             "Test Cuisine",
		"image_url":           "http://example.com/image.jpg",
		"ingredients":         []string{"ingredient1", "ingredient2"},
		"instructions":        []string{"step1", "step2"},
		"calories":            500,
		"protein":             20,
		"carbs":               30,
		"fat":                 10,
		"dietary_preferences": []string{"vegetarian", "gluten-free"},
		"tags":                []string{"quick", "healthy"},
		"embedding":           defaultEmbedding,
	}

	// Create recipe with auth token
	jsonData, err := json.Marshal(createRecipe)
	if err != nil {
		t.Fatalf("Failed to marshal recipe: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/v1/recipes", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipeData := response["recipe"].(map[string]interface{})
	recipeID := recipeData["id"].(string)

	// Delete the recipe with auth token
	req = httptest.NewRequest("DELETE", "/api/v1/recipes/"+recipeID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	// Verify the recipe was deleted with auth token
	req = httptest.NewRequest("GET", "/api/v1/recipes/"+recipeID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestListRecipes(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	// Create test recipes
	recipes := []map[string]interface{}{
		{
			"name":                "Test Recipe 1",
			"description":         "Test Description 1",
			"category":            "Category 1",
			"cuisine":             "Cuisine 1",
			"image_url":           "http://example.com/image1.jpg",
			"ingredients":         []string{"ingredient1", "ingredient2"},
			"instructions":        []string{"step1", "step2"},
			"calories":            500,
			"protein":             20,
			"carbs":               30,
			"fat":                 10,
			"dietary_preferences": []string{"vegetarian", "gluten-free"},
			"tags":                []string{"quick", "healthy"},
			"embedding":           defaultEmbedding,
		},
		{
			"name":                "Test Recipe 2",
			"description":         "Test Description 2",
			"category":            "Category 2",
			"cuisine":             "Cuisine 2",
			"image_url":           "http://example.com/image2.jpg",
			"ingredients":         []string{"ingredient3", "ingredient4"},
			"instructions":        []string{"step3", "step4"},
			"calories":            600,
			"protein":             25,
			"carbs":               35,
			"fat":                 15,
			"dietary_preferences": []string{"vegan", "dairy-free"},
			"tags":                []string{"dinner", "protein-rich"},
			"embedding":           defaultEmbedding,
		},
	}

	// Create recipes with auth token
	for _, recipe := range recipes {
		jsonData, err := json.Marshal(recipe)
		if err != nil {
			t.Fatalf("Failed to marshal recipe: %v", err)
		}
		req := httptest.NewRequest("POST", "/api/v1/recipes", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, 201, w.Code)
	}

	// Test listing recipes with auth token
	req := httptest.NewRequest("GET", "/api/v1/recipes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func setupRecipeTestRouter(t *testing.T) (*gin.Engine, *TestDB) {
	// Setup test database
	testDB := SetupTestDB(t)
	router := gin.Default()

	// Create a mock embedding service
	mockEmbeddingService := &mockEmbeddingService{}
	
	// Register recipe routes
	recipeService := service.NewRecipeService(testDB.DB, mockEmbeddingService)
	recipeHandler := &RecipeHandler{recipeService: recipeService, authService: testDB.AuthService}
	api := router.Group("/api/v1")
	{
		recipes := api.Group("/recipes")
		// Public routes (no auth middleware)
		recipes.GET("", recipeHandler.ListRecipes)
		recipes.GET("/search", recipeHandler.SearchRecipes)
		recipes.GET("/:id", recipeHandler.GetRecipe)
		
		// Protected routes (with auth middleware)
		protected := recipes.Group("")
		protected.Use(middleware.AuthMiddleware(testDB.AuthService))
		{
			protected.POST("", recipeHandler.CreateRecipe)
			protected.PUT("/:id", recipeHandler.UpdateRecipe)
			protected.DELETE("/:id", recipeHandler.DeleteRecipe)
			protected.POST("/:id/favorite", recipeHandler.FavoriteRecipe)
			protected.DELETE("/:id/favorite", recipeHandler.UnfavoriteRecipe)
		}
	}

	return router, testDB
}

func TestFavoriteRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a test recipe first
	recipeID := createTestRecipe(t, router, token)

	// Test adding to favorites
	req := httptest.NewRequest("POST", "/api/v1/recipes/"+recipeID+"/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Recipe added to favorites", response["message"])
	assert.Equal(t, true, response["is_favorite"])

	// Test adding the same recipe to favorites again (should return 409)
	req = httptest.NewRequest("POST", "/api/v1/recipes/"+recipeID+"/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 409, w.Code)
}

func TestUnfavoriteRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a test recipe first
	recipeID := createTestRecipe(t, router, token)

	// Add to favorites first
	req := httptest.NewRequest("POST", "/api/v1/recipes/"+recipeID+"/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Test removing from favorites
	req = httptest.NewRequest("DELETE", "/api/v1/recipes/"+recipeID+"/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Recipe removed from favorites", response["message"])
	assert.Equal(t, false, response["is_favorite"])

	// Test removing from favorites again (should return 404)
	req = httptest.NewRequest("DELETE", "/api/v1/recipes/"+recipeID+"/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestFavoriteRecipeNotFound(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Test favoriting a non-existent recipe with valid UUID format
	req := httptest.NewRequest("POST", "/api/v1/recipes/00000000-0000-0000-0000-000000000000/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestFavoriteRecipeUnauthorized(t *testing.T) {
	router, _ := setupRecipeTestRouter(t)

	// Test without authentication
	req := httptest.NewRequest("POST", "/api/v1/recipes/some-id/favorite", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)
}

func TestGetRecipeWithFavoriteStatus(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a test recipe
	recipeID := createTestRecipe(t, router, token)

	// Get recipe (should not be favorite initially)
	req := httptest.NewRequest("GET", "/api/v1/recipes/"+recipeID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipeData := response["recipe"].(map[string]interface{})
	assert.Equal(t, false, recipeData["is_favorite"])

	// Add to favorites
	req = httptest.NewRequest("POST", "/api/v1/recipes/"+recipeID+"/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Get recipe again (should be favorite now)
	req = httptest.NewRequest("GET", "/api/v1/recipes/"+recipeID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipeData = response["recipe"].(map[string]interface{})
	assert.Equal(t, true, recipeData["is_favorite"])
}

// Helper function to create a test recipe and return its ID
func createTestRecipe(t *testing.T, router *gin.Engine, token string) string {
	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	recipe := map[string]interface{}{
		"name":                "Test Recipe",
		"description":         "Test Description",
		"category":            "Test Category",
		"cuisine":             "Test Cuisine",
		"image_url":           "http://example.com/image.jpg",
		"ingredients":         []string{"ingredient1", "ingredient2"},
		"instructions":        []string{"step1", "step2"},
		"calories":            500,
		"protein":             20,
		"carbs":               30,
		"fat":                 10,
		"dietary_preferences": []string{"vegetarian", "gluten-free"},
		"tags":                []string{"quick", "healthy"},
		"embedding":           defaultEmbedding,
	}

	jsonData, err := json.Marshal(recipe)
	if err != nil {
		t.Fatalf("Failed to marshal recipe: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/recipes", io.NopCloser(bytes.NewBuffer(jsonData)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("Failed to create test recipe: %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	recipeData := response["recipe"].(map[string]interface{})
	return recipeData["id"].(string)
}

func TestSearchRecipes(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

	// Create a default embedding vector with 1536 dimensions
	defaultEmbedding := make([]float32, 1536)
	for i := range defaultEmbedding {
		defaultEmbedding[i] = float32(i) / 1536.0
	}

	// Create test recipes with different names and categories
	recipes := []map[string]interface{}{
		{
			"name":                "Pasta Carbonara",
			"description":         "Creamy pasta dish",
			"category":            "Dinner",
			"cuisine":             "Italian",
			"image_url":           "http://example.com/pasta.jpg",
			"ingredients":         []string{"pasta", "bacon", "eggs"},
			"instructions":        []string{"Cook pasta", "Mix with sauce"},
			"calories":            500,
			"protein":             20,
			"carbs":               60,
			"fat":                 15,
			"dietary_preferences": []string{},
			"tags":                []string{"quick", "dinner"},
			"embedding":           defaultEmbedding,
		},
		{
			"name":                "Chicken Salad",
			"description":         "Fresh healthy salad",
			"category":            "Lunch",
			"cuisine":             "American",
			"image_url":           "http://example.com/salad.jpg",
			"ingredients":         []string{"chicken", "lettuce", "tomato"},
			"instructions":        []string{"Grill chicken", "Mix salad"},
			"calories":            300,
			"protein":             25,
			"carbs":               10,
			"fat":                 12,
			"dietary_preferences": []string{"gluten-free"},
			"tags":                []string{"healthy", "lunch"},
			"embedding":           defaultEmbedding,
		},
	}

	// Create recipes
	for _, recipe := range recipes {
		jsonData, err := json.Marshal(recipe)
		if err != nil {
			t.Fatalf("Failed to marshal recipe: %v", err)
		}
		req := httptest.NewRequest("POST", "/api/v1/recipes", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, 201, w.Code)
	}

	// Test search with query parameter
	req := httptest.NewRequest("GET", "/api/v1/recipes/search?q=pasta", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "recipes")
	recipes_result := response["recipes"].([]interface{})
	assert.GreaterOrEqual(t, len(recipes_result), 1) // Should find at least the pasta recipe

	// Test search with category filter
	req = httptest.NewRequest("GET", "/api/v1/recipes/search?category=Dinner", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipes_result = response["recipes"].([]interface{})
	assert.GreaterOrEqual(t, len(recipes_result), 1) // Should find the dinner recipe

	// Test search with sort parameter
	req = httptest.NewRequest("GET", "/api/v1/recipes/search?sort=name", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipes_result = response["recipes"].([]interface{})
	assert.GreaterOrEqual(t, len(recipes_result), 2) // Should find both recipes sorted by name

	// Test search with no parameters (should behave like ListRecipes)
	req = httptest.NewRequest("GET", "/api/v1/recipes/search", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	recipes_result = response["recipes"].([]interface{})
	assert.GreaterOrEqual(t, len(recipes_result), 2) // Should find all recipes
}
