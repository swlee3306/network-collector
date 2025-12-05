// +build performance

package performance

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/internal/services/topology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupPerformanceDB sets up a test database for performance testing
func setupPerformanceDB(t *testing.T) (*gorm.DB, *storage.Repository) {
	config := database.Config{
		Host:     getEnvOrDefault("PERF_DB_HOST", "localhost"),
		Port:     getEnvAsIntOrDefault("PERF_DB_PORT", 3306),
		User:     getEnvOrDefault("PERF_DB_USER", "root"),
		Password: getEnvOrDefault("PERF_DB_PASSWORD", ""),
		Database: getEnvOrDefault("PERF_DB_NAME", "openstack_monitor_perf"),
	}

	db, err := database.Connect(config)
	require.NoError(t, err, "Failed to connect to performance test database")

	repository := storage.NewRepository(db)

	return db, repository
}

// createTestTopology creates a test topology with specified number of VMs
func createTestTopology(t *testing.T, repo *storage.Repository, numVMs int) []string {
	var instanceIDs []string

	// Create a test project
	project := &models.Project{
		OpenStackID: "test-project-perf",
		Name:        "Performance Test Project",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CollectedAt: time.Now(),
	}
	err := repo.UpsertProject(project)
	require.NoError(t, err)

	// Create a test network
	network := &models.Network{
		OpenStackID: "test-network-perf",
		Name:        "Performance Test Network",
		Status:      "ACTIVE",
		ProjectID:   project.ID,
		Shared:      false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CollectedAt: time.Now(),
	}
	err = repo.UpsertNetwork(network)
	require.NoError(t, err)

	// Create instances and topology
	for i := 0; i < numVMs; i++ {
		instance := &models.Instance{
			OpenStackID:  fmt.Sprintf("test-instance-%d", i),
			Name:         fmt.Sprintf("Test Instance %d", i),
			Status:       "ACTIVE",
			ProjectID:    project.ID,
			FlavorID:     "test-flavor",
			HypervisorID: fmt.Sprintf("test-hypervisor-%d", i%10), // 10 hypervisors
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			CollectedAt:  time.Now(),
		}
		err := repo.UpsertInstance(instance)
		require.NoError(t, err)
		instanceIDs = append(instanceIDs, instance.ID)

		// Create port for instance
		port := &models.Port{
			OpenStackID: fmt.Sprintf("test-port-%d", i),
			Name:        fmt.Sprintf("Test Port %d", i),
			Status:      "ACTIVE",
			NetworkID:   network.ID,
			DeviceID:    instance.ID,
			DeviceOwner: "compute:nova",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			CollectedAt: time.Now(),
		}
		err = repo.UpsertPort(port)
		require.NoError(t, err)

		// Create topology nodes
		vmNode := &models.TopologyNode{
			NodeType:    models.NodeTypeVM,
			OpenStackID: instance.OpenStackID,
			Name:        instance.Name,
			InstanceID:  instance.ID,
		}
		err = repo.UpsertTopologyNode(vmNode)
		require.NoError(t, err)

		portNode := &models.TopologyNode{
			NodeType:    models.NodeTypePort,
			OpenStackID: port.OpenStackID,
			Name:        port.Name,
			PortID:      port.ID,
		}
		err = repo.UpsertTopologyNode(portNode)
		require.NoError(t, err)

		// Create topology edge
		edge := &models.TopologyEdge{
			SourceNodeID: vmNode.ID,
			TargetNodeID: portNode.ID,
			EdgeType:     "VIRTUAL",
		}
		err = repo.UpsertTopologyEdge(edge)
		require.NoError(t, err)
	}

	return instanceIDs
}

// TestTopologyQueryPerformance_SC001 tests SC-001: VM topology query within 5 seconds
func TestTopologyQueryPerformance_SC001(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create test topology with 100 VMs
	instanceIDs := createTestTopology(t, repo, 100)
	require.NotEmpty(t, instanceIDs)

	analyzer := topology.NewAnalyzer(repo)
	pathfinder := topology.NewPathfinder(repo)

	// Test topology query for a single VM
	instanceID := instanceIDs[0]

	start := time.Now()
	err := analyzer.BuildTopologyForVM(instanceID)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.LessOrEqual(t, duration, 5*time.Second, "Topology query should complete within 5 seconds (SC-001)")

	t.Logf("Topology query completed in %v (target: <5s)", duration)
}

// TestTopologyQueryPerformance_SC002 tests SC-002: Topology query for 10,000 VMs within 10 seconds
func TestTopologyQueryPerformance_SC002(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create test topology with 10,000 VMs
	t.Log("Creating test topology with 10,000 VMs...")
	instanceIDs := createTestTopology(t, repo, 10000)
	require.NotEmpty(t, instanceIDs)

	analyzer := topology.NewAnalyzer(repo)

	// Test topology query for a VM in large-scale environment
	instanceID := instanceIDs[0]

	start := time.Now()
	err := analyzer.BuildTopologyForVM(instanceID)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.LessOrEqual(t, duration, 10*time.Second, "Topology query should complete within 10 seconds for 10,000 VMs (SC-002)")

	t.Logf("Topology query for 10,000 VM environment completed in %v (target: <10s)", duration)
}

// TestTopologyPathfindingPerformance tests pathfinding performance
func TestTopologyPathfindingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, repo := setupPerformanceDB(t)
	defer database.Close()

	// Create test topology
	instanceIDs := createTestTopology(t, repo, 1000)
	require.NotEmpty(t, instanceIDs)

	pathfinder := topology.NewPathfinder(repo)

	// Get topology nodes
	vmNode, err := repo.GetTopologyNodeByInstanceID(instanceIDs[0])
	require.NoError(t, err)

	// Test pathfinding
	start := time.Now()
	graph, err := pathfinder.GetTopologyGraph(vmNode.ID, 10)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.LessOrEqual(t, duration, 2*time.Second, "Pathfinding should complete within 2 seconds")

	t.Logf("Pathfinding completed in %v", duration)
	if graph != nil {
		t.Logf("Graph found with %d nodes and %d edges", len(graph.Nodes), len(graph.Edges))
	}
}

// BenchmarkTopologyQuery benchmarks topology query performance
func BenchmarkTopologyQuery(b *testing.B) {
	db, repo := setupPerformanceDB(&testing.T{})
	defer database.Close()

	instanceIDs := createTestTopology(&testing.T{}, repo, 1000)
	analyzer := topology.NewAnalyzer(repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instanceID := instanceIDs[i%len(instanceIDs)]
		_ = analyzer.BuildTopologyForVM(instanceID)
	}
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

