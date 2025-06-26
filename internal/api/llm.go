package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// LLMHandler handles LLM-related requests
type LLMHandler struct {
	db              *gorm.DB
	llmService      service.LLMServiceInterface
	authService     *service.AuthService
	recipeService   service.IRecipeService
	creationLimiter *middleware.RateLimiter
}

// NewLLMHandler creates a new LLM handler
func NewLLMHandler(db *gorm.DB, authService *service.AuthService, llmService service.LLMServiceInterface, recipeService service.IRecipeService) *LLMHandler {
	var svc service.LLMServiceInterface
	if llmService != nil {
		svc = llmService
	} else {
		var err error
		svc, err = service.NewLLMService()
		if err != nil {
			panic(err)
		}
	}
	return &LLMHandler{
		db:            db,
		llmService:    svc,
		authService:   authService,
		recipeService: recipeService,
	}
}

// NewLLMHandlerWithRateLimit creates a new LLM handler with rate limiting
func NewLLMHandlerWithRateLimit(db *gorm.DB, authService *service.AuthService, llmService service.LLMServiceInterface, recipeService service.IRecipeService, creationLimiter *middleware.RateLimiter) *LLMHandler {
	var svc service.LLMServiceInterface
	if llmService != nil {
		svc = llmService
	} else {
		var err error
		svc, err = service.NewLLMService()
		if err != nil {
			panic(err)
		}
	}
	return &LLMHandler{
		db:              db,
		llmService:      svc,
		authService:     authService,
		recipeService:   recipeService,
		creationLimiter: creationLimiter,
	}
}

// SetLLMService sets the LLM service (used for testing)
func (h *LLMHandler) SetLLMService(service service.LLMServiceInterface) {
	h.llmService = service
}

// RegisterRoutes registers the LLM routes
func (h *LLMHandler) RegisterRoutes(router *gin.RouterGroup) {
	llm := router.Group("/llm")
	llm.Use(middleware.AuthMiddleware(h.authService))
	{
		// Legacy recipe generation (maintains backward compatibility)
		llm.POST("/query", middleware.RequireEmailVerification(h.db), h.Query)
		
		// Multi-call recipe generation endpoints
		llm.POST("/generate-basic", middleware.RequireEmailVerification(h.db), h.GenerateBasicRecipe)
		llm.POST("/calculate-nutrition", middleware.RequireEmailVerification(h.db), h.CalculateNutrition)
		llm.POST("/finalize-recipe", middleware.RequireEmailVerification(h.db), h.FinalizeRecipe)
		
		// Draft operations don't require verification (user can view their drafts before verifying)
		llm.GET("/drafts/:id", h.GetDraft)
		llm.DELETE("/drafts/:id", h.DeleteDraft)
	}
}

// QueryRequest represents a request to query the LLM
type QueryRequest struct {
	Query    string `json:"query" binding:"required"`
	Intent   string `json:"intent" binding:"required"`
	DraftID  string `json:"draft_id,omitempty"`
	RecipeID string `json:"recipe_id,omitempty"`
}

// BasicRecipeRequest represents a request for basic recipe generation
type BasicRecipeRequest struct {
	Query string `json:"query" binding:"required"`
}

// NutritionRequest represents a request for nutrition calculation
type NutritionRequest struct {
	DraftID string `json:"draft_id" binding:"required"`
}

// FinalizeRequest represents a request to finalize a recipe
type FinalizeRequest struct {
	DraftID string `json:"draft_id" binding:"required"`
}

// BasicRecipeResponse represents the response for basic recipe generation
type BasicRecipeResponse struct {
	DraftID string              `json:"draft_id"`
	Recipe  *service.BasicRecipe `json:"recipe"`
	Status  string              `json:"status"`
}

// NutritionResponse represents the response for nutrition calculation
type NutritionResponse struct {
	DraftID   string  `json:"draft_id"`
	Calories  float64 `json:"calories"`
	Protein   float64 `json:"protein"`
	Carbs     float64 `json:"carbs"`
	Fat       float64 `json:"fat"`
	Status    string  `json:"status"`
}

// FinalizeResponse represents the response for recipe finalization
type FinalizeResponse struct {
	DraftID string                `json:"draft_id"`
	Recipe  *service.RecipeDraft  `json:"recipe"`
	Status  string                `json:"status"`
}

