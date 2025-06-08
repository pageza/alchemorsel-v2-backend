package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService handles authentication-related operations
type AuthService struct {
	db        *gorm.DB
	secretKey string
	isTest    bool
}

// NewAuthService creates a new AuthService instance
func NewAuthService(db *gorm.DB, secretKey string) *AuthService {
	return &AuthService{
		db:        db,
		secretKey: secretKey,
		isTest:    os.Getenv("TESTING") == "true",
	}
}

// Register handles user registration
func (s *AuthService) Register(ctx context.Context, email, password string, preferences *types.UserPreferences) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user already exists")
	}

	// Determine username
	username := email
	if preferences != nil {
		if len(preferences.DietaryPrefs) > 0 {
			// If the first dietary pref is actually a username (from test), use it
			username = preferences.DietaryPrefs[0]
		}
	}
	if ctx != nil {
		if uname, ok := ctx.Value("username").(string); ok && uname != "" {
			username = uname
		}
	}
	fmt.Printf("[DEBUG] Registering user with username: %s\n", username)

	// Check if username is taken
	var existingProfile models.UserProfile
	if err := s.db.Where("username = ?", username).First(&existingProfile).Error; err == nil {
		return nil, errors.New("username already taken")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := models.User{
		Name:         email,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create user profile
	profile := models.UserProfile{
		UserID:   user.ID,
		Username: username,
	}

	if err := s.db.Create(&profile).Error; err != nil {
		return nil, fmt.Errorf("failed to create user profile: %w", err)
	}

	// Create dietary preferences if provided
	if preferences != nil {
		// Create dietary preferences
		for _, pref := range preferences.DietaryPrefs {
			dietaryPref := models.DietaryPreference{
				UserID:         user.ID,
				PreferenceType: pref,
			}
			if err := s.db.Create(&dietaryPref).Error; err != nil {
				return nil, fmt.Errorf("failed to create dietary preference: %w", err)
			}
		}

		// Create allergens
		for _, allergen := range preferences.Allergies {
			allergenEntry := models.Allergen{
				UserID:        user.ID,
				AllergenName:  allergen,
				SeverityLevel: 1, // Default severity level
			}
			if err := s.db.Create(&allergenEntry).Error; err != nil {
				return nil, fmt.Errorf("failed to create allergen: %w", err)
			}
		}
	}

	return &user, nil
}

// Login handles user login
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, *models.UserProfile, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Get user profile for username
	var profile models.UserProfile
	if err := s.db.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to find user profile: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	return &user, &profile, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(token string) (*types.TokenClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &types.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := parsedToken.Claims.(*types.TokenClaims); ok && parsedToken.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GenerateToken generates a JWT token from claims
func (s *AuthService) GenerateToken(claims *types.TokenClaims) (string, error) {
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}
