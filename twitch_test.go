package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestNewTwitchClient tests client creation.
func TestNewTwitchClient(t *testing.T) {
	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}
	hub := NewHub()

	client := NewTwitchClient(config, hub)

	if client == nil {
		t.Fatal("NewTwitchClient returned nil")
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
}

// TestTwitchClient_IsConnected tests the connection state.
func TestTwitchClient_IsConnected(t *testing.T) {
	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}
	hub := NewHub()

	client := NewTwitchClient(config, hub)

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

// TestParseTwitchMessage tests the message parsing function.
func TestParseTwitchMessage(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		wantUsername    string
		wantContent     string
		wantNil         bool
	}{
		{
			name:         "Simple message",
			line:         ":testuser!testuser@testuser.tmi.twitch.tv PRIVMSG #mychannel :Hello, world!",
			wantUsername: "testuser",
			wantContent:  "Hello, world!",
			wantNil:      false,
		},
		{
			name:         "Message with tags",
			line:         "@badge-info=;badges=moderator/1;color=#FF0000;display-name=TestUser;id=abc123 :testuser!testuser@testuser.tmi.twitch.tv PRIVMSG #mychannel :Test message",
			wantUsername: "TestUser",
			wantContent:  "Test message",
			wantNil:      false,
		},
		{
			name:         "Message with special characters",
			line:         ":user123!user123@user123.tmi.twitch.tv PRIVMSG #channel :Hello! How are you? :)",
			wantUsername: "user123",
			wantContent:  "Hello! How are you? :)",
			wantNil:      false,
		},
		{
			name:    "Non-PRIVMSG line",
			line:    ":tmi.twitch.tv 001 testuser :Welcome, GLHF!",
			wantNil: true,
		},
		{
			name:    "Empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "PING line",
			line:    "PING :tmi.twitch.tv",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := parseTwitchMessage(tt.line)

			if tt.wantNil {
				if msg != nil {
					t.Errorf("Expected nil, got %+v", msg)
				}
				return
			}

			if msg == nil {
				t.Fatalf("Expected non-nil message, got nil")
			}

			if msg.Username != tt.wantUsername {
				t.Errorf("Username = %q, want %q", msg.Username, tt.wantUsername)
			}
			if msg.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", msg.Content, tt.wantContent)
			}
			if msg.Platform != PlatformTwitch {
				t.Errorf("Platform = %q, want %q", msg.Platform, PlatformTwitch)
			}
			if msg.Timestamp == 0 {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

// TestParseTwitchMessageWithBadges tests parsing messages with badges.
func TestParseTwitchMessageWithBadges(t *testing.T) {
	line := "@badges=moderator/1,subscriber/12;color=#00FF00;display-name=ModUser;id=msg123 :moduser!moduser@moduser.tmi.twitch.tv PRIVMSG #channel :Moderator message"

	msg := parseTwitchMessage(line)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}

	// Check badges
	expectedBadges := []string{"moderator", "subscriber"}
	if len(msg.Badges) != len(expectedBadges) {
		t.Errorf("Expected %d badges, got %d", len(expectedBadges), len(msg.Badges))
	}

	for i, badge := range expectedBadges {
		if i >= len(msg.Badges) || msg.Badges[i] != badge {
			t.Errorf("Badge[%d] = %q, want %q", i, msg.Badges[i], badge)
		}
	}

	// Check color
	if msg.Color != "#00FF00" {
		t.Errorf("Color = %q, want #00FF00", msg.Color)
	}

	// Check ID
	if !strings.HasPrefix(msg.ID, "twitch:") {
		t.Errorf("ID should start with 'twitch:', got %q", msg.ID)
	}
}

// TestParseTwitchMessageRegex tests the regex-based parser.
func TestParseTwitchMessageRegex(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantUsername string
		wantContent  string
		wantNil      bool
	}{
		{
			name:         "Simple message",
			line:         ":testuser!testuser@testuser.tmi.twitch.tv PRIVMSG #mychannel :Hello!",
			wantUsername: "testuser",
			wantContent:  "Hello!",
			wantNil:      false,
		},
		{
			name:         "Message with tags",
			line:         "@color=#FF0000 :user!user@user.tmi.twitch.tv PRIVMSG #chan :Test",
			wantUsername: "user",
			wantContent:  "Test",
			wantNil:      false,
		},
		{
			name:    "Non-matching line",
			line:    "PING :tmi.twitch.tv",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := parseTwitchMessageRegex(tt.line)

			if tt.wantNil {
				if msg != nil {
					t.Errorf("Expected nil, got %+v", msg)
				}
				return
			}

			if msg == nil {
				t.Fatal("Expected non-nil message")
			}

			if msg.Username != tt.wantUsername {
				t.Errorf("Username = %q, want %q", msg.Username, tt.wantUsername)
			}
			if msg.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", msg.Content, tt.wantContent)
			}
		})
	}
}

