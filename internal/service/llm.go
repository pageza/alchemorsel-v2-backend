package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
	"github.com/trustsight-io/deepseek-go"
)

// RecipeData represents the structure of a recipe as returned by the LLM
type RecipeData struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Cuisine      string   `json:"cuisine"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
	PrepTime     string   `json:"prep_time"`
	CookTime     string   `json:"cook_time"`
	Servings     string   `json:"servings"`
	Difficulty   string   `json:"difficulty"`
}

// LLMService handles interactions with the DeepSeek API
type LLMService struct {
	client         *deepseek.Client
	redis          *redis.Client
	jsonExtractor  *deepseek.JSONExtractor
}

// NewLLMService creates a new LLMService instance
func NewLLMService() (*LLMService, error) {
	// Get API key with fallback to file
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKeyFile := os.Getenv("DEEPSEEK_API_KEY_FILE")
		if apiKeyFile == "" {
			return nil, fmt.Errorf("DEEPSEEK_API_KEY or DEEPSEEK_API_KEY_FILE must be set")
		}

		apiKeyBytes, err := os.ReadFile(apiKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read API key file: %w", err)
		}

		apiKey = strings.TrimSpace(string(apiKeyBytes))
		if apiKey == "" {
			return nil, fmt.Errorf("API key file is empty")
		}
	}

	// Create DeepSeek client with options
	var clientOpts []deepseek.ClientOption
	
	// Set custom API URL if provided
	if apiURL := os.Getenv("DEEPSEEK_API_URL"); apiURL != "" {
		clientOpts = append(clientOpts, deepseek.WithBaseURL(apiURL))
	}
	
	// Enable debug mode if requested
	if os.Getenv("DEEPSEEK_DEBUG") == "true" {
		clientOpts = append(clientOpts, deepseek.WithDebug(true))
	}
	
	// Configure retry logic for handling rate limits and temporary failures
	clientOpts = append(clientOpts, deepseek.WithMaxRetries(3))
	clientOpts = append(clientOpts, deepseek.WithRetryWaitTime(2*time.Second))

	client, err := deepseek.NewClient(apiKey, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek client: %w", err)
	}

	// Define JSON schema for recipe data validation
	recipeSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 1},
			"description": {"type": "string", "minLength": 1},
			"category": {"type": "string", "minLength": 1},
			"cuisine": {"type": "string", "minLength": 1},
			"ingredients": {
				"type": "array",
				"items": {"type": "string", "minLength": 1},
				"minItems": 1
			},
			"instructions": {
				"type": "array",
				"items": {"type": "string", "minLength": 1},
				"minItems": 1
			},
			"prep_time": {"type": "string", "minLength": 1},
			"cook_time": {"type": "string", "minLength": 1},
			"servings": {"type": "string", "minLength": 1},
			"difficulty": {"type": "string", "enum": ["Easy", "Medium", "Hard"]},
			"calories": {"type": "number", "minimum": 0},
			"protein": {"type": "number", "minimum": 0},
			"carbs": {"type": "number", "minimum": 0},
			"fat": {"type": "number", "minimum": 0}
		},
		"required": ["name", "description", "category", "ingredients", "instructions", "prep_time", "cook_time", "servings", "difficulty", "calories", "protein", "carbs", "fat"],
		"additionalProperties": false
	}`)

	// Create JSON extractor with schema validation
	jsonExtractor := deepseek.NewJSONExtractor(recipeSchema)

	// Initialize Redis client with environment variables
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       redisDB,
	})

	return &LLMService{
		client:        client,
		redis:         redisClient,
		jsonExtractor: jsonExtractor,
	}, nil
}


// Macros represents nutritional macros information
type Macros struct {
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Carbs    float64 `json:"carbs"`
	Fat      float64 `json:"fat"`
}

// ServingsType can handle both string and number values for servings
type ServingsType struct {
	Value string
}

func (s *ServingsType) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as number first
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		s.Value = fmt.Sprintf("%d", int(num))
		return nil
	}

	// Try to unmarshal as string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.Value = str
		return nil
	}

	// Try to unmarshal as object with Value field
	var obj struct {
		Value string `json:"Value"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		s.Value = obj.Value
		return nil
	}

	return fmt.Errorf("invalid servings format")
}

// RecipeDraft represents a recipe in draft state
type RecipeDraft struct {
	ID           string          `json:"id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	Ingredients  []string        `json:"ingredients"`
	Instructions []string        `json:"instructions"`
	PrepTime     string          `json:"prep_time"`
	CookTime     string          `json:"cook_time"`
	Servings     ServingsType    `json:"servings"`
	Difficulty   string          `json:"difficulty"`
	Calories     float64         `json:"calories"`
	Protein      float64         `json:"protein"`
	Carbs        float64         `json:"carbs"`
	Fat          float64         `json:"fat"`
	UserID       string          `json:"user_id"`
	Embedding    pgvector.Vector `json:"embedding"`
}

