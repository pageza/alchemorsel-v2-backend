package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pgvector/pgvector-go"
)

// EmbeddingServiceInterface defines the interface for embedding services
type EmbeddingServiceInterface interface {
	GenerateEmbedding(text string) (pgvector.Vector, error)
	GenerateEmbeddingFromRecipe(name, description string, ingredients []string, category string, dietary []string) (pgvector.Vector, error)
}

// EmbeddingService handles interactions with the OpenAI API for embeddings
type EmbeddingService struct {
	apiKey string
	apiURL string
}

// NewEmbeddingService creates a new EmbeddingService instance
func NewEmbeddingService() (*EmbeddingService, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKeyFile := os.Getenv("OPENAI_API_KEY_FILE")
		if apiKeyFile == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY or OPENAI_API_KEY_FILE must be set")
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

	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/embeddings"
	}

	return &EmbeddingService{
		apiKey: apiKey,
		apiURL: apiURL,
	}, nil
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// GenerateEmbedding generates an embedding using OpenAI's Ada model
func (s *EmbeddingService) GenerateEmbedding(text string) (pgvector.Vector, error) {
	reqBody := embeddingRequest{
		Model: "text-embedding-ada-002",
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return pgvector.Vector{}, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return pgvector.Vector{}, fmt.Errorf("no embedding data in response")
	}

	return pgvector.NewVector(result.Data[0].Embedding), nil
}

// GenerateEmbeddingFromRecipe generates an embedding from a recipe's name and description
func (s *EmbeddingService) GenerateEmbeddingFromRecipe(name, description string, ingredients []string, category string, dietary []string) (pgvector.Vector, error) {
	// Combine all relevant recipe information for better semantic matching
	text := fmt.Sprintf("%s %s Ingredients: %s Category: %s Dietary: %s",
		name,
		description,
		strings.Join(ingredients, ", "),
		category,
		strings.Join(dietary, ", "),
	)
	return s.GenerateEmbedding(text)
}
