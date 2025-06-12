package config

import (
	"fmt"
	"os"
	"path/filepath"
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

// loadCIConfig loads configuration for CI environment using ONLY GitHub Actions secrets
func loadCIConfig(cfg *Config) error {
	// GitHub Actions variables
	cfg.ServerPort = os.Getenv("SERVER_PORT")
	cfg.ServerHost = os.Getenv("SERVER_HOST")
	cfg.DBHost = os.Getenv("DB_HOST")
	cfg.DBPort = os.Getenv("DB_PORT")
	cfg.DBUser = os.Getenv("DB_USER")
	cfg.DBName = os.Getenv("DB_NAME")
	cfg.DBSSLMode = os.Getenv("DB_SSL_MODE")
	cfg.RedisHost = os.Getenv("REDIS_HOST")
	cfg.RedisPort = os.Getenv("REDIS_PORT")

	// GitHub Actions secrets - use environment variables directly
	cfg.DBPassword = os.Getenv("TEST_DB_PASSWORD")
	if cfg.DBPassword == "" {
		return fmt.Errorf("TEST_DB_PASSWORD environment variable is required in CI environment")
	}
	cfg.JWTSecret = os.Getenv("TEST_JWT_SECRET")
	cfg.RedisPassword = os.Getenv("TEST_REDIS_PASSWORD")
	cfg.RedisURL = os.Getenv("TEST_REDIS_URL")
	cfg.RedisDB = 0 // This is a constant, not a secret

	return nil
}

// loadDevConfig loads configuration for development environment
func loadDevConfig(cfg *Config) error {
	secretsDir := os.Getenv("SECRETS_DIR")
	if secretsDir == "" {
		secretsDir = "/run/secrets"
	}

	// Load secrets from Docker secrets
	secrets := make(map[string]string)
	secretFiles := []string{
		"db_user",
		"db_password",
		"jwt_secret",
		"redis_password",
		"db_host",
		"db_port",
		"db_name",
		"db_ssl_mode",
		"redis_host",
		"redis_port",
		"redis_url",
		"server_port",
		"server_host",
	}

	for _, name := range secretFiles {
		content, err := os.ReadFile(filepath.Join(secretsDir, name))
		if err != nil {
			return fmt.Errorf("failed to read secret %s: %v", name, err)
		}
		secrets[name] = strings.TrimSpace(string(content))
	}

	cfg.ServerPort = secrets["server_port"]
	cfg.ServerHost = secrets["server_host"]
	cfg.DBHost = secrets["db_host"]
	cfg.DBPort = secrets["db_port"]
	cfg.DBUser = secrets["db_user"]
	cfg.DBPassword = secrets["db_password"]
	cfg.DBName = secrets["db_name"]
	cfg.DBSSLMode = secrets["db_ssl_mode"]
	cfg.RedisHost = secrets["redis_host"]
	cfg.RedisPort = secrets["redis_port"]
	cfg.RedisPassword = secrets["redis_password"]
	cfg.RedisDB = 0 // This is a constant, not a secret
	cfg.JWTSecret = secrets["jwt_secret"]
	cfg.RedisURL = secrets["redis_url"]

	return nil
}

// loadProdConfig loads configuration for production environment using ONLY Docker secrets
func loadProdConfig(cfg *Config) error {
	cfg.ServerPort = readSecret("server_port")
	cfg.ServerHost = readSecret("server_host")
	cfg.DBHost = readSecret("db_host")
	cfg.DBPort = readSecret("db_port")
	cfg.DBUser = readSecret("db_user")
	cfg.DBPassword = readSecret("db_password")
	cfg.DBName = readSecret("db_name")
	cfg.DBSSLMode = readSecret("db_ssl_mode")
	cfg.RedisHost = readSecret("redis_host")
	cfg.RedisPort = readSecret("redis_port")
	cfg.RedisPassword = readSecret("redis_password")
	cfg.RedisDB = 0 // This is a constant, not a secret
	cfg.JWTSecret = readSecret("jwt_secret")
	cfg.RedisURL = readSecret("redis_url")

	return nil
}

// readSecret reads a Docker secret from the secrets directory
func readSecret(name string) string {
	secretsDir := os.Getenv("SECRETS_DIR")
	if secretsDir == "" {
		secretsDir = "/run/secrets"
	}
	secretPath := filepath.Join(secretsDir, name)
	if data, err := os.ReadFile(secretPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}
