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
			RequiredEnvVars: []string{}, // No environment variables allowed
			RequiredSecrets: []string{
				"db_user",
				"db_password",
				"jwt_secret",
				"redis_password",
			},
		},
		Test: {
			RequiredEnvVars: []string{}, // No environment variables allowed
			RequiredSecrets: []string{
				"db_user",
				"db_password",
				"jwt_secret",
				"redis_password",
			},
		},
		CI: {
			RequiredEnvVars: []string{}, // No environment variables allowed
			RequiredSecrets: []string{}, // GitHub Actions secrets are injected by CI system
		},
		Production: {
			RequiredEnvVars: []string{}, // No environment variables allowed
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

	// Validate environment variables - none should be used
	for _, envVar := range reqs.RequiredEnvVars {
		if value := os.Getenv(envVar); value != "" {
			errors = append(errors, fmt.Sprintf("environment variable %s should not be used", envVar))
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
		// In CI, sensitive values must come from GitHub Actions secrets
		if cfg.DBPassword == "" {
			errors = append(errors, "TEST_DB_PASSWORD GitHub Actions secret is required in CI environment")
		}
		if cfg.JWTSecret == "" {
			errors = append(errors, "TEST_JWT_SECRET GitHub Actions secret is required in CI environment")
		}
		// E2E-FIX-2025-E: Allow empty Redis password in CI since Redis runs without auth in test setup
		// Redis password is optional in CI environment when Redis is configured without authentication
		// This prevents CI failures where empty Redis password was being incorrectly rejected
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