// Query handles recipe generation and modification requests
func (h *LLMHandler) Query(c *gin.Context) {
	println("[DEBUG] LLMHandler.Query called")
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[LLMHandler] Failed to bind JSON: %v\n", err)
		if err.Error() == "EOF" || strings.Contains(err.Error(), "invalid character") {
			fmt.Println("[LLMHandler] Responding 400: Invalid request body")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		fmt.Println("[LLMHandler] Responding 400:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[LLMHandler] Parsed QueryRequest: %+v\n", req)
	userIDVal, exists := c.Get("user_id")
	fmt.Printf("[LLMHandler] Context keys: %v\n", c.Keys)
	if !exists {
		fmt.Println("[LLMHandler] user_id missing from context. Responding 401 Unauthorized.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	fmt.Printf("[LLMHandler] user_id value: %#v\n", userIDVal)
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		fmt.Printf("[LLMHandler] user_id is not uuid.UUID, got: %T\n", userIDVal)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	fmt.Printf("[LLMHandler] user_id string: %s\n", userID.String())

	// Fetch user with dietary preferences and allergens
	var user models.User
	err := h.db.Preload("DietaryPrefs").Where("id = ?", userID).First(&user).Error
	if err != nil {
		fmt.Printf("[LLMHandler] Failed to fetch user with dietary preferences: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user preferences"})
		return
	}
	
	// Try to preload allergens separately to avoid breaking if table doesn't exist
	_ = h.db.Model(&user).Association("Allergens").Find(&user.Allergens)

	// Extract dietary preferences and allergens into string arrays
	var dietaryPrefs []string
	for _, pref := range user.DietaryPrefs {
		if pref.PreferenceType != "" {
			dietaryPrefs = append(dietaryPrefs, pref.PreferenceType)
		} else if pref.CustomName != "" {
			dietaryPrefs = append(dietaryPrefs, pref.CustomName)
		}
	}

	var allergens []string
	for _, allergen := range user.Allergens {
		allergens = append(allergens, allergen.AllergenName)
	}

	fmt.Printf("[LLMHandler] User dietary preferences: %v\n", dietaryPrefs)
	fmt.Printf("[LLMHandler] User allergens: %v\n", allergens)

	switch req.Intent {
	case "fork":
		fmt.Println("[LLMHandler] Intent: fork")
		if req.RecipeID == "" {
			fmt.Println("[LLMHandler] recipe_id is required for forking. Responding 400.")
			c.JSON(http.StatusBadRequest, gin.H{"error": "recipe_id is required for forking"})
			return
		}

		// Check rate limiting before attempting fork (without incrementing)
		if h.creationLimiter != nil {
			allowed, remaining, resetTime, err := h.creationLimiter.CheckOnly(c.Request.Context(), userID.String())
			if err != nil {
				fmt.Printf("[LLMHandler] Rate limit check failed for user %s: %v\n", userID.String(), err)
				// Continue without rate limiting on error
			} else if !allowed {
				fmt.Printf("[LLMHandler] Rate limit exceeded for user %s\n", userID.String())
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":                "rate limit exceeded",
					"rate_limit_remaining": remaining,
					"rate_limit_reset":     resetTime.Unix(),
				})
				return
			}
		}

		// Get the original recipe from database
		recipeUUID, err := uuid.Parse(req.RecipeID)
		if err != nil {
			fmt.Printf("[LLMHandler] Invalid recipe_id format: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe_id format"})
			return
		}

		originalRecipe, err := h.recipeService.GetRecipe(c.Request.Context(), recipeUUID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error getting original recipe: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "original recipe not found"})
			return
		}

		// Convert the original recipe to a draft format for modification
		originalDraft := &service.RecipeDraft{
			Name:         originalRecipe.Name,
			Description:  originalRecipe.Description,
			Category:     originalRecipe.Category,
			Ingredients:  []string(originalRecipe.Ingredients),
			Instructions: []string(originalRecipe.Instructions),
			PrepTime:     "", // These fields may not be in the same format
			CookTime:     "",
			Servings:     service.ServingsType{Value: "4"}, // Default servings
			Difficulty:   "Medium",                         // Default difficulty since Recipe model doesn't have this field
			Calories:     originalRecipe.Calories,
			Protein:      originalRecipe.Protein,
			Carbs:        originalRecipe.Carbs,
			Fat:          originalRecipe.Fat,
			UserID:       userID.String(),
		}

		// Generate modified recipe using LLM with dietary restrictions
		recipeJSON, err := h.llmService.GenerateRecipe(req.Query, dietaryPrefs, allergens, originalDraft)
		if err != nil {
			fmt.Printf("[LLMHandler] Error generating forked recipe: %v\n", err)
			// Don't increment rate limit counter on generation failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Printf("[LLMHandler] Forked Recipe JSON: %s\n", recipeJSON)

		var newRecipe service.RecipeDraft
		if err := json.Unmarshal([]byte(recipeJSON), &newRecipe); err != nil {
			fmt.Printf("[LLMHandler] Failed to parse forked recipe JSON: %v\n", err)
			// Don't increment rate limit counter on parsing failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse recipe"})
			return
		}

		newRecipe.UserID = userID.String()
		if err := h.llmService.SaveDraft(c.Request.Context(), &newRecipe); err != nil {
			fmt.Printf("[LLMHandler] Error saving forked draft: %v\n", err)
			// Don't increment rate limit counter on save failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Calculate nutrition for the forked recipe using USDA data
		fmt.Printf("[LLMHandler] Calculating nutrition for forked recipe with draft ID: %s\n", newRecipe.ID)
		_, err = h.llmService.CalculateRecipeNutrition(c.Request.Context(), newRecipe.ID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error calculating nutrition for forked recipe: %v\n", err)
			// Don't fail the fork - nutrition calculation is enhancement
		} else {
			fmt.Printf("[LLMHandler] Successfully calculated nutrition for forked recipe\n")
		}

		// Fetch the updated draft with nutrition data
		updatedFork, err := h.llmService.GetDraft(c.Request.Context(), newRecipe.ID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error fetching updated fork with nutrition: %v\n", err)
			// Use original recipe if fetch fails
			updatedFork = &newRecipe
		} else {
			fmt.Printf("[LLMHandler] Successfully fetched updated fork with nutrition data\n")
		}

		// Only increment rate limit counter on successful fork generation and save
		if h.creationLimiter != nil {
			if err := h.creationLimiter.IncrementUsage(c.Request.Context(), userID.String()); err != nil {
				fmt.Printf("[LLMHandler] Failed to increment rate limit for user %s: %v\n", userID.String(), err)
			} else {
				fmt.Printf("[LLMHandler] Rate limit incremented for successful fork by user %s\n", userID.String())
			}
		}

		fmt.Printf("[LLMHandler] Successfully forked recipe. New Draft ID: %s\n", updatedFork.ID)
		c.JSON(http.StatusOK, gin.H{
			"recipe":   updatedFork,
			"draft_id": updatedFork.ID,
		})
		fmt.Println("[LLMHandler] Responded 200 OK with forked recipe and draft_id.")

	case "generate":
		fmt.Println("[LLMHandler] Intent: generate")

		// Check rate limiting before attempting generation (without incrementing)
		if h.creationLimiter != nil {
			allowed, remaining, resetTime, err := h.creationLimiter.CheckOnly(c.Request.Context(), userID.String())
			if err != nil {
				fmt.Printf("[LLMHandler] Rate limit check failed for user %s: %v\n", userID.String(), err)
				// Continue without rate limiting on error
			} else if !allowed {
				fmt.Printf("[LLMHandler] Rate limit exceeded for user %s\n", userID.String())
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":                "rate limit exceeded",
					"rate_limit_remaining": remaining,
					"rate_limit_reset":     resetTime.Unix(),
				})
				return
			}
		}

		// Use new multi-call approach internally for backward compatibility
		// Step 1: Generate basic recipe
		draft, err := h.llmService.GenerateBasicRecipe(c.Request.Context(), req.Query, dietaryPrefs, allergens, userID.String())
		if err != nil {
			fmt.Printf("[LLMHandler] Error generating basic recipe: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Step 2: Calculate nutrition
		_, err = h.llmService.CalculateRecipeNutrition(c.Request.Context(), draft.ID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error calculating nutrition: %v\n", err)
			// Continue without nutrition data for backward compatibility
		}

		// Step 3: Finalize recipe
		recipe, err := h.llmService.FinalizeRecipe(c.Request.Context(), draft.ID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error finalizing recipe: %v\n", err)
			// Use basic draft if finalization fails
			recipe = draft
		}

		// Only increment rate limit counter on successful generation and save
		if h.creationLimiter != nil {
			if err := h.creationLimiter.IncrementUsage(c.Request.Context(), userID.String()); err != nil {
				fmt.Printf("[LLMHandler] Failed to increment rate limit for user %s: %v\n", userID.String(), err)
			} else {
				fmt.Printf("[LLMHandler] Rate limit incremented for successful generation by user %s\n", userID.String())
			}
		}

		fmt.Printf("[LLMHandler] Successfully generated and saved draft. Recipe ID: %s\n", recipe.ID)
		c.JSON(http.StatusOK, gin.H{
			"recipe":   recipe,
			"draft_id": recipe.ID,
		})
		fmt.Println("[LLMHandler] Responded 200 OK with recipe and draft_id.")
	case "modify":
		fmt.Println("[LLMHandler] Intent: modify")
		if req.DraftID == "" {
			fmt.Println("[LLMHandler] draft_id is required for modifications. Responding 400.")
			c.JSON(http.StatusBadRequest, gin.H{"error": "draft_id is required for modifications"})
			return
		}

		// Check rate limiting before attempting modification (without incrementing)
		if h.creationLimiter != nil {
			allowed, remaining, resetTime, err := h.creationLimiter.CheckOnly(c.Request.Context(), userID.String())
			if err != nil {
				fmt.Printf("[LLMHandler] Rate limit check failed for user %s: %v\n", userID.String(), err)
				// Continue without rate limiting on error
			} else if !allowed {
				fmt.Printf("[LLMHandler] Rate limit exceeded for user %s\n", userID.String())
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":                "rate limit exceeded",
					"rate_limit_remaining": remaining,
					"rate_limit_reset":     resetTime.Unix(),
				})
				return
			}
		}

		draft, err := h.llmService.GetDraft(c.Request.Context(), req.DraftID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error getting draft: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
			return
		}
		if draft.UserID != userID.String() {
			fmt.Printf("[LLMHandler] Unauthorized: draft.UserID=%s, userID=%s\n", draft.UserID, userID.String())
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		recipeJSON, err := h.llmService.GenerateRecipe(req.Query, dietaryPrefs, allergens, draft)
		if err != nil {
			fmt.Printf("[LLMHandler] Error generating modified recipe: %v\n", err)
			// Don't increment rate limit counter on generation failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Printf("[LLMHandler] Modified Recipe JSON: %s\n", recipeJSON)
		var updatedRecipe service.RecipeDraft
		if err := json.Unmarshal([]byte(recipeJSON), &updatedRecipe); err != nil {
			fmt.Printf("[LLMHandler] Failed to parse modified recipe JSON: %v\n", err)
			// Don't increment rate limit counter on parsing failure
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
		draft.Servings = updatedRecipe.Servings // Important: Update servings for nutrition calculation
		// Skip nutrition fields - they will be calculated with proper USDA data
		
		if err := h.llmService.UpdateDraft(c.Request.Context(), draft); err != nil {
			fmt.Printf("[LLMHandler] Error updating draft: %v\n", err)
			// Don't increment rate limit counter on save failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Recalculate nutrition based on modified ingredients using USDA data
		fmt.Printf("[LLMHandler] Recalculating nutrition for modified recipe with draft ID: %s\n", draft.ID)
		fmt.Printf("[LLMHandler] Draft servings before nutrition calc: %+v\n", draft.Servings)
		nutritionData, err := h.llmService.CalculateRecipeNutrition(c.Request.Context(), draft.ID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error recalculating nutrition for modified recipe: %v\n", err)
			// Don't fail the modification - nutrition calculation is enhancement
			// Continue with the modification but log the issue
		} else {
			fmt.Printf("[LLMHandler] Successfully recalculated nutrition for modified recipe\n")
			fmt.Printf("[LLMHandler] Nutrition data returned: %+v\n", nutritionData)
		}

		// Fetch the updated draft with nutrition data
		updatedDraft, err := h.llmService.GetDraft(c.Request.Context(), draft.ID)
		if err != nil {
			fmt.Printf("[LLMHandler] Error fetching updated draft with nutrition: %v\n", err)
			// Use original draft if fetch fails
			updatedDraft = draft
		} else {
			fmt.Printf("[LLMHandler] Successfully fetched updated draft with nutrition data\n")
			fmt.Printf("[LLMHandler] Updated draft nutrition - Calories: %f, Protein: %f, Carbs: %f, Fat: %f\n", 
				updatedDraft.Calories, updatedDraft.Protein, updatedDraft.Carbs, updatedDraft.Fat)
			fmt.Printf("[LLMHandler] Updated draft per-serving - CaloriesPerServing: %f, ProteinPerServing: %f\n", 
				updatedDraft.CaloriesPerServing, updatedDraft.ProteinPerServing)
		}

		// Only increment rate limit counter on successful modification and save
		if h.creationLimiter != nil {
			if err := h.creationLimiter.IncrementUsage(c.Request.Context(), userID.String()); err != nil {
				fmt.Printf("[LLMHandler] Failed to increment rate limit for user %s: %v\n", userID.String(), err)
			} else {
				fmt.Printf("[LLMHandler] Rate limit incremented for successful modification by user %s\n", userID.String())
			}
		}

		fmt.Printf("[LLMHandler] Successfully modified and updated draft. Draft ID: %s\n", updatedDraft.ID)
		c.JSON(http.StatusOK, gin.H{
			"recipe":   updatedDraft,
			"draft_id": updatedDraft.ID,
		})
		fmt.Println("[LLMHandler] Responded 200 OK with modified recipe and draft_id.")
	default:
		fmt.Printf("[LLMHandler] Invalid intent: %s. Responding 400.\n", req.Intent)
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

	draft, err := h.llmService.GetDraft(c.Request.Context(), draftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	// Verify ownership
	if draft.UserID != userID.String() {
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

	// Get draft to verify ownership
	draft, err := h.llmService.GetDraft(c.Request.Context(), draftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	// Verify ownership
	if draft.UserID != userID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.llmService.DeleteDraft(c.Request.Context(), draftID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "draft deleted"})
}

// GenerateBasicRecipe handles basic recipe generation without nutrition data
func (h *LLMHandler) GenerateBasicRecipe(c *gin.Context) {
	var req BasicRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	// Fetch user with dietary preferences and allergens
	var user models.User
	err := h.db.Preload("DietaryPrefs").Where("id = ?", userID).First(&user).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user preferences"})
		return
	}
	
	// Try to preload allergens separately to avoid breaking if table doesn't exist
	_ = h.db.Model(&user).Association("Allergens").Find(&user.Allergens)

	// Extract dietary preferences and allergens into string arrays
	var dietaryPrefs []string
	for _, pref := range user.DietaryPrefs {
		if pref.PreferenceType != "" {
			dietaryPrefs = append(dietaryPrefs, pref.PreferenceType)
		} else if pref.CustomName != "" {
			dietaryPrefs = append(dietaryPrefs, pref.CustomName)
		}
	}

	var allergens []string
	for _, allergen := range user.Allergens {
		allergens = append(allergens, allergen.AllergenName)
	}

	// Check rate limiting before attempting generation
	if h.creationLimiter != nil {
		allowed, remaining, resetTime, err := h.creationLimiter.CheckOnly(c.Request.Context(), userID.String())
		if err != nil {
			// Continue without rate limiting on error
		} else if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":                "rate limit exceeded",
				"rate_limit_remaining": remaining,
				"rate_limit_reset":     resetTime.Unix(),
			})
			return
		}
	}

	// Generate basic recipe without nutrition data
	draft, err := h.llmService.GenerateBasicRecipe(c.Request.Context(), req.Query, dietaryPrefs, allergens, userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Only increment rate limit counter on successful generation and save
	if h.creationLimiter != nil {
		if err := h.creationLimiter.IncrementUsage(c.Request.Context(), userID.String()); err != nil {
			// Log error but don't fail the request
		}
	}

	response := BasicRecipeResponse{
		DraftID: draft.ID,
		Recipe: &service.BasicRecipe{
			Name:         draft.Name,
			Description:  draft.Description,
			Category:     draft.Category,
			Cuisine:      draft.Cuisine,
			Ingredients:  draft.Ingredients,
			Instructions: draft.Instructions,
			PrepTime:     draft.PrepTime,
			CookTime:     draft.CookTime,
			Servings:     draft.Servings,
			Difficulty:   draft.Difficulty,
		},
		Status: "basic_generated",
	}

	c.JSON(http.StatusOK, response)
}

// CalculateNutrition handles nutrition calculation for an existing draft
func (h *LLMHandler) CalculateNutrition(c *gin.Context) {
	var req NutritionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	// Verify draft ownership
	draft, err := h.llmService.GetDraft(c.Request.Context(), req.DraftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	if draft.UserID != userID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Calculate nutrition for the draft
	macros, err := h.llmService.CalculateRecipeNutrition(c.Request.Context(), req.DraftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := NutritionResponse{
		DraftID:  req.DraftID,
		Calories: macros.Calories,
		Protein:  macros.Protein,
		Carbs:    macros.Carbs,
		Fat:      macros.Fat,
		Status:   "nutrition_calculated",
	}

	c.JSON(http.StatusOK, response)
}

// FinalizeRecipe handles finalizing a recipe with all data
func (h *LLMHandler) FinalizeRecipe(c *gin.Context) {
	var req FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	// Verify draft ownership
	draft, err := h.llmService.GetDraft(c.Request.Context(), req.DraftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	if draft.UserID != userID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Finalize the recipe
	finalDraft, err := h.llmService.FinalizeRecipe(c.Request.Context(), req.DraftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := FinalizeResponse{
		DraftID: req.DraftID,
		Recipe:  finalDraft,
		Status:  "finalized",
	}

	c.JSON(http.StatusOK, response)
}
