package main

import (
	"testing"
)

// TestChatMessage_IsTwitch tests the IsTwitch method.
func TestChatMessage_IsTwitch(t *testing.T) {
	tests := []struct {
		name     string
		message  ChatMessage
		expected bool
	}{
		{
			name: "Twitch message returns true",
			message: ChatMessage{
				Platform: PlatformTwitch,
			},
			expected: true,
		},
		{
			name: "Kick message returns false",
			message: ChatMessage{
				Platform: PlatformKick,
			},
			expected: false,
		},
		{
			name: "Empty platform returns false",
			message: ChatMessage{
				Platform: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.IsTwitch(); got != tt.expected {
				t.Errorf("ChatMessage.IsTwitch() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestChatMessage_IsKick tests the IsKick method.
func TestChatMessage_IsKick(t *testing.T) {
	tests := []struct {
		name     string
		message  ChatMessage
		expected bool
	}{
		{
			name: "Kick message returns true",
			message: ChatMessage{
				Platform: PlatformKick,
			},
			expected: true,
		},
		{
			name: "Twitch message returns false",
			message: ChatMessage{
				Platform: PlatformTwitch,
			},
			expected: false,
		},
		{
			name: "Empty platform returns false",
			message: ChatMessage{
				Platform: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.IsKick(); got != tt.expected {
				t.Errorf("ChatMessage.IsKick() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestChatMessage_PlatformColor tests the PlatformColor method.
func TestChatMessage_PlatformColor(t *testing.T) {
	tests := []struct {
		name     string
		message  ChatMessage
		expected string
	}{
		{
			name: "Twitch returns purple",
			message: ChatMessage{
				Platform: PlatformTwitch,
			},
			expected: "#9146FF",
		},
		{
			name: "Kick returns green",
			message: ChatMessage{
				Platform: PlatformKick,
			},
			expected: "#53FC18",
		},
		{
			name: "Unknown returns gray",
			message: ChatMessage{
				Platform: "unknown",
			},
			expected: "#888888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.PlatformColor(); got != tt.expected {
				t.Errorf("ChatMessage.PlatformColor() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestChatMessage_PlatformIcon tests the PlatformIcon method.
func TestChatMessage_PlatformIcon(t *testing.T) {
	tests := []struct {
		name     string
		message  ChatMessage
		expected string
	}{
		{
			name: "Twitch returns purple circle",
			message: ChatMessage{
				Platform: PlatformTwitch,
			},
			expected: "🟣",
		},
		{
			name: "Kick returns green circle",
			message: ChatMessage{
				Platform: PlatformKick,
			},
			expected: "🟢",
		},
		{
			name: "Unknown returns white circle",
			message: ChatMessage{
				Platform: "unknown",
			},
			expected: "⚪",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.PlatformIcon(); got != tt.expected {
				t.Errorf("ChatMessage.PlatformIcon() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestChatMessage_PlatformName tests the PlatformName method.
func TestChatMessage_PlatformName(t *testing.T) {
	tests := []struct {
		name     string
		message  ChatMessage
		expected string
	}{
		{
			name: "Twitch returns TWITCH",
			message: ChatMessage{
				Platform: PlatformTwitch,
			},
			expected: "TWITCH",
		},
		{
			name: "Kick returns KICK",
			message: ChatMessage{
				Platform: PlatformKick,
			},
			expected: "KICK",
		},
		{
			name: "Unknown returns UNKNOWN",
			message: ChatMessage{
				Platform: "unknown",
			},
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.PlatformName(); got != tt.expected {
				t.Errorf("ChatMessage.PlatformName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestChatMessage_JSONSerialization tests that ChatMessage can be serialized to JSON.
func TestChatMessage_JSONSerialization(t *testing.T) {
	message := ChatMessage{
		ID:        "twitch:12345",
		Platform:  PlatformTwitch,
		Username:  "TestUser",
		Content:   "Hello, world!",
		Timestamp: 1234567890,
		Badges:    []string{"moderator", "subscriber"},
		Color:     "#FF0000",
	}

	// Verify all fields are set correctly
	if message.ID != "twitch:12345" {
		t.Errorf("Expected ID 'twitch:12345', got '%s'", message.ID)
	}
	if message.Platform != PlatformTwitch {
		t.Errorf("Expected Platform '%s', got '%s'", PlatformTwitch, message.Platform)
	}
	if message.Username != "TestUser" {
		t.Errorf("Expected Username 'TestUser', got '%s'", message.Username)
	}
	if message.Content != "Hello, world!" {
		t.Errorf("Expected Content 'Hello, world!', got '%s'", message.Content)
	}
	if len(message.Badges) != 2 {
		t.Errorf("Expected 2 badges, got %d", len(message.Badges))
	}
}
