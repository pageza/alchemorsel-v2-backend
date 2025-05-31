package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"gorm.io/gorm"
)

type ProfileService struct {
	db *gorm.DB
}

func NewProfileService(db *gorm.DB) *ProfileService {
	return &ProfileService{db: db}
}

func (s *ProfileService) GetProfile(userID uuid.UUID) (*models.UserProfile, error) {
	var profile models.UserProfile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
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
	return nil
}

func (s *ProfileService) ValidateToken(token string) (*middleware.TokenClaims, error) {
	return nil, fmt.Errorf("not implemented")
}
