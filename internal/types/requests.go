package types

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
