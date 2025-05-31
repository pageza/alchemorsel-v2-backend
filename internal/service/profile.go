package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
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
func (s *ProfileService) ValidateToken(token string) (*middleware.TokenClaims, error) {
	// TODO: Implement token validation
	return &middleware.TokenClaims{
		UserID:   1,
		Username: "testuser",
	}, nil
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
func (s *ProfileService) GetProfileHistory(userID uint) ([]map[string]interface{}, error) {
	// TODO: Implement profile history tracking
	return []map[string]interface{}{}, nil
}

// UpdateProfile updates a user's profile and records the changes
func (s *ProfileService) UpdateProfile(userID uint, profile *models.Profile) error {
	profile.UserID = userID
	return s.db.Save(profile).Error
}

func (s *ProfileService) GetProfile(userID uint) (*models.Profile, error) {
	var profile models.Profile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}
