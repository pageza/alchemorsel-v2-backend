package api

// RecipeResponse represents the response structure for recipe-related API endpoints
type RecipeResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Category     string    `json:"category"`
	Ingredients  []string  `json:"ingredients"`
	Instructions []string  `json:"instructions"`
	Calories     int       `json:"calories"`
	Protein      int       `json:"protein"`
	Carbs        int       `json:"carbs"`
	Fat          int       `json:"fat"`
	Embedding    []float32 `json:"embedding"`
}
