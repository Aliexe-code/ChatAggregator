package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConn is an interface for WebSocket connections.
// This allows mocking in tests without requiring real network connections.
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// websocketConnWrapper wraps gorilla/websocket.Conn to implement WebSocketConn.
type websocketConnWrapper struct {
	*websocket.Conn
}

func (w *websocketConnWrapper) ReadMessage() (int, []byte, error) {
	return w.Conn.ReadMessage()
}

func (w *websocketConnWrapper) WriteMessage(messageType int, data []byte) error {
	return w.Conn.WriteMessage(messageType, data)
}

func (w *websocketConnWrapper) Close() error {
	return w.Conn.Close()
}

// TwitchClient connects to Twitch IRC and receives chat messages.
// It uses Twitch's IRC-over-WebSocket interface.
//
// Connection Flow:
// 1. Connect to wss://irc-ws.chat.twitch.tv:443
// 2. Request capabilities (tags, commands, membership)
// 3. Authenticate with PASS and NICK
// 4. JOIN the specified channel
// 5. Listen for PRIVMSG messages
// 6. Respond to PING with PONG
//
// Security:
// - OAuth token is sent only to Twitch servers
// - Messages are sanitized before broadcasting
// - Connection uses TLS (wss://)
const (
	// TwitchIRCAddress is the Twitch IRC WebSocket endpoint.
	TwitchIRCAddress = "irc-ws.chat.twitch.tv:443"

	// TwitchIRCScheme is the WebSocket scheme (secure).
	TwitchIRCScheme = "wss"

	// TwitchReconnectDelay is the delay before reconnecting.
	TwitchReconnectDelay = 5 * time.Second
)

// TwitchClient manages the connection to Twitch IRC.
type TwitchClient struct {
	// config holds the Twitch configuration.
	config *Config

	// hub is the message hub for broadcasting messages.
	hub *Hub

	// conn is the WebSocket connection (interface for mocking).
	conn WebSocketConn

	// done signals the client to stop.
	done chan struct{}

	// mu protects connection state.
	mu sync.Mutex

	// connected indicates if we're currently connected.
	connected bool

	// dialer is the WebSocket dialer (can be mocked for testing)
	dialer *websocket.Dialer
}

// NewTwitchClient creates a new Twitch IRC client.
func NewTwitchClient(config *Config, hub *Hub) *TwitchClient {
	return &TwitchClient{
		config:  config,
		hub:     hub,
		done:    make(chan struct{}),
		dialer:  websocket.DefaultDialer,
	}
}

// NewTwitchClientWithDialer creates a new Twitch IRC client with a custom dialer.
// This is used for testing with mock WebSocket connections.
func NewTwitchClientWithDialer(config *Config, hub *Hub, dialer *websocket.Dialer) *TwitchClient {
	return &TwitchClient{
		config:  config,
		hub:     hub,
		done:    make(chan struct{}),
		dialer:  dialer,
	}
}

// NewTwitchClientWithConn creates a Twitch IRC client with a pre-existing connection.
// This is used for testing with mock WebSocket connections.
func NewTwitchClientWithConn(config *Config, hub *Hub, conn WebSocketConn) *TwitchClient {
	return &TwitchClient{
		config:    config,
		hub:       hub,
		done:      make(chan struct{}),
		conn:      conn,
		connected: true,
		dialer:    websocket.DefaultDialer,
	}
}

// Connect establishes the connection to Twitch IRC.
func (c *TwitchClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	u := url.URL{Scheme: TwitchIRCScheme, Host: TwitchIRCAddress}
	log.Printf("🔌 Connecting to Twitch IRC: %s", u.String())

	conn, _, err := c.dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Twitch IRC: %w", err)
	}

	c.conn = &websocketConnWrapper{Conn: conn}
	c.connected = true
	log.Println("✅ Connected to Twitch IRC")

	return nil
}