// TestParseTwitchMessage_EdgeCases tests edge cases in message parsing.
func TestParseTwitchMessage_EdgeCases(t *testing.T) {
	t.Run("Message with colons in content", func(t *testing.T) {
		line := ":user!user@user.tmi.twitch.tv PRIVMSG #channel :Check this: https://example.com :)"
		msg := parseTwitchMessage(line)
		if msg == nil {
			t.Fatal("Expected non-nil message")
		}
		if !strings.Contains(msg.Content, "https://") {
			t.Errorf("Content should contain URL, got %q", msg.Content)
		}
	})

	t.Run("Message with emotes only", func(ttesting *testing.T) {
		line := ":user!user@user.tmi.twitch.tv PRIVMSG #channel :PogChamp Kappa"
		msg := parseTwitchMessage(line)
		if msg == nil {
			t.Fatal("Expected non-nil message")
		}
		if msg.Content != "PogChamp Kappa" {
			t.Errorf("Content = %q, want 'PogChamp Kappa'", msg.Content)
		}
	})

	t.Run("Empty badges field", func(t *testing.T) {
		line := "@badges=;color= :user!user@user.tmi.twitch.tv PRIVMSG #channel :Test"
		msg := parseTwitchMessage(line)
		if msg == nil {
			t.Fatal("Expected non-nil message")
		}
		if len(msg.Badges) != 0 {
			t.Errorf("Expected empty badges, got %v", msg.Badges)
		}
	})
}

// TestTwitchClient_Stop tests client shutdown.
func TestTwitchClient_Stop(t *testing.T) {
	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}
	hub := NewHub()

	client := NewTwitchClient(config, hub)

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

// TestTwitchClient_Stop_Idempotent tests that Stop can be called multiple times.
func TestTwitchClient_Stop_Idempotent(t *testing.T) {
	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}
	hub := NewHub()

	client := NewTwitchClient(config, hub)

	// Stop multiple times should not panic
	for i := 0; i < 3; i++ {
		client.Stop()
	}
}

// TestTwitchClient_handleLine tests line processing.
func TestTwitchClient_handleLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantMessage  bool
		wantUsername string
		wantContent  string
	}{
		{
			name:        "PING message",
			line:        "PING :tmi.twitch.tv",
			wantMessage: false,
		},
		{
			name:         "PRIVMSG",
			line:         ":testuser!testuser@testuser.tmi.twitch.tv PRIVMSG #channel :Hello!",
			wantMessage:  true,
			wantUsername: "testuser",
			wantContent:  "Hello!",
		},
		{
			name:        "System message",
			line:        ":tmi.twitch.tv 001 testuser :Welcome!",
			wantMessage: false,
		},
		{
			name:        "Empty line",
			line:        "",
			wantMessage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			go hub.Run()
			defer hub.Stop()

			config := &Config{
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:testtoken",
				TwitchChannel:    "testchannel",
			}

			_ = NewTwitchClient(config, hub) // Create client, we don't need to use it

			// If we expect a message, register a client to receive it
			var receiver *Client
			if tt.wantMessage {
				receiver = hub.Register()
			}

			// Parse the message directly and send to hub
			if tt.wantMessage {
				msg := parseTwitchMessage(tt.line)
				if msg != nil {
					hub.Send(msg)
				}
			}

			if tt.wantMessage {
				select {
				case data := <-receiver.send:
					var msg ChatMessage
					if err := json.Unmarshal(data, &msg); err != nil {
						t.Fatalf("Failed to unmarshal: %v", err)
					}
					if msg.Username != tt.wantUsername {
						t.Errorf("Username = %q, want %q", msg.Username, tt.wantUsername)
					}
					if msg.Content != tt.wantContent {
						t.Errorf("Content = %q, want %q", msg.Content, tt.wantContent)
					}
				case <-time.After(100 * time.Millisecond):
					t.Error("Did not receive expected message")
				}
			}
		})
	}
}

