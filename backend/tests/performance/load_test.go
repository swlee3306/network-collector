// +build performance

package performance

import (
	"fmt"
	"testing"
	"time"

	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/internal/services/topology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLargeScaleEnvironment_10000VMs simulates a large-scale environment with 10,000 VMs
func TestLargeScaleEnvironment_10000VMs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large-scale performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	t.Log("Setting up large-scale environment with 10,000 VMs...")
	start := time.Now()

	// Create test topology with 10,000 VMs
	instanceIDs := createTestTopology(t, repo, 10000)

	setupDuration := time.Since(start)
	t.Logf("Environment setup completed in %v", setupDuration)

	require.Len(t, instanceIDs, 10000, "Should create 10,000 instances")

	// Test topology query performance
	analyzer := topology.NewAnalyzer(repo)
	pathfinder := topology.NewPathfinder(repo)

	// Test multiple topology queries
	numQueries := 10
	var totalDuration time.Duration

	for i := 0; i < numQueries; i++ {
		instanceID := instanceIDs[i*1000] // Sample different instances

		queryStart := time.Now()
		err := analyzer.BuildTopologyForVM(instanceID)
		queryDuration := time.Since(queryStart)

		require.NoError(t, err)
		assert.LessOrEqual(t, queryDuration, 10*time.Second, 
			"Topology query should complete within 10 seconds (SC-002)")

		totalDuration += queryDuration
	}

	avgDuration := totalDuration / time.Duration(numQueries)
	t.Logf("Average topology query time: %v (target: <10s)", avgDuration)
	assert.LessOrEqual(t, avgDuration, 10*time.Second, 
		"Average query time should be within 10 seconds")

	// Test pathfinding performance
	vmNode, err := repo.GetTopologyNodeByInstanceID(instanceIDs[0])
	require.NoError(t, err)

	pathStart := time.Now()
	graph, err := pathfinder.GetTopologyGraph(vmNode.ID, 10)
	pathDuration := time.Since(pathStart)

	require.NoError(t, err)
	assert.LessOrEqual(t, pathDuration, 5*time.Second, 
		"Pathfinding should complete within 5 seconds in large-scale environment")

	t.Logf("Pathfinding completed in %v", pathDuration)
	if graph != nil {
		t.Logf("Graph found with %d nodes and %d edges", len(graph.Nodes), len(graph.Edges))
	}
}

// TestConcurrentTopologyQueries tests concurrent topology queries
func TestConcurrentTopologyQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create test topology
	instanceIDs := createTestTopology(t, repo, 1000)
	analyzer := topology.NewAnalyzer(repo)

	// Test concurrent queries
	numConcurrent := 10
	results := make(chan time.Duration, numConcurrent)

	start := time.Now()
	for i := 0; i < numConcurrent; i++ {
		go func(idx int) {
			instanceID := instanceIDs[idx*100]
			queryStart := time.Now()
			err := analyzer.BuildTopologyForVM(instanceID)
			queryDuration := time.Since(queryStart)
			
			require.NoError(t, err)
			results <- queryDuration
		}(i)
	}

	// Collect results
	var maxDuration time.Duration
	for i := 0; i < numConcurrent; i++ {
		duration := <-results
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	totalDuration := time.Since(start)

	t.Logf("Concurrent queries completed in %v", totalDuration)
	t.Logf("Max individual query time: %v", maxDuration)
	
	assert.LessOrEqual(t, maxDuration, 5*time.Second, 
		"Each concurrent query should complete within 5 seconds")
}

// TestDatabaseQueryPerformance tests database query performance under load
func TestDatabaseQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create large dataset
	instanceIDs := createTestTopology(t, repo, 5000)

	// Test list instances query
	start := time.Now()
	instances, err := repo.ListInstances()
	duration := time.Since(start)

	require.NoError(t, err)
	assert.LessOrEqual(t, duration, 2*time.Second, 
		"List instances query should complete within 2 seconds")
	assert.Len(t, instances, 5000, "Should return all instances")

	t.Logf("List instances query completed in %v", duration)

	// Test get instance by ID query
	start = time.Now()
	instance, err := repo.GetInstanceByID(instanceIDs[0])
	duration = time.Since(start)

	require.NoError(t, err)
	assert.LessOrEqual(t, duration, 100*time.Millisecond, 
		"Get instance by ID should complete within 100ms")
	assert.NotNil(t, instance, "Should return instance")

	t.Logf("Get instance by ID query completed in %v", duration)
}

// BenchmarkLargeScaleTopology benchmarks topology operations in large-scale environment
func BenchmarkLargeScaleTopology(b *testing.B) {
	db, repo := setupPerformanceDB(&testing.T{})
	defer database.Close()

	instanceIDs := createTestTopology(&testing.T{}, repo, 10000)
	analyzer := topology.NewAnalyzer(repo)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			instanceID := instanceIDs[i%len(instanceIDs)]
			_ = analyzer.BuildTopologyForVM(instanceID)
			i++
		}
	})
}

