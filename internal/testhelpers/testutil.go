package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDatabase represents a test database instance with its configuration
type TestDatabase struct {
	db          *gorm.DB
	config      *config.Config
	authService *service.AuthService
}

// DB returns the underlying *gorm.DB instance
func (db *TestDatabase) DB() *gorm.DB {
	return db.db
}

// setupDatabase handles database initialization and migration
func setupDatabase(t *testing.T, db *gorm.DB, cfg *config.Config) *TestDatabase {
	t.Log("[DEBUG] Starting database setup...")

	// Create test database instance
	testDB := &TestDatabase{
		db:          db,
		config:      cfg,
		authService: service.NewAuthService(db, cfg.JWTSecret),
	}
	t.Log("[DEBUG] Test database instance created")

	// Install pgvector extension
	t.Log("[DEBUG] Installing pgvector extension...")
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		t.Logf("[ERROR] Failed to install pgvector extension: %v", err)
		t.Fatalf("failed to install pgvector extension: %v", err)
	}
	t.Log("[DEBUG] pgvector extension installed successfully")

	// Verify pgvector installation
	t.Log("[DEBUG] Verifying pgvector installation...")
	var extensionExists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&extensionExists).Error; err != nil {
		t.Logf("[ERROR] Failed to verify pgvector extension: %v", err)
		t.Fatalf("failed to verify pgvector extension: %v", err)
	}
	if !extensionExists {
		t.Log("[ERROR] pgvector extension not found after installation")
		t.Fatal("pgvector extension not found after installation")
	}
	t.Log("[DEBUG] pgvector extension verified successfully")

	// Create dietary preference type enum
	t.Log("[DEBUG] Creating dietary preference type enum...")
	if err := db.Exec(`
		DO $$ BEGIN
			CREATE TYPE dietary_preference_type AS ENUM (
				'vegetarian',
				'vegan',
				'pescatarian',
				'gluten-free',
				'dairy-free',
				'nut-free',
				'soy-free',
				'egg-free',
				'shellfish-free',
				'custom'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
	`).Error; err != nil {
		t.Logf("[ERROR] Failed to create dietary preference type: %v", err)
		t.Fatalf("failed to create dietary preference type: %v", err)
	}
	t.Log("[DEBUG] Dietary preference type enum created successfully")

	// Verify dietary preference type
	t.Log("[DEBUG] Verifying dietary preference type...")
	var typeExists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_type WHERE typname = 'dietary_preference_type')").Scan(&typeExists).Error; err != nil {
		t.Logf("[ERROR] Failed to verify dietary preference type: %v", err)
		t.Fatalf("failed to verify dietary preference type: %v", err)
	}
	if !typeExists {
		t.Log("[ERROR] dietary_preference_type not found after creation")
		t.Fatal("dietary_preference_type not found after creation")
	}
	t.Log("[DEBUG] Dietary preference type verified successfully")

	// Auto-migrate the schema
	t.Log("[DEBUG] Starting schema migration...")
	t.Log("[DEBUG] Models to migrate: User, UserProfile, Recipe, RecipeFavorite, DietaryPreference, Allergen")

	// First, create tables without the vector column
	t.Log("[DEBUG] Creating initial tables without vector column...")
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.RecipeFavorite{},
		&models.DietaryPreference{},
		&models.Allergen{},
	); err != nil {
		t.Logf("[ERROR] Initial migration failed: %v", err)
		t.Fatalf("failed to migrate initial tables: %v", err)
	}
	t.Log("[DEBUG] Initial tables created successfully")

	// Then, create the recipes table with vector column
	t.Log("[DEBUG] Creating recipes table with vector column...")
	if err := db.AutoMigrate(&models.Recipe{}); err != nil {
		t.Logf("[ERROR] Recipe table migration failed: %v", err)
		t.Logf("[DEBUG] Database connection details - Host: %s, Port: %s, User: %s, DB: %s, SSL: %s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName, cfg.DBSSLMode)
		t.Fatalf("failed to migrate recipes table: %v", err)
	}
	t.Log("[DEBUG] Recipes table created successfully")

	// Verify tables were created
	t.Log("[DEBUG] Verifying table creation...")
	var tables []string
	if err := db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error; err != nil {
		t.Logf("[ERROR] Failed to verify tables: %v", err)
		t.Fatalf("failed to verify tables: %v", err)
	}

	expectedTables := []string{
		"users",
		"user_profiles",
		"recipes",
		"recipe_favorites",
		"dietary_preferences",
		"allergens",
	}

	t.Logf("[DEBUG] Found tables: %v", tables)
	t.Logf("[DEBUG] Expected tables: %v", expectedTables)

	for _, expected := range expectedTables {
		found := false
		for _, table := range tables {
			if table == expected {
				found = true

				break
			}
		}
		if !found {
			t.Logf("[ERROR] Expected table %s was not found in database", expected)
			t.Fatalf("expected table %s was not created", expected)
		}
	}
	t.Log("[DEBUG] All expected tables verified successfully")

	// Verify vector column in recipes table
	t.Log("[DEBUG] Verifying vector column in recipes table...")
	var columnExists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'recipes' AND column_name = 'embedding' AND data_type = 'USER-DEFINED')").Scan(&columnExists).Error; err != nil {
		t.Logf("[ERROR] Failed to verify vector column: %v", err)
		t.Fatalf("failed to verify vector column: %v", err)
	}
	if !columnExists {
		t.Log("[ERROR] vector column not found in recipes table")
		t.Fatal("vector column not found in recipes table")
	}
	t.Log("[DEBUG] Vector column verified successfully")

	// Verify vector column type
	t.Log("[DEBUG] Verifying vector column type...")
	var columnType string
	if err := db.Raw("SELECT udt_name FROM information_schema.columns WHERE table_name = 'recipes' AND column_name = 'embedding'").Scan(&columnType).Error; err != nil {
		t.Logf("[ERROR] Failed to verify vector column type: %v", err)
		t.Fatalf("failed to verify vector column type: %v", err)
	}
	if columnType != "vector" {
		t.Logf("[ERROR] Expected vector column type 'vector', got '%s'", columnType)
		t.Fatalf("expected vector column type 'vector', got '%s'", columnType)
	}
	t.Log("[DEBUG] Vector column type verified successfully")

	return testDB
}

