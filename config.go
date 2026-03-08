package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// ioWriter and ioReader are used for testing
var (
	stdout io.Writer = os.Stdout
	stdin  io.Reader = os.Stdin
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

// ========================================
// Setup Wizard
// ========================================

// ConfigFileExists checks if .env file exists.
func ConfigFileExists() bool {
	_, err := os.Stat(".env")
	return err == nil
}

// RunSetupWizard runs the interactive configuration wizard.
// Returns a Config if successful, or an error.
func RunSetupWizard() (*Config, error) {
	return runSetupWizardWith(bufio.NewReader(stdin), stdout)
}

// runSetupWizardWith runs the setup wizard with the provided reader/writer.
// This is used for testing with mocked input.
func runSetupWizardWith(r *bufio.Reader, w io.Writer) (*Config, error) {
	printSetupBannerTo(w)

	cfg := &Config{
		Port:        8080,
		EnableTwitch: true,
		EnableKick:  true,
	}

	// Ask which platforms to configure
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Which platforms do you want to configure?")
	fmt.Fprintln(w, "  1. Both Twitch and Kick")
	fmt.Fprintln(w, "  2. Twitch only")
	fmt.Fprintln(w, "  3. Kick only")
	fmt.Fprintln(w)

	choice := readInputFrom(r, w, "Enter choice (1-3)", "1")

	switch strings.TrimSpace(choice) {
	case "1":
		// Configure both
		if err := configureTwitchWith(cfg, r, w); err != nil {
			return nil, err
		}
		if err := configureKickWith(cfg, r, w); err != nil {
			return nil, err
		}
	case "2":
		// Twitch only
		cfg.EnableKick = false
		if err := configureTwitchWith(cfg, r, w); err != nil {
			return nil, err
		}
	case "3":
		// Kick only
		cfg.EnableTwitch = false
		if err := configureKickWith(cfg, r, w); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid choice: %s", choice)
	}

	// Ask about port
	portStr := readInputFrom(r, w, "Server port", "8080")
	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		return nil, fmt.Errorf("invalid port number: %s", portStr)
	}
	cfg.Port = port

	// Save configuration
	if err := SaveConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "✅ Configuration saved to .env file!")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(w)

	return cfg, nil
}

// printSetupBannerTo prints the setup banner to the provided writer.
func printSetupBannerTo(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "╔═══════════════════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║                                                           ║")
	fmt.Fprintln(w, "║         🔀  Chat Aggregator Setup Wizard                  ║")
	fmt.Fprintln(w, "║                                                           ║")
	fmt.Fprintln(w, "║   No configuration file found. Let's set things up!       ║")
	fmt.Fprintln(w, "║                                                           ║")
	fmt.Fprintln(w, "╚═══════════════════════════════════════════════════════════╝")
}

// configureTwitch collects Twitch configuration from user.
func configureTwitch(cfg *Config) error {
	return configureTwitchWith(cfg, bufio.NewReader(stdin), stdout)
}

// configureTwitchWith collects Twitch configuration using the provided reader/writer.
func configureTwitchWith(cfg *Config, r *bufio.Reader, w io.Writer) error {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "🟣 TWITCH CONFIGURATION")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(w)

	// Show OAuth instructions
	fmt.Fprintln(w, "To get your Twitch OAuth token:")
	fmt.Fprintln(w, "  1. Go to: https://twitchtokengenerator.com/")
	fmt.Fprintln(w, "  2. Click 'Connect with Twitch'")
	fmt.Fprintln(w, "  3. Select 'Chat:Read' scope")
	fmt.Fprintln(w, "  4. Copy the token (starts with 'oauth:')")
	fmt.Fprintln(w)

	cfg.TwitchUsername = strings.TrimSpace(readInputFrom(r, w, "Your Twitch username", ""))
	if cfg.TwitchUsername == "" {
		return fmt.Errorf("Twitch username is required")
	}

	cfg.TwitchChannel = strings.TrimSpace(readInputFrom(r, w, "Twitch channel to join", ""))
	if cfg.TwitchChannel == "" {
		return fmt.Errorf("Twitch channel is required")
	}

	cfg.TwitchOAuthToken = strings.TrimSpace(readInputFrom(r, w, "Twitch OAuth token", ""))
	if cfg.TwitchOAuthToken == "" {
		return fmt.Errorf("Twitch OAuth token is required")
	}

	// Ensure token has oauth: prefix
	if !strings.HasPrefix(cfg.TwitchOAuthToken, "oauth:") {
		cfg.TwitchOAuthToken = "oauth:" + cfg.TwitchOAuthToken
	}

	return nil
}