// Authenticate sends authentication and joins the channel.
func (c *TwitchClient) Authenticate() error {
	// Step 1: Request capabilities for rich message data
	// - twitch.tv/tags: Adds metadata (badges, colors, etc.)
	// - twitch.tv/commands: Enables Twitch-specific commands
	// - twitch.tv/membership: Enables JOIN/PART messages
	capReq := "CAP REQ :twitch.tv/tags twitch.tv/commands twitch.tv/membership"
	if err := c.send(capReq); err != nil {
		return fmt.Errorf("failed to request capabilities: %w", err)
	}

	// Step 2: Send authentication
	// PASS must be sent before NICK
	passCmd := fmt.Sprintf("PASS %s", c.config.TwitchOAuthToken)
	if err := c.send(passCmd); err != nil {
		return fmt.Errorf("failed to send PASS: %w", err)
	}

	nickCmd := fmt.Sprintf("NICK %s", strings.ToLower(c.config.TwitchUsername))
	if err := c.send(nickCmd); err != nil {
		return fmt.Errorf("failed to send NICK: %w", err)
	}

	// Step 3: Join the channel
	// Channel name must be lowercase and prefixed with #
	joinCmd := fmt.Sprintf("JOIN #%s", strings.ToLower(c.config.TwitchChannel))
	if err := c.send(joinCmd); err != nil {
		return fmt.Errorf("failed to JOIN channel: %w", err)
	}

	log.Printf("📺 Joined Twitch channel: #%s", c.config.TwitchChannel)
	return nil
}

// Run starts the message reading loop.
// This should be called after Connect and Authenticate.
func (c *TwitchClient) Run() {
	for {
		if !c.runIteration() {
			return
		}
	}
}

// runIteration performs one iteration of the message reading loop.
// Returns false if the client should stop, true to continue.
func (c *TwitchClient) runIteration() bool {
	select {
	case <-c.done:
		return false
	default:
		if err := c.readMessages(); err != nil {
			log.Printf("❌ Twitch read error: %v", err)
			c.handleReconnect()
		}
		return true
	}
}

// Stop gracefully stops the client.
// Safe to call multiple times.
func (c *TwitchClient) Stop() {
	select {
	case <-c.done:
		// Already closed
		return
	default:
		close(c.done)
	}
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connected = false
	c.mu.Unlock()
	log.Println("🛑 Twitch client stopped")
}

// IsConnected returns the current connection state.
func (c *TwitchClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// send sends a raw IRC command.
func (c *TwitchClient) send(cmd string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(cmd+"\r\n"))
}

// readMessages reads and processes incoming messages.
func (c *TwitchClient) readMessages() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	// Twitch may send multiple IRC messages in one WebSocket frame
	// Messages are separated by \r\n
	lines := strings.Split(string(message), "\r\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		c.handleLine(line)
	}

	return nil
}

// handleLine processes a single IRC line.
func (c *TwitchClient) handleLine(line string) {
	// Handle PING (keepalive)
	if strings.HasPrefix(line, "PING") {
		c.handlePing(line)
		return
	}

	// Handle PRIVMSG (chat message)
	if strings.Contains(line, "PRIVMSG") {
		msg := parseTwitchMessage(line)
		if msg != nil {
			c.hub.Send(msg)
		}
		return
	}

	// Handle other messages (for logging/debugging)
	if strings.Contains(line, "001") {
		// Welcome message - authentication successful
		log.Println("✅ Twitch authentication successful")
	} else if strings.Contains(line, "NOTICE") {
		// Notice messages (errors, warnings)
		log.Printf("📢 Twitch NOTICE: %s", line)
	}
}

// handlePing responds to Twitch keepalive PING.
func (c *TwitchClient) handlePing(line string) {
	// PING format: PING :tmi.twitch.tv
	// PONG format: PONG :tmi.twitch.tv
	pong := strings.Replace(line, "PING", "PONG", 1)
	if err := c.send(pong); err != nil {
		log.Printf("❌ Failed to send PONG: %v", err)
	}
}

// handleReconnect handles reconnection logic.
func (c *TwitchClient) handleReconnect() {
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	log.Printf("🔄 Reconnecting to Twitch in %v...", TwitchReconnectDelay)
	time.Sleep(TwitchReconnectDelay)

	for {
		select {
		case <-c.done:
			return
		default:
			if err := c.Connect(); err != nil {
				log.Printf("❌ Twitch reconnect failed: %v", err)
				time.Sleep(TwitchReconnectDelay)
				continue
			}
			if err := c.Authenticate(); err != nil {
				log.Printf("❌ Twitch re-auth failed: %v", err)
				time.Sleep(TwitchReconnectDelay)
				continue
			}
			log.Println("✅ Twitch reconnected successfully")
			return
		}
	}
}

