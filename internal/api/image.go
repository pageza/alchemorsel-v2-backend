package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// ImageHandler handles image generation requests
type ImageHandler struct {
	db           *gorm.DB
	imageService service.IImageService
	llmService   service.LLMServiceInterface
	authService  *service.AuthService
	rateLimiter  *middleware.RateLimiter
}

// NewImageHandler creates a new image handler
func NewImageHandler(db *gorm.DB, imageService service.IImageService, llmService service.LLMServiceInterface, authService *service.AuthService, rateLimiter *middleware.RateLimiter) *ImageHandler {
	return &ImageHandler{
		db:           db,
		imageService: imageService,
		llmService:   llmService,
		authService:  authService,
		rateLimiter:  rateLimiter,
	}
}

// GenerateRecipeImageRequest represents the request body for recipe image generation
type GenerateRecipeImageRequest struct {
	DraftID string `json:"draft_id" binding:"required"`
}

// GenerateRecipeImageResponse represents the response for recipe image generation
type GenerateRecipeImageResponse struct {
	DraftID  string `json:"draft_id"`
	ImageURL string `json:"image_url"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// GenerateImageFromPromptRequest represents the request for generating image from prompt
type GenerateImageFromPromptRequest struct {
	Prompt string `json:"prompt" binding:"required"`
	Size   string `json:"size,omitempty"` // Optional, defaults to "1024x1024"
}

// GenerateImageFromPromptResponse represents the response for prompt-based image generation
type GenerateImageFromPromptResponse struct {
	ImageURL string `json:"image_url"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// GenerateRecipeImage generates an image for a recipe draft
func (h *ImageHandler) GenerateRecipeImage(c *gin.Context) {

	var req GenerateRecipeImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Get the recipe draft
	draft, err := h.llmService.GetDraft(c.Request.Context(), req.DraftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Draft not found",
			"message": "The specified recipe draft could not be found",
		})
		return
	}

	// Generate image for the recipe
	imageURL, err := h.imageService.GenerateRecipeImage(c.Request.Context(), draft)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Image generation failed",
			"message": err.Error(),
		})
		return
	}

	// Update the draft with the image URL
	draft.ImageURL = imageURL
	if err := h.llmService.UpdateDraft(c.Request.Context(), draft); err != nil {
		// Log the error but don't fail the request since we have the image
		// The frontend can still use the image URL
		c.JSON(http.StatusOK, GenerateRecipeImageResponse{
			DraftID:  req.DraftID,
			ImageURL: imageURL,
			Status:   "success",
			Message:  "Image generated successfully, but draft update failed",
		})
		return
	}

	c.JSON(http.StatusOK, GenerateRecipeImageResponse{
		DraftID:  req.DraftID,
		ImageURL: imageURL,
		Status:   "success",
	})
}

// GenerateImageFromPrompt generates an image from a text prompt
func (h *ImageHandler) GenerateImageFromPrompt(c *gin.Context) {

	var req GenerateImageFromPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Default size if not specified
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	// Validate size
	validSizes := []string{"1024x1024", "1024x1792", "1792x1024"}
	validSize := false
	for _, validSz := range validSizes {
		if size == validSz {
			validSize = true
			break
		}
	}
	if !validSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid size",
			"message": "Size must be one of: 1024x1024, 1024x1792, 1792x1024",
		})
		return
	}

	// Generate image
	imageURL, err := h.imageService.GenerateImageFromPrompt(c.Request.Context(), req.Prompt, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Image generation failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateImageFromPromptResponse{
		ImageURL: imageURL,
		Status:   "success",
	})
}

// RegisterRoutes registers the image generation routes
func (h *ImageHandler) RegisterRoutes(router *gin.RouterGroup) {
	imageRoutes := router.Group("/images")
	imageRoutes.Use(middleware.AuthMiddleware(h.authService))
	
	// Apply rate limiting if available
	if h.rateLimiter != nil {
		imageRoutes.Use(h.rateLimiter.RateLimitMiddleware())
	}
	
	{
		imageRoutes.POST("/generate-recipe", h.GenerateRecipeImage)
		imageRoutes.POST("/generate", h.GenerateImageFromPrompt)
	}
}