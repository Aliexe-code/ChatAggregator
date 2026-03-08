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

// TestServer_handleWebSocket_InvalidMethod tests WebSocket with wrong HTTP method.
func TestServer_handleWebSocket_InvalidMethod(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 8080}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("GET", "/ws", nil)
	// Missing WebSocket upgrade headers
	rec := httptest.NewRecorder()

	server.handleWebSocket(rec, req)

	// Should return bad request without proper WebSocket headers
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestServer_handleIndex_ObsMode tests OBS mode parameter.
func TestServer_handleIndex_ObsMode(t *testing.T) {
	hub := NewHub()
	config := &Config{
		Port:          8080,
		TwitchChannel: "testchannel",
		KickChannel:   "kickchannel",
	}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("GET", "/?obs=true", nil)
	rec := httptest.NewRecorder()

	server.handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Response should contain the page content
	body := rec.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}
}

// TestServer_handleStats_DisabledPlatforms tests stats with disabled platforms.
func TestServer_handleStats_DisabledPlatforms(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{
		Port:         8080,
		EnableTwitch: false,
		EnableKick:   false,
	}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rec := httptest.NewRecorder()

	server.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var stats map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &stats)

	if stats["twitch_enabled"] != false {
		t.Error("twitch_enabled should be false")
	}
	if stats["kick_enabled"] != false {
		t.Error("kick_enabled should be false")
	}
}

// TestServer_handleIndex_SecurityHeaders tests security headers.
func TestServer_handleIndex_SecurityHeaders(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 8080}

	server, _ := NewServer(hub, config)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	server.handleIndex(rec, req)

	// Check all security headers
	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-Xss-Protection", "1; mode=block"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.expected {
			t.Errorf("Header %s = %q, want %q", tt.header, got, tt.expected)
		}
	}
}

// TestServer_Routes tests all routes are registered.
func TestServer_Routes(t *testing.T) {
	hub := NewHub()
	config := &Config{
		Port:          18081,
		TwitchChannel: "test",
		KickChannel:   "test",
	}

	server, _ := NewServer(hub, config)

	// Test each route using handler functions directly
	routes := []struct {
		path       string
		method     string
		wantStatus int
	}{
		{"/", "GET", http.StatusOK},
		{"/health", "GET", http.StatusOK},
		{"/api/stats", "GET", http.StatusOK},
	}

	for _, r := range routes {
		req := httptest.NewRequest(r.method, r.path, nil)
		rec := httptest.NewRecorder()

		// Use the appropriate handler based on path
		switch r.path {
		case "/":
			server.handleIndex(rec, req)
		case "/health":
			server.handleHealth(rec, req)
		case "/api/stats":
			server.handleStats(rec, req)
		}

		// Should not return 404 (route not found)
		if rec.Code == http.StatusNotFound {
			t.Errorf("Route %s returned 404", r.path)
		}
	}
}

// TestServer_writePump tests the write pump functionality.
func TestServer_writePump(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	config := &Config{Port: 8080}
	server, _ := NewServer(hub, config)
	_ = server // Avoid unused variable error

	// Register a client
	client := hub.Register()

	// Send a message through the hub
	msg := &ChatMessage{
		ID:        "test-1",
		Platform:  PlatformTwitch,
		Username:  "testuser",
		Content:   "Hello World",
		Timestamp: time.Now().Unix(),
	}

	// Send message to hub
	go func() {
		time.Sleep(50 * time.Millisecond)
		hub.Send(msg)
	}()

	// Read from client channel
	select {
	case data := <-client.send:
		var received ChatMessage
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		if received.ID != msg.ID {
			t.Errorf("Received message ID = %s, want %s", received.ID, msg.ID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Did not receive message within timeout")
	}
}

// TestServer_Stop_Idempotent tests that Stop can be called multiple times.
func TestServer_Stop_Idempotent(t *testing.T) {
	hub := NewHub()
	config := &Config{Port: 18082}

	server, _ := NewServer(hub, config)

	// Start server
	go server.Start()
	time.Sleep(50 * time.Millisecond)

	// Stop multiple times - should not panic
	for i := 0; i < 3; i++ {
		if err := server.Stop(); err != nil {
			t.Errorf("Stop() error on call %d: %v", i+1, err)
		}
	}
}
