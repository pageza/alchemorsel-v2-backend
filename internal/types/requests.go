package types

import (
	"time"

	"github.com/google/uuid"
)

// CreateRecipeRequest represents the request body for creating a recipe
type CreateRecipeRequest struct {
	Name               string   `json:"name" binding:"required"`
	Description        string   `json:"description" binding:"required"`
	Category           string   `json:"category" binding:"required"`
	Cuisine            string   `json:"cuisine"`
	ImageURL           string   `json:"image_url"`
	Ingredients        []string `json:"ingredients" binding:"required"`
	Instructions       []string `json:"instructions" binding:"required"`
	Calories           float64  `json:"calories"`
	Protein            float64  `json:"protein"`
	Carbs              float64  `json:"carbs"`
	Fat                float64  `json:"fat"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Tags               []string `json:"tags"`
}

// UpdateRecipeRequest represents the request body for updating a recipe
type UpdateRecipeRequest struct {
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

// Feedback API types
type CreateFeedbackRequest struct {
	Type        string `json:"type" binding:"required,oneof=bug feature general"`
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description" binding:"required,max=2000"`
	Priority    string `json:"priority" binding:"oneof=low medium high critical"`
	UserAgent   string `json:"user_agent"`
	URL         string `json:"url"`
}

type FeedbackResponse struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	UserAgent   string     `json:"user_agent,omitempty"`
	URL         string     `json:"url,omitempty"`
	AdminNotes  string     `json:"admin_notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
}

type UpdateFeedbackStatusRequest struct {
	Status     string `json:"status" binding:"required,oneof=open in_progress resolved closed"`
	AdminNotes string `json:"admin_notes"`
}
