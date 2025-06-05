package model

// Macros represents nutrition information for a recipe.
type Macros struct {
	Calories int     `json:"calories"`
	Protein  float64 `json:"protein"`
	Fat      float64 `json:"fat"`
	Carbs    float64 `json:"carbs"`
}
