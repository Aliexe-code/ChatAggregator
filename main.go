// Package main is the entry point for the Multi-Platform Chat Aggregator.
// This application connects to Twitch and Kick chat simultaneously and
// displays messages from both platforms in a unified interface.
//
// Usage:
//
//	chat-aggregator [flags]
//
// Flags:
//
//	-h, --help     Show help message
//
// Environment Variables:
//
//	TWITCH_USERNAME     Twitch bot username
//	TWITCH_OAUTH_TOKEN  Twitch OAuth token (format: oauth:xxxxx)
//	TWITCH_CHANNEL      Twitch channel to join
//	KICK_CHANNEL        Kick channel to join
//	PORT                HTTP server port (default: 8080)
//
// Example .env file:
//
//	TWITCH_USERNAME=mybot
//	TWITCH_OAUTH_TOKEN=oauth:abc123xyz
//	TWITCH_CHANNEL=mystream
//	KICK_CHANNEL=mystream
//	PORT=8080
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Version information (set at build time)
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	// Print banner
	printBanner()

	// Load configuration from environment
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("❌ Configuration error: %v", err)
	}

	log.Printf("📋 Configuration loaded: %s", config.Sanitized())

	// Create the message hub
	hub := NewHub()

	// Start the hub in a goroutine
	go hub.Run()
	log.Println("🔄 Message hub started")

	// Start platform clients
	var twitchClient *TwitchClient
	var kickClient *KickClient

	if config.EnableTwitch {
		twitchClient = startTwitchClient(config, hub)
	}

	if config.EnableKick {
		kickClient = startKickClient(config, hub)
	}

	// Create and start the web server
	server, err := NewServer(hub, config)
	if err != nil {
		log.Fatalf("❌ Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	go handleShutdown(hub, server, twitchClient, kickClient)

	// Start the server (blocking)
	log.Printf("🌐 Starting web server on port %d", config.Port)
	log.Printf("🔗 Open http://localhost:%d in your browser", config.Port)
	log.Printf("📺 For OBS, add as Browser Source: http://localhost:%d?obs=true", config.Port)

	if err := server.Start(); err != nil {
		log.Printf("⚠️ Server stopped: %v", err)
	}
}

// printBanner displays the application banner.
func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   🔀  Multi-Platform Chat Aggregator                      ║
║                                                           ║
║   Combine Twitch and Kick chat in one view                ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	log.Print(banner)
	log.Printf("📌 Version: %s | Built: %s", Version, BuildDate)
}

// startTwitchClient initializes and starts the Twitch client.
func startTwitchClient(config *Config, hub *Hub) *TwitchClient {
	if err := config.ValidateTwitch(); err != nil {
		log.Printf("⚠️ Twitch configuration invalid: %v", err)
		return nil
	}

	client := NewTwitchClient(config, hub)

	go func() {
		// Connect and authenticate
		if err := client.Connect(); err != nil {
			log.Printf("❌ Failed to connect to Twitch: %v", err)
			return
		}

		if err := client.Authenticate(); err != nil {
			log.Printf("❌ Failed to authenticate with Twitch: %v", err)
			return
		}

		// Start reading messages
		client.Run()
	}()

	log.Printf("🟣 Twitch client started for channel: #%s", config.TwitchChannel)
	return client
}

// startKickClient initializes and starts the Kick client.
func startKickClient(config *Config, hub *Hub) *KickClient {
	if err := config.ValidateKick(); err != nil {
		log.Printf("⚠️ Kick configuration invalid: %v", err)
		return nil
	}

	client := NewKickClient(config, hub)

	go func() {
		// Connect (includes getting chatroom ID and subscribing)
		if err := client.Connect(); err != nil {
			log.Printf("❌ Failed to connect to Kick: %v", err)
			return
		}

		// Start reading messages
		client.Run()
	}()

	log.Printf("🟢 Kick client started for channel: %s", config.KickChannel)
	return client
}

// handleShutdown gracefully shuts down all components.
func handleShutdown(hub *Hub, server *Server, twitchClient *TwitchClient, kickClient *KickClient) {
	// Create channel for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	log.Printf("\n🛑 Received signal: %v", sig)
	log.Println("📦 Shutting down gracefully...")

	// Stop clients first
	if twitchClient != nil {
		twitchClient.Stop()
	}
	if kickClient != nil {
		kickClient.Stop()
	}

	// Stop the hub
	hub.Stop()

	// Stop the server
	if err := server.Stop(); err != nil {
		log.Printf("⚠️ Server stop error: %v", err)
	}

	log.Println("✅ Shutdown complete. Goodbye!")
	os.Exit(0)
}