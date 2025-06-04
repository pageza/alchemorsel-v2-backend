package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token has expired")
)

// ProfileService handles profile-related business logic
type ProfileService struct {
	db        *gorm.DB
	jwtSecret []byte
}

// NewProfileService creates a new profile service
func NewProfileService(db *gorm.DB, jwtSecret string) *ProfileService {
	return &ProfileService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user
func (s *ProfileService) GenerateToken(userID, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token expires in 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims
func (s *ProfileService) ValidateToken(tokenString string) (*middleware.TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			return nil, ErrInvalidToken
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, ErrInvalidToken
		}

		username, _ := claims["username"].(string)
		return &middleware.TokenClaims{
			UserID:   userID,
			Username: username,
		}, nil
	}

	return nil, ErrInvalidToken
}

// SanitizeProfile sanitizes profile data before sending to client
func (s *ProfileService) SanitizeProfile(profile map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	// Only include safe fields
	if username, ok := profile["username"]; ok {
		sanitized["username"] = username
	}
	if email, ok := profile["email"]; ok {
		sanitized["email"] = email
	}
	if id, ok := profile["id"]; ok {
		sanitized["id"] = id
	}

	return sanitized
}

// RecordProfileChange records a change to a user's profile
func (s *ProfileService) RecordProfileChange(userID, field, oldValue, newValue, changedBy string) error {
	history := &models.ProfileHistory{
		UserID:    userID,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedAt: time.Now(),
		ChangedBy: changedBy,
	}

	return s.db.Create(history).Error
}

// GetProfileHistory retrieves the change history for a user's profile
func (s *ProfileService) GetProfileHistory(userID uuid.UUID) ([]map[string]interface{}, error) {
	var histories []models.ProfileHistory
	if err := s.db.Where("user_id = ?", userID.String()).Order("changed_at desc").Find(&histories).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(histories))
	for i, h := range histories {
		result[i] = map[string]interface{}{
			"id":         h.ID,
			"user_id":    h.UserID,
			"field":      h.Field,
			"old_value":  h.OldValue,
			"new_value":  h.NewValue,
			"changed_at": h.ChangedAt,
			"changed_by": h.ChangedBy,
		}
	}

	return result, nil
}

func (s *ProfileService) GetProfile(userID uuid.UUID) (*models.UserProfile, error) {
	var profile models.UserProfile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create a new profile if it doesn't exist
			profile = models.UserProfile{
				UserID: userID,
			}
			if err := s.db.Create(&profile).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &profile, nil
}

func (s *ProfileService) UpdateProfile(userID uuid.UUID, updates map[string]interface{}) error {
	return s.db.Model(&models.UserProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (s *ProfileService) Logout(userID uuid.UUID) error {
	// In a real application, you might want to invalidate the user's session or token
	// For now, we'll just return nil as the token invalidation is handled by the client
	return nil
}

// GetUserRecipes returns all recipes created by the given user
func (s *ProfileService) GetUserRecipes(userID uuid.UUID) ([]model.Recipe, error) {
	var recipes []model.Recipe
	if err := s.db.Where("user_id = ?", userID).Find(&recipes).Error; err != nil {
		return nil, err
	}
	return recipes, nil
}
