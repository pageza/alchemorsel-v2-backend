package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
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

// QueryRequest represents a request to query the LLM
type QueryRequest struct {
	Query   string `json:"query" binding:"required"`
	Intent  string `json:"intent" binding:"required"`
	DraftID string `json:"draft_id,omitempty"`
}

// Query handles recipe generation and modification requests
func (h *LLMHandler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[LLMHandler] Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		fmt.Println("[LLMHandler] user_id missing from context. Unauthorized.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(uuid.UUID).String()
	fmt.Printf("[LLMHandler] user_id from context: %s\n", userID)
	fmt.Printf("[LLMHandler] QueryRequest: %+v\n", req)

	switch req.Intent {
	case "generate":
		recipeJSON, err := h.llmService.GenerateRecipe(req.Query, []string{}, []string{}, nil)
		if err != nil {
			fmt.Printf("[LLMHandler] Error generating recipe: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var recipe service.RecipeDraft
		if err := json.Unmarshal([]byte(recipeJSON), &recipe); err != nil {
			fmt.Printf("[LLMHandler] Failed to parse recipe JSON: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse recipe"})
			return
		}

		recipe.UserID = userID
		if err := h.llmService.SaveDraft(c.Request.Context(), &recipe); err != nil {
			fmt.Printf("[LLMHandler] Error saving draft: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("[LLMHandler] Successfully generated and saved draft. Recipe ID: %s\n", recipe.ID)
		c.JSON(http.StatusOK, gin.H{
			"recipe":   recipe,
			"draft_id": recipe.ID,
		})

	case "modify":
		if req.DraftID == "" {
			fmt.Println("[LLMHandler] draft_id is required for modifications")
			c.JSON(http.StatusBadRequest, gin.H{"error": "draft_id is required for modifications"})
			return
		}

		draft, err := h.llmService.GetDraft(c.Request.Context(), req.DraftID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error getting draft: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
			return
		}

		if draft.UserID != userID {
			fmt.Printf("[LLMHandler] Unauthorized: draft.UserID=%s, userID=%s\n", draft.UserID, userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}

		recipeJSON, err := h.llmService.GenerateRecipe(req.Query, []string{}, []string{}, draft)
		if err != nil {
			fmt.Printf("[LLMHandler] Error generating modified recipe: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var updatedRecipe service.RecipeDraft
		if err := json.Unmarshal([]byte(recipeJSON), &updatedRecipe); err != nil {
			fmt.Printf("[LLMHandler] Failed to parse modified recipe JSON: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse recipe"})
			return
		}

		draft.Name = updatedRecipe.Name
		draft.Description = updatedRecipe.Description
		draft.Category = updatedRecipe.Category
		draft.Ingredients = updatedRecipe.Ingredients
		draft.Instructions = updatedRecipe.Instructions
		draft.PrepTime = updatedRecipe.PrepTime
		draft.CookTime = updatedRecipe.CookTime
		draft.Difficulty = updatedRecipe.Difficulty
		draft.Calories = updatedRecipe.Calories
		draft.Protein = updatedRecipe.Protein
		draft.Carbs = updatedRecipe.Carbs
		draft.Fat = updatedRecipe.Fat

		if err := h.llmService.UpdateDraft(c.Request.Context(), draft); err != nil {
			fmt.Printf("[LLMHandler] Error updating draft: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("[LLMHandler] Successfully modified and updated draft. Draft ID: %s\n", draft.ID)
		c.JSON(http.StatusOK, gin.H{
			"recipe":   draft,
			"draft_id": draft.ID,
		})

	default:
		fmt.Printf("[LLMHandler] Invalid intent: %s\n", req.Intent)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid intent"})
	}
}

// GetDraft retrieves a recipe draft
func (h *LLMHandler) GetDraft(c *gin.Context) {
	draftID := c.Param("id")
	if draftID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "draft_id is required"})
		return
	}

	// Get user ID from context
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	draft, err := h.llmService.GetDraft(c.Request.Context(), draftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	// Verify ownership
	if draft.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"draft": draft})
}

// DeleteDraft removes a recipe draft
func (h *LLMHandler) DeleteDraft(c *gin.Context) {
	draftID := c.Param("id")
	if draftID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "draft_id is required"})
		return
	}

	// Get user ID from context
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get draft to verify ownership
	draft, err := h.llmService.GetDraft(c.Request.Context(), draftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	// Verify ownership
	if draft.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.llmService.DeleteDraft(c.Request.Context(), draftID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "draft deleted"})
}
