package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the chat aggregator.
// All sensitive values are loaded from environment variables.
//
// Security considerations:
// - OAuth tokens are never logged or printed
// - Config values are validated at startup
// - Required fields cause immediate failure if missing
type Config struct {
	// Twitch configuration
	TwitchUsername   string // Bot/account username for Twitch
	TwitchOAuthToken string // OAuth token (format: oauth:xxxxxx)
	TwitchChannel    string // Channel to join (without #)

	// Kick configuration
	KickChannel string // Channel to join on Kick

	// Server configuration
	Port int // HTTP server port (default: 8080)

	// Feature flags
	EnableTwitch bool // Whether to connect to Twitch
	EnableKick   bool // Whether to connect to Kick
}

// LoadConfig reads configuration from environment variables.
// It first tries to load .env file, then falls back to system env vars.
//
// Security: OAuth tokens are trimmed but otherwise stored as-is.
// The caller should never log the config directly.
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	// This allows the app to work with just system env vars
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg := &Config{
		TwitchUsername:   getEnv("TWITCH_USERNAME", ""),
		TwitchOAuthToken: getEnv("TWITCH_OAUTH_TOKEN", ""),
		TwitchChannel:    getEnv("TWITCH_CHANNEL", ""),
		KickChannel:      getEnv("KICK_CHANNEL", ""),
		EnableTwitch:     true,
		EnableKick:       true,
	}

	// Parse port with default
	portStr := getEnv("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT value: %s", portStr)
	}
	cfg.Port = port

	// Determine which platforms are enabled
	// Twitch is enabled only if we have all required credentials
	cfg.EnableTwitch = cfg.TwitchUsername != "" && 
		cfg.TwitchOAuthToken != "" && 
		cfg.TwitchChannel != ""

	// Kick is enabled if channel is specified
	cfg.EnableKick = cfg.KickChannel != ""

	// Validate: at least one platform must be enabled
	if !cfg.EnableTwitch && !cfg.EnableKick {
		return nil, fmt.Errorf("at least one platform must be configured")
	}

	return cfg, nil
}

// Validate checks if Twitch configuration is complete.
func (c *Config) ValidateTwitch() error {
	if c.TwitchUsername == "" {
		return fmt.Errorf("TWITCH_USERNAME is required for Twitch")
	}
	if c.TwitchOAuthToken == "" {
		return fmt.Errorf("TWITCH_OAUTH_TOKEN is required for Twitch")
	}
	if c.TwitchChannel == "" {
		return fmt.Errorf("TWITCH_CHANNEL is required for Twitch")
	}
	if !strings.HasPrefix(c.TwitchOAuthToken, "oauth:") {
		return fmt.Errorf("TWITCH_OAUTH_TOKEN must start with 'oauth:'")
	}
	return nil
}

// ValidateKick checks if Kick configuration is complete.
func (c *Config) ValidateKick() error {
	if c.KickChannel == "" {
		return fmt.Errorf("KICK_CHANNEL is required for Kick")
	}
	return nil
}

// Sanitized returns a copy of the config safe for logging.
// OAuth tokens are masked to prevent accidental exposure.
func (c *Config) Sanitized() string {
	twitchToken := ""
	if c.TwitchOAuthToken != "" {
		// Show only last 4 characters
		if len(c.TwitchOAuthToken) > 8 {
			twitchToken = "oauth:****" + c.TwitchOAuthToken[len(c.TwitchOAuthToken)-4:]
		} else {
			twitchToken = "oauth:****"
		}
	}

	return fmt.Sprintf(
		"Config{TwitchUser: %s, TwitchToken: %s, TwitchChannel: %s, KickChannel: %s, Port: %d}",
		c.TwitchUsername,
		twitchToken,
		c.TwitchChannel,
		c.KickChannel,
		c.Port,
	)
}

// getEnv retrieves an environment variable or returns the default value.
// Values are trimmed of whitespace for robustness.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

// String implements fmt.Stringer for Config.
// Uses Sanitized() to prevent token exposure.
func (c *Config) String() string {
	return c.Sanitized()
}
