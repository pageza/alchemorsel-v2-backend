package testhelpers

import (
	"testing"

	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseSetup(t *testing.T) {
	// Use the exported SetupTestDB from testhelpers package
	db := SetupTestDB(t)
	assert.NotNil(t, db)

	// Test creating a user
	user := &models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}
	err := db.DB().Create(user).Error
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)

	// Cleanup is handled by SetupTestDB
}
