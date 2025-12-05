package models

import (
	"github.com/google/uuid"
)

// generateUUID generates a new UUID string
func generateUUID() string {
	return uuid.New().String()
}

// AllModels returns a slice of all model types for auto-migration
func AllModels() []interface{} {
	return []interface{}{
		&Project{},
		&Instance{},
		&Network{},
		&Subnet{},
		&Port{},
		&Router{},
		&Hypervisor{},
		&Flavor{},
		&Volume{},
		&TopologyNode{},
		&TopologyEdge{},
		&InstanceMetrics{},
		&NetworkMetrics{},
		&HypervisorMetrics{},
	}
}

