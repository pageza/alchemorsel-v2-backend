package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	ServerPort string
	ServerHost string

	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// JWT configuration
	JWTSecret string

	// New fields
	RedisURL string
}

// LoadConfig creates a new Config instance with values from environment variables or secrets
func LoadConfig() (*Config, error) {
	env := GetEnvironment()
	cfg := &Config{}

	// Load configuration based on environment
	switch env {
	case CI:
		if err := loadCIConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to load CI configuration: %w", err)
		}
	case Development, Test:
		if err := loadDevConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to load development configuration: %w", err)
		}
	case Production:
		if err := loadProdConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to load production configuration: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown environment: %s", env)
	}

	// Validate the configuration
	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// loadCIConfig loads configuration for CI environment
func loadCIConfig(cfg *Config) error {
	redisDB, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))

	cfg.ServerPort = getEnvOrDefault("SERVER_PORT", "8080")
	cfg.ServerHost = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
	cfg.DBHost = getEnvOrDefault("DB_HOST", "localhost")
	cfg.DBPort = getEnvOrDefault("DB_PORT", "5432")
	cfg.DBUser = getEnvOrDefault("DB_USER", "postgres")
	cfg.DBPassword = getEnvOrDefault("DB_PASSWORD", "")
	cfg.DBName = getEnvOrDefault("DB_NAME", "alchemorsel")
	cfg.DBSSLMode = getEnvOrDefault("DB_SSL_MODE", "disable")
	cfg.RedisHost = getEnvOrDefault("REDIS_HOST", "localhost")
	cfg.RedisPort = getEnvOrDefault("REDIS_PORT", "6379")
	cfg.RedisPassword = getEnvOrDefault("REDIS_PASSWORD", "")
	cfg.RedisDB = redisDB
	cfg.JWTSecret = getEnvOrDefault("JWT_SECRET", "")
	cfg.RedisURL = getEnvOrDefault("REDIS_URL", "redis://localhost:6379")

	return nil
}

// loadDevConfig loads configuration for development environment
func loadDevConfig(cfg *Config) error {
	redisDB, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))

	cfg.ServerPort = getEnvOrDefault("SERVER_PORT", "8080")
	cfg.ServerHost = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
	cfg.DBHost = getEnvOrDefault("DB_HOST", "localhost")
	cfg.DBPort = getEnvOrDefault("DB_PORT", "5432")
	cfg.DBUser = getEnvOrSecret("DB_USER", "db_user")
	cfg.DBPassword = getEnvOrSecret("DB_PASSWORD", "db_password")
	cfg.DBName = getEnvOrDefault("DB_NAME", "alchemorsel")
	cfg.DBSSLMode = getEnvOrDefault("DB_SSL_MODE", "disable")
	cfg.RedisHost = getEnvOrDefault("REDIS_HOST", "localhost")
	cfg.RedisPort = getEnvOrDefault("REDIS_PORT", "6379")
	cfg.RedisPassword = getEnvOrSecret("REDIS_PASSWORD", "redis_password")
	cfg.RedisDB = redisDB
	cfg.JWTSecret = getEnvOrSecret("JWT_SECRET", "jwt_secret")
	cfg.RedisURL = getEnvOrDefault("REDIS_URL", "redis://localhost:6379")

	return nil
}

// loadProdConfig loads configuration for production environment
func loadProdConfig(cfg *Config) error {
	redisDB, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))

	cfg.ServerPort = getEnvOrDefault("SERVER_PORT", "8080")
	cfg.ServerHost = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
	cfg.DBHost = getEnvOrDefault("DB_HOST", "localhost")
	cfg.DBPort = getEnvOrDefault("DB_PORT", "5432")
	cfg.DBUser = getEnvOrSecret("DB_USER", "db_user")
	cfg.DBPassword = getEnvOrSecret("DB_PASSWORD", "db_password")
	cfg.DBName = getEnvOrDefault("DB_NAME", "alchemorsel")
	cfg.DBSSLMode = getEnvOrDefault("DB_SSL_MODE", "disable")
	cfg.RedisHost = getEnvOrDefault("REDIS_HOST", "localhost")
	cfg.RedisPort = getEnvOrDefault("REDIS_PORT", "6379")
	cfg.RedisPassword = getEnvOrSecret("REDIS_PASSWORD", "redis_password")
	cfg.RedisDB = redisDB
	cfg.JWTSecret = getEnvOrSecret("JWT_SECRET", "jwt_secret")
	cfg.RedisURL = getEnvOrDefault("REDIS_URL", "redis://localhost:6379")

	return nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrSecret tries to get an environment variable, then falls back to Docker secret
func getEnvOrSecret(envKey, secretName string) string {
	if value, exists := os.LookupEnv(envKey); exists {
		return value
	}
	return readSecret(secretName)
}

// readSecret reads a Docker secret from the default secrets directory
func readSecret(name string) string {
	secretPath := filepath.Join("/run/secrets", name)
	if data, err := os.ReadFile(secretPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}
