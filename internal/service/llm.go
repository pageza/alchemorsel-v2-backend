package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pageza/alchemorsel-v2/backend/internal/model"
)

// LLMService handles interactions with the DeepSeek API
type LLMService struct {
	apiKey string
	apiURL string
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

	return &LLMService{
		apiKey: apiKey,
		apiURL: apiURL,
	}, nil
}

// Message represents a message in the chat
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents a request to the DeepSeek API
type Request struct {
	Model          string            `json:"model"`
	Messages       []Message         `json:"messages"`
	ResponseFormat map[string]string `json:"response_format"`
}

// GenerateRecipe generates a recipe using the DeepSeek API
func (s *LLMService) GenerateRecipe(query string) (string, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are a professional chef. Please provide your response in JSON format with the following structure:
{
    "name": "Recipe name",
    "description": "Brief description of the recipe",
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
    "macros": {
        "calories": 0,
        "protein": 0,
        "fat": 0,
        "carbs": 0
    }
}`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Generate a recipe for: %s", query),
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
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return result.Choices[0].Message.Content, nil
}

// CalculateMacros uses the LLM to estimate nutritional macros for a list of ingredients.
func (s *LLMService) CalculateMacros(ingredients []string) (*model.Macros, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: `You are a nutritionist. Given a list of ingredients, provide the total calories, protein, fat and carbs in grams as JSON in the form {"calories":0,"protein":0,"fat":0,"carbs":0}.`,
		},
		{
			Role:    "user",
			Content: strings.Join(ingredients, ", "),
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
		bodyBytes, _ := io.ReadAll(resp.Body)
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

	var macros model.Macros
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &macros); err != nil {
		return nil, fmt.Errorf("failed to parse macros: %w", err)
	}
	return &macros, nil
}
