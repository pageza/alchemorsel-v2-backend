package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestDatabase(t *testing.T) {
	// Use the exported SetupTestDatabase from testhelpers package
	db := testhelpers.SetupTestDatabase(t)
	assert.NotNil(t, db)

	// Test database operations
	user := models.User{
		ID:           uuid.New(),
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
	}

	err := db.DB().Create(&user).Error
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)

	// Cleanup is handled by SetupTestDatabase
}
