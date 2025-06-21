package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

	// Hash password for test users
	password := "testpassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	now := time.Now()

	// Test users with different verification statuses
	testUsers := []struct {
		name       string
		email      string
		username   string
		verified   bool
		verifiedAt *time.Time
		role       string
	}{
		{
			name:       "John Doe",
			email:      "john.doe@example.com",
			username:   "johndoe",
			verified:   true,
			verifiedAt: &now,
			role:       "user",
		},
		{
			name:       "Jane Smith",
			email:      "jane.smith@example.com",
			username:   "janesmith",
			verified:   true,
			verifiedAt: &now,
			role:       "user",
		},
		{
			name:       "Bob Wilson",
			email:      "bob.wilson@example.com",
			username:   "bobwilson",
			verified:   false,
			verifiedAt: nil,
			role:       "user",
		},
		{
			name:       "Alice Cooper",
			email:      "alice.cooper@example.com",
			username:   "alicecooper",
			verified:   false,
			verifiedAt: nil,
			role:       "user",
		},
		{
			name:       "Admin User",
			email:      "admin@example.com",
			username:   "admin",
			verified:   true,
			verifiedAt: &now,
			role:       "admin",
		},
		{
			name:       "Test Verified",
			email:      "verified@example.com",
			username:   "verified_user",
			verified:   true,
			verifiedAt: &now,
			role:       "user",
		},
		{
			name:       "Test Unverified",
			email:      "unverified@example.com",
			username:   "unverified_user",
			verified:   false,
			verifiedAt: nil,
			role:       "user",
		},
	}

	log.Println("Creating test users with mixed verification statuses...")

	for _, userData := range testUsers {
		// Check if user already exists
		var existingUser models.User
		if err := db.Where("email = ?", userData.email).First(&existingUser).Error; err == nil {
			log.Printf("User %s already exists, skipping...", userData.email)
			continue
		}

		// Create user
		userID := uuid.New()
		user := models.User{
			ID:              userID,
			Name:            userData.name,
			Email:           userData.email,
			PasswordHash:    string(hashedPassword),
			EmailVerified:   userData.verified,
			EmailVerifiedAt: userData.verifiedAt,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("Failed to create user %s: %v", userData.email, err)
			continue
		}

		// Create user profile
		profile := models.UserProfile{
			ID:                uuid.New(),
			UserID:            userID,
			Username:          userData.username,
			Bio:               fmt.Sprintf("Test user for development - %s", userData.role),
			ProfilePictureURL: "",
			PrivacyLevel:      "public",
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if err := db.Create(&profile).Error; err != nil {
			log.Printf("Failed to create profile for %s: %v", userData.email, err)
			continue
		}

		// Add some dietary preferences for testing
		if userData.verified {
			dietaryPrefs := []string{"vegetarian", "gluten-free"}
			for _, pref := range dietaryPrefs {
				dietPref := models.DietaryPreference{
					UserID:         userID,
					PreferenceType: pref,
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				db.Create(&dietPref)
			}

			// Add some allergens for testing
			allergens := []string{"nuts", "dairy"}
			for _, allergen := range allergens {
				allergenEntry := models.Allergen{
					UserID:        userID,
					AllergenName:  allergen,
					SeverityLevel: 2,
					CreatedAt:     now,
					UpdatedAt:     now,
				}
				db.Create(&allergenEntry)
			}
		}

		status := "unverified"
		if userData.verified {
			status = "verified"
		}
		log.Printf("✅ Created %s user: %s (%s) - %s", userData.role, userData.name, userData.email, status)
	}

	log.Println("\n📋 Test Users Summary:")
	log.Println("======================")

	var verifiedCount, unverifiedCount int64
	db.Model(&models.User{}).Where("email_verified = ?", true).Count(&verifiedCount)
	db.Model(&models.User{}).Where("email_verified = ?", false).Count(&unverifiedCount)

	log.Printf("✅ Verified users: %d", verifiedCount)
	log.Printf("❌ Unverified users: %d", unverifiedCount)
	log.Printf("📧 Total users: %d", verifiedCount+unverifiedCount)

	log.Println("\n🔑 Test Credentials:")
	log.Println("Email: Any of the above emails")
	log.Println("Password: testpassword123")

	log.Println("\nTest users created successfully!")
}
