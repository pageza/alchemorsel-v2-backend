package types

import (
	"encoding/json"

	"github.com/google/uuid"
)

// TestUser represents a test user
type TestUser struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
}

// TestUserPreference represents a test user preference
type TestUserPreference struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	DietaryPrefs    []string  `json:"dietary_prefs"`
	Allergies       []string  `json:"allergies"`
	FavoriteCuisine string    `json:"favorite_cuisine"`
}

// TestRecipe represents a test recipe
type TestRecipe struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Ingredients  []string  `json:"ingredients"`
	Instructions []string  `json:"instructions"`
	ImageURL     string    `json:"image_url"`
	PrepTime     int       `json:"prep_time"`
	CookTime     int       `json:"cook_time"`
	Servings     int       `json:"servings"`
	Difficulty   string    `json:"difficulty"`
}

// TestRecipeFavorite represents a test recipe favorite
type TestRecipeFavorite struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	RecipeID uuid.UUID `json:"recipe_id"`
}

// TestJSONStringArray is a custom type for JSON string arrays
type TestJSONStringArray []string

// MarshalJSON implements the json.Marshaler interface
func (a TestJSONStringArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (a *TestJSONStringArray) UnmarshalJSON(data []byte) error {
	var s []string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*a = TestJSONStringArray(s)
	return nil
}
