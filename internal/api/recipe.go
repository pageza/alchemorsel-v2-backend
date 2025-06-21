package api

import (
	"fmt"
	"net/http"
	"strings"

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
	recipeService           service.IRecipeService
	authService             service.IAuthService
	llmService              service.LLMServiceInterface
	embeddingService        service.EmbeddingServiceInterface
	db                      *gorm.DB
	creationRateLimiter     *middleware.RateLimiter
	modificationRateLimiter *middleware.RateLimiter
}

// NewRecipeHandler creates a new RecipeHandler
func NewRecipeHandler(recipeService service.IRecipeService, authService service.IAuthService, llmService service.LLMServiceInterface, embeddingService service.EmbeddingServiceInterface, db *gorm.DB) *RecipeHandler {
	return &RecipeHandler{
		recipeService:    recipeService,
		authService:      authService,
		llmService:       llmService,
		embeddingService: embeddingService,
		db:               db,
	}
}

// NewRecipeHandlerWithRateLimit creates a new RecipeHandler with rate limiting
func NewRecipeHandlerWithRateLimit(recipeService service.IRecipeService, authService service.IAuthService, llmService service.LLMServiceInterface, embeddingService service.EmbeddingServiceInterface, db *gorm.DB, creationLimiter *middleware.RateLimiter, modificationLimiter *middleware.RateLimiter) *RecipeHandler {
	return &RecipeHandler{
		recipeService:           recipeService,
		authService:             authService,
		llmService:              llmService,
		embeddingService:        embeddingService,
		db:                      db,
		creationRateLimiter:     creationLimiter,
		modificationRateLimiter: modificationLimiter,
	}
}

