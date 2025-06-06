package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

type RecipeHandler struct {
	db               *gorm.DB
	authService      *service.AuthService
	llmService       *service.LLMService
	embeddingService *service.EmbeddingService
}

func NewRecipeHandler(db *gorm.DB, authService *service.AuthService) (*RecipeHandler, error) {
	llmService, _ := service.NewLLMService()
	embeddingService, err := service.NewEmbeddingService()
	if err != nil {
		return nil, err
	}
	return &RecipeHandler{
		db:               db,
		authService:      authService,
		llmService:       llmService,
		embeddingService: embeddingService,
	}, nil
}

func (h *RecipeHandler) RegisterRoutes(router *gin.RouterGroup) {
	recipes := router.Group("/recipes")
	{
		recipes.GET("", h.ListRecipes)
		recipes.GET("/:id", h.GetRecipe)
		recipes.POST("", middleware.AuthMiddleware(h.authService), h.CreateRecipe)
		recipes.PUT("/:id", middleware.AuthMiddleware(h.authService), h.UpdateRecipe)
		recipes.DELETE("/:id", middleware.AuthMiddleware(h.authService), h.DeleteRecipe)
		recipes.POST("/:id/favorite", middleware.AuthMiddleware(h.authService), h.FavoriteRecipe)
		recipes.DELETE("/:id/favorite", middleware.AuthMiddleware(h.authService), h.UnfavoriteRecipe)
	}
}

func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	var recipes []models.Recipe

	query := h.db

	if search := c.Query("q"); search != "" {
		if h.db.Dialector.Name() == "postgres" {
			// Generate embedding for semantic search
			vec, err := h.embeddingService.GenerateEmbedding(search)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate embedding"})
				return
			}

			// Combine semantic and keyword search
			// Use a subquery to get both semantic and keyword matches
			subQuery := h.db.Model(&models.Recipe{}).
				Select("id, embedding <-> ? as similarity", vec).
				Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(ingredients::text) LIKE ?",
					"%"+strings.ToLower(search)+"%",
					"%"+strings.ToLower(search)+"%",
					"%"+strings.ToLower(search)+"%",
				)

			// Join with the main query and order by similarity
			query = query.Joins("JOIN (?) as search ON recipes.id = search.id", subQuery).
				Order("search.similarity ASC")
		} else {
			// Fallback to keyword search for non-PostgreSQL databases
			like := "%" + strings.ToLower(search) + "%"
			query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(ingredients) LIKE ?",
				like, like, like)
		}
	}

	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	if cuisine := c.Query("cuisine"); cuisine != "" {
		query = query.Where("cuisine = ?", cuisine)
	}

	if prefs := c.Query("dietary"); prefs != "" {
		for _, p := range strings.Split(prefs, ",") {
			like := "%" + strings.ToLower(strings.TrimSpace(p)) + "%"
			query = query.Where("LOWER(dietary_preferences::text) LIKE ?", like)
		}
	}

	if ex := c.Query("exclude"); ex != "" {
		for _, a := range strings.Split(ex, ",") {
			like := "%" + strings.ToLower(strings.TrimSpace(a)) + "%"
			if h.db.Dialector.Name() == "postgres" {
				query = query.Where("LOWER(ingredients::text) NOT LIKE ?", like)
			} else {
				query = query.Where("LOWER(ingredients) NOT LIKE ?", like)
			}
		}
	}

	result := query.Find(&recipes)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recipes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recipes": recipes,
	})
}

func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	id := c.Param("id")
	var recipe models.Recipe
	result := h.db.First(&recipe, "id = ?", id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Set the user ID on the recipe
	recipe.UserID = userID.(uuid.UUID)

	// calculate macros if not provided
	if recipe.Calories == 0 && recipe.Protein == 0 && recipe.Carbs == 0 && recipe.Fat == 0 {
		if h.llmService != nil {
			macros, err := h.llmService.CalculateMacros([]string(recipe.Ingredients))
			if err == nil && macros != nil {
				recipe.Calories = macros.Calories
				recipe.Protein = macros.Protein
				recipe.Carbs = macros.Carbs
				recipe.Fat = macros.Fat
			}
		}
	}

	// Generate tags from category, cuisine, and dietary preferences
	tags := make([]string, 0)
	if recipe.Category != "" {
		tags = append(tags, recipe.Category)
	}
	if recipe.Cuisine != "" {
		tags = append(tags, recipe.Cuisine)
	}
	if len(recipe.DietaryPreferences) > 0 {
		tags = append(tags, []string(recipe.DietaryPreferences)...)
	}
	recipe.Tags = models.JSONBStringArray(tags)

	// generate embedding with enhanced context
	embedding, err := h.embeddingService.GenerateEmbeddingFromRecipe(
		recipe.Name,
		recipe.Description,
		recipe.Ingredients,
		recipe.Category,
		recipe.DietaryPreferences,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate embedding"})
		return
	}
	recipe.Embedding = embedding

	result := h.db.Create(&recipe)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create recipe"})
		return
	}

	c.JSON(http.StatusCreated, recipe)
}

func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	id := c.Param("id")
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if recipe.Calories == 0 && recipe.Protein == 0 && recipe.Carbs == 0 && recipe.Fat == 0 {
		if h.llmService != nil {
			macros, err := h.llmService.CalculateMacros([]string(recipe.Ingredients))
			if err == nil && macros != nil {
				recipe.Calories = macros.Calories
				recipe.Protein = macros.Protein
				recipe.Carbs = macros.Carbs
				recipe.Fat = macros.Fat
			}
		}
	}

	// generate embedding with enhanced context
	embedding, err := h.embeddingService.GenerateEmbeddingFromRecipe(
		recipe.Name,
		recipe.Description,
		recipe.Ingredients,
		recipe.Category,
		recipe.DietaryPreferences,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate embedding"})
		return
	}
	recipe.Embedding = embedding

	result := h.db.Model(&models.Recipe{}).Where("id = ?", id).Updates(recipe)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe updated successfully",
		"id":      id,
	})
}

func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	id := c.Param("id")
	result := h.db.Delete(&models.Recipe{}, "id = ?", id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe deleted successfully",
		"id":      id,
	})
}

func (h *RecipeHandler) FavoriteRecipe(c *gin.Context) {
	idStr := c.Param("id")
	recipeID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe id"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	fav := models.RecipeFavorite{
		RecipeID: recipeID,
		UserID:   userIDVal.(uuid.UUID),
	}
	if err := h.db.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to favorite recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe favorited successfully",
		"id":      idStr,
	})
}

func (h *RecipeHandler) UnfavoriteRecipe(c *gin.Context) {
	idStr := c.Param("id")
	recipeID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe id"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.db.Where("recipe_id = ? AND user_id = ?", recipeID, userIDVal.(uuid.UUID)).Delete(&models.RecipeFavorite{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfavorite recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe unfavorited successfully",
		"id":      idStr,
	})
}
