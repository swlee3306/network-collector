package events

import (
	"log"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeResourceUpdated EventType = "resource.updated"
	EventTypeTopologyUpdated EventType = "topology.updated"
	EventTypeError           EventType = "error"
	EventTypeCollectionStart EventType = "collection.start"
	EventTypeCollectionEnd   EventType = "collection.end"
	EventTypeHeartbeat       EventType = "heartbeat"
	EventTypeConnection      EventType = "connection"
)

// Event represents a broadcast event
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Broadcaster manages event broadcasting to SSE clients
type Broadcaster struct {
	clients map[chan Event]bool
	mu      sync.RWMutex
}

// NewBroadcaster creates a new event broadcaster
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[chan Event]bool),
	}
}

// Subscribe adds a new client to receive events
func (b *Broadcaster) Subscribe() chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 10) // Buffer to prevent blocking
	b.clients[ch] = true
	log.Printf("New client subscribed. Total clients: %d", len(b.clients))
	return ch
}

// Unsubscribe removes a client from receiving events
func (b *Broadcaster) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clients[ch] {
		close(ch)
		delete(b.clients, ch)
		log.Printf("Client unsubscribed. Total clients: %d", len(b.clients))
	}
}

// Broadcast sends an event to all subscribed clients
func (b *Broadcaster) Broadcast(event Event) {
	b.mu.RLock()
	// Create a copy of clients map to avoid holding lock during send
	clients := make([]chan Event, 0, len(b.clients))
	for ch := range b.clients {
		clients = append(clients, ch)
	}
	b.mu.RUnlock()

	// Set timestamp once before sending to all clients
	timestamp := time.Now().Format(time.RFC3339)

	// Send to all clients (without holding lock to avoid blocking)
	for _, ch := range clients {
		// Create a new event instance with timestamp for each client
		// This prevents race conditions if event is modified concurrently
		eventCopy := Event{
			Type:      event.Type,
			Timestamp: timestamp,
			Data:      make(map[string]interface{}),
		}
		// Copy data map to avoid sharing mutable state
		for k, v := range event.Data {
			eventCopy.Data[k] = v
		}

		select {
		case ch <- eventCopy:
			// Event sent successfully
		default:
			// Channel is full, skip this client
			log.Printf("Client channel full, skipping broadcast")
		}
	}
}

// BroadcastJSON broadcasts a JSON event
func (b *Broadcaster) BroadcastJSON(eventType EventType, data map[string]interface{}) {
	event := Event{
		Type: eventType,
		Data: data,
	}
	b.Broadcast(event)
}

// GetClientCount returns the number of subscribed clients
func (b *Broadcaster) GetClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

