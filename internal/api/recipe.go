package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	pgvector "github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// RecipeHandler handles recipe-related requests
type RecipeHandler struct {
	recipeService    service.IRecipeService
	authService      service.IAuthService
	llmService       service.LLMServiceInterface
	embeddingService service.EmbeddingServiceInterface
}

// NewRecipeHandler creates a new RecipeHandler
func NewRecipeHandler(recipeService service.IRecipeService, authService service.IAuthService, llmService service.LLMServiceInterface, embeddingService service.EmbeddingServiceInterface) *RecipeHandler {
	return &RecipeHandler{
		recipeService:    recipeService,
		authService:      authService,
		llmService:       llmService,
		embeddingService: embeddingService,
	}
}

// RegisterRoutes registers the recipe routes
func (h *RecipeHandler) RegisterRoutes(router *gin.RouterGroup) {
	recipes := router.Group("/recipes")
	recipes.Use(middleware.AuthMiddleware(h.authService))
	{
		recipes.GET("", h.ListRecipes)
		recipes.GET("/:id", h.GetRecipe)
		recipes.POST("", h.CreateRecipe)
		recipes.PUT("/:id", h.UpdateRecipe)
		recipes.DELETE("/:id", h.DeleteRecipe)
		recipes.POST("/:id/favorite", h.FavoriteRecipe)
		recipes.DELETE("/:id/favorite", h.UnfavoriteRecipe)
	}
}

