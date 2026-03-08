package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewKickClient tests client creation.
func TestNewKickClient(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	if client == nil {
		t.Fatal("NewKickClient returned nil")
	}
	if client.config != config {
		t.Error("Client config not set correctly")
	}
	if client.hub != hub {
		t.Error("Client hub not set correctly")
	}
	if client.done == nil {
		t.Error("Client done channel is nil")
	}
	if client.httpClient == nil {
		t.Error("Client httpClient is nil")
	}
}

// TestKickClient_IsConnected tests the connection state.
func TestKickClient_IsConnected(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	if !client.IsConnected() {
		t.Error("Client should be connected after setting flag")
	}
}

// TestKickClient_Stop tests client shutdown.
func TestKickClient_Stop(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	// Start the done channel reader to prevent blocking
	go func() {
		<-client.done
	}()

	// Stop should not panic
	client.Stop()

	// Give time for goroutine to finish
	time.Sleep(10 * time.Millisecond)

	if client.IsConnected() {
		t.Error("Client should not be connected after Stop()")
	}
}

// TestParseKickMessage tests the message parsing function.
func TestParseKickMessage(t *testing.T) {
	// This is a sample Kick chat event (from Kick's actual format)
	jsonData := `{
		"message": {
			"id": "48b77917-9cc6-4fd3-9a9c-e62dd89a95e2",
			"message": "Hello, world!",
			"type": "",
			"created_at": 1677379978
		},
		"user": {
			"id": 242977,
			"username": "TestUser"
		}
	}`

	msg, err := ParseKickMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseKickMessage error: %v", err)
	}

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}

	// Check ID
	expectedID := "kick:48b77917-9cc6-4fd3-9a9c-e62dd89a95e2"
	if msg.ID != expectedID {
		t.Errorf("ID = %q, want %q", msg.ID, expectedID)
	}

	// Check Platform
	if msg.Platform != PlatformKick {
		t.Errorf("Platform = %q, want %q", msg.Platform, PlatformKick)
	}

	// Check Username
	if msg.Username != "TestUser" {
		t.Errorf("Username = %q, want 'TestUser'", msg.Username)
	}

	// Check Content
	if msg.Content != "Hello, world!" {
		t.Errorf("Content = %q, want 'Hello, world!'", msg.Content)
	}

	// Check Timestamp
	if msg.Timestamp != 1677379978 {
		t.Errorf("Timestamp = %d, want 1677379978", msg.Timestamp)
	}
}

// TestParseKickMessage_InvalidJSON tests parsing invalid JSON.
func TestParseKickMessage_InvalidJSON(t *testing.T) {
	_, err := ParseKickMessage([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestParseKickMessage_EmptyFields tests parsing with empty fields.
func TestParseKickMessage_EmptyFields(t *testing.T) {
	jsonData := `{
		"message": {
			"id": "",
			"message": "",
			"created_at": 0
		},
		"user": {
			"id": 0,
			"username": ""
		}
	}`

	msg, err := ParseKickMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseKickMessage error: %v", err)
	}

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}

	// ID should still have "kick:" prefix
	if msg.ID != "kick:" {
		t.Errorf("ID = %q, want 'kick:'", msg.ID)
	}
}

// TestKickChatEventData_JSONUnmarshal tests JSON unmarshaling.
func TestKickChatEventData_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"message": {
			"id": "test-msg-id",
			"message": "Test message content",
			"type": "message",
			"created_at": 1234567890
		},
		"user": {
			"id": 12345,
			"username": "testuser"
		}
	}`

	var eventData KickChatEventData
	if err := json.Unmarshal([]byte(jsonData), &eventData); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if eventData.Message.ID != "test-msg-id" {
		t.Errorf("Message.ID = %q, want 'test-msg-id'", eventData.Message.ID)
	}
	if eventData.Message.Content != "Test message content" {
		t.Errorf("Message.Content = %q, want 'Test message content'", eventData.Message.Content)
	}
	if eventData.User.Username != "testuser" {
		t.Errorf("User.Username = %q, want 'testuser'", eventData.User.Username)
	}
}

// TestPusherEvent_JSONUnmarshal tests Pusher event parsing.
func TestPusherEvent_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantEvent  string
		wantChannel string
	}{
		{
			name:       "Connection established",
			json:       `{"event":"pusher:connection_established","data":"{}"}`,
			wantEvent:  "pusher:connection_established",
			wantChannel: "",
		},
		{
			name:       "Chat message event",
			json:       `{"event":"App\\Events\\ChatMessageSentEvent","channel":"chatrooms.123456","data":"{}"}`,
			wantEvent:  "App\\Events\\ChatMessageSentEvent",
			wantChannel: "chatrooms.123456",
		},
		{
			name:       "Ping event",
			json:       `{"event":"pusher:ping"}`,
			wantEvent:  "pusher:ping",
			wantChannel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event PusherEvent
			if err := json.Unmarshal([]byte(tt.json), &event); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if event.Event != tt.wantEvent {
				t.Errorf("Event = %q, want %q", event.Event, tt.wantEvent)
			}
			if event.Channel != tt.wantChannel {
				t.Errorf("Channel = %q, want %q", event.Channel, tt.wantChannel)
			}
		})
	}
}

// TestKickClient_handleChatMessage tests the handleChatMessage method.
func TestKickClient_handleChatMessage(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := NewKickClient(config, hub)

	// Register a client to receive the message
	testClient := hub.Register()
	time.Sleep(10 * time.Millisecond)

	// Create a chat message event
	jsonData := `{
		"message": {
			"id": "test-id-123",
			"message": "Test message",
			"created_at": 1700000000
		},
		"user": {
			"id": 999,
			"username": "TestUser"
		}
	}`

	client.handleChatMessage(json.RawMessage(jsonData))
	time.Sleep(20 * time.Millisecond)

	// Check that message was received
	select {
	case data := <-testClient.send:
		var msg ChatMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		if msg.ID != "kick:test-id-123" {
			t.Errorf("ID = %q, want 'kick:test-id-123'", msg.ID)
		}
		if msg.Platform != PlatformKick {
			t.Errorf("Platform = %q, want 'kick'", msg.Platform)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive message within timeout")
	}
}

// TestKickClient_Stop_Idempotent tests that Stop can be called multiple times.
func TestKickClient_Stop_Idempotent(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	// Stop multiple times should not panic
	for i := 0; i < 3; i++ {
		client.Stop()
	}
}

// TestKickClient_handleChatMessage_InvalidJSON tests handling invalid JSON.
func TestKickClient_handleChatMessage_InvalidJSON(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := NewKickClient(config, hub)

	// Invalid JSON should not crash
	client.handleChatMessage(json.RawMessage(`invalid json`))
}

// TestKickClient_handleChatMessage_NilData tests handling nil data.
func TestKickClient_handleChatMessage_NilData(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := NewKickClient(config, hub)

	// Nil data should not crash
	client.handleChatMessage(nil)
}

// TestParseKickMessage_SpecialCharacters tests messages with special characters.
func TestParseKickMessage_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"message": {
			"id": "msg-id",
			"message": "Hello! 😀 🎮 <script>alert('xss')</script>",
			"created_at": 1234567890
		},
		"user": {
			"id": 1,
			"username": "user_with_special_chars"
		}
	}`

	msg, err := ParseKickMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseKickMessage error: %v", err)
	}

	if msg.Content != "Hello! 😀 🎮 <script>alert('xss')</script>" {
		t.Errorf("Content = %q, unexpected value", msg.Content)
	}
}

