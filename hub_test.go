package main

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestNewHub tests that NewHub creates a valid Hub.
func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.messages == nil {
		t.Error("Hub.messages channel is nil")
	}
	if hub.clients == nil {
		t.Error("Hub.clients map is nil")
	}
	if hub.done == nil {
		t.Error("Hub.done channel is nil")
	}
}

// TestHub_Register tests client registration.
func TestHub_Register(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := hub.Register()

	// Wait a bit for the registration to process
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.ClientCount())
	}

	if client.send == nil {
		t.Error("Client.send channel is nil")
	}
}

// TestHub_Unregister tests client unregistration.
func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := hub.Register()
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Fatalf("Expected 1 client after registration, got %d", hub.ClientCount())
	}

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after unregistration, got %d", hub.ClientCount())
	}
}

// TestHub_Send tests message broadcasting.
func TestHub_Send(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := hub.Register()
	time.Sleep(10 * time.Millisecond)

	msg := &ChatMessage{
		ID:        "test:123",
		Platform:  PlatformTwitch,
		Username:  "testuser",
		Content:   "Hello, world!",
		Timestamp: time.Now().Unix(),
	}

	hub.Send(msg)
	time.Sleep(10 * time.Millisecond)

	// Check that message was received
	select {
	case data := <-client.send:
		var received ChatMessage
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if received.ID != msg.ID {
			t.Errorf("Expected ID %s, got %s", msg.ID, received.ID)
		}
		if received.Content != msg.Content {
			t.Errorf("Expected Content %s, got %s", msg.Content, received.Content)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive message within timeout")
	}

	// Check stats
	stats := hub.Stats()
	if stats.TotalMessages != 1 {
		t.Errorf("Expected TotalMessages 1, got %d", stats.TotalMessages)
	}
}

// TestHub_Stats tests statistics tracking.
func TestHub_Stats(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Register a client to receive messages (we need this for broadcasting)
	_ = hub.Register()
	time.Sleep(10 * time.Millisecond)

	// Send Twitch message
	hub.Send(&ChatMessage{
		ID:       "twitch:1",
		Platform: PlatformTwitch,
		Content:  "test",
	})
	time.Sleep(10 * time.Millisecond)

	// Send Kick message
	hub.Send(&ChatMessage{
		ID:       "kick:1",
		Platform: PlatformKick,
		Content:  "test",
	})
	time.Sleep(10 * time.Millisecond)

	stats := hub.Stats()
	if stats.TotalMessages != 2 {
		t.Errorf("Expected TotalMessages 2, got %d", stats.TotalMessages)
	}
	if stats.TwitchMessages != 1 {
		t.Errorf("Expected TwitchMessages 1, got %d", stats.TwitchMessages)
	}
	if stats.KickMessages != 1 {
		t.Errorf("Expected KickMessages 1, got %d", stats.KickMessages)
	}
}

// TestHub_ConcurrentClients tests handling multiple concurrent clients.
func TestHub_ConcurrentClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	const numClients = 10
	var wg sync.WaitGroup

	// Register multiple clients concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := hub.Register()
			time.Sleep(5 * time.Millisecond)
			hub.Unregister(client)
		}()
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after all unregister, got %d", hub.ClientCount())
	}
}

// TestHub_BroadcastToMultipleClients tests broadcasting to multiple clients.
func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	const numClients = 5
	clients := make([]*Client, numClients)

	for i := 0; i < numClients; i++ {
		clients[i] = hub.Register()
	}
	time.Sleep(10 * time.Millisecond)

	msg := &ChatMessage{
		ID:       "test:1",
		Platform: PlatformTwitch,
		Content:  "broadcast test",
	}

	hub.Send(msg)
	time.Sleep(10 * time.Millisecond)

	// All clients should receive the message
	for i, client := range clients {
		select {
		case <-client.send:
			// Message received
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Client %d did not receive message", i)
		}
	}
}

// TestNewChatMessage tests the NewChatMessage constructor.
func TestNewChatMessage(t *testing.T) {
	msg := NewChatMessage("test:123", PlatformTwitch, "user", "Hello")

	if msg.ID != "test:123" {
		t.Errorf("Expected ID 'test:123', got '%s'", msg.ID)
	}
	if msg.Platform != PlatformTwitch {
		t.Errorf("Expected Platform Twitch, got %s", msg.Platform)
	}
	if msg.Username != "user" {
		t.Errorf("Expected Username 'user', got '%s'", msg.Username)
	}
	if msg.Content != "Hello" {
		t.Errorf("Expected Content 'Hello', got '%s'", msg.Content)
	}
	if msg.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
}

// TestNewChatMessageWithBadges tests the NewChatMessageWithBadges constructor.
func TestNewChatMessageWithBadges(t *testing.T) {
	badges := []string{"moderator", "subscriber"}
	msg := NewChatMessageWithBadges("test:456", PlatformKick, "user", "Hi", badges, "#FF0000")

	if msg.ID != "test:456" {
		t.Errorf("Expected ID 'test:456', got '%s'", msg.ID)
	}
	if len(msg.Badges) != 2 {
		t.Errorf("Expected 2 badges, got %d", len(msg.Badges))
	}
	if msg.Color != "#FF0000" {
		t.Errorf("Expected Color '#FF0000', got '%s'", msg.Color)
	}
}

