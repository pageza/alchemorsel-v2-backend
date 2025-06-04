package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// LLMHandler handles LLM-related requests
type LLMHandler struct {
	db         *gorm.DB
	llmService *service.LLMService
}

// NewLLMHandler creates a new LLMHandler instance
func NewLLMHandler(db *gorm.DB) (*LLMHandler, error) {
	llmService, err := service.NewLLMService()
	if err != nil {
		return nil, err
	}

	return &LLMHandler{
		db:         db,
		llmService: llmService,
	}, nil
}

// RegisterRoutes registers the LLM routes
func (h *LLMHandler) RegisterRoutes(router *gin.RouterGroup) {
	llm := router.Group("/llm")
	{
		llm.POST("/query", h.Query)
	}
}

// Query handles LLM query requests
func (h *LLMHandler) Query(c *gin.Context) {
	var req struct {
		Query  string `json:"query" binding:"required"`
		Intent string `json:"intent" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate recipe using LLM
	recipeJSON, err := h.llmService.GenerateRecipe(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate recipe: " + err.Error()})
		return
	}

	log.Printf("Raw recipe JSON from DeepSeek (type: %T): %s", recipeJSON, recipeJSON)

	// Parse the JSON response into a recipe struct
	type RecipeData struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		Category     string   `json:"category"`
		Ingredients  []string `json:"ingredients"`
		Instructions []string `json:"instructions"`
		PrepTime     string   `json:"prep_time"`
		CookTime     string   `json:"cook_time"`
		Servings     string   `json:"servings"`
		Difficulty   string   `json:"difficulty"`
	}

	var recipeData RecipeData
	if err := json.Unmarshal([]byte(recipeJSON), &recipeData); err != nil {
		log.Printf("Failed to unmarshal recipe JSON: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recipe: " + err.Error()})
		return
	}

	// Log detailed information about the parsed data
	log.Printf("Parsed recipe data structure:")
	log.Printf("- Name (type: %T): %v", recipeData.Name, recipeData.Name)
	log.Printf("- Description (type: %T): %v", recipeData.Description, recipeData.Description)
	log.Printf("- Category (type: %T): %v", recipeData.Category, recipeData.Category)
	log.Printf("- Ingredients (type: %T): %v", recipeData.Ingredients, recipeData.Ingredients)
	log.Printf("- Instructions (type: %T): %v", recipeData.Instructions, recipeData.Instructions)
	log.Printf("- PrepTime (type: %T): %v", recipeData.PrepTime, recipeData.PrepTime)
	log.Printf("- CookTime (type: %T): %v", recipeData.CookTime, recipeData.CookTime)
	log.Printf("- Servings (type: %T): %v", recipeData.Servings, recipeData.Servings)
	log.Printf("- Difficulty (type: %T): %v", recipeData.Difficulty, recipeData.Difficulty)

	// Convert the parsed data into a model.Recipe
	recipe := model.Recipe{
		ID:           uuid.New(),
		Name:         recipeData.Name,
		Description:  recipeData.Description,
		Category:     recipeData.Category,
		Ingredients:  model.JSONBStringArray(recipeData.Ingredients),
		Instructions: model.JSONBStringArray(recipeData.Instructions),
		UserID:       uuid.New(),
	}

	// Persist the recipe
	if err := h.db.Create(&recipe).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recipe": recipe,
	})
}