// RegisterRoutes registers the recipe routes
func (h *RecipeHandler) RegisterRoutes(router *gin.RouterGroup) {
	recipes := router.Group("/recipes")

	// Public routes (no authentication required) - ONLY for landing page
	recipes.GET("/featured", h.GetFeaturedRecipes)

	// Protected routes (authentication required) - recipe browsing
	protected := recipes.Group("")
	protected.Use(middleware.AuthMiddleware(h.authService))
	{
		protected.GET("", h.ListRecipes)
		protected.GET("/search", h.SearchRecipes)
		protected.GET("/:id", h.GetRecipe)
	}

	// Email verification required routes (authentication + email verification)
	verified := recipes.Group("")
	verified.Use(middleware.AuthMiddleware(h.authService))
	verified.Use(middleware.RequireEmailVerification(h.db))
	{
		// Recipe creation with rate limiting (2 per hour)
		createGroup := verified.Group("")
		if h.creationRateLimiter != nil {
			createGroup.Use(h.creationRateLimiter.RateLimitMiddleware())
		}
		createGroup.POST("", h.CreateRecipe)

		// Recipe modification with per-recipe rate limiting (10 per recipe per hour)
		modifyGroup := verified.Group("")
		if h.modificationRateLimiter != nil {
			modifyGroup.Use(h.modificationRateLimiter.PerRecipeRateLimitMiddleware())
		}
		modifyGroup.PUT("/:id", h.UpdateRecipe)
		modifyGroup.DELETE("/:id", h.DeleteRecipe)

		// Favorite/unfavorite operations
		verified.POST("/:id/favorite", h.FavoriteRecipe)
		verified.DELETE("/:id/favorite", h.UnfavoriteRecipe)
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

// GetRecipe handles getting a single recipe for authenticated users
func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	println("[DEBUG] GetRecipe called")
	id := c.Param("id")
	println("[DEBUG] Recipe ID param: " + id)
	fmt.Printf("[DEBUG] Context keys: %v\n", c.Keys)

	// User is always authenticated due to middleware
	userIDValue := c.MustGet("user_id")
	userID := userIDValue.(uuid.UUID)

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

	// Add favorite status
	favoriteRecipes, err := h.recipeService.GetFavoriteRecipes(c.Request.Context(), userID)
	isFavorite := false
	if err == nil {
		for _, fav := range favoriteRecipes {
			if fav.ID == recipe.ID {
				isFavorite = true
				break
			}
		}
	}

	// Return recipe with favorite status
	type RecipeResponse struct {
		*models.Recipe
		IsFavorite bool `json:"is_favorite"`
	}

	c.JSON(http.StatusOK, gin.H{"recipe": RecipeResponse{
		Recipe:     recipe,
		IsFavorite: isFavorite,
	}})
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

// ListRecipes handles listing recipes for authenticated users
func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"
	var recipes []*models.Recipe
	var err error

	// User is always authenticated due to middleware
	userIDValue := c.MustGet("user_id")
	userID := userIDValue.(uuid.UUID)

	if all {
		// Return all recipes if explicitly requested
		recipes, err = h.recipeService.ListRecipes(c.Request.Context(), nil)
	} else {
		// Return user's recipes by default
		recipes, err = h.recipeService.ListRecipes(c.Request.Context(), &userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add favorite status to each recipe
	favoriteRecipes, err := h.recipeService.GetFavoriteRecipes(c.Request.Context(), userID)
	if err == nil {
		// Create a map for quick lookup
		favoriteMap := make(map[uuid.UUID]bool)
		for _, fav := range favoriteRecipes {
			favoriteMap[fav.ID] = true
		}

		// Create response with favorite status
		type RecipeResponse struct {
			*models.Recipe
			IsFavorite bool `json:"is_favorite"`
		}

		recipesWithFavorites := make([]RecipeResponse, len(recipes))
		for i, recipe := range recipes {
			recipesWithFavorites[i] = RecipeResponse{
				Recipe:     recipe,
				IsFavorite: favoriteMap[recipe.ID],
			}
		}

		c.JSON(http.StatusOK, gin.H{"recipes": recipesWithFavorites})
		return
	}

	// If error getting favorites, return recipes without favorite status
	c.JSON(http.StatusOK, gin.H{"recipes": recipes})
}

// GetFeaturedRecipes handles getting featured recipes for the landing page
func (h *RecipeHandler) GetFeaturedRecipes(c *gin.Context) {
	// Get the last 6 recipes as featured recipes (for landing page)
	var recipes []models.Recipe
	if err := h.db.Order("created_at desc").Limit(6).Find(&recipes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch featured recipes"})
		return
	}

	// Convert to pointers for consistency with other endpoints
	featuredRecipes := make([]*models.Recipe, len(recipes))
	for i := range recipes {
		featuredRecipes[i] = &recipes[i]
	}

	c.JSON(http.StatusOK, gin.H{"recipes": featuredRecipes})
}

// SearchRecipes handles searching recipes for authenticated users
func (h *RecipeHandler) SearchRecipes(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")
	sortBy := c.DefaultQuery("sort", "newest")

	fmt.Printf("[DEBUG] SearchRecipes called with query=%s, category=%s, sort=%s\n", query, category, sortBy)

	// User is always authenticated due to middleware
	userIDValue := c.MustGet("user_id")
	userID := userIDValue.(uuid.UUID)

	var recipes []*models.Recipe
	var err error

	if query != "" {
		// Use semantic search if query is provided
		recipes, err = h.recipeService.SearchRecipes(c.Request.Context(), query)
	} else {
		// Fall back to listing all recipes if no query
		recipes, err = h.recipeService.ListRecipes(c.Request.Context(), nil)
	}

	if err != nil {
		fmt.Printf("[DEBUG] Error searching recipes: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter by category if specified
	if category != "" && category != "All" {
		filtered := make([]*models.Recipe, 0)
		for _, recipe := range recipes {
			if strings.EqualFold(recipe.Category, category) {
				filtered = append(filtered, recipe)
			}
		}
		recipes = filtered
	}

	// Sort results
	// Note: For now we'll keep the database ordering, but this could be enhanced
	// to support different sorting options like popularity, rating, etc.

	fmt.Printf("[DEBUG] Found %d recipes after filtering\n", len(recipes))

	// Add favorite status to each recipe
	favoriteRecipes, err := h.recipeService.GetFavoriteRecipes(c.Request.Context(), userID)
	if err == nil {
		// Create a map for quick lookup
		favoriteMap := make(map[uuid.UUID]bool)
		for _, fav := range favoriteRecipes {
			favoriteMap[fav.ID] = true
		}

		// Create response with favorite status
		type RecipeResponse struct {
			*models.Recipe
			IsFavorite bool `json:"is_favorite"`
		}

		recipesWithFavorites := make([]RecipeResponse, len(recipes))
		for i, recipe := range recipes {
			recipesWithFavorites[i] = RecipeResponse{
				Recipe:     recipe,
				IsFavorite: favoriteMap[recipe.ID],
			}
		}

		c.JSON(http.StatusOK, gin.H{"recipes": recipesWithFavorites})
		return
	}

	// If error getting favorites, return recipes without favorite status
	c.JSON(http.StatusOK, gin.H{"recipes": recipes})
}

// FavoriteRecipe handles favoriting a recipe
func (h *RecipeHandler) FavoriteRecipe(c *gin.Context) {
	// Get user ID from context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get recipe ID from URL parameter
	recipeIDStr := c.Param("id")
	if recipeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
		return
	}

	recipeID, err := uuid.Parse(recipeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID format"})
		return
	}

	// Check if recipe exists
	_, err = h.recipeService.GetRecipe(c.Request.Context(), recipeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify recipe"})
		return
	}

	// Add favorite using recipe service
	err = h.recipeService.FavoriteRecipe(c.Request.Context(), userID, recipeID)
	if err != nil {
		// Check if already favorited
		if err.Error() == "recipe already favorited" {
			c.JSON(http.StatusConflict, gin.H{"error": "recipe already favorited"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to favorite recipe"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "recipe favorited successfully",
		"is_favorite": true,
	})
}

// UnfavoriteRecipe handles unfavoriting a recipe
func (h *RecipeHandler) UnfavoriteRecipe(c *gin.Context) {
	// Get user ID from context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get recipe ID from URL parameter
	recipeIDStr := c.Param("id")
	if recipeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
		return
	}

	recipeID, err := uuid.Parse(recipeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe ID format"})
		return
	}

	// Remove favorite using recipe service
	err = h.recipeService.UnfavoriteRecipe(c.Request.Context(), userID, recipeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "favorite not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unfavorite recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "recipe unfavorited successfully",
		"is_favorite": false,
	})
}
