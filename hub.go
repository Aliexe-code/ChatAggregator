package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub is the central message aggregator.
// It collects chat messages from all platform clients and broadcasts them
// to all connected WebSocket clients (frontend users).
//
// Thread Safety: All operations are protected by a mutex to allow
// concurrent access from multiple goroutines (platform clients).
//
// Performance: Uses buffered channels to handle message bursts.
// The message queue size can be configured via constants.
const (
	// MessageBufferSize is the size of the message channel buffer.
	// A larger buffer handles message bursts better but uses more memory.
	MessageBufferSize = 1000

	// ClientSendBufferSize is the size of each client's send channel.
	// Larger values prevent slow clients from blocking others.
	ClientSendBufferSize = 256
)

// Hub manages connected clients and broadcasts messages.
type Hub struct {
	// messages is the incoming message channel.
	// Platform clients send messages to this channel.
	messages chan *ChatMessage

	// register is used to register new WebSocket clients.
	register chan *Client

	// unregister is used to unregister WebSocket clients.
	unregister chan *Client

	// clients is a map of all connected WebSocket clients.
	// Using a map allows O(1) client lookup/removal.
	clients map[*Client]bool

	// mu protects concurrent access to clients map.
	mu sync.RWMutex

	// done signals the hub to stop running.
	done chan struct{}

	// stats tracks hub statistics.
	stats HubStats
}

// Client represents a connected WebSocket client (frontend user).
type Client struct {
	// hub is a reference to the hub for unregistering.
	hub *Hub

	// send is a channel for sending messages to this client.
	// Buffered to prevent slow clients from blocking the hub.
	send chan []byte
}

// HubStats tracks statistics about the hub.
type HubStats struct {
	// TotalMessages is the total number of messages processed.
	TotalMessages int64

	// TwitchMessages is the number of messages from Twitch.
	TwitchMessages int64

	// KickMessages is the number of messages from Kick.
	KickMessages int64

	// PeakClients is the maximum number of concurrent clients.
	PeakClients int

	// mu protects concurrent access to stats.
	mu sync.RWMutex
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		messages:   make(chan *ChatMessage, MessageBufferSize),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop.
// This should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			// Hub is shutting down
			h.closeAllClients()
			return

		case client := <-h.register:
			// New client connected
			h.registerClient(client)

		case client := <-h.unregister:
			// Client disconnected
			h.unregisterClient(client)

		case message := <-h.messages:
			// New message from a platform
			h.broadcastMessage(message)
		}
	}
}

// Stop gracefully stops the hub.
// Safe to call multiple times.
func (h *Hub) Stop() {
	select {
	case <-h.done:
		// Already closed
	default:
		close(h.done)
	}
}

// Register creates and registers a new WebSocket client.
// Returns the client for use in WebSocket handlers.
func (h *Hub) Register() *Client {
	client := &Client{
		hub:  h,
		send: make(chan []byte, ClientSendBufferSize),
	}
	h.register <- client
	return client
}

// Unregister removes a client from the hub.
// This should be called when a WebSocket connection closes.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Send sends a chat message to the hub for broadcasting.
// This is called by platform clients when they receive messages.
// The function is non-blocking - messages are queued in a buffer.
func (h *Hub) Send(message *ChatMessage) {
	select {
	case h.messages <- message:
		// Message queued successfully
	default:
		// Buffer is full, message dropped
		// This prevents blocking if messages come too fast
		log.Printf("⚠️ Hub message buffer full, message dropped")
	}
}

// Stats returns a copy of the current hub statistics.
func (h *Hub) Stats() HubStats {
	h.stats.mu.RLock()
	defer h.stats.mu.RUnlock()
	return h.stats
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// registerClient adds a client to the hub.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	clientCount := len(h.clients)
	h.mu.Unlock()

	// Update peak clients stat
	h.stats.mu.Lock()
	if clientCount > h.stats.PeakClients {
		h.stats.PeakClients = clientCount
	}
	h.stats.mu.Unlock()

	log.Printf("👤 Client connected. Total clients: %d", clientCount)
}

// unregisterClient removes a client from the hub and closes its send channel.
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	clientCount := len(h.clients)
	h.mu.Unlock()

	log.Printf("👋 Client disconnected. Total clients: %d", clientCount)
}

// broadcastMessage sends a message to all connected clients.
func (h *Hub) broadcastMessage(message *ChatMessage) {
	// Update statistics
	h.stats.mu.Lock()
	h.stats.TotalMessages++
	if message.IsTwitch() {
		h.stats.TwitchMessages++
	} else if message.IsKick() {
		h.stats.KickMessages++
	}
	h.stats.mu.Unlock()

	// Marshal message to JSON once (performance optimization)
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal message: %v", err)
		return
	}

	// Broadcast to all clients
	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.send <- data:
			// Message sent successfully
		default:
			// Client's buffer is full, skip this client
			// The client might be slow or disconnected
		}
	}
	h.mu.RUnlock()
}

// closeAllClients closes all client connections.
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}

// NewChatMessage creates a new ChatMessage with current timestamp.
func NewChatMessage(id string, platform Platform, username, content string) *ChatMessage {
	return &ChatMessage{
		ID:        id,
		Platform:  platform,
		Username:  username,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
}

// NewChatMessageWithBadges creates a new ChatMessage with badges and color.
func NewChatMessageWithBadges(id string, platform Platform, username, content string, badges []string, color string) *ChatMessage {
	return &ChatMessage{
		ID:        id,
		Platform:  platform,
		Username:  username,
		Content:   content,
		Timestamp: time.Now().Unix(),
		Badges:    badges,
		Color:     color,
	}
}
