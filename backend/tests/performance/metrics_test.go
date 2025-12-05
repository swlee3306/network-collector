// +build performance

package performance

import (
	"testing"
	"time"

	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestMetricsQueryPerformance_SC007 tests SC-007: Historical metrics query within 3 seconds
func TestMetricsQueryPerformance_SC007(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create test instance
	instance := &models.Instance{
		OpenStackID:  "test-instance-metrics",
		Name:         "Test Instance for Metrics",
		Status:       "ACTIVE",
		ProjectID:    "test-project",
		FlavorID:     "test-flavor",
		HypervisorID: "test-hypervisor",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CollectedAt:  time.Now(),
	}
	err := repo.UpsertInstance(instance)
	require.NoError(t, err)

	// Create historical metrics data (30 days worth)
	now := time.Now()
	for i := 0; i < 30*24*60; i++ { // 30 days * 24 hours * 60 minutes
		timestamp := now.Add(-time.Duration(i) * time.Minute)
		metrics := &models.InstanceMetrics{
			InstanceID: instance.ID,
			Timestamp:  timestamp,
			CPUUsage:   float64(i % 100),
			MemoryUsage: float64(i % 100),
			DiskUsage:  float64(i % 100),
			NetworkIn:  float64(i * 1000),
			NetworkOut: float64(i * 1000),
		}
		err := repo.SaveInstanceMetrics(metrics)
		require.NoError(t, err)
	}

	// Test metrics query with time range (last 7 days)
	startTime := now.Add(-7 * 24 * time.Hour)
	endTime := now

	queryStart := time.Now()
	metrics, err := repo.GetInstanceMetrics(instance.ID, startTime, endTime)
	queryDuration := time.Since(queryStart)

	require.NoError(t, err)
	assert.LessOrEqual(t, queryDuration, 3*time.Second, "Metrics query should complete within 3 seconds (SC-007)")
	assert.NotEmpty(t, metrics, "Should return metrics data")

	t.Logf("Metrics query completed in %v (target: <3s)", queryDuration)
	t.Logf("Retrieved %d metric records", len(metrics))
}

// TestMetricsQueryWithLargeDataset tests metrics query performance with very large dataset
func TestMetricsQueryWithLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create multiple instances with metrics
	numInstances := 100
	var instanceIDs []string

	for i := 0; i < numInstances; i++ {
		instance := &models.Instance{
			OpenStackID:  "test-instance-metrics-" + string(rune(i)),
			Name:         "Test Instance " + string(rune(i)),
			Status:       "ACTIVE",
			ProjectID:    "test-project",
			FlavorID:     "test-flavor",
			HypervisorID: "test-hypervisor",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			CollectedAt:  time.Now(),
		}
		err := repo.UpsertInstance(instance)
		require.NoError(t, err)
		instanceIDs = append(instanceIDs, instance.ID)

		// Create 7 days of metrics for each instance
		now := time.Now()
		for j := 0; j < 7*24*60; j++ { // 7 days * 24 hours * 60 minutes
			timestamp := now.Add(-time.Duration(j) * time.Minute)
			metrics := &models.InstanceMetrics{
				InstanceID: instance.ID,
				Timestamp:  timestamp,
				CPUUsage:   float64(j % 100),
				MemoryUsage: float64(j % 100),
				DiskUsage:  float64(j % 100),
				NetworkIn:  float64(j * 1000),
				NetworkOut: float64(j * 1000),
			}
			err := repo.SaveInstanceMetrics(metrics)
			require.NoError(t, err)
		}
	}

	// Test query performance
	startTime := time.Now().Add(-7 * 24 * time.Hour)
	endTime := time.Now()

	queryStart := time.Now()
	metrics, err := repo.GetInstanceMetrics(instanceIDs[0], startTime, endTime)
	queryDuration := time.Since(queryStart)

	require.NoError(t, err)
	assert.LessOrEqual(t, queryDuration, 3*time.Second, "Metrics query should complete within 3 seconds even with large dataset")
	assert.NotEmpty(t, metrics, "Should return metrics data")

	t.Logf("Metrics query with large dataset completed in %v", queryDuration)
	t.Logf("Retrieved %d metric records", len(metrics))
}

// BenchmarkMetricsQuery benchmarks metrics query performance
func BenchmarkMetricsQuery(b *testing.B) {
	db, repo := setupPerformanceDB(&testing.T{})
	defer database.Close()

	// Create test instance and metrics
	instance := &models.Instance{
		OpenStackID:  "bench-instance",
		Name:         "Benchmark Instance",
		Status:       "ACTIVE",
		ProjectID:    "test-project",
		FlavorID:     "test-flavor",
		HypervisorID: "test-hypervisor",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CollectedAt:  time.Now(),
	}
	repo.UpsertInstance(instance)

	// Create 7 days of metrics
	now := time.Now()
	for i := 0; i < 7*24*60; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Minute)
		metrics := &models.InstanceMetrics{
			InstanceID: instance.ID,
			Timestamp:  timestamp,
			CPUUsage:   float64(i % 100),
			MemoryUsage: float64(i % 100),
			DiskUsage:  float64(i % 100),
			NetworkIn:  float64(i * 1000),
			NetworkOut: float64(i * 1000),
		}
		repo.SaveInstanceMetrics(metrics)
	}

	startTime := now.Add(-7 * 24 * time.Hour)
	endTime := now

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetInstanceMetrics(instance.ID, startTime, endTime)
	}
}

