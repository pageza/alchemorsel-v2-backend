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
		defer func() { _ = os.Unsetenv("CI") }()

		// Set required GitHub Actions variables
		_ = os.Setenv("SERVER_PORT", "8080")
		_ = os.Setenv("SERVER_HOST", "localhost")
		_ = os.Setenv("DB_HOST", "localhost")
		_ = os.Setenv("DB_PORT", "5432")
		_ = os.Setenv("DB_USER", "postgres")
		_ = os.Setenv("DB_NAME", "alchemorsel")
		_ = os.Setenv("DB_SSL_MODE", "disable")
		_ = os.Setenv("REDIS_HOST", "localhost")
		_ = os.Setenv("REDIS_PORT", "6379")

		// Set required GitHub Actions secrets
		_ = os.Setenv("TEST_DB_PASSWORD", "postpass")
		_ = os.Setenv("TEST_JWT_SECRET", "test-jwt-secret")
		_ = os.Setenv("TEST_REDIS_PASSWORD", "test-redis-pass")
		_ = os.Setenv("TEST_REDIS_URL", "redis://localhost:6379")

		defer func() {
			_ = os.Unsetenv("SERVER_PORT")
			_ = os.Unsetenv("SERVER_HOST")
			_ = os.Unsetenv("DB_HOST")
			_ = os.Unsetenv("DB_PORT")
			_ = os.Unsetenv("DB_USER")
			_ = os.Unsetenv("DB_NAME")
			_ = os.Unsetenv("DB_SSL_MODE")
			_ = os.Unsetenv("REDIS_HOST")
			_ = os.Unsetenv("REDIS_PORT")
			_ = os.Unsetenv("TEST_DB_PASSWORD")
			_ = os.Unsetenv("TEST_JWT_SECRET")
			_ = os.Unsetenv("TEST_REDIS_PASSWORD")
			_ = os.Unsetenv("TEST_REDIS_URL")
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
		defer func() { _ = os.Unsetenv("CI") }()

		// Set required GitHub Actions variables
		_ = os.Setenv("SERVER_PORT", "8080")
		_ = os.Setenv("SERVER_HOST", "localhost")
		_ = os.Setenv("DB_HOST", "localhost")
		_ = os.Setenv("DB_PORT", "5432")
		_ = os.Setenv("DB_USER", "postgres")
		_ = os.Setenv("DB_NAME", "alchemorsel")
		_ = os.Setenv("DB_SSL_MODE", "disable")
		_ = os.Setenv("REDIS_HOST", "localhost")
		_ = os.Setenv("REDIS_PORT", "6379")

		// Set required GitHub Actions secrets
		_ = os.Setenv("TEST_DB_PASSWORD", "postpass")
		_ = os.Setenv("TEST_JWT_SECRET", "test-jwt-secret")
		_ = os.Setenv("TEST_REDIS_PASSWORD", "test-redis-pass")
		_ = os.Setenv("TEST_REDIS_URL", "redis://localhost:6379")

		defer func() {
			_ = os.Unsetenv("SERVER_PORT")
			_ = os.Unsetenv("SERVER_HOST")
			_ = os.Unsetenv("DB_HOST")
			_ = os.Unsetenv("DB_PORT")
			_ = os.Unsetenv("DB_USER")
			_ = os.Unsetenv("DB_NAME")
			_ = os.Unsetenv("DB_SSL_MODE")
			_ = os.Unsetenv("REDIS_HOST")
			_ = os.Unsetenv("REDIS_PORT")
			_ = os.Unsetenv("TEST_DB_PASSWORD")
			_ = os.Unsetenv("TEST_JWT_SECRET")
			_ = os.Unsetenv("TEST_REDIS_PASSWORD")
			_ = os.Unsetenv("TEST_REDIS_URL")
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
