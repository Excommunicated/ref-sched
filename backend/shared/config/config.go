package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// Database
	DatabaseURL string

	// Server
	Port string
	Env  string // "development" or "production"

	// Session
	SessionSecret string

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Frontend
	FrontendURL string

	// Audit
	AuditRetentionDays int

	// Match Archival
	MatchRetentionDays int
}

// Load reads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		SessionSecret:      getEnv("SESSION_SECRET", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		AuditRetentionDays: getEnvInt("AUDIT_RETENTION_DAYS", 730),
		MatchRetentionDays: getEnvInt("MATCH_RETENTION_DAYS", 730),
	}

	// Validate required fields
	cfg.validate()

	// Ensure database URL has timezone parameter
	cfg.ensureDatabaseTimezone()

	log.Println("Configuration loaded successfully")
	return cfg
}

// validate ensures required configuration is present
func (c *Config) validate() {
	required := map[string]string{
		"DATABASE_URL":         c.DatabaseURL,
		"SESSION_SECRET":       c.SessionSecret,
		"GOOGLE_CLIENT_ID":     c.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": c.GoogleClientSecret,
		"GOOGLE_REDIRECT_URL":  c.GoogleRedirectURL,
	}

	var missing []string
	for key, value := range required {
		if value == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		log.Fatalf("Missing required environment variables: %v", missing)
	}

	if c.IsProduction() {
		knownDefaults := map[string]bool{
			"dev_session_secret_change_in_production": true,
			"changeme":       true,
			"secret":         true,
			"session_secret": true,
			"default":        true,
		}
		if knownDefaults[c.SessionSecret] {
			log.Fatalf("In production, SESSION_SECRET must be set to a secure value and not a known default")
		}
		if len(c.SessionSecret) < 32 {
			log.Fatalf("In production, SESSION_SECRET must be at least 32 characters long")
		}
	}
}

// ensureDatabaseTimezone adds timezone parameter to database URL if not present
func (c *Config) ensureDatabaseTimezone() {
	if !strings.Contains(c.DatabaseURL, "timezone=") {
		if strings.Contains(c.DatabaseURL, "?") {
			c.DatabaseURL += "&timezone=America/New_York"
		} else {
			c.DatabaseURL += "?timezone=America/New_York"
		}
	}
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// getEnv reads an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt reads an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil && intValue > 0 {
			return intValue
		}
		log.Printf("Invalid integer value for %s, using default: %d", key, defaultValue)
	}
	return defaultValue
}
