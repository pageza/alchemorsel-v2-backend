package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
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
	apiKey string
	apiURL string
	redis  *redis.Client
}

// NewLLMService creates a new LLMService instance
func NewLLMService() (*LLMService, error) {
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

	apiURL := os.Getenv("DEEPSEEK_API_URL")
	if apiURL == "" {
		apiURL = "https://api.deepseek.com/v1/chat/completions"
	}

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
		apiKey: apiKey,
		apiURL: apiURL,
		redis:  redisClient,
	}, nil
}

// Message represents a message in the chat
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents a request to the DeepSeek API
type Request struct {
	Model            string            `json:"model"`
	Messages         []Message         `json:"messages"`
	ResponseFormat   map[string]string `json:"response_format"`
	MaxTokens        int               `json:"max_tokens,omitempty"`
	Temperature      float64           `json:"temperature"`
	TopP             float64           `json:"top_p"`
	FrequencyPenalty float64           `json:"frequency_penalty"`
	PresencePenalty  float64           `json:"presence_penalty"`
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

// GenerateRecipe generates a recipe with retry logic for robustness
func (s *LLMService) GenerateRecipe(query string, dietaryPrefs, allergens []string, originalRecipe *RecipeDraft) (string, error) {
	const maxRetries = 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("[LLMHandler] Generation attempt %d/%d\n", attempt, maxRetries)

		content, err := s.generateRecipeAttempt(query, dietaryPrefs, allergens, originalRecipe)
		if err != nil {
			fmt.Printf("[LLMHandler] Attempt %d failed: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to generate recipe after %d attempts: %w", maxRetries, err)
			}
			continue
		}

		// Validate JSON by attempting to parse it
		var tempRecipe RecipeDraft
		if err := json.Unmarshal([]byte(content), &tempRecipe); err != nil {
			fmt.Printf("[LLMHandler] Attempt %d returned invalid JSON: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to generate valid JSON after %d attempts: %w", maxRetries, err)
			}
			continue
		}

		fmt.Printf("[LLMHandler] Successfully generated recipe on attempt %d\n", attempt)
		return content, nil
	}

	return "", fmt.Errorf("failed to generate recipe after %d attempts", maxRetries)
}

