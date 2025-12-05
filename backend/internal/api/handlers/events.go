package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/events"
	"github.com/network-collector/backend/internal/services/storage"
)

// StreamEvents streams real-time events using Server-Sent Events (SSE)
func StreamEvents(repo *storage.Repository, broadcaster *events.Broadcaster) gin.HandlerFunc {
	return func(c *gin.Context) {
		if broadcaster == nil {
			c.JSON(500, gin.H{"error": "Event broadcaster not initialized"})
			return
		}

		// Set headers for SSE
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// Subscribe to events
		eventChan := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(eventChan)

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

		// Use WaitGroup for proper goroutine coordination
		var wg sync.WaitGroup
		wg.Add(1)

		// Heartbeat goroutine with proper cleanup
		go func() {
			defer wg.Done()
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
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case event, ok := <-eventChan:
					if !ok {
						// Channel closed
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
						return
					default:
						// Write SSE format: "data: {json}\n\n"
						if _, err := c.Writer.WriteString("data: " + string(eventJSON) + "\n\n"); err != nil {
							log.Printf("Failed to write event: %v", err)
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

		// Wait for either context cancellation or event loop completion
		select {
		case <-ctx.Done():
			// Context cancelled, wait for goroutines to finish
			wg.Wait()
			<-done
		case <-done:
			// Event loop completed, wait for heartbeat goroutine
			wg.Wait()
		}
	}
}