// handleReconnectOnce performs a single reconnection attempt.
// Returns true if should continue trying, false if stopped or succeeded.
// This is extracted for testing.
func (c *TwitchClient) handleReconnectOnce() bool {
	select {
	case <-c.done:
		return false
	default:
		if err := c.Connect(); err != nil {
			log.Printf("❌ Twitch reconnect failed: %v", err)
			return true // Continue trying
		}
		if err := c.Authenticate(); err != nil {
			log.Printf("❌ Twitch re-auth failed: %v", err)
			return true // Continue trying
		}
		log.Println("✅ Twitch reconnected successfully")
		return false // Successfully reconnected, stop
	}
}

// cleanupConnection marks connection as disconnected and closes it.
func (c *TwitchClient) cleanupConnection() {
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

// parseTwitchMessage parses a Twitch IRC PRIVMSG into a ChatMessage.
// Example input:
// @badge-info=;badges=moderator/1;color=#FF0000;display-name=TestUser;emotes=;id=abc123;mod=1;room-id=12345;subscriber=0;turbo=0;user-id=67890;user-type=mod :testuser!testuser@testuser.tmi.twitch.tv PRIVMSG #channel :Hello, world!
func parseTwitchMessage(line string) *ChatMessage {
	// Extract tags (before the first space)
	tags := make(map[string]string)
	if strings.HasPrefix(line, "@") {
		tagEnd := strings.Index(line, " ")
		if tagEnd > 0 {
			tagStr := line[1:tagEnd]
			for _, tag := range strings.Split(tagStr, ";") {
				parts := strings.SplitN(tag, "=", 2)
				if len(parts) == 2 {
					tags[parts[0]] = parts[1]
				}
			}
			line = line[tagEnd+1:]
		}
	}

	// Parse the IRC message using a scanner
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	var parts []string
	for scanner.Scan() {
		parts = append(parts, scanner.Text())
	}

	// Find PRIVMSG index
	privmsgIdx := -1
	for i, p := range parts {
		if p == "PRIVMSG" {
			privmsgIdx = i
			break
		}
	}

	if privmsgIdx == -1 {
		return nil
	}

	// Extract username from prefix (e.g., :username!username@...)
	prefix := parts[0]
	if !strings.HasPrefix(prefix, ":") {
		return nil
	}
	username := prefix[1:]
	if bangIdx := strings.Index(username, "!"); bangIdx > 0 {
		username = username[:bangIdx]
	}

	// Extract message ID from tags
	id := tags["id"]
	if id == "" {
		id = fmt.Sprintf("twitch:%d", time.Now().UnixNano())
	} else {
		id = "twitch:" + id
	}

	// Extract channel (parts[privmsgIdx+1] with # prefix removed)
	channel := parts[privmsgIdx+1]
	channel = strings.TrimPrefix(channel, "#")

	// Extract message content (everything after the channel)
	var content string
	if len(parts) > privmsgIdx+2 {
		content = strings.Join(parts[privmsgIdx+2:], " ")
		content = strings.TrimPrefix(content, ":")
	}

	// Extract badges
	var badges []string
	if badgesStr := tags["badges"]; badgesStr != "" {
		for _, badge := range strings.Split(badgesStr, ",") {
			if badge != "" {
				// Badge format: badge_name/tier (e.g., "moderator/1")
				badgeName := strings.Split(badge, "/")[0]
				if badgeName != "" {
					badges = append(badges, badgeName)
				}
			}
		}
	}

	// Get display name (prefer over username)
	displayName := tags["display-name"]
	if displayName == "" {
		displayName = username
	}

	// Create the message
	return &ChatMessage{
		ID:        id,
		Platform:  PlatformTwitch,
		Username:  displayName,
		Content:   content,
		Timestamp: time.Now().Unix(),
		Badges:    badges,
		Color:     tags["color"],
	}
}

// twitchMessageRegex is used for parsing PRIVMSG lines.
// Matches: @tags :user!user@user.tmi.twitch.tv PRIVMSG #channel :message
var twitchMessageRegex = regexp.MustCompile(
	`^(@[^\s]+ )?:([^!]+)![^\s]+ PRIVMSG #([^\s]+) :(.*)$`)

// parseTwitchMessageRegex parses using regex (alternative method).
func parseTwitchMessageRegex(line string) *ChatMessage {
	matches := twitchMessageRegex.FindStringSubmatch(line)
	if len(matches) < 5 {
		return nil
	}

	username := matches[2]
	content := matches[4]

	return &ChatMessage{
		ID:        fmt.Sprintf("twitch:%d", time.Now().UnixNano()),
		Platform:  PlatformTwitch,
		Username:  username,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
}
