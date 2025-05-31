package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/google/uuid"
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

func (s *ProfileService) GetDietaryPreferences(userID uuid.UUID) ([]models.DietaryPreference, error) {
	var preferences []models.DietaryPreference
	if err := s.db.Where("user_id = ?", userID).Find(&preferences).Error; err != nil {
		return nil, err
	}
	return preferences, nil
}

func (s *ProfileService) UpdateDietaryPreferences(userID uuid.UUID, preferences []models.DietaryPreference) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.DietaryPreference{}).Error; err != nil {
			return err
		}
		
		for _, pref := range preferences {
			pref.UserID = userID
			if err := tx.Create(&pref).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProfileService) GetAllergens(userID uuid.UUID) ([]models.Allergen, error) {
	var allergens []models.Allergen
	if err := s.db.Where("user_id = ?", userID).Find(&allergens).Error; err != nil {
		return nil, err
	}
	return allergens, nil
}

func (s *ProfileService) UpdateAllergens(userID uuid.UUID, allergens []models.Allergen) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.Allergen{}).Error; err != nil {
			return err
		}
		
		for _, allergen := range allergens {
			allergen.UserID = userID
			if err := tx.Create(&allergen).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProfileService) GetAppliances(userID uuid.UUID) ([]models.UserAppliance, error) {
	var appliances []models.UserAppliance
	if err := s.db.Where("user_id = ?", userID).Find(&appliances).Error; err != nil {
		return nil, err
	}
	return appliances, nil
}

func (s *ProfileService) UpdateAppliances(userID uuid.UUID, appliances []models.UserAppliance) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserAppliance{}).Error; err != nil {
			return err
		}
		
		for _, appliance := range appliances {
			appliance.UserID = userID
			if err := tx.Create(&appliance).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProfileService) Logout(userID uuid.UUID) error {
	// In a real application, you might want to invalidate the user's session or token
	// For now, we'll just return nil as the token invalidation is handled by the client
	return nil
}
