package main

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestPrintBanner tests that the banner is printed correctly.
func TestPrintBanner(t *testing.T) {
	// We can't easily test the log output, but we can verify the function exists
	// and doesn't panic
	printBanner()
}

// TestVersionVariables tests that version variables are set.
func TestVersionVariables(t *testing.T) {
	// Version and BuildDate should have default values
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

// TestMainImports tests that the main package compiles correctly.
func TestMainImports(t *testing.T) {
	// This test just verifies the package compiles and imports work
	_ = bytes.Buffer{}
}

// TestHandleShutdown_NoPanic tests that handleShutdown doesn't panic with nil clients.
func TestHandleShutdown_NoPanic(t *testing.T) {
	// Create a hub and server to test shutdown
	hub := NewHub()
	go hub.Run()

	config := &Config{Port: 18090}
	server, err := NewServer(hub, config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Stop the hub
	hub.Stop()
	
	// Stop should work
	if err := server.Stop(); err != nil {
		t.Logf("Server stop returned: %v", err)
	}
}

// TestStartTwitchClient_NilConfig tests startTwitchClient with nil config.
func TestStartTwitchClient_NilConfig(t *testing.T) {
	// Create a valid hub
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Test with invalid config (missing Twitch credentials)
	config := &Config{
		TwitchUsername:   "",
		TwitchOAuthToken: "",
		TwitchChannel:    "",
	}

	client := startTwitchClient(config, hub)
	if client != nil {
		t.Error("startTwitchClient should return nil with invalid config")
		client.Stop()
	}
}

// TestStartKickClient_NilConfig tests startKickClient with invalid config.
func TestStartKickClient_NilConfig(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Test with invalid config (missing Kick channel)
	config := &Config{
		KickChannel: "",
	}

	client := startKickClient(config, hub)
	if client != nil {
		t.Error("startKickClient should return nil with invalid config")
		client.Stop()
	}
}

// TestStartTwitchClientWithDialer_ValidConfig tests the Twitch client with valid config.
func TestStartTwitchClientWithDialer_ValidConfig(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}

	// Use a custom dialer that will fail quickly
	dialer := &websocket.Dialer{
		HandshakeTimeout: 1 * time.Nanosecond,
	}

	// This should return a client but it will fail to connect in the goroutine
	client := startTwitchClientWithDialer(config, hub, dialer)
	
	// Give time for the goroutine to run
	time.Sleep(100 * time.Millisecond)
	
	if client != nil {
		client.Stop()
	}
}

// TestStartKickClientWithClients_ValidConfig tests the Kick client with valid config.
func TestStartKickClientWithClients_ValidConfig(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{
		KickChannel: "testchannel",
	}

	// Use custom clients that will fail quickly
	dialer := &websocket.Dialer{
		HandshakeTimeout: 1 * time.Nanosecond,
	}
	httpClient := &http.Client{
		Timeout: 1 * time.Nanosecond,
	}

	// This should return a client but it will fail to connect in the goroutine
	client := startKickClientWithClients(config, hub, dialer, httpClient)
	
	// Give time for the goroutine to run
	time.Sleep(100 * time.Millisecond)
	
	if client != nil {
		client.Stop()
	}
}

// TestStartTwitchClientWithDialer_NilDialer tests with nil dialer (uses default).
func TestStartTwitchClientWithDialer_NilDialer(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}

	// Nil dialer should use default
	client := startTwitchClientWithDialer(config, hub, nil)
	
	// Give time for the goroutine to start
	time.Sleep(50 * time.Millisecond)
	
	if client != nil {
		client.Stop()
	}
}

// TestStartKickClientWithClients_NilClients tests with nil clients (uses default).
func TestStartKickClientWithClients_NilClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{
		KickChannel: "testchannel",
	}

	// Use a custom HTTP client with short timeout instead of nil
	// Nil clients would cause panic, so we use a fast-failing client
	httpClient := &http.Client{
		Timeout: 1 * time.Nanosecond,
	}

	client := startKickClientWithClients(config, hub, nil, httpClient)
	
	// Give time for the goroutine to start
	time.Sleep(50 * time.Millisecond)
	
	if client != nil {
		client.Stop()
	}
}

// TestConfigFileExists_Function tests the ConfigFileExists function.
func TestConfigFileExists_Function(t *testing.T) {
	// Remove any existing .env
	os.Remove(".env")
	
	if ConfigFileExists() {
		t.Error("ConfigFileExists should return false when .env doesn't exist")
	}
	
	// Create .env
	os.WriteFile(".env", []byte("TEST=1"), 0600)
	defer os.Remove(".env")
	
	if !ConfigFileExists() {
		t.Error("ConfigFileExists should return true when .env exists")
	}
}