// CreateRecipe handles creating a new recipe
func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	println("[DEBUG] CreateRecipe called")
	fmt.Printf("[DEBUG] Context keys: %v\n", c.Keys)
	userIDVal, exists := c.Get("user_id")
	fmt.Printf("[DEBUG] user_id value: %#v\n", userIDVal)
	if !exists {
		fmt.Println("[DEBUG] user_id missing from context. Responding 401 Unauthorized.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		fmt.Printf("[DEBUG] user_id is not uuid.UUID, got: %T\n", userIDVal)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	fmt.Printf("[DEBUG] user_id: %s\n", userID.String())
	var req struct {
		Name               string    `json:"name" binding:"required"`
		Description        string    `json:"description" binding:"required"`
		Category           string    `json:"category" binding:"required"`
		Cuisine            string    `json:"cuisine"`
		ImageURL           string    `json:"image_url"`
		Ingredients        []string  `json:"ingredients" binding:"required"`
		Instructions       []string  `json:"instructions" binding:"required"`
		Calories           float64   `json:"calories"`
		Protein            float64   `json:"protein"`
		Carbs              float64   `json:"carbs"`
		Fat                float64   `json:"fat"`
		DietaryPreferences []string  `json:"dietary_preferences"`
		Tags               []string  `json:"tags"`
		Embedding          []float32 `json:"embedding"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[DEBUG] Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a copy of the request for logging without the full embedding
	logReq := req
	if len(logReq.Embedding) > 0 {
		// Only show first and last value of the embedding vector
		logReq.Embedding = []float32{logReq.Embedding[0], logReq.Embedding[len(logReq.Embedding)-1]}
	}
	fmt.Printf("[DEBUG] Parsed CreateRecipeRequest: {Name:%s Description:%s Category:%s Cuisine:%s ImageURL:%s Ingredients:%v Instructions:%v Calories:%v Protein:%v Carbs:%v Fat:%v DietaryPreferences:%v Tags:%v Embedding:%v}\n",
		logReq.Name, logReq.Description, logReq.Category, logReq.Cuisine, logReq.ImageURL,
		logReq.Ingredients, logReq.Instructions, logReq.Calories, logReq.Protein, logReq.Carbs, logReq.Fat,
		logReq.DietaryPreferences, logReq.Tags, logReq.Embedding)

	// Create embedding vector if not provided
	var embedding pgvector.Vector
	if len(req.Embedding) > 0 {
		embedding = pgvector.NewVector(req.Embedding)
	} else {
		// Generate embedding from recipe data
		var err error
		embedding, err = h.embeddingService.GenerateEmbeddingFromRecipe(
			req.Name,
			req.Description,
			req.Ingredients,
			req.Category,
			req.DietaryPreferences,
		)
		if err != nil {
			fmt.Printf("[DEBUG] Error generating embedding: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate embedding"})
			return
		}
	}

	recipe := &models.Recipe{
		Name:               req.Name,
		Description:        req.Description,
		Category:           req.Category,
		Cuisine:            req.Cuisine,
		ImageURL:           req.ImageURL,
		Ingredients:        models.JSONBStringArray(req.Ingredients),
		Instructions:       models.JSONBStringArray(req.Instructions),
		Calories:           req.Calories,
		Protein:            req.Protein,
		Carbs:              req.Carbs,
		Fat:                req.Fat,
		DietaryPreferences: models.JSONBStringArray(req.DietaryPreferences),
		Tags:               models.JSONBStringArray(req.Tags),
		UserID:             userID,
		Embedding:          embedding,
	}

	createdRecipe, err := h.recipeService.CreateRecipe(c.Request.Context(), recipe)
	if err != nil {
		fmt.Printf("[DEBUG] Error creating recipe: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fmt.Printf("[DEBUG] Successfully created recipe. Recipe ID: %s\n", createdRecipe.ID)
	c.JSON(http.StatusCreated, gin.H{"recipe": createdRecipe})
}

// GetRecipe handles getting a single recipe
func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	println("[DEBUG] GetRecipe called")
	fmt.Printf("[DEBUG] Context keys: %v\n", c.Keys)
	userIDVal, exists := c.Get("user_id")
	fmt.Printf("[DEBUG] user_id value: %#v\n", userIDVal)
	if !exists {
		fmt.Println("[DEBUG] user_id missing from context. Responding 401 Unauthorized.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		fmt.Printf("[DEBUG] user_id is not uuid.UUID, got: %T\n", userIDVal)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	fmt.Printf("[DEBUG] user_id: %s\n", userID.String())
	id := c.Param("id")
	fmt.Printf("[DEBUG] Recipe ID param: %s\n", id)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
		return
	}

	recipeID, err := uuid.Parse(id)
	if err != nil {
		fmt.Printf("[DEBUG] Invalid recipe ID format: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID format"})
		return
	}

	recipe, err := h.recipeService.GetRecipe(c.Request.Context(), recipeID)
	if err != nil {
		fmt.Printf("[DEBUG] Error getting recipe: %v\n", err)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log recipe details without the embedding vector
	fmt.Printf("[DEBUG] Successfully retrieved recipe: {ID:%s Name:%s Description:%s Category:%s Cuisine:%s ImageURL:%s Ingredients:%v Instructions:%v Calories:%v Protein:%v Carbs:%v Fat:%v UserID:%s DietaryPreferences:%v Tags:%v}\n",
		recipe.ID, recipe.Name, recipe.Description, recipe.Category, recipe.Cuisine, recipe.ImageURL,
		recipe.Ingredients, recipe.Instructions, recipe.Calories, recipe.Protein, recipe.Carbs, recipe.Fat,
		recipe.UserID, recipe.DietaryPreferences, recipe.Tags)

	c.JSON(http.StatusOK, gin.H{"recipe": recipe})
}

// UpdateRecipe handles updating a recipe
func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
		return
	}

	recipeID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID"})
		return
	}

	var req struct {
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		Category           string   `json:"category"`
		Cuisine            string   `json:"cuisine"`
		ImageURL           string   `json:"image_url"`
		Ingredients        []string `json:"ingredients"`
		Instructions       []string `json:"instructions"`
		Calories           float64  `json:"calories"`
		Protein            float64  `json:"protein"`
		Carbs              float64  `json:"carbs"`
		Fat                float64  `json:"fat"`
		DietaryPreferences []string `json:"dietary_preferences"`
		Tags               []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipe := &models.Recipe{
		Name:               req.Name,
		Description:        req.Description,
		Category:           req.Category,
		Cuisine:            req.Cuisine,
		ImageURL:           req.ImageURL,
		Ingredients:        models.JSONBStringArray(req.Ingredients),
		Instructions:       models.JSONBStringArray(req.Instructions),
		Calories:           req.Calories,
		Protein:            req.Protein,
		Carbs:              req.Carbs,
		Fat:                req.Fat,
		DietaryPreferences: models.JSONBStringArray(req.DietaryPreferences),
		Tags:               models.JSONBStringArray(req.Tags),
	}

	updatedRecipe, err := h.recipeService.UpdateRecipe(c.Request.Context(), recipeID, recipe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedRecipe)
}

// DeleteRecipe handles deleting a recipe
func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
		return
	}

	recipeID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID"})
		return
	}

	err = h.recipeService.DeleteRecipe(c.Request.Context(), recipeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		fmt.Printf("[DEBUG] Error deleting recipe: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListRecipes handles listing recipes with optional filters
func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	recipes, err := h.recipeService.ListRecipes(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recipes": recipes})
}

// FavoriteRecipe handles favoriting a recipe
func (h *RecipeHandler) FavoriteRecipe(c *gin.Context) {
	// TODO: Implement favorite functionality
	c.Status(http.StatusNotImplemented)
}

// UnfavoriteRecipe handles unfavoriting a recipe
func (h *RecipeHandler) UnfavoriteRecipe(c *gin.Context) {
	// TODO: Implement unfavorite functionality
	c.Status(http.StatusNotImplemented)
}
