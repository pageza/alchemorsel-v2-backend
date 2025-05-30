package config

import (
	"os"
	"path/filepath"
)

// Config holds all configuration for the application
type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

// New creates a new Config with values from Docker Secrets
func New() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     readSecret("postgres_user"),
		DBPassword: readSecret("postgres_password"),
		DBName:     readSecret("postgres_db"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// readSecret reads a Docker secret from the default secrets directory
func readSecret(name string) string {
	secretPath := filepath.Join("/run/secrets", name)
	if data, err := os.ReadFile(secretPath); err == nil {
		return string(data)
	}
	return ""
}