// configureKick collects Kick configuration from user.
func configureKick(cfg *Config) error {
	return configureKickWith(cfg, bufio.NewReader(stdin), stdout)
}

// configureKickWith collects Kick configuration using the provided reader/writer.
func configureKickWith(cfg *Config, r *bufio.Reader, w io.Writer) error {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "🟢 KICK CONFIGURATION")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "No authentication needed for Kick - just enter the channel name!")
	fmt.Fprintln(w)

	cfg.KickChannel = strings.TrimSpace(readInputFrom(r, w, "Kick channel to join", ""))
	if cfg.KickChannel == "" {
		return fmt.Errorf("Kick channel is required")
	}

	return nil
}

// SaveConfig writes the configuration to .env file.
func SaveConfig(cfg *Config) error {
	var lines []string

	lines = append(lines, "# Chat Aggregator Configuration")
	lines = append(lines, "# Generated by setup wizard")
	lines = append(lines, "")

	if cfg.TwitchUsername != "" {
		lines = append(lines, "# Twitch Configuration")
		lines = append(lines, fmt.Sprintf("TWITCH_USERNAME=%s", cfg.TwitchUsername))
		lines = append(lines, fmt.Sprintf("TWITCH_OAUTH_TOKEN=%s", cfg.TwitchOAuthToken))
		lines = append(lines, fmt.Sprintf("TWITCH_CHANNEL=%s", cfg.TwitchChannel))
		lines = append(lines, "")
	}

	if cfg.KickChannel != "" {
		lines = append(lines, "# Kick Configuration")
		lines = append(lines, fmt.Sprintf("KICK_CHANNEL=%s", cfg.KickChannel))
		lines = append(lines, "")
	}

	lines = append(lines, "# Server Configuration")
	lines = append(lines, fmt.Sprintf("PORT=%d", cfg.Port))

	content := strings.Join(lines, "\n")
	return os.WriteFile(".env", []byte(content), 0600) // 0600 = owner read/write only
}

// printSetupBanner displays the setup wizard banner.
func printSetupBanner() {
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "╔═══════════════════════════════════════════════════════════╗")
	fmt.Fprintln(stdout, "║                                                           ║")
	fmt.Fprintln(stdout, "║         🔀  Chat Aggregator Setup Wizard                  ║")
	fmt.Fprintln(stdout, "║                                                           ║")
	fmt.Fprintln(stdout, "║   No configuration file found. Let's set things up!       ║")
	fmt.Fprintln(stdout, "║                                                           ║")
	fmt.Fprintln(stdout, "╚═══════════════════════════════════════════════════════════╝")
}

// readInput reads user input with a prompt and default value.
func readInput(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(stdout, "%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Fprintf(stdout, "%s: ", prompt)
	}

	// Use a shared buffered reader for stdin
	if sharedBufReader == nil {
		sharedBufReader = bufio.NewReader(stdin)
	}

	input, _ := sharedBufReader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// sharedBufReader is a shared buffered reader for stdin
var sharedBufReader *bufio.Reader

// readInputFrom reads user input from a buffered reader.
// This is used for testing with mocked input.
func readInputFrom(r *bufio.Reader, w io.Writer, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(w, "%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Fprintf(w, "%s: ", prompt)
	}

	input, _ := r.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}
