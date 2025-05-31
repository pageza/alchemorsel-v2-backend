package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "alchemorsel")
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("REDIS_URL", "redis://localhost:6379")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test database configuration
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "alchemorsel", cfg.DBName)
	assert.Equal(t, "disable", cfg.DBSSLMode)

	// Test JWT configuration
	assert.Equal(t, "test-secret", cfg.JWTSecret)

	// Test Redis configuration
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_SSL_MODE")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("REDIS_URL")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test default values
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "alchemorsel", cfg.DBName)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, "your-secret-key", cfg.JWTSecret)
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
}