// generateRecipeAttempt performs a single attempt at recipe generation
func (s *LLMService) generateRecipeAttempt(query string, dietaryPrefs, allergens []string, originalRecipe *RecipeDraft) (string, error) {
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

	messages := []Message{
		{
			Role: "system",
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
			Role:    "user",
			Content: prompt,
		},
	}

	reqBody := Request{
		Model:    "deepseek-chat",
		Messages: messages,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
		MaxTokens:        4096, // Much higher limit to prevent cutoff
		Temperature:      0.2,  // Low temperature for reliable JSON formatting
		TopP:             0.8,  // Focused sampling for structured output
		FrequencyPenalty: 0.5,  // Penalize repeated tokens
		PresencePenalty:  0.5,  // Encourage new topics
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// Create HTTP client with longer timeout for complex recipe generation
	client := &http.Client{
		Timeout: 120 * time.Second, // Increased to 120 seconds for complex recipes
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Read response with detailed logging
	fmt.Printf("[LLMHandler] Response status: %d\n", resp.StatusCode)
	fmt.Printf("[LLMHandler] Response headers: %v\n", resp.Header)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("[LLMHandler] Raw response length: %d bytes\n", len(body))
	fmt.Printf("[LLMHandler] RAW DEEPSEEK RESPONSE: %s\n", string(body))

	// Check if response was potentially truncated
	if len(body) > 0 && !strings.HasSuffix(string(body), "}") {
		fmt.Printf("[LLMHandler] WARNING: Response appears truncated (doesn't end with })\n")
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	// Apply JSON fixes before returning
	content := result.Choices[0].Message.Content
	fmt.Printf("[LLMHandler] EXTRACTED CONTENT BEFORE FIXES: %s\n", content)
	content = fixDeepSeekJSON(content)
	fmt.Printf("[LLMHandler] EXTRACTED CONTENT AFTER FIXES: %s\n", content)

	return content, nil
}

// fixDeepSeekJSON fixes common JSON formatting issues from DeepSeek API
func fixDeepSeekJSON(content string) string {
	fmt.Printf("[LLMHandler] Fixing DeepSeek JSON formatting issues...\n")

	// 1. Fix incomplete JSON (missing closing brace)
	trimmed := strings.TrimSpace(content)
	if !strings.HasSuffix(trimmed, "}") {
		fmt.Printf("[LLMHandler] Adding missing closing brace\n")
		// LINT-FIX-2025: Use unconditional TrimSuffix instead of conditional check
		// gosimple suggests this pattern is more idiomatic and handles edge cases better
		trimmed = strings.TrimSuffix(trimmed, ",")
		// Handle incomplete field values by closing them properly
		if strings.HasSuffix(trimmed, `"difficulty": "Easy"`) ||
			strings.HasSuffix(trimmed, `"difficulty": "Medium"`) ||
			strings.HasSuffix(trimmed, `"difficulty": "Hard"`) {
			content = trimmed + "\n}"
		} else if strings.Contains(trimmed, `"difficulty":`) {
			// If difficulty field is incomplete, fix it
			lastCommaIndex := strings.LastIndex(trimmed, ",")
			if lastCommaIndex > 0 {
				content = trimmed[:lastCommaIndex] + "\n}"
			} else {
				content = trimmed + "\n}"
			}
		} else {
			content = trimmed + "\n}"
		}
	}

	// 2. Fix single quotes to double quotes for JSON compliance
	if strings.Contains(content, "'") {
		fmt.Printf("[LLMHandler] Converting single quotes to double quotes\n")
		content = strings.ReplaceAll(content, "'", "\"")
	}

	// 3. Fix double double quotes (if any)
	if strings.Contains(content, `""`) {
		fmt.Printf("[LLMHandler] Fixing double quotes\n")
		content = strings.ReplaceAll(content, `""`, `"`)
	}

	// 4. Remove empty string entries in arrays
	if strings.Contains(content, `""`) || strings.Contains(content, `"",`) {
		fmt.Printf("[LLMHandler] Cleaning up empty entries\n")
		content = strings.ReplaceAll(content, `,\n        ""`, "")
		content = strings.ReplaceAll(content, `""`, "")
	}

	// 5. Fix incomplete field values with missing quotes
	if strings.Contains(content, `"difficulty": Easy`) {
		fmt.Printf("[LLMHandler] Fixing missing quotes around difficulty value\n")
		content = strings.ReplaceAll(content, `"difficulty": Easy`, `"difficulty": "Easy"`)
	}
	if strings.Contains(content, `"difficulty": Medium`) {
		content = strings.ReplaceAll(content, `"difficulty": Medium`, `"difficulty": "Medium"`)
	}
	if strings.Contains(content, `"difficulty": Hard`) {
		content = strings.ReplaceAll(content, `"difficulty": Hard`, `"difficulty": "Hard"`)
	}

	fmt.Printf("[LLMHandler] JSON formatting fixes applied\n")
	return content
}

// CalculateMacros estimates the macronutrients for a set of ingredients
func (s *LLMService) CalculateMacros(ingredients []string) (*Macros, error) {
	prompt := "Provide an approximate macronutrient breakdown as JSON with fields calories, protein, carbs and fat for the following ingredients:" + "\n" + strings.Join(ingredients, "\n")
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a nutrition expert. Respond only with JSON like {\"calories\":0,\"protein\":0,\"carbs\":0,\"fat\":0}",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	reqBody := Request{
		Model:    "deepseek-chat",
		Messages: messages,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read error response: %w", readErr)
		}
		log.Printf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	var macros Macros
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &macros); err != nil {
		return nil, fmt.Errorf("failed to parse macros: %w", err)
	}

	return &macros, nil
}

// GenerateRecipesBatch generates multiple recipes in a single batch
func (s *LLMService) GenerateRecipesBatch(prompts []string) ([]string, error) {
	// Create a batch request with all prompts
	messages := []Message{
		{
			Role: "system",
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
		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Generate a recipe for: %s", prompt),
		})
	}

	// Create the request
	req := Request{
		Model:    "deepseek-chat",
		Messages: messages,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
		Temperature:      0.2, // Low temperature for reliable JSON formatting
		TopP:             0.8, // Focused sampling for structured output
		FrequencyPenalty: 0.5, // Penalize repeated tokens
		PresencePenalty:  0.5, // Encourage new topics
	}

	// Marshal the request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// After receiving the response from DeepSeek, before parsing:
	log.Printf("Raw DeepSeek response: %s", string(body))

	// Parse response
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
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
