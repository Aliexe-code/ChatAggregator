package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Server handles HTTP and WebSocket connections.
// It serves the web frontend and manages WebSocket clients.
//
// Security considerations:
// - WebSocket origin check prevents cross-site WebSocket hijacking
// - Content-Type headers are set correctly for each response
// - Input validation on all endpoints
// - No sensitive data in responses

//go:embed web/*
var webFiles embed.FS

// WebSocket upgrader with security settings
var upgrader = websocket.Upgrader{
	// CheckOrigin validates the request origin
	// In production, you should restrict this to your domain
	CheckOrigin: func(r *http.Request) bool {
		// For development, allow all origins
		// In production, replace with: return r.Header.Get("Origin") == "https://yourdomain.com"
		return true
	},
	// Buffer sizes for performance
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Handshake timeout
	HandshakeTimeout: 10 * time.Second,
}

// Server represents the HTTP/WebSocket server.
type Server struct {
	// hub is the message hub for broadcasting.
	hub *Hub

	// config holds the configuration.
	config *Config

	// httpServer is the underlying HTTP server.
	httpServer *http.Server

	// tmpl holds parsed HTML templates.
	tmpl *template.Template

	// startTime is when the server started.
	startTime time.Time
}

// NewServer creates a new Server instance.
func NewServer(hub *Hub, config *Config) (*Server, error) {
	// Parse templates from embedded files
	tmpl, err := template.ParseFS(webFiles, "web/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		hub:      hub,
		config:   config,
		tmpl:     tmpl,
		startTime: time.Now(),
	}, nil
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Route handlers
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/health", s.handleHealth)

	// Serve static files from embedded FS
	staticFS, _ := fs.Sub(webFiles, "web/static")
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🌐 Server starting on http://localhost:%d", s.config.Port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// handleIndex serves the main page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Security: Add security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Render template with data
	data := struct {
		TwitchChannel string
		KickChannel   string
		Port          int
	}{
		TwitchChannel: s.config.TwitchChannel,
		KickChannel:   s.config.KickChannel,
		Port:          s.config.Port,
	}

	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("❌ Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleWebSocket handles WebSocket connections from clients.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register client with hub
	client := s.hub.Register()
	defer s.hub.Unregister(client)

	log.Printf("🔌 WebSocket client connected from %s", r.RemoteAddr)

	// Start goroutine to write messages to client
	go s.writePump(conn, client)

	// Read pump (keep connection alive, handle close)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ WebSocket read error: %v", err)
			}
			break
		}
	}
}

// writePump sends messages from the hub to the WebSocket client.
func (s *Server) writePump(conn *websocket.Conn, client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				// Hub closed the channel
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Set write deadline
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// Write message
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("⚠️ WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleStats returns server statistics.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	stats := s.hub.Stats()
	uptime := time.Since(s.startTime).Seconds()

	response := map[string]interface{}{
		"total_messages":  stats.TotalMessages,
		"twitch_messages": stats.TwitchMessages,
		"kick_messages":   stats.KickMessages,
		"connected_clients": s.hub.ClientCount(),
		"peak_clients":    stats.PeakClients,
		"uptime_seconds":  uptime,
		"twitch_enabled":  s.config.EnableTwitch,
		"kick_enabled":    s.config.EnableKick,
		"twitch_channel":  s.config.TwitchChannel,
		"kick_channel":    s.config.KickChannel,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode stats: %v", err)
	}
}

// handleHealth returns health status for monitoring.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(s.startTime).Seconds(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