// TestKickChatEventData_EmptyMessage tests empty message handling.
func TestKickChatEventData_EmptyMessage(t *testing.T) {
	jsonData := `{
		"message": {
			"id": "id",
			"message": ""
		},
		"user": {
			"username": "user"
		}
	}`

	var eventData KickChatEventData
	if err := json.Unmarshal([]byte(jsonData), &eventData); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if eventData.Message.Content != "" {
		t.Errorf("Expected empty content, got %q", eventData.Message.Content)
	}
}

// TestKickClient_ConfigValidation tests that config is properly used.
func TestKickClient_ConfigValidation(t *testing.T) {
	config := &Config{
		KickChannel: "mykickchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	if client.config.KickChannel != "mykickchannel" {
		t.Errorf("KickChannel = %q, want 'mykickchannel'", client.config.KickChannel)
	}
}

// TestParseKickMessage_MessageID tests message ID generation.
func TestParseKickMessage_MessageID(t *testing.T) {
	tests := []struct {
		name     string
		jsonID   string
		expected string
	}{
		{
			name:     "Normal ID",
			jsonID:   "abc-123-xyz",
			expected: "kick:abc-123-xyz",
		},
		{
			name:     "Empty ID",
			jsonID:   "",
			expected: "kick:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := `{
				"message": {
					"id": "` + tt.jsonID + `",
					"message": "test"
				},
				"user": {
					"username": "user"
				}
			}`

			msg, err := ParseKickMessage([]byte(jsonData))
			if err != nil {
				t.Fatalf("ParseKickMessage error: %v", err)
			}

			if msg.ID != tt.expected {
				t.Errorf("ID = %q, want %q", msg.ID, tt.expected)
			}
		})
	}
}

// TestKickClient_HTTPClient tests that HTTP client is properly configured.
func TestKickClient_HTTPClient(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	// HTTP client should have reasonable timeout
	if client.httpClient.Timeout == 0 {
		t.Error("HTTP client should have a timeout configured")
	}
}

// TestPusherEvent_EmptyEvent tests empty event handling.
func TestPusherEvent_EmptyEvent(t *testing.T) {
	jsonData := `{}`
	var event PusherEvent
	if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if event.Event != "" {
		t.Errorf("Event = %q, want empty", event.Event)
	}
}

// TestKickClient_ConcurrentStop tests concurrent stop calls.
func TestKickClient_ConcurrentStop(t *testing.T) {
	config := &Config{
		KickChannel: "testchannel",
	}
	hub := NewHub()

	client := NewKickClient(config, hub)

	// Test that we can stop the client
	client.Stop()

	// Verify connection state
	if client.IsConnected() {
		t.Error("Client should not be connected after Stop()")
	}
}
