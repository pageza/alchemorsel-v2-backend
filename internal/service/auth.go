package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
func (s *AuthService) Register(ctx context.Context, email, password, username string, preferences *types.UserPreferences) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user already exists")
	}

	// Use provided username, default to email if empty
	if username == "" {
		username = email
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
		Name:         username,
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

// GenerateVerificationToken generates a new verification token for a user
func (s *AuthService) GenerateVerificationToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(bytes)

	// Set expiration to 24 hours from now
	expiresAt := time.Now().Add(24 * time.Hour)

	// Update user with new verification token
	result := s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"verification_token":            token,
			"verification_token_expires_at": expiresAt,
		})

	if result.Error != nil {
		return "", fmt.Errorf("failed to update verification token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return "", errors.New("user not found")
	}

	return token, nil
}

// ValidateVerificationToken validates a verification token and marks email as verified
func (s *AuthService) ValidateVerificationToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	now := time.Now()

	// Find user with valid token
	err := s.db.Where("verification_token = ? AND verification_token_expires_at > ?", token, now).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid or expired verification token")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Update user to mark email as verified and clear token
	verifiedAt := time.Now()
	result := s.db.Model(&user).Updates(map[string]interface{}{
		"is_email_verified":             true,
		"email_verified_at":             verifiedAt,
		"verification_token":            nil,
		"verification_token_expires_at": nil,
	})

	if result.Error != nil {
		return nil, fmt.Errorf("failed to update user verification status: %w", result.Error)
	}

	// Update the returned user object
	user.EmailVerified = true
	user.EmailVerifiedAt = &verifiedAt
	user.VerificationToken = nil
	user.VerificationTokenExpiresAt = nil

	return &user, nil
}

// ResendVerificationEmail generates a new token and sends verification email
func (s *AuthService) ResendVerificationEmail(ctx context.Context, email string, emailService IEmailService) error {
	var user models.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return errors.New("email already verified")
	}

	// Generate new verification token
	token, err := s.GenerateVerificationToken(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Send verification email
	if err := emailService.SendVerificationEmail(&user, token); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email address
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}
