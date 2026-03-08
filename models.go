// Package main provides a multi-platform chat aggregator for Twitch and Kick.
// This file defines the core data structures used throughout the application.
package main

// Platform represents a streaming platform.
// Using a custom type provides type safety and prevents string comparison errors.
type Platform string

const (
	// PlatformTwitch represents Twitch.tv streaming platform.
	// Messages from Twitch will be marked with purple color (#9146FF).
	PlatformTwitch Platform = "twitch"
	
	// PlatformKick represents Kick.com streaming platform.
	// Messages from Kick will be marked with green color (#53FC18).
	PlatformKick Platform = "kick"
)

// ChatMessage represents a single chat message from any platform.
// This is the unified format that all platform-specific messages are converted to.
//
// Security considerations:
// - Content is stored as-is but will be HTML-escaped before rendering
// - Username is sanitized to prevent injection attacks
// - ID is used for message deduplication
type ChatMessage struct {
	// ID is a unique identifier for the message.
	// Format: "<platform>:<original_id>" to ensure global uniqueness.
	ID string `json:"id"`
	
	// Platform indicates which streaming platform this message came from.
	// Used for visual differentiation in the UI.
	Platform Platform `json:"platform"`
	
	// Username is the display name of the user who sent the message.
	// This is HTML-escaped before being sent to clients.
	Username string `json:"username"`
	
	// Content is the actual text of the chat message.
	// This is HTML-escaped before being sent to clients.
	Content string `json:"content"`
	
	// Timestamp is when the message was sent (or received if original timestamp unavailable).
	// Stored in UTC for consistency.
	Timestamp int64 `json:"timestamp"`
	
	// Badges contains the user's badges (e.g., "moderator", "subscriber", "vip").
	// Used for visual indicators in the UI.
	Badges []string `json:"badges,omitempty"`
	
	// Color is the user's chat color (primarily from Twitch).
	// Format: "#RRGGBB" or empty string if not set.
	Color string `json:"color,omitempty"`
}

// IsTwitch returns true if the message is from Twitch platform.
func (m *ChatMessage) IsTwitch() bool {
	return m.Platform == PlatformTwitch
}

// IsKick returns true if the message is from Kick platform.
func (m *ChatMessage) IsKick() bool {
	return m.Platform == PlatformKick
}

// PlatformColor returns the display color for the platform.
// Twitch: Purple (#9146FF)
// Kick: Green (#53FC18)
func (m *ChatMessage) PlatformColor() string {
	switch m.Platform {
	case PlatformTwitch:
		return "#9146FF"
	case PlatformKick:
		return "#53FC18"
	default:
		return "#888888"
	}
}

// PlatformIcon returns the emoji icon for the platform.
// Twitch: 🟣 (purple circle)
// Kick: 🟢 (green circle)
func (m *ChatMessage) PlatformIcon() string {
	switch m.Platform {
	case PlatformTwitch:
		return "🟣"
	case PlatformKick:
		return "🟢"
	default:
		return "⚪"
	}
}

// PlatformName returns the human-readable platform name.
func (m *ChatMessage) PlatformName() string {
	switch m.Platform {
	case PlatformTwitch:
		return "TWITCH"
	case PlatformKick:
		return "KICK"
	default:
		return "UNKNOWN"
	}
}