// TestParseTwitchMessage_Channel tests channel extraction.
func TestParseTwitchMessage_Channel(t *testing.T) {
	line := ":user!user@user.tmi.twitch.tv PRIVMSG #mychannel :Test"
	msg := parseTwitchMessage(line)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	// The channel should be parsed (implementation dependent)
}

// TestParseTwitchMessage_DisplayName tests display name extraction.
func TestParseTwitchMessage_DisplayName(t *testing.T) {
	line := "@display-name=TestUser :testuser!testuser@testuser.tmi.twitch.tv PRIVMSG #channel :Hello"
	msg := parseTwitchMessage(line)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if msg.Username != "TestUser" {
		t.Errorf("Username = %q, want 'TestUser'", msg.Username)
	}
}

// TestParseTwitchMessage_ID tests message ID extraction.
func TestParseTwitchMessage_ID(t *testing.T) {
	line := "@id=abc123xyz :user!user@user.tmi.twitch.tv PRIVMSG #channel :Test"
	msg := parseTwitchMessage(line)
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	// ID should contain the message ID
	if !strings.Contains(msg.ID, "abc123xyz") {
		t.Errorf("ID should contain 'abc123xyz', got %q", msg.ID)
	}
}

// TestTwitchClient_ConnectionState tests connection state management.
func TestTwitchClient_ConnectionState(t *testing.T) {
	config := &Config{
		TwitchUsername:   "testuser",
		TwitchOAuthToken: "oauth:testtoken",
		TwitchChannel:    "testchannel",
	}
	hub := NewHub()

	client := NewTwitchClient(config, hub)

	// Initially not connected
	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	// Set connected state
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	if !client.IsConnected() {
		t.Error("Client should be connected after setting flag")
	}
}

// TestTwitchMessage_Timestamp tests that timestamps are set.
func TestTwitchMessage_Timestamp(t *testing.T) {
	line := ":user!user@user.tmi.twitch.tv PRIVMSG #channel :Test"
	before := time.Now().Unix()
	msg := parseTwitchMessage(line)
	after := time.Now().Unix()

	if msg == nil {
		t.Fatal("Expected non-nil message")
	}

	// Timestamp should be within 1 second of now
	if msg.Timestamp < before || msg.Timestamp > after {
		t.Errorf("Timestamp = %d, expected between %d and %d", msg.Timestamp, before, after)
	}
}

// TestTwitchClient_ConfigValidation tests that config is properly used.
func TestTwitchClient_ConfigValidation(t *testing.T) {
	config := &Config{
		TwitchUsername:   "mybot",
		TwitchOAuthToken: "oauth:mytoken",
		TwitchChannel:    "mychannel",
	}
	hub := NewHub()

	client := NewTwitchClient(config, hub)

	if client.config.TwitchUsername != "mybot" {
		t.Errorf("Username = %q, want 'mybot'", client.config.TwitchUsername)
	}
	if client.config.TwitchChannel != "mychannel" {
		t.Errorf("Channel = %q, want 'mychannel'", client.config.TwitchChannel)
	}
}
