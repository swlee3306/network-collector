// +build integration

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/database/migrations"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, *storage.Repository) {
	// Get test database configuration from environment
	config := database.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsIntOrDefault("TEST_DB_PORT", 3306),
		User:     getEnvOrDefault("TEST_DB_USER", "root"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", ""),
		Database: getEnvOrDefault("TEST_DB_NAME", "openstack_monitor_test"),
	}

	db, err := database.Connect(config)
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations
	err = migrations.RunMigrations(db)
	require.NoError(t, err, "Failed to run migrations")

	// Clean up test data before test
	cleanupTestData(t, db)

	repository := storage.NewRepository(db)

	return db, repository
}

func cleanupTestData(t *testing.T, db *gorm.DB) {
	// Clean up test data
	if db != nil {
		// Delete test data in reverse order of dependencies
		db.Exec("DELETE FROM topology_edges")
		db.Exec("DELETE FROM topology_nodes")
		db.Exec("DELETE FROM instance_metrics")
		db.Exec("DELETE FROM network_metrics")
		db.Exec("DELETE FROM hypervisor_metrics")
		db.Exec("DELETE FROM volumes")
		db.Exec("DELETE FROM instances")
		db.Exec("DELETE FROM ports")
		db.Exec("DELETE FROM routers")
		db.Exec("DELETE FROM networks")
		db.Exec("DELETE FROM subnets")
		db.Exec("DELETE FROM hypervisors")
		db.Exec("DELETE FROM flavors")
		db.Exec("DELETE FROM projects")
	}
}

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
	// Simple conversion - in production, use strconv.Atoi
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := database.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsIntOrDefault("TEST_DB_PORT", 3306),
		User:     getEnvOrDefault("TEST_DB_USER", "root"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", ""),
		Database: getEnvOrDefault("TEST_DB_NAME", "openstack_monitor_test"),
	}

	db, err := database.Connect(config)
	require.NoError(t, err, "Should connect to database")
	defer database.Close()

	// Test basic query
	var count int64
	err = db.Raw("SELECT 1").Scan(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRepository_InstanceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, repo := setupTestDB(t)
	defer func() {
		cleanupTestData(t, db)
		database.Close()
	}()

	t.Run("create and retrieve instance", func(t *testing.T) {
		instance := &models.Instance{
			OpenStackID: "test-instance-1",
			Name:        "test-vm",
			Status:      "ACTIVE",
		}

		err := repo.UpsertInstance(instance)
		assert.NoError(t, err)
		assert.NotEmpty(t, instance.ID)

		// Retrieve by OpenStack ID
		retrieved, err := repo.GetInstanceByOpenStackID("test-instance-1")
		assert.NoError(t, err)
		assert.Equal(t, instance.Name, retrieved.Name)
		assert.Equal(t, instance.Status, retrieved.Status)
	})

	t.Run("list instances", func(t *testing.T) {
		instances, err := repo.ListInstances()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(instances), 1)
	})
}

func TestRepository_NetworkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, repo := setupTestDB(t)
	defer func() {
		cleanupTestData(t, db)
		database.Close()
	}()

	t.Run("create and retrieve network", func(t *testing.T) {
		network := &models.Network{
			OpenStackID: "test-network-1",
			Name:        "test-network",
			Status:      "ACTIVE",
		}

		err := repo.UpsertNetwork(network)
		assert.NoError(t, err)
		assert.NotEmpty(t, network.ID)

		// Retrieve by OpenStack ID
		retrieved, err := repo.GetNetworkByOpenStackID("test-network-1")
		assert.NoError(t, err)
		assert.Equal(t, network.Name, retrieved.Name)
		assert.Equal(t, network.Status, retrieved.Status)
	})
}

func TestRepository_TopologyOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, repo := setupTestDB(t)
	defer func() {
		cleanupTestData(t, db)
		database.Close()
	}()

	t.Run("create topology node and edge", func(t *testing.T) {
		// Create instance first
		instance := &models.Instance{
			OpenStackID: "test-instance-topology",
			Name:        "test-vm-topology",
			Status:      "ACTIVE",
		}
		err := repo.UpsertInstance(instance)
		require.NoError(t, err)

		// Create topology node
		node := &models.TopologyNode{
			NodeType:    models.NodeTypeVM,
			InstanceID:  instance.ID,
			Name:        "test-vm-node",
			IsAccessible: true,
		}

		err = repo.UpsertTopologyNode(node)
		assert.NoError(t, err)
		assert.NotEmpty(t, node.ID)

		// Create another node
		portNode := &models.TopologyNode{
			NodeType:    models.NodeTypePort,
			Name:        "test-port-node",
			IsAccessible: true,
		}
		err = repo.UpsertTopologyNode(portNode)
		require.NoError(t, err)

		// Create edge
		edge := &models.TopologyEdge{
			SourceNodeID: node.ID,
			TargetNodeID: portNode.ID,
			EdgeType:     "VIRTUAL",
		}

		err = repo.UpsertTopologyEdge(edge)
		assert.NoError(t, err)
		assert.NotEmpty(t, edge.ID)
	})
}

