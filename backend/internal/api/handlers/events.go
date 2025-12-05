package handlers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/events"
	"github.com/network-collector/backend/internal/services/storage"
)

var globalBroadcaster *events.Broadcaster

// SetEventBroadcaster sets the global event broadcaster
func SetEventBroadcaster(broadcaster *events.Broadcaster) {
	globalBroadcaster = broadcaster
}

// StreamEvents streams real-time events using Server-Sent Events (SSE)
func StreamEvents(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalBroadcaster == nil {
			c.JSON(500, gin.H{"error": "Event broadcaster not initialized"})
			return
		}

		// Set headers for SSE
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// Subscribe to events
		eventChan := globalBroadcaster.Subscribe()
		defer globalBroadcaster.Unsubscribe(eventChan)

		// Send initial connection event
		initialEvent := events.Event{
			Type: events.EventTypeConnection,
			Data: map[string]interface{}{
				"message": "Connected to event stream",
			},
		}
		initialEventJSON, err := json.Marshal(initialEvent)
		if err != nil {
			log.Printf("Failed to marshal initial event: %v", err)
			c.JSON(500, gin.H{"error": "Failed to initialize event stream"})
			return
		}
		
		// Write initial event - check for write errors
		if _, err := c.Writer.WriteString("data: " + string(initialEventJSON) + "\n\n"); err != nil {
			log.Printf("Failed to write initial event: %v", err)
			return
		}
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}

		// Send periodic heartbeat to keep connection alive
		// Use request context for cancellation
		ctx := c.Request.Context()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Heartbeat goroutine with proper cleanup
		heartbeatDone := make(chan struct{})
		go func() {
			defer close(heartbeatDone)
			for {
				select {
				case <-ticker.C:
					heartbeat := events.Event{
						Type: events.EventTypeHeartbeat,
						Data: map[string]interface{}{
							"message": "heartbeat",
						},
					}
					heartbeatJSON, err := json.Marshal(heartbeat)
					if err != nil {
						log.Printf("Failed to marshal heartbeat event: %v", err)
						continue
					}
					// Check if context is cancelled before writing
					select {
					case <-ctx.Done():
						return
					default:
						if _, err := c.Writer.WriteString("data: " + string(heartbeatJSON) + "\n\n"); err != nil {
							log.Printf("Failed to write heartbeat: %v", err)
							return
						}
						if flusher, ok := c.Writer.(http.Flusher); ok {
							flusher.Flush()
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// Stream events to client
		for {
			select {
			case event, ok := <-eventChan:
				if !ok {
					// Channel closed, wait for heartbeat goroutine to finish
					<-heartbeatDone
					return
				}

				eventJSON, err := json.Marshal(event)
				if err != nil {
					log.Printf("Failed to marshal event: %v", err)
					continue
				}

				// Check if context is cancelled before writing
				select {
				case <-ctx.Done():
					// Wait for heartbeat goroutine to finish
					<-heartbeatDone
					return
				default:
					// Write SSE format: "data: {json}\n\n"
					if _, err := c.Writer.WriteString("data: " + string(eventJSON) + "\n\n"); err != nil {
						log.Printf("Failed to write event: %v", err)
						// Wait for heartbeat goroutine to finish
						<-heartbeatDone
						return
					}
					if flusher, ok := c.Writer.(http.Flusher); ok {
						flusher.Flush()
					}
				}

			case <-ctx.Done():
				// Wait for heartbeat goroutine to finish
				<-heartbeatDone
				return
			}
		}
	}
}

