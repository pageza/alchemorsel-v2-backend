package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// LLMHandler handles LLM-related requests
type LLMHandler struct {
	db          *gorm.DB
	llmService  *service.LLMService
	authService *service.AuthService
}

// NewLLMHandler creates a new LLMHandler instance
func NewLLMHandler(db *gorm.DB, authService *service.AuthService) (*LLMHandler, error) {
	llmService, err := service.NewLLMService()
	if err != nil {
		return nil, err
	}

	return &LLMHandler{
		db:          db,
		llmService:  llmService,
		authService: authService,
	}, nil
}

// RegisterRoutes registers the LLM routes
func (h *LLMHandler) RegisterRoutes(router *gin.RouterGroup) {
	llm := router.Group("/llm")
	{
		llm.POST("/query", middleware.AuthMiddleware(h.authService), h.Query)
	}
}

// Query handles LLM query requests
func (h *LLMHandler) Query(c *gin.Context) {
	var req struct {
		Query    string `json:"query" binding:"required"`
		Intent   string `json:"intent" binding:"required"`
		RecipeID string `json:"recipe_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the authenticated user ID from context
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

	var prefs []models.DietaryPreference
	h.db.Where("user_id = ?", userID).Find(&prefs)
	dietaryList := make([]string, 0, len(prefs))
	for _, p := range prefs {
		if p.PreferenceType == "custom" {
			dietaryList = append(dietaryList, p.CustomName)
		} else {
			dietaryList = append(dietaryList, p.PreferenceType)
		}
	}

	var alls []models.Allergen
	h.db.Where("user_id = ?", userID).Find(&alls)
	allergenList := make([]string, 0, len(alls))
	for _, a := range alls {
		allergenList = append(allergenList, a.AllergenName)
	}

	// Generate recipe using LLM
	recipeJSON, err := h.llmService.GenerateRecipe(req.Query, dietaryList, allergenList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate recipe: " + err.Error()})
		return
	}

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
		Calories     float64  `json:"calories"`
		Protein      float64  `json:"protein"`
		Carbs        float64  `json:"carbs"`
		Fat          float64  `json:"fat"`
	}

	var recipeData RecipeData
	if err := json.Unmarshal([]byte(recipeJSON), &recipeData); err != nil {
		log.Printf("Failed to unmarshal recipe JSON: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recipe: " + err.Error()})
		return
	}

	// If macros are missing try to calculate them using the LLM
	if recipeData.Calories == 0 && recipeData.Protein == 0 && recipeData.Carbs == 0 && recipeData.Fat == 0 {
		macros, err := h.llmService.CalculateMacros(recipeData.Ingredients)
		if err == nil && macros != nil {
			recipeData.Calories = macros.Calories
			recipeData.Protein = macros.Protein
			recipeData.Carbs = macros.Carbs
			recipeData.Fat = macros.Fat
		}
	}

	if req.Intent == "modify" {
		if req.RecipeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "recipe_id required"})
			return
		}
		rid, err := uuid.Parse(req.RecipeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe_id"})
			return
		}

		var existing model.Recipe
		if err := h.db.First(&existing, "id = ?", rid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}

		existing.Name = recipeData.Name
		existing.Description = recipeData.Description
		existing.Category = recipeData.Category
		existing.Ingredients = model.JSONBStringArray(recipeData.Ingredients)
		existing.Instructions = model.JSONBStringArray(recipeData.Instructions)
		existing.Embedding = service.GenerateEmbedding(recipeData.Name + " " + recipeData.Description)
		if recipeData.Calories == 0 && recipeData.Protein == 0 && recipeData.Carbs == 0 && recipeData.Fat == 0 {
			macros, err := h.llmService.CalculateMacros(recipeData.Ingredients)
			if err == nil && macros != nil {
				recipeData.Calories = macros.Calories
				recipeData.Protein = macros.Protein
				recipeData.Carbs = macros.Carbs
				recipeData.Fat = macros.Fat
			}
		}
		existing.Calories = recipeData.Calories
		existing.Protein = recipeData.Protein
		existing.Carbs = recipeData.Carbs
		existing.Fat = recipeData.Fat

		if err := h.db.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recipe"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"recipe": existing})
		return
	}

	// Convert the parsed data into a model.Recipe
	recipe := model.Recipe{
		ID:           uuid.New(),
		Name:         recipeData.Name,
		Description:  recipeData.Description,
		Category:     recipeData.Category,
		Ingredients:  model.JSONBStringArray(recipeData.Ingredients),
		Instructions: model.JSONBStringArray(recipeData.Instructions),
		Calories:     recipeData.Calories,
		Protein:      recipeData.Protein,
		Carbs:        recipeData.Carbs,
		Fat:          recipeData.Fat,
		UserID:       userID,
		Embedding:    service.GenerateEmbedding(recipeData.Name + " " + recipeData.Description),
	}

	// Persist the recipe
	if err := h.db.Create(&recipe).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recipe"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"recipe": recipe,
	})
}