// SetupTestDB creates a new test database using PostgreSQL testcontainer
func SetupTestDB(t *testing.T) *TestDatabase {
	t.Log("[DEBUG] Starting test database setup...")
	ctx := context.Background()

	// Set CI environment and required secrets
	t.Log("[DEBUG] Setting up environment variables...")
	os.Setenv("CI", "true")
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("TEST_DB_PASSWORD", "testpass")
	os.Setenv("TEST_JWT_SECRET", "test-jwt-secret")
	os.Setenv("TEST_REDIS_PASSWORD", "test-redis-pass")
	os.Setenv("TEST_REDIS_URL", "redis://localhost:6379")
	t.Log("[DEBUG] Environment variables set successfully")

	defer func() {
		t.Log("[DEBUG] Cleaning up environment variables...")
		os.Unsetenv("CI")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSL_MODE")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("TEST_DB_PASSWORD")
		os.Unsetenv("TEST_JWT_SECRET")
		os.Unsetenv("TEST_REDIS_PASSWORD")
		os.Unsetenv("TEST_REDIS_URL")
		t.Log("[DEBUG] Environment variables cleaned up")
	}()

	// Load configuration
	t.Log("[DEBUG] Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Logf("[ERROR] Failed to load configuration: %v", err)
		t.Fatalf("failed to load configuration: %v", err)
	}
	t.Log("[DEBUG] Configuration loaded successfully")

	// Debug logging
	t.Logf("[DEBUG] DB_HOST: %s", cfg.DBHost)
	t.Logf("[DEBUG] DB_PORT: %s", cfg.DBPort)
	t.Logf("[DEBUG] DB_USER: %s", cfg.DBUser)
	t.Logf("[DEBUG] DB_PASSWORD: %s", cfg.DBPassword)
	t.Logf("[DEBUG] DB_NAME: %s", cfg.DBName)
	t.Logf("[DEBUG] DB_SSL_MODE: %s", cfg.DBSSLMode)

	// Use testcontainers for both CI and local environments
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     cfg.DBUser,
				"POSTGRES_PASSWORD": cfg.DBPassword,
				"POSTGRES_DB":       cfg.DBName,
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections"),
				wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
					return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
						cfg.DBUser,
						cfg.DBPassword,
						host,
						port.Port(),
						cfg.DBName,
						cfg.DBSSLMode)
				}).WithStartupTimeout(30*time.Second),
				wait.ForExec([]string{"pg_isready", "-U", cfg.DBUser, "-d", cfg.DBName}).
					WithStartupTimeout(10*time.Second).
					WithPollInterval(2*time.Second),
			).WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Get container host and port
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Connect to database with retry logic
	maxRetries := 5
	retryDelay := 2 * time.Second
	var db *gorm.DB

	for i := 0; i < maxRetries; i++ {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host,
			mappedPort.Port(),
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode)

		t.Logf("Attempting to connect to database at %s:%s as user %s (attempt %d/%d)",
			host,
			mappedPort.Port(),
			cfg.DBUser,
			i+1,
			maxRetries)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			break
		}

		t.Logf("Connection attempt %d failed: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to database after %d attempts: %v", maxRetries, err)
	}

	// Register cleanup
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}

		// Wait for container to be fully terminated
		<-ctx.Done()
	})

	t.Log("[DEBUG] Setting up database...")
	return setupDatabase(t, db, cfg)
}

