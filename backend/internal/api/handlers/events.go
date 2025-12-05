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
			Type: events.EventTypeCollectionEnd,
			Data: map[string]interface{}{
				"message": "Connected to event stream",
			},
		}
		initialEventJSON, _ := json.Marshal(initialEvent)
		c.Writer.WriteString("data: " + string(initialEventJSON) + "\n\n")
		c.Writer.Flush()

		// Send periodic heartbeat to keep connection alive
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ticker.C:
					heartbeat := events.Event{
						Type: events.EventTypeCollectionEnd,
						Data: map[string]interface{}{
							"message": "heartbeat",
						},
					}
					heartbeatJSON, _ := json.Marshal(heartbeat)
					c.Writer.WriteString("data: " + string(heartbeatJSON) + "\n\n")
					c.Writer.Flush()
				case <-c.Request.Context().Done():
					return
				}
			}
		}()

		// Stream events to client
		for {
			select {
			case event, ok := <-eventChan:
				if !ok {
					return
				}

				eventJSON, err := json.Marshal(event)
				if err != nil {
					log.Printf("Failed to marshal event: %v", err)
					continue
				}

				// Write SSE format: "data: {json}\n\n"
				c.Writer.WriteString("data: " + string(eventJSON) + "\n\n")
				c.Writer.Flush()

			case <-c.Request.Context().Done():
				return
			}
		}
	}
}

