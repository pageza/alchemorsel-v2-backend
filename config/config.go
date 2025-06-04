package config

import (
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

// New creates a new Config instance with values from environment variables
func New() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	return &Config{
		// Server configuration
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),

		// Database configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "alchemorsel"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Redis configuration
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,

		// JWT configuration
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),

		// New fields
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
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

func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		ServerHost: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
		DBHost:     getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:     getEnvOrDefault("DB_PORT", "5432"),
		DBUser:     getEnvOrSecret("DB_USER", "db_user"),
		DBPassword: getEnvOrSecret("DB_PASSWORD", "db_password"),
		DBName:     getEnvOrSecret("DB_NAME", "db_name"),
		DBSSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
		JWTSecret:  getEnvOrSecret("JWT_SECRET", "jwt_secret"),
		RedisURL:   getEnvOrSecret("REDIS_URL", "redis_url"),
	}
	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
