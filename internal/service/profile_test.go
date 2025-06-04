package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateTokenValid(t *testing.T) {
	secret := "test-secret"
	svc := NewProfileService(nil, secret)
	userID := uuid.New()
	token, err := svc.GenerateToken(userID.String(), "tester")
	assert.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "tester", claims.Username)
}

func TestValidateTokenInvalid(t *testing.T) {
	secret := "test-secret"
	svc := NewProfileService(nil, secret)

	claims, err := svc.ValidateToken("invalid.token")
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
