package collector

import (
	"fmt"
	"log"
	"time"

	"github.com/network-collector/backend/pkg/errors"
)

// CollectionResult represents the result of a collection operation
type CollectionResult struct {
	Success      bool
	ResourceType string
	Error        error
	Retryable    bool
}

// HandleCollectionError handles errors during collection with retry logic
func HandleCollectionError(resourceType string, err error, retryable bool) *CollectionResult {
	if err == nil {
		return &CollectionResult{
			Success:      true,
			ResourceType: resourceType,
		}
	}

	// Check if it's an OpenStack error
	if openstackErr, ok := err.(*errors.OpenStackError); ok {
		retryable = openstackErr.Retryable
	}

	log.Printf("Collection error for %s: %v (retryable: %v)", resourceType, err, retryable)

	return &CollectionResult{
		Success:      false,
		ResourceType: resourceType,
		Error:        err,
		Retryable:    retryable,
	}
}

// RetryCollection retries a collection operation with exponential backoff
func RetryCollection(operation func() error, maxRetries int, resourceType string) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("Retrying %s collection (attempt %d/%d) after %v", resourceType, attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		err := operation()
		if err == nil {
			if attempt > 0 {
				log.Printf("%s collection succeeded after %d retries", resourceType, attempt)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !errors.IsRetryable(err) {
			log.Printf("%s collection error is not retryable: %v", resourceType, err)
			return err
		}

		log.Printf("%s collection failed (attempt %d/%d): %v", resourceType, attempt+1, maxRetries, err)
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// ClassifyOpenStackError classifies an OpenStack error and determines if it's retryable
func ClassifyOpenStackError(service, operation string, err error) *errors.OpenStackError {
	if err == nil {
		return nil
	}

	// Determine if error is retryable based on error type
	retryable := false
	errorMsg := err.Error()

	// Network errors, timeouts, and 5xx errors are usually retryable
	if containsAny(errorMsg, []string{
		"timeout", "connection refused", "connection reset",
		"network", "temporary", "503", "504", "502",
	}) {
		retryable = true
	}

	// 4xx errors (except 401, 403) are usually not retryable
	if containsAny(errorMsg, []string{"400", "401", "403", "404", "409"}) {
		retryable = false
	}

	return errors.NewOpenStackError(service, operation, err, retryable)
}

// Helper function to check if string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