// SaveDraft saves a recipe draft to Redis
func (s *LLMService) SaveDraft(ctx context.Context, draft *RecipeDraft) error {
	draft.ID = uuid.New().String()
	draft.CreatedAt = time.Now()
	draft.UpdatedAt = time.Now()

	data, err := json.Marshal(draft)
	if err != nil {
		return fmt.Errorf("failed to marshal draft: %w", err)
	}

	key := fmt.Sprintf("recipe:draft:%s", draft.ID)
	err = s.redis.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save draft to Redis: %w", err)
	}

	return nil
}

// GetDraft retrieves a recipe draft from Redis
func (s *LLMService) GetDraft(ctx context.Context, id string) (*RecipeDraft, error) {
	key := fmt.Sprintf("recipe:draft:%s", id)
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get draft from Redis: %w", err)
	}

	var draft RecipeDraft
	if err := json.Unmarshal(data, &draft); err != nil {
		return nil, fmt.Errorf("failed to unmarshal draft: %w", err)
	}

	return &draft, nil
}

// UpdateDraft updates a recipe draft in Redis
func (s *LLMService) UpdateDraft(ctx context.Context, draft *RecipeDraft) error {
	draft.UpdatedAt = time.Now()

	data, err := json.Marshal(draft)
	if err != nil {
		return fmt.Errorf("failed to marshal draft: %w", err)
	}

	key := fmt.Sprintf("recipe:draft:%s", draft.ID)
	err = s.redis.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to update draft in Redis: %w", err)
	}

	return nil
}

// DeleteDraft removes a recipe draft from Redis
func (s *LLMService) DeleteDraft(ctx context.Context, id string) error {
	key := fmt.Sprintf("recipe:draft:%s", id)
	err := s.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete draft from Redis: %w", err)
	}

	return nil
}

// GenerateRecipe generates a recipe using the DeepSeek API with proper JSON extraction
func (s *LLMService) GenerateRecipe(query string, dietaryPrefs, allergens []string, originalRecipe *RecipeDraft) (string, error) {
	// Build the user prompt based on request type
	var prompt string
	if originalRecipe != nil {
		// For modifications, include the original recipe in the prompt
		prompt = fmt.Sprintf("Modify this recipe: %s\n\nOriginal recipe:\nName: %s\nDescription: %s\nIngredients: %s\nInstructions: %s\n\nModification request: %s",
			originalRecipe.Name,
			originalRecipe.Name,
			originalRecipe.Description,
			strings.Join(originalRecipe.Ingredients, "\n"),
			strings.Join(originalRecipe.Instructions, "\n"),
			query)
	} else {
		// For new recipes
		prompt = fmt.Sprintf("Generate a recipe for: %s", query)
		if len(dietaryPrefs) > 0 {
			prompt += ". The recipe should be suitable for: " + strings.Join(dietaryPrefs, ", ")
		}
		if len(allergens) > 0 {
			prompt += ". Avoid using: " + strings.Join(allergens, ", ")
		}
	}

	// Create messages using the DeepSeek client Message type
	messages := []deepseek.Message{
		{
			Role: deepseek.RoleSystem,
			Content: `You are a professional chef and nutritionist. Please provide your response in JSON format with the following structure:
{
    "name": "Recipe name",
    "description": "Brief description of the recipe",
    "category": "One of: Main Course, Dessert, Snack, Appetizer, Breakfast, Lunch, Dinner, Side Dish, Beverage, Soup, Salad, Bread, Pasta, Seafood, Meat, Vegetarian, Vegan, Gluten-Free",
    "cuisine": "One of: Italian, French, Chinese, Japanese, Thai, Indian, Mexican, Mediterranean, American, British, German, Korean, Spanish, Brazilian, Moroccan, Fusion, or Other",
    "ingredients": [
        "2 cups flour",
        "1 cup sugar",
        "3 eggs"
    ],
    "instructions": [
        "Step 1: Mix the dry ingredients",
        "Step 2: Add the wet ingredients",
        "Step 3: Bake at 350°F for 30 minutes"
    ],
    "prep_time": "Preparation time",
    "cook_time": "Cooking time",
    "servings": "Number of servings",
    "difficulty": "Easy/Medium/Hard",
    "calories": 350,
    "protein": 15,
    "carbs": 45,
    "fat": 12
}

Note: The calories, protein, carbs, and fat fields must be numbers, not strings.
The category field MUST be one of the listed categories above.
The cuisine field MUST be one of the listed cuisines above.`,
		},
		{
			Role:    deepseek.RoleUser,
			Content: prompt,
		},
	}

	// Create request using the DeepSeek client
	request := deepseek.ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: messages,
		ResponseFormat: &struct {
			Type string `json:"type"`
		}{
			Type: "json_object",
		},
		MaxTokens:        4096, // Prevent cutoff
		Temperature:      0.7,  // Balanced creativity
		TopP:             0.9,  // Diverse outputs
		FrequencyPenalty: 0.5,  // Penalize repetition
		PresencePenalty:  0.5,  // Encourage new topics
	}

	// Make the API call using the client with enhanced error handling
	log.Printf("[LLMService] Making recipe generation request for query: %s", query)
	ctx := context.Background()
	response, err := s.client.CreateChatCompletion(ctx, &request)
	if err != nil {
		log.Printf("[LLMService] API call failed: %v", err)
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	// Validate response structure
	if len(response.Choices) == 0 {
		log.Printf("[LLMService] No choices returned in API response")
		return "", fmt.Errorf("no response choices from API")
	}

	log.Printf("[LLMService] Received response with %d choices", len(response.Choices))

	// Extract JSON using the JSONExtractor with schema validation
	var recipeData RecipeData
	if err := s.jsonExtractor.ExtractJSON(response, &recipeData); err != nil {
		log.Printf("[LLMService] JSON extraction failed: %v", err)
		log.Printf("[LLMService] Raw response content: %s", response.Choices[0].Message.Content)
		return "", fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	log.Printf("[LLMService] Successfully extracted recipe: %s", recipeData.Name)

	// Marshal the validated recipe data back to JSON string
	recipeJSON, err := json.Marshal(recipeData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validated recipe data: %w", err)
	}

	return string(recipeJSON), nil
}


// CalculateMacros estimates the macronutrients for a set of ingredients using the DeepSeek client
func (s *LLMService) CalculateMacros(ingredients []string) (*Macros, error) {
	prompt := "Provide an approximate macronutrient breakdown as JSON with fields calories, protein, carbs and fat for the following ingredients:" + "\n" + strings.Join(ingredients, "\n")
	
	messages := []deepseek.Message{
		{
			Role:    deepseek.RoleSystem,
			Content: "You are a nutrition expert. Respond only with JSON like {\"calories\":0,\"protein\":0,\"carbs\":0,\"fat\":0}",
		},
		{
			Role:    deepseek.RoleUser,
			Content: prompt,
		},
	}

	request := deepseek.ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: messages,
		ResponseFormat: &struct {
			Type string `json:"type"`
		}{
			Type: "json_object",
		},
		Temperature: 0.3, // Lower temperature for more consistent nutrition data
	}

	ctx := context.Background()
	response, err := s.client.CreateChatCompletion(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	// Extract the content from the first choice
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from API")
	}

	var macros Macros
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &macros); err != nil {
		return nil, fmt.Errorf("failed to parse macros from response: %w", err)
	}

	return &macros, nil
}