// CreateTestUserAndToken creates a test user and returns their ID and a valid JWT token
func CreateTestUserAndToken(t *testing.T, db *TestDatabase) (uuid.UUID, string) {
	// Create a test user with configured password
	userID := uuid.New()
	password := db.config.DBPassword // Use the configured password from secrets
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		ID:           userID,
		Name:         "Test User",
		Email:        fmt.Sprintf("testuser+%s@example.com", userID.String()),
		PasswordHash: string(hashedPassword),
	}
	if err := db.db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create user profile
	profile := models.UserProfile{
		ID:       uuid.New(),
		UserID:   userID,
		Username: fmt.Sprintf("testuser_%s", userID.String()),
	}
	if err := db.db.Create(&profile).Error; err != nil {
		t.Fatalf("failed to create test user profile: %v", err)
	}

	// Generate JWT token
	token, err := db.authService.GenerateToken(&types.TokenClaims{
		UserID:   user.ID,
		Username: profile.Username,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return userID, token
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)
	return user
}

// CreateTestProfile creates a test user profile in the database
func CreateTestProfile(t *testing.T, db *gorm.DB, userID uuid.UUID) *models.UserProfile {
	profile := &models.UserProfile{
		UserID:   userID,
		Username: "testuser",
	}
	err := db.Create(profile).Error
	assert.NoError(t, err)
	return profile
}

// CreateTestRecipe creates a test recipe with the given user ID
func CreateTestRecipe(t *testing.T, db *gorm.DB, userID uuid.UUID) *models.Recipe {
	t.Log("[DEBUG] Creating test recipe...")
	recipe := &models.Recipe{
		Name:               "Test Recipe",
		Description:        "Test Description",
		UserID:             userID,
		Category:           "Main Course",
		Cuisine:            "Italian",
		Ingredients:        []string{"ingredient1", "ingredient2"},
		Instructions:       []string{"step1", "step2"},
		Calories:           100,
		Protein:            10,
		Carbs:              20,
		Fat:                5,
		Embedding:          pgvector.NewVector([]float32{1.0, 2.0, 3.0}), // Example embedding
		DietaryPreferences: []string{"vegetarian"},
		Tags:               []string{"healthy", "quick"},
	}

	t.Log("[DEBUG] Saving test recipe to database...")
	if err := db.Create(recipe).Error; err != nil {
		t.Logf("[ERROR] Failed to create test recipe: %v", err)
		t.Fatalf("failed to create test recipe: %v", err)
	}
	t.Log("[DEBUG] Test recipe created successfully")

	// Verify recipe was created with vector
	t.Log("[DEBUG] Verifying test recipe creation...")
	var savedRecipe models.Recipe
	if err := db.First(&savedRecipe, recipe.ID).Error; err != nil {
		t.Logf("[ERROR] Failed to verify test recipe: %v", err)
		t.Fatalf("failed to verify test recipe: %v", err)
	}

	// Verify vector embedding
	t.Log("[DEBUG] Verifying vector embedding...")
	// The vector should be initialized with zeros by the BeforeCreate hook
	zeroVector := make([]float32, 1536)
	expectedVector := pgvector.NewVector(zeroVector)

	// Compare vectors by converting to slices
	savedVec := savedRecipe.Embedding.Slice()
	expectedVec := expectedVector.Slice()

	if len(savedVec) != len(expectedVec) {
		t.Logf("[ERROR] Vector length mismatch: got %d, want %d", len(savedVec), len(expectedVec))
		t.Fatal("vector length mismatch")
	}

	for i := range savedVec {
		if savedVec[i] != expectedVec[i] {
			t.Logf("[ERROR] Vector mismatch at index %d: got %f, want %f", i, savedVec[i], expectedVec[i])
			t.Fatal("vector mismatch")
		}
	}
	t.Log("[DEBUG] Vector embedding verified successfully")

	return recipe
}

// CreateTestRecipeFavorite creates a test recipe favorite
func CreateTestRecipeFavorite(t *testing.T, userID, recipeID uuid.UUID) *models.RecipeFavorite {
	return &models.RecipeFavorite{
		ID:       uuid.New(),
		UserID:   userID,
		RecipeID: recipeID,
	}
}

// MockTokenValidator is a mock token validator for testing
type MockTokenValidator struct {
	Claims *types.TokenClaims
	Error  error
}

// ValidateToken validates a token and returns claims
func (m *MockTokenValidator) ValidateToken(token string) (*types.TokenClaims, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Claims, nil
}

// JSONMarshal is a helper function to marshal JSON for testing
func JSONMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return data
}
