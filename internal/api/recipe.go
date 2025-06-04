package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

type RecipeHandler struct {
	db          *gorm.DB
	authService *service.AuthService
}

func NewRecipeHandler(db *gorm.DB, authService *service.AuthService) *RecipeHandler {
	return &RecipeHandler{
		db:          db,
		authService: authService,
	}
}

func (h *RecipeHandler) RegisterRoutes(router *gin.RouterGroup) {
	recipes := router.Group("/recipes")
	{
		recipes.GET("", h.ListRecipes)
		recipes.GET("/:id", h.GetRecipe)
		recipes.POST("", AuthMiddleware(h.authService), h.CreateRecipe)
		recipes.PUT("/:id", AuthMiddleware(h.authService), h.UpdateRecipe)
		recipes.DELETE("/:id", AuthMiddleware(h.authService), h.DeleteRecipe)
		recipes.POST("/:id/favorite", AuthMiddleware(h.authService), h.FavoriteRecipe)
		recipes.DELETE("/:id/favorite", AuthMiddleware(h.authService), h.UnfavoriteRecipe)
	}
}

func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	var recipes []model.Recipe
	result := h.db.Find(&recipes)
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
	var recipe model.Recipe
	result := h.db.First(&recipe, "id = ?", id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var recipe model.Recipe
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

	result := h.db.Create(&recipe)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create recipe"})
		return
	}

	c.JSON(http.StatusCreated, recipe)
}

func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	id := c.Param("id")
	var recipe model.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&model.Recipe{}).Where("id = ?", id).Updates(recipe)
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
	result := h.db.Delete(&model.Recipe{}, "id = ?", id)
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

	fav := model.RecipeFavorite{
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

	if err := h.db.Where("recipe_id = ? AND user_id = ?", recipeID, userIDVal.(uuid.UUID)).Delete(&model.RecipeFavorite{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfavorite recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe unfavorited successfully",
		"id":      idStr,
	})
}
