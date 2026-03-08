package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// KickClient connects to Kick chat via Pusher WebSocket.
//
// Connection Flow:
// 1. Fetch chatroom_id from Kick API (GET /api/v1/channels/{username})
// 2. Connect to Pusher WebSocket (ws-us*.pusher.com)
// 3. Subscribe to chatroom channel: chatrooms.{chatroom_id}
// 4. Listen for ChatMessageSentEvent events
//
// Security:
// - No authentication required for reading (public chat)
// - Connection uses TLS (wss://)
// - Messages are sanitized before broadcasting
const (
	// KickAPIBase is the base URL for Kick API.
	KickAPIBase = "https://kick.com"

	// KickChannelEndpoint is the API endpoint for channel info.
	KickChannelEndpoint = "/api/v1/channels/"

	// PusherAppKey is Kick's Pusher application key.
	// This is public and embedded in Kick's frontend.
	PusherAppKey = "eb1d5f283081a78b932c"

	// PusherHost is the Pusher WebSocket host.
	PusherHost = "ws-us1.pusher.com"

	// PusherPath is the WebSocket path.
	PusherPath = "/app/" + PusherAppKey + "?protocol=7"

	// PusherReconnectDelay is the delay before reconnecting.
	PusherReconnectDelay = 5 * time.Second
)

// KickChannelInfo contains channel information from Kick API.
type KickChannelInfo struct {
	ChatroomID int    `json:"chatroom_id"`
	Username   string `json:"username"`
	UserID     int    `json:"user_id"`
}

// PusherEvent represents a Pusher WebSocket event.
type PusherEvent struct {
	Event   string          `json:"event"`
	Channel string          `json:"channel,omitempty"`
	Data    json.RawMessage `json:"data"`
}

// KickChatEventData contains the chat message data from Kick.
type KickChatEventData struct {
	Message KickChatMessage `json:"message"`
	User    KickChatUser    `json:"user"`
}

// KickChatMessage contains message details.
type KickChatMessage struct {
	ID        string `json:"id"`
	Content   string `json:"message"`
	Type      string `json:"type"`
	CreatedAt int64  `json:"created_at"`
}

// KickChatUser contains user details.
type KickChatUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// HTTPClient is an interface for making HTTP requests.
// This allows mocking in tests without using real network calls.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// KickClient manages the connection to Kick chat.
type KickClient struct {
	// config holds the configuration.
	config *Config

	// hub is the message hub for broadcasting messages.
	hub *Hub

	// conn is the WebSocket connection (interface for mocking).
	conn WebSocketConn

	// chatroomID is the Kick chatroom ID.
	chatroomID int

	// done signals the client to stop.
	done chan struct{}

	// mu protects connection state.
	mu sync.Mutex

	// connected indicates if we're currently connected.
	connected bool

	// httpClient is used for API requests (interface for mocking).
	httpClient HTTPClient

	// dialer is the WebSocket dialer (can be mocked for testing)
	dialer *websocket.Dialer
}

// NewKickClient creates a new Kick chat client.
func NewKickClient(config *Config, hub *Hub) *KickClient {
	return &KickClient{
		config: config,
		hub:    hub,
		done:   make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
		dialer: websocket.DefaultDialer,
	}
}

// NewKickClientWithHTTP creates a new Kick chat client with a custom HTTP client.
// This is used for testing with mock HTTP servers.
func NewKickClientWithHTTP(config *Config, hub *Hub, httpClient HTTPClient) *KickClient {
	return &KickClient{
		config:     config,
		hub:        hub,
		done:       make(chan struct{}),
		httpClient: httpClient,
		dialer:     websocket.DefaultDialer,
	}
}

// NewKickClientWithClients creates a new Kick chat client with custom HTTP client and dialer.
// This is used for testing with mock connections.
func NewKickClientWithClients(config *Config, hub *Hub, httpClient HTTPClient, dialer *websocket.Dialer) *KickClient {
	return &KickClient{
		config:     config,
		hub:        hub,
		done:       make(chan struct{}),
		httpClient: httpClient,
		dialer:     dialer,
	}
}

// NewKickClientWithConn creates a Kick chat client with a pre-existing connection.
// This is used for testing with mock WebSocket connections.
func NewKickClientWithConn(config *Config, hub *Hub, conn WebSocketConn, chatroomID int) *KickClient {
	return &KickClient{
		config:     config,
		hub:        hub,
		done:       make(chan struct{}),
		conn:       conn,
		chatroomID: chatroomID,
		connected:  true,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		dialer:     websocket.DefaultDialer,
	}
}

// Connect establishes the connection to Kick chat.
func (c *KickClient) Connect() error {
	// Step 1: Get chatroom ID from Kick API
	chatroomID, err := c.getChatroomID()
	if err != nil {
		return fmt.Errorf("failed to get chatroom ID: %w", err)
	}
	c.chatroomID = chatroomID
	log.Printf("📺 Kick chatroom ID: %d", chatroomID)

	// Step 2: Connect to Pusher WebSocket
	if err := c.connectPusher(); err != nil {
		return fmt.Errorf("failed to connect to Pusher: %w", err)
	}

	// Step 3: Subscribe to the chatroom channel
	if err := c.subscribeToChannel(); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()

	log.Printf("✅ Connected to Kick channel: %s", c.config.KickChannel)
	return nil
}