// TestHub_PeakClients tests that peak clients stat is updated.
func TestHub_PeakClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Register 3 clients
	for i := 0; i < 3; i++ {
		hub.Register()
		time.Sleep(5 * time.Millisecond)
	}

	stats := hub.Stats()
	if stats.PeakClients != 3 {
		t.Errorf("Expected PeakClients 3, got %d", stats.PeakClients)
	}
}

// TestHub_SendToEmptyClient tests sending when no clients are registered.
func TestHub_SendToEmptyClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Send without any clients - should not panic
	msg := &ChatMessage{
		ID:       "test:1",
		Platform: PlatformTwitch,
		Content:  "test",
	}

	hub.Send(msg)
	time.Sleep(10 * time.Millisecond)

	// Should still count the message
	stats := hub.Stats()
	if stats.TotalMessages != 1 {
		t.Errorf("Expected TotalMessages 1, got %d", stats.TotalMessages)
	}
}

// TestHub_Stop_Idempotent tests that Stop can be called multiple times.
func TestHub_Stop_Idempotent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	time.Sleep(10 * time.Millisecond)

	// Stop multiple times should not panic
	for i := 0; i < 3; i++ {
		hub.Stop()
	}
}

// TestHub_UnregisterNonExistentClient tests unregistering a client that doesn't exist.
func TestHub_UnregisterNonExistentClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Create a fake client that was never registered
	fakeClient := &Client{
		send: make(chan []byte, 256),
	}

	// Should not panic
	hub.Unregister(fakeClient)
	time.Sleep(10 * time.Millisecond)
}

// TestHub_RapidMessageSending tests rapid message sending.
func TestHub_RapidMessageSending(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := hub.Register()
	time.Sleep(10 * time.Millisecond)

	// Send many messages rapidly
	const numMessages = 100
	for i := 0; i < numMessages; i++ {
		msg := &ChatMessage{
			ID:       string(rune(i)),
			Platform: PlatformTwitch,
			Content:  "test",
		}
		hub.Send(msg)
	}

	// Give time to process
	time.Sleep(100 * time.Millisecond)

	// Check stats
	stats := hub.Stats()
	if stats.TotalMessages != numMessages {
		t.Errorf("Expected TotalMessages %d, got %d", numMessages, stats.TotalMessages)
	}

	// Receive messages (may not get all due to buffer)
	received := 0
	for {
		select {
		case <-client.send:
			received++
		default:
			goto done
		}
	}
done:
	if received == 0 {
		t.Error("Should have received at least one message")
	}
}

// TestHub_ClientBufferFull tests behavior when client buffer is full.
func TestHub_ClientBufferFull(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	_ = hub.Register() // Register client but we don't need to use it directly
	time.Sleep(10 * time.Millisecond)

	// Fill the client's buffer
	const bufferSize = 256
	for i := 0; i < bufferSize+10; i++ {
		msg := &ChatMessage{
			ID:       string(rune(i)),
			Platform: PlatformTwitch,
			Content:  "test message content here",
		}
		hub.Send(msg)
	}

	// Hub should handle full buffer gracefully (drop messages)
	time.Sleep(50 * time.Millisecond)

	// Hub should still be running
	stats := hub.Stats()
	if stats.TotalMessages == 0 {
		t.Error("Should have counted some messages")
	}
}

// TestClient_Send tests the client send channel.
func TestClient_Send(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := hub.Register()
	time.Sleep(10 * time.Millisecond)

	// Test that client can receive
	msg := &ChatMessage{
		ID:       "test:1",
		Platform: PlatformKick,
		Username: "user",
		Content:  "Hello",
	}

	hub.Send(msg)

	select {
	case data := <-client.send:
		var received ChatMessage
		json.Unmarshal(data, &received)
		if received.ID != "test:1" {
			t.Errorf("ID = %s, want test:1", received.ID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive message")
	}
}

// TestHub_Stats_ZeroClients tests stats when no clients connected.
func TestHub_Stats_ZeroClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	stats := hub.Stats()

	// PeakClients should be 0 when no clients connected
	if stats.PeakClients != 0 {
		t.Errorf("PeakClients = %d, want 0", stats.PeakClients)
	}
	// Use ClientCount() for current client count
	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount = %d, want 0", hub.ClientCount())
	}
}

// TestHub_DoubleRun tests calling Run twice.
func TestHub_DoubleRun(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Second Run should return immediately because done is not closed yet
	// This is a sanity check - we can't easily test this without race conditions
	defer hub.Stop()
}

// TestHub_MessageOrder tests that message order is preserved.
func TestHub_MessageOrder(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := hub.Register()
	time.Sleep(10 * time.Millisecond)

	// Send messages in order
	for i := 0; i < 5; i++ {
		msg := &ChatMessage{
			ID:       string(rune('A' + i)),
			Platform: PlatformTwitch,
			Content:  "test",
		}
		hub.Send(msg)
	}

	time.Sleep(50 * time.Millisecond)

	// Receive messages
	prevID := ""
	count := 0
	for {
		select {
		case data := <-client.send:
			var msg ChatMessage
			json.Unmarshal(data, &msg)
			if prevID != "" && msg.ID < prevID {
				t.Errorf("Message order: got %s after %s", msg.ID, prevID)
			}
			prevID = msg.ID
			count++
		default:
			goto done
		}
	}
done:
	if count < 1 {
		t.Error("Should have received at least one message")
	}
}
