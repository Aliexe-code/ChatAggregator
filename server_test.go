package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewServer tests server creation.
func TestNewServer(t *testing.T) {
	hub := NewHub()
	config := &Config{
		Port:          8080,
		TwitchChannel: "testchannel",
		KickChannel:   "testchannel",
	}

	server, err := NewServer(hub, config)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.hub != hub {
		t.Error("Server hub not set correctly")
	}
	if server.config != config {
		t.Error("Server config not set correctly")
	}
}

// TestServer_handleIndex tests the index page handler.
func TestServer_handleIndex(t *testing.T) {
	hub := NewHub()
	config := &Config{
		Port:          8080,
		TwitchChannel: "testchannel",
		KickChannel:   "kickchannel",
	}

	server, err := NewServer(hub, config)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	server.handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Check content type
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	// Check security headers
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Missing X-Content-Type-Options header")
	}
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("Missing X-Frame-Options header")
	}
}

// TestServer_handleIndex_MethodNotAllowed tests POST rejection.
func TestServer_handleIndex_MethodNotAllowed(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 8080}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()

	server.handleIndex(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestServer_handleStats tests the stats endpoint.
func TestServer_handleStats(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{
		Port:          8080,
		TwitchChannel: "twitchchan",
		KickChannel:   "kickchan",
		EnableTwitch:  true,
		EnableKick:    true,
	}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rec := httptest.NewRecorder()

	server.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Check content type
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	// Parse response
	var stats map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to parse stats: %v", err)
	}

	// Check expected fields
	if stats["twitch_channel"] != "twitchchan" {
		t.Errorf("twitch_channel = %v, want 'twitchchan'", stats["twitch_channel"])
	}
	if stats["kick_channel"] != "kickchan" {
		t.Errorf("kick_channel = %v, want 'kickchan'", stats["kick_channel"])
	}
}

// TestServer_handleHealth tests the health endpoint.
func TestServer_handleHealth(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 8080}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Parse response
	var health map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("Failed to parse health: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("status = %v, want 'healthy'", health["status"])
	}
}

// TestServer_handleHealth_MethodNotAllowed tests POST rejection.
func TestServer_handleHealth_MethodNotAllowed(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 8080}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("POST", "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestServer_handleStats_MethodNotAllowed tests POST rejection.
func TestServer_handleStats_MethodNotAllowed(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 8080}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("DELETE", "/api/stats", nil)
	rec := httptest.NewRecorder()

	server.handleStats(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestServer_StartStop tests server lifecycle.
func TestServer_StartStop(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 18080} // Use non-standard port to avoid conflicts

	server, err := NewServer(hub, config)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Start server in goroutine
	go func() {
		server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is running
	resp, err := http.Get("http://localhost:18080/health")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Server stop error: %v", err)
	}
}