// getChatroomID fetches the chatroom ID from Kick API.
func (c *KickClient) getChatroomID() (int, error) {
	url := KickAPIBase + KickChannelEndpoint + c.config.KickChannel

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// Set headers to mimic a browser request
	// This helps avoid rate limiting
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var info struct {
		ChatroomID int `json:"chatroom_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if info.ChatroomID == 0 {
		return 0, fmt.Errorf("channel not found: %s", c.config.KickChannel)
	}

	return info.ChatroomID, nil
}

// connectPusher establishes WebSocket connection to Pusher.
func (c *KickClient) connectPusher() error {
	u := url.URL{
		Scheme:   "wss",
		Host:     PusherHost,
		Path:     PusherPath,
		RawQuery: "client=js&version=7.4.0&protocol=7",
	}

	log.Printf("🔌 Connecting to Pusher: %s", u.Host)

	conn, _, err := c.dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.conn = &websocketConnWrapper{Conn: conn}
	log.Println("✅ Connected to Pusher")
	return nil
}

// subscribeToChannel sends subscription message to Pusher.
func (c *KickClient) subscribeToChannel() error {
	channelName := fmt.Sprintf("chatrooms.%d", c.chatroomID)

	subscribeMsg := map[string]interface{}{
		"event":   "pusher:subscribe",
		"data":    map[string]string{"channel": channelName},
		"channel": channelName,
	}

	data, err := json.Marshal(subscribeMsg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return err
	}

	log.Printf("📺 Subscribed to Kick channel: %s", channelName)
	return nil
}

// Run starts the message reading loop.
func (c *KickClient) Run() {
	for {
		if !c.runIteration() {
			return
		}
	}
}

// runIteration performs one iteration of the message reading loop.
// Returns false if the client should stop, true to continue.
func (c *KickClient) runIteration() bool {
	select {
	case <-c.done:
		return false
	default:
		if err := c.readMessages(); err != nil {
			log.Printf("❌ Kick read error: %v", err)
			c.handleReconnect()
		}
		return true
	}
}

// Stop gracefully stops the client.
// Safe to call multiple times.
func (c *KickClient) Stop() {
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
	log.Println("🛑 Kick client stopped")
}

// IsConnected returns the current connection state.
func (c *KickClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// readMessages reads and processes incoming Pusher events.
func (c *KickClient) readMessages() error {
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

	// Parse the Pusher event
	var event PusherEvent
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("⚠️ Failed to parse Pusher event: %v", err)
		return nil // Don't return error, just skip this message
	}

	// Handle different event types
	switch event.Event {
	case "pusher:connection_established":
		log.Println("✅ Pusher connection established")

	case "pusher_internal:subscription_succeeded":
		log.Println("✅ Pusher subscription succeeded")

	case "App\\Events\\ChatMessageSentEvent":
		// This is a chat message!
		c.handleChatMessage(event.Data)

	case "pusher:ping":
		// Respond to ping with pong
		c.sendPong()

	default:
		// Log unknown events for debugging (only in verbose mode)
		if strings.Contains(event.Event, "error") {
			log.Printf("⚠️ Pusher event: %s", event.Event)
		}
	}

	return nil
}

// handleChatMessage processes a chat message event.
func (c *KickClient) handleChatMessage(data json.RawMessage) {
	var eventData KickChatEventData
	if err := json.Unmarshal(data, &eventData); err != nil {
		log.Printf("⚠️ Failed to parse chat message: %v", err)
		return
	}

	msg := &ChatMessage{
		ID:        "kick:" + eventData.Message.ID,
		Platform:  PlatformKick,
		Username:  eventData.User.Username,
		Content:   eventData.Message.Content,
		Timestamp: eventData.Message.CreatedAt,
	}

	// If timestamp is 0, use current time
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	c.hub.Send(msg)
}

// sendPong sends a pong response to Pusher.
func (c *KickClient) sendPong() {
	pong := `{"event":"pusher:pong"}`
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.WriteMessage(websocket.TextMessage, []byte(pong))
	}
}

// WriteMessage writes a message to the WebSocket connection.
// This is a helper for testing.
func (c *KickClient) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.conn.WriteMessage(messageType, data)
}

// handleReconnect handles reconnection logic.
func (c *KickClient) handleReconnect() {
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	log.Printf("🔄 Reconnecting to Kick in %v...", PusherReconnectDelay)
	time.Sleep(PusherReconnectDelay)

	for {
		select {
		case <-c.done:
			return
		default:
			if err := c.Connect(); err != nil {
				log.Printf("❌ Kick reconnect failed: %v", err)
				time.Sleep(PusherReconnectDelay)
				continue
			}
			log.Println("✅ Kick reconnected successfully")
			return
		}
	}
}

// handleReconnectOnce performs a single reconnection attempt.
// Returns true if reconnected, false if stopped or failed.
// This is extracted for testing.
func (c *KickClient) handleReconnectOnce() bool {
	select {
	case <-c.done:
		return false
	default:
		if err := c.Connect(); err != nil {
			log.Printf("❌ Kick reconnect failed: %v", err)
			return true // Continue trying
		}
		log.Println("✅ Kick reconnected successfully")
		return false // Successfully reconnected, stop
	}
}

// cleanupConnection marks connection as disconnected and closes it.
func (c *KickClient) cleanupConnection() {
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

// ParseKickMessage is a helper function for testing.
// It parses a Kick chat event JSON into a ChatMessage.
func ParseKickMessage(jsonData []byte) (*ChatMessage, error) {
	var eventData KickChatEventData
	if err := json.Unmarshal(jsonData, &eventData); err != nil {
		return nil, err
	}

	return &ChatMessage{
		ID:        "kick:" + eventData.Message.ID,
		Platform:  PlatformKick,
		Username:  eventData.User.Username,
		Content:   eventData.Message.Content,
		Timestamp: eventData.Message.CreatedAt,
	}, nil
}
