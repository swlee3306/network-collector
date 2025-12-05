package retention

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/network-collector/backend/internal/services/storage"
)

// CleanupService handles data retention and cleanup
type CleanupService struct {
	repository *storage.Repository
	retentionDays int
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(repository *storage.Repository, retentionDays int) *CleanupService {
	return &CleanupService{
		repository:    repository,
		retentionDays: retentionDays,
	}
}

// CleanupOldMetrics removes metrics older than retention period
func (s *CleanupService) CleanupOldMetrics() error {
	cutoffDate := time.Now().AddDate(0, 0, -s.retentionDays)
	log.Printf("Cleaning up metrics older than %d days (before %s)", s.retentionDays, cutoffDate.Format(time.RFC3339))

	// Cleanup instance metrics
	if err := s.repository.DeleteInstanceMetricsBefore(cutoffDate); err != nil {
		return fmt.Errorf("failed to cleanup instance metrics: %w", err)
	}

	// Cleanup network metrics
	if err := s.repository.DeleteNetworkMetricsBefore(cutoffDate); err != nil {
		return fmt.Errorf("failed to cleanup network metrics: %w", err)
	}

	// Cleanup hypervisor metrics
	if err := s.repository.DeleteHypervisorMetricsBefore(cutoffDate); err != nil {
		return fmt.Errorf("failed to cleanup hypervisor metrics: %w", err)
	}

	log.Println("Metrics cleanup completed successfully")
	return nil
}

// RunPeriodicCleanup runs cleanup on a schedule
func (s *CleanupService) RunPeriodicCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial cleanup
	if err := s.CleanupOldMetrics(); err != nil {
		log.Printf("Initial cleanup failed: %v", err)
	}

	// Run periodic cleanup
	for range ticker.C {
		if err := s.CleanupOldMetrics(); err != nil {
			log.Printf("Periodic cleanup failed: %v", err)
		}
	}
}

// RunPeriodicCleanupWithContext runs cleanup on a schedule with context support for graceful shutdown
func (s *CleanupService) RunPeriodicCleanupWithContext(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial cleanup
	if err := s.CleanupOldMetrics(); err != nil {
		log.Printf("Initial cleanup failed: %v", err)
	}

	// Run periodic cleanup
	for {
		select {
		case <-ticker.C:
			if err := s.CleanupOldMetrics(); err != nil {
				log.Printf("Periodic cleanup failed: %v", err)
			}
		case <-ctx.Done():
			log.Println("Cleanup service stopping due to context cancellation")
			return
		}
	}
}

