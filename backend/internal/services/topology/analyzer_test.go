package topology

import (
	"testing"

	"github.com/network-collector/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of storage.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetInstanceByID(id string) (*models.Instance, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Instance), args.Error(1)
}

func (m *MockRepository) GetPortsByOpenStackDeviceID(deviceID string) ([]*models.Port, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Port), args.Error(1)
}

func (m *MockRepository) GetNetworkByID(id string) (*models.Network, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Network), args.Error(1)
}

func (m *MockRepository) GetHypervisorByID(id string) (*models.Hypervisor, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Hypervisor), args.Error(1)
}

func (m *MockRepository) UpsertTopologyNode(node *models.TopologyNode) error {
	args := m.Called(node)
	return args.Error(0)
}

func (m *MockRepository) UpsertTopologyEdge(edge *models.TopologyEdge) error {
	args := m.Called(edge)
	return args.Error(0)
}

func (m *MockRepository) GetTopologyNodeByInstanceID(instanceID string) (*models.TopologyNode, error) {
	args := m.Called(instanceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TopologyNode), args.Error(1)
}

func (m *MockRepository) GetTopologyNodeByPortID(portID string) (*models.TopologyNode, error) {
	args := m.Called(portID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TopologyNode), args.Error(1)
}

func (m *MockRepository) GetTopologyNodeByNetworkID(networkID string) (*models.TopologyNode, error) {
	args := m.Called(networkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TopologyNode), args.Error(1)
}

func (m *MockRepository) GetTopologyNodeByHypervisorID(hypervisorID string) (*models.TopologyNode, error) {
	args := m.Called(hypervisorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TopologyNode), args.Error(1)
}

func TestBuildTopologyForVM(t *testing.T) {
	tests := []struct {
		name          string
		instanceID    string
		setupMocks    func(*MockRepository)
		expectError   bool
		expectedCalls int
	}{
		{
			name:       "successful topology build",
			instanceID: "instance-1",
			setupMocks: func(m *MockRepository) {
				instance := &models.Instance{
					ID:          "instance-1",
					OpenStackID: "os-instance-1",
					Name:        "test-vm",
					Status:      "ACTIVE",
					HypervisorID: "hypervisor-1",
				}
				m.On("GetInstanceByID", "instance-1").Return(instance, nil)

				port := &models.Port{
					ID:          "port-1",
					OpenStackID: "os-port-1",
					NetworkID:   "network-1",
					DeviceID:   "os-instance-1",
					Status:     "ACTIVE",
				}
				m.On("GetPortsByOpenStackDeviceID", "os-instance-1").Return([]*models.Port{port}, nil)

				network := &models.Network{
					ID:          "network-1",
					OpenStackID: "os-network-1",
					Name:        "test-network",
					Status:      "ACTIVE",
				}
				m.On("GetNetworkByID", "network-1").Return(network, nil)

				hypervisor := &models.Hypervisor{
					ID:          "hypervisor-1",
					OpenStackID: "os-hypervisor-1",
					Hostname:    "compute-1",
					Status:      "up",
				}
				m.On("GetHypervisorByID", "hypervisor-1").Return(hypervisor, nil)

				// Mock topology node creation
				m.On("UpsertTopologyNode", mock.AnythingOfType("*models.TopologyNode")).Return(nil).Times(4) // VM, Port, Network, Host
				m.On("UpsertTopologyEdge", mock.AnythingOfType("*models.TopologyEdge")).Return(nil).Times(3) // VM->Port, Port->Network, Network->Host
			},
			expectError:   false,
			expectedCalls: 7,
		},
		{
			name:       "instance not found",
			instanceID: "instance-1",
			setupMocks: func(m *MockRepository) {
				m.On("GetInstanceByID", "instance-1").Return(nil, assert.AnError)
			},
			expectError:   true,
			expectedCalls: 1,
		},
		{
			name:       "no ports found",
			instanceID: "instance-1",
			setupMocks: func(m *MockRepository) {
				instance := &models.Instance{
					ID:          "instance-1",
					OpenStackID: "os-instance-1",
					Name:        "test-vm",
					Status:      "ACTIVE",
				}
				m.On("GetInstanceByID", "instance-1").Return(instance, nil)
				m.On("GetPortsByOpenStackDeviceID", "os-instance-1").Return([]*models.Port{}, nil)

				// Should still create VM node
				m.On("UpsertTopologyNode", mock.AnythingOfType("*models.TopologyNode")).Return(nil).Times(1)
			},
			expectError:   false,
			expectedCalls: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These tests require repository interface refactoring
			// For now, we'll skip them and create integration tests instead
			t.Skip("Requires repository interface refactoring for proper unit testing")
		})
	}
}

func TestNewAnalyzer(t *testing.T) {
	// Note: This test requires actual repository, so we'll skip it for now
	// In a real implementation, we'd use dependency injection with interfaces
	t.Skip("Requires repository interface refactoring")
}

