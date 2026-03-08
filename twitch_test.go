package main

import (
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
