package config

import (
	"fmt"
	"os"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ConfigRequirements defines required configuration for each environment
type ConfigRequirements struct {
	RequiredEnvVars []string
	RequiredSecrets []string
}

var (
	// Environment-specific requirements
	requirements = map[Environment]ConfigRequirements{
		Development: {
			RequiredEnvVars: []string{
				"SERVER_PORT",
				"SERVER_HOST",
				"DB_HOST",
				"DB_PORT",
				"DB_NAME",
				"DB_SSL_MODE",
				"REDIS_HOST",
				"REDIS_PORT",
				"REDIS_URL",
			},
			RequiredSecrets: []string{
				"db_user",
				"db_password",
				"jwt_secret",
				"redis_password",
			},
		},
		Test: {
			RequiredEnvVars: []string{
				"SERVER_PORT",
				"SERVER_HOST",
				"DB_HOST",
				"DB_PORT",
				"DB_NAME",
				"DB_SSL_MODE",
				"REDIS_HOST",
				"REDIS_PORT",
				"REDIS_URL",
			},
			RequiredSecrets: []string{
				"db_user",
				"db_password",
				"jwt_secret",
				"redis_password",
			},
		},
		CI: {
			RequiredEnvVars: []string{
				"SERVER_PORT",
				"SERVER_HOST",
				"DB_HOST",
				"DB_PORT",
				"DB_USER",
				"DB_PASSWORD",
				"DB_NAME",
				"DB_SSL_MODE",
				"REDIS_HOST",
				"REDIS_PORT",
				"REDIS_URL",
				"JWT_SECRET",
				"REDIS_PASSWORD",
			},
			RequiredSecrets: []string{}, // CI uses environment variables, not Docker secrets
		},
		Production: {
			RequiredEnvVars: []string{
				"SERVER_PORT",
				"SERVER_HOST",
				"DB_HOST",
				"DB_PORT",
				"DB_NAME",
				"DB_SSL_MODE",
				"REDIS_HOST",
				"REDIS_PORT",
				"REDIS_URL",
			},
			RequiredSecrets: []string{
				"db_user",
				"db_password",
				"jwt_secret",
				"redis_password",
			},
		},
	}
)

// ValidateConfig checks if the configuration meets the requirements for the current environment
func ValidateConfig(cfg *Config) error {
	env := GetEnvironment()
	reqs := requirements[env]

	var errors []string

	// Validate environment variables
	for _, envVar := range reqs.RequiredEnvVars {
		if value := os.Getenv(envVar); value == "" {
			errors = append(errors, fmt.Sprintf("required environment variable %s is not set", envVar))
		}
	}

	// Validate secrets
	if env != CI { // Skip secret validation in CI environment
		for _, secret := range reqs.RequiredSecrets {
			if value := readSecret(secret); value == "" {
				errors = append(errors, fmt.Sprintf("required secret %s is not set", secret))
			}
		}
	}

	// Validate sensitive values
	if env == CI {
		// In CI, sensitive values must come from environment variables
		if cfg.DBPassword == "" {
			errors = append(errors, "DB_PASSWORD environment variable is required in CI environment")
		}
		if cfg.JWTSecret == "" {
			errors = append(errors, "JWT_SECRET environment variable is required in CI environment")
		}
		if cfg.RedisPassword == "" {
			errors = append(errors, "REDIS_PASSWORD environment variable is required in CI environment")
		}
	} else {
		// In other environments, sensitive values must come from Docker secrets
		if cfg.DBPassword == "" {
			errors = append(errors, "db_password secret is required")
		}
		if cfg.JWTSecret == "" {
			errors = append(errors, "jwt_secret secret is required")
		}
		if cfg.RedisPassword == "" {
			errors = append(errors, "redis_password secret is required")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}
