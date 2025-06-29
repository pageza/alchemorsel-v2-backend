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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/config"
)

// ImageGenerationRequest represents a request to the DALL-E API
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size"`
	Quality        string `json:"quality"`
	ResponseFormat string `json:"response_format"`
}

// ImageGenerationResponse represents the response from DALL-E API
type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url,omitempty"`
		B64JSON       string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}

// ImageService handles image generation and storage operations
type ImageService struct {
	apiKey    string
	apiURL    string
	s3Config  *config.S3Config
	client    *http.Client
}

// NewImageService creates a new ImageService instance
func NewImageService(s3Config *config.S3Config) (*ImageService, error) {
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

	apiURL := os.Getenv("OPENAI_IMAGES_API_URL")
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/images/generations"
	}

	return &ImageService{
		apiKey:   apiKey,
		apiURL:   apiURL,
		s3Config: s3Config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// GenerateRecipeImage generates an image for a recipe based on its data
func (s *ImageService) GenerateRecipeImage(ctx context.Context, recipeData *RecipeDraft) (string, error) {
	// Build a descriptive prompt for food image generation
	prompt := s.buildRecipeImagePrompt(recipeData)
	log.Printf("[ImageService] Generating image for recipe '%s' with prompt: %s", recipeData.Name, prompt)

	// Generate the image
	imageURL, err := s.GenerateImageFromPrompt(ctx, prompt, "1024x1024")
	if err != nil {
		return "", fmt.Errorf("failed to generate recipe image: %w", err)
	}

	return imageURL, nil
}

// GenerateImageFromPrompt generates an image from a text prompt
func (s *ImageService) GenerateImageFromPrompt(ctx context.Context, prompt string, size string) (string, error) {
	const maxRetries = 3
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[ImageService] Image generation attempt %d/%d", attempt, maxRetries)
		
		imageURL, err := s.generateImageAttempt(ctx, prompt, size)
		if err != nil {
			log.Printf("[ImageService] Attempt %d failed: %v", attempt, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to generate image after %d attempts: %w", maxRetries, err)
			}
			// Wait before retry
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		
		log.Printf("[ImageService] Successfully generated image on attempt %d", attempt)
		return imageURL, nil
	}
	
	return "", fmt.Errorf("failed to generate image after %d attempts", maxRetries)
}

// generateImageAttempt performs a single image generation attempt
func (s *ImageService) generateImageAttempt(ctx context.Context, prompt string, size string) (string, error) {
	// Prepare the request payload
	reqBody := ImageGenerationRequest{
		Model:          "dall-e-3",
		Prompt:         prompt,
		N:              1,
		Size:           size,
		Quality:        "standard", // Use standard quality to control costs
		ResponseFormat: "url",      // Get URL instead of base64 for efficiency
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ImageService] API request failed with status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result ImageGenerationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("no image data in API response")
	}

	imageURL := result.Data[0].URL
	if imageURL == "" {
		return "", fmt.Errorf("empty image URL in API response")
	}

	// Download the image and upload to S3
	s3URL, err := s.downloadAndUploadToS3(ctx, imageURL)
	if err != nil {
		log.Printf("[ImageService] Failed to upload to S3, returning original URL: %v", err)
		// Return the original URL as fallback
		return imageURL, nil
	}

	return s3URL, nil
}

// downloadAndUploadToS3 downloads an image from URL and uploads it to S3
func (s *ImageService) downloadAndUploadToS3(ctx context.Context, imageURL string) (string, error) {
	// Download the image
	resp, err := s.client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Generate unique filename
	fileName := fmt.Sprintf("recipe-images/%s.png", uuid.New().String())

	// Upload to S3
	return s.UploadImageToS3(ctx, imageData, fileName)
}

// UploadImageToS3 uploads image data to S3 and returns the public URL
func (s *ImageService) UploadImageToS3(ctx context.Context, imageData []byte, fileName string) (string, error) {
	_, err := s.s3Config.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.s3Config.BucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return public URL
	publicURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.s3Config.BucketName, fileName)
	log.Printf("[ImageService] Successfully uploaded image to S3: %s", publicURL)
	
	return publicURL, nil
}

// buildRecipeImagePrompt creates a detailed prompt for recipe image generation
func (s *ImageService) buildRecipeImagePrompt(recipeData *RecipeDraft) string {
	// Base prompt for food photography style
	basePrompt := "A professional food photography shot of "
	
	// Add recipe name and description
	recipeDescription := strings.ToLower(recipeData.Name)
	if recipeData.Description != "" {
		recipeDescription += ", " + strings.ToLower(recipeData.Description)
	}
	
	// Add cuisine style if available
	cuisineStyle := ""
	if recipeData.Cuisine != "" && recipeData.Cuisine != "unknown" {
		cuisineStyle = fmt.Sprintf(", %s style", strings.ToLower(recipeData.Cuisine))
	}
	
	// Add category context if available
	categoryContext := ""
	if recipeData.Category != "" && recipeData.Category != "unknown" {
		switch strings.ToLower(recipeData.Category) {
		case "dessert":
			categoryContext = ", beautifully plated dessert"
		case "breakfast":
			categoryContext = ", appetizing breakfast dish"
		case "main course", "lunch", "dinner":
			categoryContext = ", elegantly presented main dish"
		case "appetizer":
			categoryContext = ", attractive appetizer"
		case "snack":
			categoryContext = ", delicious snack"
		case "beverage":
			categoryContext = ", refreshing beverage"
		case "soup":
			categoryContext = ", steaming bowl of soup"
		case "salad":
			categoryContext = ", fresh and colorful salad"
		}
	}
	
	// Photography style specifications
	stylePrompt := ", shot with natural lighting, shallow depth of field, garnished beautifully, restaurant quality presentation, high resolution, food styling, appetizing colors"
	
	// Combine all parts
	fullPrompt := basePrompt + recipeDescription + cuisineStyle + categoryContext + stylePrompt
	
	// Ensure prompt doesn't exceed typical limits (around 1000 characters)
	if len(fullPrompt) > 900 {
		fullPrompt = fullPrompt[:900]
	}
	
	return fullPrompt
}