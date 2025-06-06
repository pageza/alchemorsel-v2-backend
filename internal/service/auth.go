package service

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrMissingPreferences = errors.New("dietary preferences or allergens required")

type AuthService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) Register(name, email, password, username, dietaryPrefs, allergies string) (string, error) {
	// Start a transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return "", tx.Error
	}

	// Defer a rollback in case anything fails
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user already exists
	var existingUser models.User
	if err := tx.Where("email = ?", email).First(&existingUser).Error; err == nil {
		tx.Rollback()
		return "", errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	// Create user
	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	// Create user profile
	profile := models.UserProfile{
		UserID:   user.ID,
		Username: username,
		Email:    email,
	}
	if err := tx.Create(&profile).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	// Insert dietary preferences if any
	if dietaryPrefs != "" {
		prefs := strings.Split(dietaryPrefs, ",")
		for _, p := range prefs {
			pref := strings.TrimSpace(p)
			if pref == "" {
				continue
			}
			dp := models.DietaryPreference{
				UserID:         user.ID,
				PreferenceType: pref,
			}
			// Only set CustomName if the preference type is 'custom'
			if pref == "custom" {
				dp.CustomName = "Custom Diet"
			} else {
				dp.CustomName = "" // Ensure it's empty for non-custom preferences
			}
			if err := tx.Create(&dp).Error; err != nil {
				tx.Rollback()
				return "", err
			}
		}
	}

	// Insert allergens if any
	if allergies != "" {
		alls := strings.Split(allergies, ",")
		for _, a := range alls {
			all := strings.TrimSpace(a)
			if all == "" {
				continue
			}
			record := models.Allergen{
				UserID:        user.ID,
				AllergenName:  all,
				SeverityLevel: 1,
			}
			if err := tx.Create(&record).Error; err != nil {
				tx.Rollback()
				return "", err
			}
		}
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return "", err
	}

	return token, nil
}

func (s *AuthService) Login(email, password string) (string, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return "", errors.New("invalid credentials")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*middleware.TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, err
		}

		return &middleware.TokenClaims{
			UserID: userID,
		}, nil
	}

	return nil, errors.New("invalid token")
}
