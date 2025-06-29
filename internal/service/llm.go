package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
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
	apiKey           string
	apiURL           string
	redis            *redis.Client
	embeddingService EmbeddingServiceInterface
	imageService     IImageService
}

// NewLLMService creates a new LLMService instance
func NewLLMService() (*LLMService, error) {
	return NewLLMServiceWithEmbedding(nil)
}

// NewLLMServiceWithEmbedding creates a new LLMService instance with embedding service
func NewLLMServiceWithEmbedding(embeddingService EmbeddingServiceInterface) (*LLMService, error) {
	return NewLLMServiceWithServices(embeddingService, nil)
}

// NewLLMServiceWithServices creates a new LLMService instance with embedding and image services
func NewLLMServiceWithServices(embeddingService EmbeddingServiceInterface, imageService IImageService) (*LLMService, error) {
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
		apiKey:           apiKey,
		apiURL:           apiURL,
		redis:            redisClient,
		embeddingService: embeddingService,
		imageService:     imageService,
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

// BasicRecipe represents a simplified recipe without nutrition data
type BasicRecipe struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Category     string       `json:"category"`
	Cuisine      string       `json:"cuisine"`
	Ingredients  []string     `json:"ingredients"`
	Instructions []string     `json:"instructions"`
	PrepTime     string       `json:"prep_time"`
	CookTime     string       `json:"cook_time"`
	Servings     ServingsType `json:"servings"`
	Difficulty   string       `json:"difficulty"`
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
	Cuisine      string          `json:"cuisine"`
	ImageURL     string          `json:"image_url"`
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
	// Per-serving nutrition (calculated from total / servings)
	CaloriesPerServing float64    `json:"calories_per_serving"`
	ProteinPerServing  float64    `json:"protein_per_serving"`
	CarbsPerServing    float64    `json:"carbs_per_serving"`
	FatPerServing      float64    `json:"fat_per_serving"`
	UserID             string     `json:"user_id"`
	Embedding          pgvector.Vector `json:"embedding"`
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

// GenerateBasicRecipe generates a basic recipe without nutrition data for faster response
func (s *LLMService) GenerateBasicRecipe(ctx context.Context, query string, dietaryPrefs []string, allergens []string, userID string) (*RecipeDraft, error) {
	// Generate basic recipe with simplified prompt
	basicRecipeJSON, err := s.generateBasicRecipeAttempt(query, dietaryPrefs, allergens)
	if err != nil {
		return nil, fmt.Errorf("failed to generate basic recipe: %w", err)
	}

	// Parse the basic recipe
	var basicRecipe BasicRecipe
	if err := json.Unmarshal([]byte(basicRecipeJSON), &basicRecipe); err != nil {
		// Log the JSON that failed to parse for debugging
		fmt.Printf("[LLMHandler] Failed to parse basic recipe JSON. Raw JSON: %s\n", basicRecipeJSON)
		fmt.Printf("[LLMHandler] Parse error: %v\n", err)
		return nil, fmt.Errorf("failed to parse basic recipe JSON: %w", err)
	}

	// Convert to RecipeDraft (with zero nutrition values initially)
	draft := &RecipeDraft{
		Name:         basicRecipe.Name,
		Description:  basicRecipe.Description,
		Category:     basicRecipe.Category,
		Cuisine:      basicRecipe.Cuisine,
		Ingredients:  basicRecipe.Ingredients,
		Instructions: basicRecipe.Instructions,
		PrepTime:     basicRecipe.PrepTime,
		CookTime:     basicRecipe.CookTime,
		Servings:     basicRecipe.Servings,
		Difficulty:   basicRecipe.Difficulty,
		Calories:     0, // Will be calculated separately
		Protein:      0,
		Carbs:        0,
		Fat:          0,
		CaloriesPerServing: 0, // Will be calculated separately
		ProteinPerServing:  0,
		CarbsPerServing:    0,
		FatPerServing:      0,
		UserID:       userID,
	}

	// Save the draft
	if err := s.SaveDraft(ctx, draft); err != nil {
		return nil, fmt.Errorf("failed to save basic recipe draft: %w", err)
	}

	return draft, nil
}

// CalculateRecipeNutrition calculates nutrition data for an existing recipe draft
func (s *LLMService) CalculateRecipeNutrition(ctx context.Context, draftID string) (*Macros, error) {
	log.Printf("NUTRITION DEBUG: CalculateRecipeNutrition called with draftID: %s", draftID)
	fmt.Printf("NUTRITION DEBUG: CalculateRecipeNutrition called with draftID: %s\n", draftID)
	
	// Get the draft
	fmt.Printf("[LLMHandler] CalculateRecipeNutrition - Getting draft ID: %s\n", draftID)
	draft, err := s.GetDraft(ctx, draftID)
	if err != nil {
		log.Printf("NUTRITION DEBUG: Failed to get draft: %v", err)
		fmt.Printf("[LLMHandler] CalculateRecipeNutrition - Failed to get draft: %v\n", err)
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}
	fmt.Printf("[LLMHandler] CalculateRecipeNutrition - Successfully got draft, calling CalculateMacros\n")

	// Calculate nutrition based on ingredients
	log.Printf("NUTRITION DEBUG: About to call CalculateMacros with %d ingredients", len(draft.Ingredients))
	macros, err := s.CalculateMacros(draft.Ingredients)
	if err != nil {
		log.Printf("NUTRITION DEBUG: CalculateMacros failed: %v", err)
		return nil, fmt.Errorf("failed to calculate nutrition: %w", err)
	}

	// Update the draft with total nutrition data
	draft.Calories = macros.Calories
	draft.Protein = macros.Protein
	draft.Carbs = macros.Carbs
	draft.Fat = macros.Fat
	
	// Calculate per-serving nutrition
	// Extract numeric value from serving string (e.g., "24 cookies" -> 24, "4 servings" -> 4)
	servingsStr := draft.Servings.Value
	var servings float64 = 1
	
	// Use regex to extract the first number from the string
	re := regexp.MustCompile(`\d+\.?\d*`)
	if matches := re.FindString(servingsStr); matches != "" {
		if parsedServings, err := strconv.ParseFloat(matches, 64); err == nil && parsedServings > 0 {
			servings = parsedServings
			log.Printf("NUTRITION DEBUG: Parsed servings from '%s' as %g", servingsStr, servings)
		} else {
			log.Printf("NUTRITION DEBUG: Could not parse extracted number '%s' from '%s', defaulting to 1", matches, servingsStr)
		}
	} else {
		log.Printf("NUTRITION DEBUG: No number found in servings '%s', defaulting to 1", servingsStr)
	}
	
	// Round to 1 decimal place for cleaner display
	draft.CaloriesPerServing = math.Round(draft.Calories/servings*10) / 10
	draft.ProteinPerServing = math.Round(draft.Protein/servings*10) / 10
	draft.CarbsPerServing = math.Round(draft.Carbs/servings*10) / 10
	draft.FatPerServing = math.Round(draft.Fat/servings*10) / 10
	
	log.Printf("NUTRITION DEBUG: Total calories=%.0f, servings=%.0f, per serving=%.0f", 
		draft.Calories, servings, draft.CaloriesPerServing)

	// Save the updated draft
	if err := s.UpdateDraft(ctx, draft); err != nil {
		return nil, fmt.Errorf("failed to update draft with nutrition: %w", err)
	}

	return macros, nil
}

// FinalizeRecipe performs final processing and optimization of a recipe draft
func (s *LLMService) FinalizeRecipe(ctx context.Context, draftID string) (*RecipeDraft, error) {
	// Get the current draft
	draft, err := s.GetDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	// Generate embedding if we have an embedding service and the draft doesn't have one
	// Check if embedding is empty by converting to slice and checking length
	var embeddingEmpty bool
	if embeddingSlice := draft.Embedding.Slice(); len(embeddingSlice) == 0 {
		embeddingEmpty = true
	}
	
	if s.embeddingService != nil && embeddingEmpty {
		// Extract dietary preferences from category or other fields if needed
		// For now, we'll use an empty array since this info isn't stored in the draft
		dietaryPrefs := []string{}
		
		embedding, err := s.embeddingService.GenerateEmbeddingFromRecipe(
			draft.Name,
			draft.Description,
			draft.Ingredients,
			draft.Category,
			dietaryPrefs,
		)
		if err != nil {
			// Log the error but don't fail the finalization
			log.Printf("Warning: Failed to generate embedding for recipe %s: %v", draft.ID, err)
		} else {
			draft.Embedding = embedding
			draft.UpdatedAt = time.Now()
			
			// Save the updated draft with embedding
			if err := s.UpdateDraft(ctx, draft); err != nil {
				return nil, fmt.Errorf("failed to update draft with embedding: %w", err)
			}
		}
	}

	// Generate image if image service is available and no image exists
	if s.imageService != nil && draft.ImageURL == "" {
		log.Printf("Generating image for recipe %s", draft.ID)
		imageURL, err := s.imageService.GenerateRecipeImage(ctx, draft)
		if err != nil {
			// Log the error but don't fail the finalization
			log.Printf("Warning: Failed to generate image for recipe %s: %v", draft.ID, err)
		} else {
			draft.ImageURL = imageURL
			draft.UpdatedAt = time.Now()
			
			// Save the updated draft with image
			if err := s.UpdateDraft(ctx, draft); err != nil {
				log.Printf("Warning: Failed to update draft with image URL: %v", err)
			}
		}
	}

	return draft, nil
}

// generateBasicRecipeAttempt generates a basic recipe without nutrition data
func (s *LLMService) generateBasicRecipeAttempt(query string, dietaryPrefs, allergens []string) (string, error) {
	// Build dietary restrictions message
	var dietaryRestrictions string
	if len(dietaryPrefs) > 0 || len(allergens) > 0 {
		dietaryRestrictions = "\n\n⚠️ CRITICAL DIETARY REQUIREMENTS (MUST BE FOLLOWED):\n"
		if len(dietaryPrefs) > 0 {
			dietaryRestrictions += fmt.Sprintf("- This recipe MUST be suitable for: %s\n", strings.Join(dietaryPrefs, ", "))
			dietaryRestrictions += "- NEVER include ingredients that violate these dietary preferences\n"
		}
		if len(allergens) > 0 {
			dietaryRestrictions += fmt.Sprintf("- ABSOLUTELY AVOID these allergens: %s\n", strings.Join(allergens, ", "))
			dietaryRestrictions += "- Check ALL ingredients and sub-ingredients for these allergens\n"
		}
		dietaryRestrictions += "\nFAILURE TO FOLLOW THESE RESTRICTIONS COULD CAUSE SERIOUS HARM!"
	}

	prompt := fmt.Sprintf("Generate a recipe for: %s%s", query, dietaryRestrictions)

	messages := []Message{
		{
			Role: "system",
			Content: `You are a professional chef who STRICTLY RESPECTS dietary restrictions and allergens.

⚠️ CRITICAL SAFETY RULES:
1. When a user has dietary restrictions (vegan, vegetarian, gluten-free, etc.), you MUST ensure ALL ingredients comply
2. For vegan recipes: NO meat, dairy, eggs, honey, or ANY animal products
3. For vegetarian recipes: NO meat, poultry, or fish (dairy and eggs are allowed unless specified otherwise)
4. For gluten-free: NO wheat, barley, rye, or ingredients containing gluten
5. For dairy-free: NO milk, cheese, butter, cream, yogurt, or ANY dairy products
6. For allergens: NEVER include the specified allergens in ANY form, including traces or derivatives
7. ALWAYS suggest appropriate substitutes that maintain the recipe's integrity
8. BE DECISIVE: Choose specific substitute ingredients instead of giving options (e.g., use "seitan" not "seitan or tofu")
9. ADAPT RECIPE NAMES: Update the recipe name to reflect the actual ingredients used (e.g., "Seitan Noodle Soup" not "Vegan Chicken Noodle Soup")

🔧 REQUIRED FORMAT - CUSTOM STRUCTURED FORMAT ONLY:

You MUST respond using this EXACT Custom Structured Format:

NAME: Recipe Name Here
DESCRIPTION: Brief description of the recipe
CATEGORY: Main Course
CUISINE: Italian
INGREDIENTS: 2 cups flour|1 cup sugar|3 large eggs|1/2 cup butter
INSTRUCTIONS: step1|step2|step3|step4
PREP_TIME: 15 minutes
COOK_TIME: 30 minutes
SERVINGS: 4
DIFFICULTY: Easy

⚠️ CRITICAL FORMATTING RULES:
- Use exactly "NAME:", "DESCRIPTION:", etc. (uppercase with colon)
- INGREDIENTS MUST include specific quantities (e.g., "2 cups flour", "1 tbsp salt")
- Separate ingredients with pipe symbols (|)
- Separate instructions with pipe symbols (|)
- NO JSON, NO YAML, NO other formats - ONLY this Custom Structured Format
- Each field on its own line
- No extra text before or after the structured data

REMEMBER: User safety depends on you following dietary restrictions EXACTLY!`,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	reqBody := Request{
		Model:    "deepseek-chat",
		Messages: messages,
		// ResponseFormat: map[string]string{
		//	"type": "json_object",
		// }, // DISABLED - Testing alternative formats
		MaxTokens:        3072, // Reduced for basic recipe
		Temperature:      0.2,
		TopP:             0.8,
		FrequencyPenalty: 0.5,
		PresencePenalty:  0.5,
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

	client := &http.Client{
		Timeout: 60 * time.Second, // Shorter timeout for basic recipe
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
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

	content := result.Choices[0].Message.Content
	
	// Debug logging to see the raw DeepSeek response
	fmt.Printf("[LLMHandler] RAW DEEPSEEK RESPONSE: %s\n", content)
	
	// CUSTOM STRUCTURED FORMAT PROCESSING
	fmt.Printf("[LLMHandler] RAW DEEPSEEK RESPONSE: %s\n", content)
	
	// Parse Custom Structured Format and convert to JSON for existing system
	if jsonResult, success := parseCustomStructuredFormat(content); success {
		fmt.Printf("[LLMHandler] CUSTOM FORMAT PARSED SUCCESSFULLY: %s\n", jsonResult)
		return jsonResult, nil
	}
	
	// FALLBACK: Comment out JSON processing for now, return error instead
	/*
	// LEGACY JSON PROCESSING (COMMENTED OUT)
	content = fixDeepSeekJSON(content)
	fmt.Printf("[LLMHandler] FIXED JSON: %s\n", content)
	return content, nil
	*/
	
	fmt.Printf("[LLMHandler] FAILED TO PARSE CUSTOM FORMAT, CONTENT: %s\n", content)
	return "", fmt.Errorf("failed to parse custom structured format from DeepSeek response")
}

// generateRecipeAttempt performs a single attempt at recipe generation
func (s *LLMService) generateRecipeAttempt(query string, dietaryPrefs, allergens []string, originalRecipe *RecipeDraft) (string, error) {
	var prompt string
	
	// Build dietary restrictions message that applies to ALL recipe types
	var dietaryRestrictions string
	if len(dietaryPrefs) > 0 || len(allergens) > 0 {
		dietaryRestrictions = "\n\n⚠️ CRITICAL DIETARY REQUIREMENTS (MUST BE FOLLOWED):\n"
		if len(dietaryPrefs) > 0 {
			dietaryRestrictions += fmt.Sprintf("- This recipe MUST be suitable for: %s\n", strings.Join(dietaryPrefs, ", "))
			dietaryRestrictions += "- NEVER include ingredients that violate these dietary preferences\n"
		}
		if len(allergens) > 0 {
			dietaryRestrictions += fmt.Sprintf("- ABSOLUTELY AVOID these allergens: %s\n", strings.Join(allergens, ", "))
			dietaryRestrictions += "- Check ALL ingredients and sub-ingredients for these allergens\n"
		}
		dietaryRestrictions += "\nFAILURE TO FOLLOW THESE RESTRICTIONS COULD CAUSE SERIOUS HARM!"
	}
	
	if originalRecipe != nil {
		// For modifications, include the original recipe in the prompt
		prompt = fmt.Sprintf("Modify this recipe: %s\n\nOriginal recipe:\nName: %s\nDescription: %s\nIngredients: %s\nInstructions: %s\n\nModification request: %s%s",
			originalRecipe.Name,
			originalRecipe.Name,
			originalRecipe.Description,
			strings.Join(originalRecipe.Ingredients, "\n"),
			strings.Join(originalRecipe.Instructions, "\n"),
			query,
			dietaryRestrictions)
	} else {
		// For new recipes
		prompt = fmt.Sprintf("Generate a recipe for: %s%s", query, dietaryRestrictions)
	}

	messages := []Message{
		{
			Role: "system",
			Content: `You are a professional chef and nutritionist who STRICTLY RESPECTS dietary restrictions and allergens.

⚠️ CRITICAL SAFETY RULES:
1. When a user has dietary restrictions (vegan, vegetarian, gluten-free, etc.), you MUST ensure ALL ingredients comply
2. For vegan recipes: NO meat, dairy, eggs, honey, or ANY animal products
3. For vegetarian recipes: NO meat, poultry, or fish (dairy and eggs are allowed unless specified otherwise)
4. For gluten-free: NO wheat, barley, rye, or ingredients containing gluten
5. For dairy-free: NO milk, cheese, butter, cream, yogurt, or ANY dairy products
6. For allergens: NEVER include the specified allergens in ANY form, including traces or derivatives
7. ALWAYS suggest appropriate substitutes that maintain the recipe's integrity
8. BE DECISIVE: Choose specific substitute ingredients instead of giving options (e.g., use "seitan" not "seitan or tofu")
9. ADAPT RECIPE NAMES: Update the recipe name to reflect the actual ingredients used (e.g., "Seitan Noodle Soup" not "Vegan Chicken Noodle Soup")

🔧 REQUIRED FORMAT - CUSTOM STRUCTURED FORMAT ONLY:

You MUST respond using this EXACT Custom Structured Format:

NAME: Recipe Name Here
DESCRIPTION: Brief description of the recipe
CATEGORY: Main Course
CUISINE: Italian
INGREDIENTS: 2 cups flour|1 cup sugar|3 large eggs|1/2 cup butter
INSTRUCTIONS: step1|step2|step3|step4
PREP_TIME: 15 minutes
COOK_TIME: 30 minutes
SERVINGS: 4
DIFFICULTY: Easy

⚠️ CRITICAL FORMATTING RULES:
- Use exactly "NAME:", "DESCRIPTION:", etc. (uppercase with colon)
- INGREDIENTS MUST include specific quantities (e.g., "2 cups flour", "1 tbsp salt")
- Separate ingredients with pipe symbols (|)
- Separate instructions with pipe symbols (|)
- NO JSON, NO YAML, NO other formats - ONLY this Custom Structured Format
- Each field on its own line
- No extra text before or after the structured data

REMEMBER: User safety depends on you following dietary restrictions EXACTLY!`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf(`%s

Generate the recipe using the Custom Structured Format specified above.`, prompt),
		},
	}

	reqBody := Request{
		Model:    "deepseek-chat",
		Messages: messages,
		// ResponseFormat: map[string]string{
		//	"type": "json_object",
		// }, // DISABLED - Testing alternative formats
		MaxTokens:        3000, // Sufficient for recipe but not excessive
		Temperature:      0.1,  // Very low temperature for precise JSON formatting
		TopP:             0.7,  // More focused sampling for structured output
		FrequencyPenalty: 0.0,  // Don't penalize repetition of JSON structure
		PresencePenalty:  0.0,  // Don't discourage standard JSON format
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
	defer func() { _ = resp.Body.Close() }()

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

	// CUSTOM STRUCTURED FORMAT PROCESSING
	content := result.Choices[0].Message.Content
	fmt.Printf("[LLMHandler] EXTRACTED CONTENT: %s\n", content)
	
	// Parse Custom Structured Format and convert to JSON for existing system
	if jsonResult, success := parseCustomStructuredFormat(content); success {
		fmt.Printf("[LLMHandler] CUSTOM FORMAT PARSED SUCCESSFULLY: %s\n", jsonResult)
		return jsonResult, nil
	}
	
	// FALLBACK: Comment out JSON processing for now, return error instead
	/*
	// LEGACY JSON PROCESSING (COMMENTED OUT)
	content = fixDeepSeekJSON(content)
	fmt.Printf("[LLMHandler] EXTRACTED CONTENT (NO FIXES APPLIED): %s\n", content)
	return content, nil
	*/
	
	fmt.Printf("[LLMHandler] FAILED TO PARSE CUSTOM FORMAT, CONTENT: %s\n", content)
	return "", fmt.Errorf("failed to parse custom structured format from DeepSeek response")
}

// parseCustomStructuredFormat parses the Custom Structured Format that DeepSeek naturally prefers
func parseCustomStructuredFormat(content string) (string, bool) {
	fmt.Printf("[LLMHandler] Attempting to parse Custom Structured Format...\n")
	
	lines := strings.Split(content, "\n")
	data := make(map[string]interface{})
	
	// Find the start of the structured data (skip any preamble)
	startIdx := -1
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NAME:") {
			startIdx = i
			break
		}
	}
	
	if startIdx == -1 {
		fmt.Printf("[LLMHandler] No NAME: field found, not Custom Structured Format\n")
		return "", false
	}
	
	// Parse from NAME: onwards
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		// Stop parsing if we hit non-structured content
		if !strings.Contains(line, ":") && !strings.HasPrefix(line, "-") {
			// Check if it's a continuation line or end of structured data
			if strings.Contains(line, "safety") || strings.Contains(line, "note") || strings.Contains(line, "✔️") {
				break
			}
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "name":
			data["name"] = value
		case "description":
			data["description"] = value
		case "category":
			data["category"] = value
		case "cuisine":
			data["cuisine"] = value
		case "ingredients":
			if strings.Contains(value, "|") {
				ingredients := strings.Split(value, "|")
				for j, ing := range ingredients {
					ingredients[j] = strings.TrimSpace(ing)
				}
				data["ingredients"] = ingredients
			} else {
				// Handle multi-line ingredient lists
				ingredients := []string{}
				if value != "" {
					ingredients = append(ingredients, value)
				}
				// Check following lines for ingredient continuation
				for j := i + 1; j < len(lines); j++ {
					nextLine := strings.TrimSpace(lines[j])
					if nextLine == "" || strings.Contains(nextLine, ":") {
						break
					}
					if strings.HasPrefix(nextLine, "-") {
						ingredient := strings.TrimSpace(strings.TrimPrefix(nextLine, "-"))
						if ingredient != "" {
							ingredients = append(ingredients, ingredient)
						}
					}
				}
				if len(ingredients) > 0 {
					data["ingredients"] = ingredients
				}
			}
		case "instructions":
			if strings.Contains(value, "|") {
				instructions := strings.Split(value, "|")
				for j, inst := range instructions {
					instructions[j] = strings.TrimSpace(inst)
				}
				data["instructions"] = instructions
			} else {
				// Handle multi-line instruction lists
				instructions := []string{}
				if value != "" {
					instructions = append(instructions, value)
				}
				// Check following lines for numbered instructions
				for j := i + 1; j < len(lines); j++ {
					nextLine := strings.TrimSpace(lines[j])
					if nextLine == "" || (strings.Contains(nextLine, ":") && !strings.Contains(nextLine, ".")) {
						break
					}
					if strings.HasPrefix(nextLine, "1.") || strings.HasPrefix(nextLine, "2.") || 
					   strings.HasPrefix(nextLine, "3.") || strings.HasPrefix(nextLine, "4.") ||
					   strings.HasPrefix(nextLine, "5.") || strings.HasPrefix(nextLine, "6.") ||
					   strings.HasPrefix(nextLine, "7.") || strings.HasPrefix(nextLine, "8.") {
						instructions = append(instructions, nextLine)
					}
				}
				if len(instructions) > 0 {
					data["instructions"] = instructions
				}
			}
		case "prep_time", "preptime":
			data["prep_time"] = value
		case "cook_time", "cooktime":
			data["cook_time"] = value
		case "servings":
			data["servings"] = value
		case "difficulty":
			data["difficulty"] = value
		case "calories":
			if calories, err := strconv.ParseFloat(value, 64); err == nil {
				data["calories"] = calories
			}
		case "protein":
			if protein, err := strconv.ParseFloat(value, 64); err == nil {
				data["protein"] = protein
			}
		case "carbs", "carbohydrates":
			if carbs, err := strconv.ParseFloat(value, 64); err == nil {
				data["carbs"] = carbs
			}
		case "fat":
			if fat, err := strconv.ParseFloat(value, 64); err == nil {
				data["fat"] = fat
			}
		}
	}
	
	// Check if we have minimum required fields
	if data["name"] == nil || data["ingredients"] == nil {
		fmt.Printf("[LLMHandler] Missing required fields - name: %v, ingredients: %v\n", 
			data["name"] != nil, data["ingredients"] != nil)
		return "", false
	}
	
	// Convert to JSON
	if jsonBytes, err := json.Marshal(data); err == nil {
		fmt.Printf("[LLMHandler] Successfully parsed Custom Structured Format with %d fields\n", len(data))
		return string(jsonBytes), true
	}
	
	fmt.Printf("[LLMHandler] Failed to marshal parsed data to JSON\n")
	return "", false
}


// fixDeepSeekJSON fixes common JSON formatting issues from DeepSeek API
// Used by GenerateRecipe and GenerateBasicRecipe methods
//nolint:unused // False positive - function is used by GenerateRecipe methods
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

	// 6. Fix unquoted instruction steps (match lines that start with Step without opening quote)
	// Pattern: \n        Step 1: text -> \n        "Step 1: text"
	stepPattern := regexp.MustCompile(`(\n\s+)(Step \d+:[^"\n]+?)(\n|\s*$)`)
	if stepPattern.MatchString(content) {
		fmt.Printf("[LLMHandler] Fixing unquoted instruction steps\n")
		content = stepPattern.ReplaceAllString(content, `${1}"${2}",${3}`)
	}

	// 9. Fix unquoted property names (e.g., difficulty: -> "difficulty":)
	unquotedProps := regexp.MustCompile(`(\n\s+)([a-z_]+):\s*([^,\n}]+)`)
	if unquotedProps.MatchString(content) {
		fmt.Printf("[LLMHandler] Fixing unquoted property names\n")
		content = unquotedProps.ReplaceAllStringFunc(content, func(match string) string {
			// Extract parts
			parts := unquotedProps.FindStringSubmatch(match)
			if len(parts) != 4 {
				return match
			}
			whitespace := parts[1]
			prop := parts[2]
			value := parts[3]
			
			// Quote the value if it's not already quoted and not a number
			if !strings.HasPrefix(value, `"`) && !regexp.MustCompile(`^\d+$`).MatchString(strings.TrimSpace(value)) {
				value = `"` + strings.TrimSpace(value) + `"`
			}
			
			return fmt.Sprintf(`%s"%s": %s`, whitespace, prop, value)
		})
	}

	// 10. Fix unescaped quotes in string values (e.g., it"s -> it's)
	unescapedQuotes := regexp.MustCompile(`"([^"]*)"([st])\s+([^"]*)"`)
	if unescapedQuotes.MatchString(content) {
		fmt.Printf("[LLMHandler] Fixing unescaped quotes in strings\n")
		content = unescapedQuotes.ReplaceAllString(content, `"${1}'${2} ${3}"`)
	}

	// 7. Fix malformed field values with embedded newlines
	// Pattern: "field": "\nvalue" -> "field": "value"
	newlinePattern := regexp.MustCompile(`("[^"]+"):\s*"\s*\n\s*([^"]+)"`)
	if newlinePattern.MatchString(content) {
		fmt.Printf("[LLMHandler] Fixing field values with embedded newlines\n")
		content = newlinePattern.ReplaceAllString(content, `$1: "$2"`)
	}

	// 8. Fix trailing commas and field syntax issues
	// Pattern: prep_time": "value", -> "prep_time": "value",
	missingQuotePattern := regexp.MustCompile(`(\n\s*)([a-z_]+"):\s*"`)
	if missingQuotePattern.MatchString(content) {
		fmt.Printf("[LLMHandler] Fixing missing opening quotes on field names\n")
		content = missingQuotePattern.ReplaceAllString(content, `${1}"${2}: "`)
	}

	fmt.Printf("[LLMHandler] JSON formatting fixes applied\n")
	return content
}

// CalculateMacros estimates the macronutrients for a set of ingredients
func (s *LLMService) CalculateMacros(ingredients []string) (*Macros, error) {
	fmt.Println("=== CalculateMacros CALLED ===")
	log.Printf("=== CalculateMacros CALLED with %d ingredients ===", len(ingredients))
	// Create a more detailed prompt for better nutrition calculation
	ingredientsList := strings.Join(ingredients, "\n")
	prompt := fmt.Sprintf(`Calculate the TOTAL nutritional content for this COMPLETE recipe based on these ingredients:

%s

Provide accurate nutritional values considering:
- The actual quantities specified (cups, tablespoons, etc.)
- That this represents the ENTIRE recipe, not per serving
- Common nutritional databases for each ingredient
- Realistic calorie counts (e.g., avocados are ~320 cal each, maple syrup is ~52 cal/tbsp)

Return ONLY a JSON object with these fields:`, ingredientsList)
	
	messages := []Message{
		{
			Role:    "system",
			Content: `You are a professional nutritionist who calculates PRECISE macronutrient values based on USDA nutritional databases.

CRITICAL REQUIREMENTS:
1. Calculate the EXACT nutritional content for each ingredient based on:
   - The specific quantity given (e.g., "1 cup", "2 tbsp", "400g")
   - Standard nutritional values from USDA or similar databases
   - DO NOT use estimates or ranges - calculate actual values

2. For each ingredient, mentally calculate:
   - Calories per unit × quantity
   - Protein grams per unit × quantity
   - Carb grams per unit × quantity
   - Fat grams per unit × quantity

3. Sum ALL ingredients to get TOTAL recipe nutrition

4. Return ONLY a JSON object: {"calories":0,"protein":0,"carbs":0,"fat":0}
   - NO markdown formatting, NO code blocks, NO explanations
   - Just the raw JSON object

Example calculation process (DO NOT include in response):
- 2 cups pasta (400g) = 740 calories, 26g protein, 156g carbs, 2g fat
- 1 cup tomato sauce = 70 calories, 3g protein, 16g carbs, 0g fat
- 2 tbsp olive oil = 240 calories, 0g protein, 0g carbs, 28g fat
TOTAL = {"calories":1050,"protein":29,"carbs":172,"fat":30}`,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	reqBody := Request{
		Model:    "deepseek-chat",
		Messages: messages,
		// ResponseFormat: map[string]string{
		//	"type": "json_object",
		// }, // DISABLED - Testing alternative formats
		MaxTokens:        1024,
		Temperature:      0.1,
		TopP:             0.9,  // Changed from 0.8 to 0.9 to test if this fixes the issue
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
	}

	fmt.Printf("[LLMHandler] CalculateMacros - About to marshal request with TopP: %f\n", reqBody.TopP)
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("[LLMHandler] CalculateMacros - Marshal error: %v\n", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Debug logging to see the actual JSON being sent
	fmt.Printf("[LLMHandler] CalculateMacros - Sending JSON: %s\n", string(jsonData))

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
	defer func() { _ = resp.Body.Close() }()

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
	nutritionContent := result.Choices[0].Message.Content
	fmt.Printf("[LLMHandler] CalculateMacros - LLM returned nutrition: %s\n", nutritionContent)
	
	// Strip markdown code blocks if present
	nutritionContent = strings.TrimSpace(nutritionContent)
	if strings.HasPrefix(nutritionContent, "```json") {
		nutritionContent = strings.TrimPrefix(nutritionContent, "```json")
		nutritionContent = strings.TrimSuffix(nutritionContent, "```")
		nutritionContent = strings.TrimSpace(nutritionContent)
	} else if strings.HasPrefix(nutritionContent, "```") {
		nutritionContent = strings.TrimPrefix(nutritionContent, "```")
		nutritionContent = strings.TrimSuffix(nutritionContent, "```")
		nutritionContent = strings.TrimSpace(nutritionContent)
	}
	
	fmt.Printf("[LLMHandler] CalculateMacros - Cleaned nutrition JSON: %s\n", nutritionContent)
	
	if err := json.Unmarshal([]byte(nutritionContent), &macros); err != nil {
		fmt.Printf("[LLMHandler] CalculateMacros - Failed to parse nutrition JSON: %v\n", err)
		return nil, fmt.Errorf("failed to parse macros: %w", err)
	}
	
	fmt.Printf("[LLMHandler] CalculateMacros - Parsed nutrition: calories=%f, protein=%f, carbs=%f, fat=%f\n", 
		macros.Calories, macros.Protein, macros.Carbs, macros.Fat)

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
		// ResponseFormat: map[string]string{
		//	"type": "json_object",
		// }, // DISABLED - Testing alternative formats
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
	defer func() { _ = resp.Body.Close() }()

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
