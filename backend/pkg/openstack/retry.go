package openstack

import (
	"fmt"
	"math"
	"time"

	"github.com/gophercloud/gophercloud"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []int // HTTP status codes that should be retried
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        8 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []int{500, 502, 503, 504, 429},
	}
}

// RetryFunc is a function that can be retried
type RetryFunc func() error

// Retry executes a function with exponential backoff retry logic
func Retry(fn RetryFunc, config RetryConfig) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, config) {
			return err
		}

		// Don't sleep after last attempt
		if attempt < config.MaxRetries {
			time.Sleep(delay)
			delay = time.Duration(math.Min(float64(delay.Nanoseconds())*config.BackoffFactor, float64(config.MaxDelay.Nanoseconds())))
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error, config RetryConfig) bool {
	// Check if it's a gophercloud error with retryable status code
	if gophercloudErr, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
		for _, code := range config.RetryableErrors {
			if gophercloudErr.Actual == code {
				return true
			}
		}
	}

	// Network errors are generally retryable
	if _, ok := err.(gophercloud.ErrDefault500); ok {
		return true
	}
	if _, ok := err.(gophercloud.ErrDefault503); ok {
		return true
	}

	return false
}

