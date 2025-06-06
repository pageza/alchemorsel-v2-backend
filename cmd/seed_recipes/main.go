package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	numRecipes = 25 // Number of recipes to generate
	batchSize  = 5  // Number of recipes to generate in each batch
)

var recipePrompts = []string{
	"Create a traditional Italian pasta recipe with a unique twist",
	"Create a healthy vegan salad recipe with seasonal ingredients",
	"Create a quick breakfast smoothie recipe with protein",
	"Create a spicy Indian curry recipe with a modern twist",
	"Create a classic French dessert recipe with a contemporary presentation",
	"Create a gluten-free bread recipe with alternative flours",
	"Create a keto-friendly dinner recipe with high protein",
	"Create a Mediterranean seafood recipe with fresh herbs",
	"Create a vegetarian stir-fry recipe with Asian flavors",
	"Create a traditional Mexican recipe with authentic spices",
	"Create a Japanese sushi recipe with fusion elements",
	"Create a Thai soup recipe with bold flavors",
	"Create a Middle Eastern appetizer recipe with mezze",
	"Create a traditional American comfort food recipe with a healthy twist",
	"Create a raw vegan dessert recipe with superfoods",
	"Create a paleo-friendly recipe with ancient grains",
	"Create a traditional Chinese recipe with regional variations",
	"Create a Greek salad recipe with Mediterranean ingredients",
	"Create a traditional Russian recipe with modern techniques",
	"Create a Korean BBQ recipe with homemade marinades",
	"Create a traditional Spanish tapas recipe with local ingredients",
	"Create a traditional German recipe with seasonal produce",
	"Create a traditional British recipe with contemporary plating",
	"Create a traditional Brazilian recipe with street food influence",
	"Create a traditional Moroccan recipe with aromatic spices",
	"Create a fusion recipe combining Italian and Japanese flavors",
	"Create a healthy recipe using ancient grains and superfoods",
	"Create a quick and easy recipe for busy weeknights",
	"Create a recipe perfect for meal prep and batch cooking",
	"Create a recipe using only pantry staples",
	"Create a recipe that's both kid-friendly and nutritious",
	"Create a recipe that's perfect for entertaining guests",
	"Create a recipe that's both budget-friendly and delicious",
	"Create a recipe that's perfect for summer barbecues",
	"Create a recipe that's ideal for winter comfort food",
	"Create a recipe that's perfect for spring picnics",
	"Create a recipe that's great for fall harvest",
	"Create a recipe that's perfect for holiday celebrations",
	"Create a recipe that's ideal for romantic dinners",
	"Create a recipe that's perfect for brunch gatherings",
	"Create a recipe that's great for potlucks",
	"Create a recipe that's perfect for camping trips",
	"Create a recipe that's ideal for beach picnics",
	"Create a recipe that's perfect for office lunches",
	"Create a recipe that's great for family gatherings",
	"Create a recipe that's perfect for date nights",
	"Create a recipe that's ideal for Sunday dinners",
	"Create a recipe that's perfect for game day",
	"Create a recipe that's great for afternoon tea",
	"Create a recipe that's perfect for midnight snacks",
}

type RecipeData struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
	PrepTime     string   `json:"prep_time"`
	CookTime     string   `json:"cook_time"`
	Servings     string   `json:"servings"`
	Difficulty   string   `json:"difficulty"`
	Tags         []string `json:"tags"`
}

func main() {
	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:your_secure_password_here@localhost:5432/alchemorsel?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize services
	llmService, err := service.NewLLMService()
	if err != nil {
		log.Fatalf("Failed to create LLM service: %v", err)
	}

	embeddingService, err := service.NewEmbeddingService()
	if err != nil {
		log.Fatalf("Failed to create embedding service: %v", err)
	}

	// Create a test user
	userID := uuid.New()
	user := models.User{
		ID:           userID,
		Name:         "Test User",
		Email:        fmt.Sprintf("test_%d@example.com", time.Now().Unix()),
		PasswordHash: "dummy_hash", // This is just for seeding
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create test user: %v", err)
	}

	// Create user profile
	profile := models.UserProfile{
		ID:                uuid.New(),
		UserID:            userID,
		Username:          fmt.Sprintf("testuser_%d", time.Now().Unix()),
		Email:             user.Email,
		Bio:               "Test user for recipe seeding",
		ProfilePictureURL: "",
		PrivacyLevel:      "public",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := db.Create(&profile).Error; err != nil {
		log.Fatalf("Failed to create user profile: %v", err)
	}

	// Generate and save recipes in batches
	for i := 0; i < numRecipes; i += batchSize {
		batchEnd := i + batchSize
		if batchEnd > numRecipes {
			batchEnd = numRecipes
		}

		log.Printf("Generating batch of recipes %d-%d", i+1, batchEnd)

		// Prepare batch of prompts with randomization
		var batchPrompts []string
		for j := i; j < batchEnd; j++ {
			// Add some randomization to make each prompt unique
			basePrompt := recipePrompts[j%len(recipePrompts)]
			randomPrompt := fmt.Sprintf("%s (Make it unique and different from any previous recipes)", basePrompt)
			batchPrompts = append(batchPrompts, randomPrompt)
		}

		// Generate recipes in batch
		recipesJSON, err := llmService.GenerateRecipesBatch(batchPrompts)
		if err != nil {
			log.Printf("Failed to generate batch of recipes: %v", err)
			continue
		}

		// Process each recipe in the batch
		for _, recipeJSON := range recipesJSON {
			var recipeData RecipeData
			if err := json.Unmarshal([]byte(recipeJSON), &recipeData); err != nil {
				log.Printf("Failed to parse recipe JSON: %v", err)
				continue
			}

			// Calculate macros
			macros, err := llmService.CalculateMacros(recipeData.Ingredients)
			if err != nil {
				log.Printf("Failed to calculate macros: %v", err)
				continue
			}

			// Generate embedding
			embedding, err := embeddingService.GenerateEmbeddingFromRecipe(
				recipeData.Name,
				recipeData.Description,
				recipeData.Ingredients,
				recipeData.Category,
				[]string{}, // Empty dietary preferences for now
			)
			if err != nil {
				log.Printf("Failed to generate embedding: %v", err)
				continue
			}

			// Create recipe record
			recipe := models.Recipe{
				ID:           uuid.New(),
				Name:         recipeData.Name,
				Description:  recipeData.Description,
				Category:     recipeData.Category,
				Ingredients:  models.JSONBStringArray(recipeData.Ingredients),
				Instructions: models.JSONBStringArray(recipeData.Instructions),
				Calories:     macros.Calories,
				Protein:      macros.Protein,
				Carbs:        macros.Carbs,
				Fat:          macros.Fat,
				Embedding:    embedding,
				UserID:       userID,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Tags:         models.JSONBStringArray([]string{recipeData.Category, "dietary_preference", "meal_type"}),
			}

			if err := db.Create(&recipe).Error; err != nil {
				log.Printf("Failed to save recipe: %v", err)
				continue
			}

			log.Printf("Successfully created recipe: %s", recipe.Name)
		}

		// Add a small delay between batches to avoid rate limiting
		time.Sleep(2 * time.Second)
	}

	log.Printf("Successfully seeded %d recipes", numRecipes)
}
