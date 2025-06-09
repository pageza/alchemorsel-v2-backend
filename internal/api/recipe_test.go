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
	"github.com/stretchr/testify/assert"
)

func TestCreateRecipe(t *testing.T) {
	router, testDB := setupRecipeTestRouter(t)

	// Create test user and get token
	_, token := CreateTestUserAndToken(t, testDB)

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

	updateRecipe := map[string]interface{}{
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
		"dietary_preferences": []string{"vegan", "gluten-free"},
		"tags":                []string{"quick", "healthy", "updated"},
	}

	jsonData, err = json.Marshal(updateRecipe)
	if err != nil {
		t.Fatalf("Failed to marshal update recipe: %v", err)
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

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "recipes")
	recipesList := response["recipes"].([]interface{})
	assert.Len(t, recipesList, 2)
}

func setupRecipeTestRouter(t *testing.T) (*gin.Engine, *TestDB) {
	// Setup test database
	testDB := SetupTestDB(t)
	router := gin.Default()

	// Register recipe routes
	recipeService := service.NewRecipeService(testDB.DB, nil)
	recipeHandler := &RecipeHandler{recipeService: recipeService, authService: testDB.AuthService}
	api := router.Group("/api/v1")
	{
		recipes := api.Group("/recipes")
		recipes.Use(middleware.AuthMiddleware(testDB.AuthService))
		{
			recipes.GET("", recipeHandler.ListRecipes)
			recipes.GET(":id", recipeHandler.GetRecipe)
			recipes.POST("", recipeHandler.CreateRecipe)
			recipes.PUT(":id", recipeHandler.UpdateRecipe)
			recipes.DELETE(":id", recipeHandler.DeleteRecipe)
			recipes.POST(":id/favorite", recipeHandler.FavoriteRecipe)
			recipes.DELETE(":id/favorite", recipeHandler.UnfavoriteRecipe)
		}
	}

	return router, testDB
}
