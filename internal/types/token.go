package types

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	IsEmailVerified bool      `json:"is_email_verified"`
}

// GetAudience implements jwt.Claims
func (c *TokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return c.RegisteredClaims.GetAudience()
}

// GetExpirationTime implements jwt.Claims
func (c *TokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetExpirationTime()
}

// GetNotBefore implements jwt.Claims
func (c *TokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetNotBefore()
}

// GetIssuedAt implements jwt.Claims
func (c *TokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetIssuedAt()
}

// GetIssuer implements jwt.Claims
func (c *TokenClaims) GetIssuer() (string, error) {
	return c.RegisteredClaims.GetIssuer()
}

// GetSubject implements jwt.Claims
func (c *TokenClaims) GetSubject() (string, error) {
	return c.RegisteredClaims.GetSubject()
}
