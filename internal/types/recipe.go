package types

import (
	"time"

	"github.com/google/uuid"
)

// Recipe represents a recipe in the system
type Recipe struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Category           string    `json:"category"`
	Cuisine            string    `json:"cuisine"`
	ImageURL           string    `json:"image_url"`
	Ingredients        []string  `json:"ingredients"`
	Instructions       []string  `json:"instructions"`
	Calories           float64   `json:"calories"`
	Protein            float64   `json:"protein"`
	Carbs              float64   `json:"carbs"`
	Fat                float64   `json:"fat"`
	DietaryPreferences []string  `json:"dietary_preferences"`
	Tags               []string  `json:"tags"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