// GenerateRecipesBatch generates multiple recipes in a single batch using the DeepSeek client
func (s *LLMService) GenerateRecipesBatch(prompts []string) ([]string, error) {
	// Create a batch request with all prompts
	messages := []deepseek.Message{
		{
			Role: deepseek.RoleSystem,
			Content: `You are a professional chef and nutritionist. For each recipe prompt, provide your response in JSON format with the following structure:
{
    "name": "Recipe name",
    "description": "Brief description of the recipe",
    "category": "One of: Main Course, Dessert, Snack, Appetizer, Breakfast, Lunch, Dinner, Side Dish, Beverage, Soup, Salad, Bread, Pasta, Seafood, Meat, Vegetarian, Vegan, Gluten-Free",
    "ingredients": [
        "2 cups flour",
        "1 cup sugar",
        "3 eggs"
    ],
    "instructions": [
        "Step 1: Mix the dry ingredients",
        "Step 2: Add the wet ingredients",
        "Step 3: Bake at 350°F for 30 minutes"
    ],
    "prep_time": "Preparation time",
    "cook_time": "Cooking time",
    "servings": "Number of servings",
    "difficulty": "Easy/Medium/Hard"
}

Please provide each recipe as a separate JSON object in an array.`,
		},
	}

	// Add all prompts as user messages
	for _, prompt := range prompts {
		messages = append(messages, deepseek.Message{
			Role:    deepseek.RoleUser,
			Content: fmt.Sprintf("Generate a recipe for: %s", prompt),
		})
	}

	// Create the request
	request := deepseek.ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: messages,
		ResponseFormat: &struct {
			Type string `json:"type"`
		}{
			Type: "json_object",
		},
		Temperature:      0.9, // Higher temperature for more creativity
		TopP:             0.9, // Higher top_p for more diverse outputs
		FrequencyPenalty: 0.5, // Penalize repeated tokens
		PresencePenalty:  0.5, // Encourage new topics
	}

	// Make the API call with the client
	ctx := context.Background()
	response, err := s.client.CreateChatCompletion(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in API response")
	}

	content := response.Choices[0].Message.Content

	// Parse the content which contains the recipes array
	var recipesWrapper struct {
		Recipes []RecipeData `json:"recipes"`
	}
	if err := json.Unmarshal([]byte(content), &recipesWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse recipes array: %v", err)
	}

	// Convert each recipe to JSON string
	var recipesJSON []string
	for _, recipe := range recipesWrapper.Recipes {
		recipeJSON, err := json.Marshal(recipe)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal recipe: %v", err)
		}
		recipesJSON = append(recipesJSON, string(recipeJSON))
	}

	return recipesJSON, nil
}
