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

	// Check for retryable patterns first (network errors, timeouts, 5xx errors)
	hasRetryablePattern := containsAny(errorMsg, []string{
		"timeout", "connection refused", "connection reset",
		"network", "temporary", "503", "504", "502",
	})

	// Check for non-retryable 4xx errors (excluding 401, 403 as per comment)
	hasNonRetryable4xx := containsAny(errorMsg, []string{"400", "404", "409"})

	// Check for 401/403 (authentication/authorization errors)
	hasAuthError := containsAny(errorMsg, []string{"401", "403"})

	// Determine retryability:
	// 1. If retryable patterns found, set to true
	// 2. If non-retryable 4xx found (and no retryable patterns), set to false
	// 3. 401/403: retryable if retryable patterns also present, otherwise false
	if hasRetryablePattern {
		retryable = true
		// Even if 401/403 present, keep retryable = true if retryable patterns found
		// (e.g., "timeout 401" or "network 403" - temporary auth failures)
	} else if hasNonRetryable4xx {
		retryable = false
	} else if hasAuthError {
		// 401/403 without retryable patterns are not retryable
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

