package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Test CI environment with GitHub Actions secrets
	t.Run("CI Environment with GitHub Actions Secrets", func(t *testing.T) {
		// Set CI environment
		os.Setenv("CI", "true")
		defer os.Unsetenv("CI")

		// Set required GitHub Actions variables
		os.Setenv("SERVER_PORT", "8080")
		os.Setenv("SERVER_HOST", "localhost")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "postgres")
		os.Setenv("DB_NAME", "alchemorsel")
		os.Setenv("DB_SSL_MODE", "disable")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("REDIS_PORT", "6379")

		// Set required GitHub Actions secrets
		os.Setenv("TEST_DB_PASSWORD", "postpass")
		os.Setenv("TEST_JWT_SECRET", "test-jwt-secret")
		os.Setenv("TEST_REDIS_PASSWORD", "test-redis-pass")
		os.Setenv("TEST_REDIS_URL", "redis://localhost:6379")

		defer func() {
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("SERVER_HOST")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_SSL_MODE")
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_PORT")
			os.Unsetenv("TEST_DB_PASSWORD")
			os.Unsetenv("TEST_JWT_SECRET")
			os.Unsetenv("TEST_REDIS_PASSWORD")
			os.Unsetenv("TEST_REDIS_URL")
		}()

		cfg, err := LoadConfig()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Test database configuration
		assert.Equal(t, "localhost", cfg.DBHost)
		assert.Equal(t, "5432", cfg.DBPort)
		assert.Equal(t, "postgres", cfg.DBUser)
		assert.Equal(t, "postpass", cfg.DBPassword)
		assert.Equal(t, "alchemorsel", cfg.DBName)
		assert.Equal(t, "disable", cfg.DBSSLMode)

		// Test JWT configuration
		assert.Equal(t, "test-jwt-secret", cfg.JWTSecret)

		// Test Redis configuration
		assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
		assert.Equal(t, "test-redis-pass", cfg.RedisPassword)
	})
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Test CI environment with GitHub Actions secrets
	t.Run("CI Environment with GitHub Actions Secrets", func(t *testing.T) {
		// Set CI environment
		os.Setenv("CI", "true")
		defer os.Unsetenv("CI")

		// Set required GitHub Actions variables
		os.Setenv("SERVER_PORT", "8080")
		os.Setenv("SERVER_HOST", "localhost")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "postgres")
		os.Setenv("DB_NAME", "alchemorsel")
		os.Setenv("DB_SSL_MODE", "disable")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("REDIS_PORT", "6379")

		// Set required GitHub Actions secrets
		os.Setenv("TEST_DB_PASSWORD", "postpass")
		os.Setenv("TEST_JWT_SECRET", "test-jwt-secret")
		os.Setenv("TEST_REDIS_PASSWORD", "test-redis-pass")
		os.Setenv("TEST_REDIS_URL", "redis://localhost:6379")

		defer func() {
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("SERVER_HOST")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_SSL_MODE")
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_PORT")
			os.Unsetenv("TEST_DB_PASSWORD")
			os.Unsetenv("TEST_JWT_SECRET")
			os.Unsetenv("TEST_REDIS_PASSWORD")
			os.Unsetenv("TEST_REDIS_URL")
		}()

		cfg, err := LoadConfig()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Test database configuration
		assert.Equal(t, "localhost", cfg.DBHost)
		assert.Equal(t, "5432", cfg.DBPort)
		assert.Equal(t, "postgres", cfg.DBUser)
		assert.Equal(t, "postpass", cfg.DBPassword)
		assert.Equal(t, "alchemorsel", cfg.DBName)
		assert.Equal(t, "disable", cfg.DBSSLMode)

		// Test JWT configuration
		assert.Equal(t, "test-jwt-secret", cfg.JWTSecret)

		// Test Redis configuration
		assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
		assert.Equal(t, "test-redis-pass", cfg.RedisPassword)
	})
}
