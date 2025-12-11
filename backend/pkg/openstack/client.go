package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/network-collector/backend/pkg/errors"
)

// Config holds OpenStack client configuration
type Config struct {
	AuthURL      string
	Username     string
	Password     string
	ProjectID    string
	DomainName   string
	EndpointType string // "public", "internal", or "admin" - controls which endpoint to use
}

// Client wraps OpenStack API clients
type Client struct {
	provider *gophercloud.ProviderClient
	nova     *gophercloud.ServiceClient
	neutron  *gophercloud.ServiceClient
	cinder   *gophercloud.ServiceClient
	keystone *gophercloud.ServiceClient
}

// NewClient creates a new OpenStack client
func NewClient(config Config) (*Client, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.Password,
		DomainName:       config.DomainName,
		TenantID:         config.ProjectID,
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, errors.NewOpenStackError("keystone", "authenticate", err, true)
	}

	// Set token expiration and refresh logic
	provider.HTTPClient.Timeout = 30 * time.Second

	// Determine endpoint type (default to "public" if not specified)
	endpointType := config.EndpointType
	if endpointType == "" {
		endpointType = "public"
	}

	// Validate endpoint type
	if endpointType != "public" && endpointType != "internal" && endpointType != "admin" {
		endpointType = "public" // Default to public if invalid
	}

	// Try to create service clients with specified endpoint type
	// If the endpoint type is not available, fallback to "public"
	endpointOpts := gophercloud.EndpointOpts{
		Type: endpointType,
	}

	log.Printf("OpenStack client: attempting to use endpoint type '%s'", endpointType)

	// Helper function to try creating a service client with fallback
	tryCreateServiceClient := func(serviceName string, createFunc func(*gophercloud.ProviderClient, gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error)) (*gophercloud.ServiceClient, error) {
		// Try with specified endpoint type first
		client, err := createFunc(provider, endpointOpts)
		if err != nil {
			errMsg := err.Error()
			log.Printf("OpenStack %s client: failed to create with endpoint type '%s': %v", serviceName, endpointType, err)
			
			// Check if this is an endpoint not found error
			isEndpointNotFound := contains(errMsg, "No suitable endpoint") ||
				contains(errMsg, "could not find") ||
				contains(errMsg, "endpoint could not be found") ||
				contains(errMsg, "service catalog")
			
			// If endpoint type is not "public" and we get endpoint not found error, try public
			if endpointType != "public" && isEndpointNotFound {
				log.Printf("OpenStack %s client: endpoint type '%s' not found, falling back to 'public'", serviceName, endpointType)
				// Fallback to public endpoint
				publicOpts := gophercloud.EndpointOpts{Type: "public"}
				client, fallbackErr := createFunc(provider, publicOpts)
				if fallbackErr == nil {
					log.Printf("OpenStack %s client: successfully created with 'public' endpoint type", serviceName)
					return client, nil
				}
				// If public also fails, return a combined error message
				log.Printf("OpenStack %s client: fallback to 'public' also failed: %v", serviceName, fallbackErr)
				return nil, fmt.Errorf("failed to create %s client with '%s' endpoint (fallback to 'public' also failed: %v)", serviceName, endpointType, fallbackErr)
			}
			// Return original error if it's not an endpoint not found error, or if we're already trying public
			return nil, err
		}
		log.Printf("OpenStack %s client: successfully created with endpoint type '%s'", serviceName, endpointType)
		return client, nil
	}

	// Get service clients with fallback support
	nova, err := tryCreateServiceClient("nova", openstack.NewComputeV2)
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "create_client", err, true)
	}

	neutron, err := tryCreateServiceClient("neutron", openstack.NewNetworkV2)
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "create_client", err, true)
	}

	cinder, err := tryCreateServiceClient("cinder", openstack.NewBlockStorageV3)
	if err != nil {
		return nil, errors.NewOpenStackError("cinder", "create_client", err, true)
	}

	keystone, err := tryCreateServiceClient("keystone", openstack.NewIdentityV3)
	if err != nil {
		return nil, errors.NewOpenStackError("keystone", "create_client", err, true)
	}

	return &Client{
		provider: provider,
		nova:     nova,
		neutron:  neutron,
		cinder:   cinder,
		keystone: keystone,
	}, nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// GetNovaClient returns the Nova (Compute) service client
func (c *Client) GetNovaClient() *gophercloud.ServiceClient {
	return c.nova
}

// GetNeutronClient returns the Neutron (Networking) service client
func (c *Client) GetNeutronClient() *gophercloud.ServiceClient {
	return c.neutron
}

// GetCinderClient returns the Cinder (Block Storage) service client
func (c *Client) GetCinderClient() *gophercloud.ServiceClient {
	return c.cinder
}

// GetKeystoneClient returns the Keystone (Identity) service client
func (c *Client) GetKeystoneClient() *gophercloud.ServiceClient {
	return c.keystone
}

// RefreshToken refreshes the authentication token if needed
func (c *Client) RefreshToken() error {
	// Token refresh logic will be handled by gophercloud automatically
	// This method can be used for manual refresh if needed
	return nil
}

// Nova API wrappers

// ListServers lists all servers from all projects
// AllTenants is set to true to collect instances from all projects, not just the authenticated project
func (c *Client) ListServers() ([]servers.Server, error) {
	opts := servers.ListOpts{
		AllTenants: true, // List servers from all projects
	}
	allPages, err := servers.List(c.nova, opts).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "list_servers", err, true)
	}

	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "extract_servers", err, false)
	}

	return allServers, nil
}

// GetServer gets a specific server by ID
func (c *Client) GetServer(serverID string) (*servers.Server, error) {
	server, err := servers.Get(c.nova, serverID).Extract()
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "get_server", err, true)
	}
	return server, nil
}

// ServerDiagnostics represents server diagnostics data
type ServerDiagnostics struct {
	CPUUsagePercent float64
	MemoryUsageMB   int64
	MemoryTotalMB   int64
	DiskReadBytes   int64
	DiskWriteBytes  int64
	NetworkRxBytes  int64
	NetworkTxBytes  int64
}

// GetServerDiagnostics gets server diagnostics (CPU, memory, disk, network metrics)
// Note: This requires admin privileges or the server owner
// Diagnostics format varies by hypervisor (libvirt, vmware, etc.)
// Uses direct HTTP request since gophercloud may not have Diagnostics function in all versions
func (c *Client) GetServerDiagnostics(serverID string) (*ServerDiagnostics, error) {
	// Make direct HTTP request to Nova diagnostics endpoint
	// GET /servers/{server_id}/diagnostics
	url := c.nova.ServiceURL("servers", serverID, "diagnostics")
	
	var diagnostics map[string]interface{}
	resp, err := c.nova.Get(url, &diagnostics, nil)
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "get_server_diagnostics", err, true)
	}
	defer resp.Body.Close()

	// Parse diagnostics map to extract metrics
	// Diagnostics format varies by hypervisor, but typically includes:
	// - cpu: CPU usage information
	// - memory: Memory usage information
	// - disk: Disk I/O information
	// - network: Network I/O information
	
	metrics := &ServerDiagnostics{}

	// Extract CPU usage (format varies by hypervisor)
	if cpuData, ok := diagnostics["cpu"].(map[string]interface{}); ok {
		// Try different CPU metric formats
		if cpuPercent, ok := cpuData["cpu_percent"].(float64); ok {
			metrics.CPUUsagePercent = cpuPercent
		} else if cpuTime, ok := cpuData["cpu_time"].(float64); ok {
			// CPU time in nanoseconds - would need previous measurement to calculate percentage
			// For now, skip if we can't get percentage directly
			_ = cpuTime
		}
	}

	// Extract memory usage
	if memoryData, ok := diagnostics["memory"].(map[string]interface{}); ok {
		if memUsed, ok := memoryData["used"].(float64); ok {
			metrics.MemoryUsageMB = int64(memUsed / 1024 / 1024) // Convert bytes to MB
		}
		if memTotal, ok := memoryData["total"].(float64); ok {
			metrics.MemoryTotalMB = int64(memTotal / 1024 / 1024) // Convert bytes to MB
		}
	}

	// Extract disk I/O
	if diskData, ok := diagnostics["disk"].(map[string]interface{}); ok {
		if readBytes, ok := diskData["read_bytes"].(float64); ok {
			metrics.DiskReadBytes = int64(readBytes)
		}
		if writeBytes, ok := diskData["write_bytes"].(float64); ok {
			metrics.DiskWriteBytes = int64(writeBytes)
		}
	}

	// Extract network I/O
	if networkData, ok := diagnostics["network"].(map[string]interface{}); ok {
		// Network data is typically a map of interface names
		var totalRx, totalTx int64
		for _, ifaceData := range networkData {
			if iface, ok := ifaceData.(map[string]interface{}); ok {
				if rxBytes, ok := iface["rx_bytes"].(float64); ok {
					totalRx += int64(rxBytes)
				}
				if txBytes, ok := iface["tx_bytes"].(float64); ok {
					totalTx += int64(txBytes)
				}
			}
		}
		metrics.NetworkRxBytes = totalRx
		metrics.NetworkTxBytes = totalTx
	}

	return metrics, nil
}

// ListHypervisors lists all hypervisors
func (c *Client) ListHypervisors() ([]hypervisors.Hypervisor, error) {
	allPages, err := hypervisors.List(c.nova, hypervisors.ListOpts{}).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "list_hypervisors", err, true)
	}

	allHypervisors, err := hypervisors.ExtractHypervisors(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "extract_hypervisors", err, false)
	}

	return allHypervisors, nil
}

// ListFlavors lists all flavors
func (c *Client) ListFlavors() ([]flavors.Flavor, error) {
	allPages, err := flavors.ListDetail(c.nova, flavors.ListOpts{}).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "list_flavors", err, true)
	}

	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("nova", "extract_flavors", err, false)
	}

	return allFlavors, nil
}

// Neutron API wrappers

// ListNetworks lists all networks from all projects
// Note: Neutron doesn't have AllTenants option, but omitting TenantID allows admin users to see all projects
func (c *Client) ListNetworks() ([]networks.Network, error) {
	// Empty ListOpts means no tenant filtering - admin users will see all networks
	opts := networks.ListOpts{}
	allPages, err := networks.List(c.neutron, opts).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "list_networks", err, true)
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "extract_networks", err, false)
	}

	return allNetworks, nil
}

// ListPorts lists all ports
func (c *Client) ListPorts(opts ports.ListOpts) ([]ports.Port, error) {
	allPages, err := ports.List(c.neutron, opts).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "list_ports", err, true)
	}

	allPorts, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "extract_ports", err, false)
	}

	return allPorts, nil
}

// ListRouters lists all routers from all projects
// Note: Neutron doesn't have AllTenants option, but omitting TenantID allows admin users to see all projects
func (c *Client) ListRouters() ([]routers.Router, error) {
	// Empty ListOpts means no tenant filtering - admin users will see all routers
	opts := routers.ListOpts{}
	allPages, err := routers.List(c.neutron, opts).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "list_routers", err, true)
	}

	allRouters, err := routers.ExtractRouters(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("neutron", "extract_routers", err, false)
	}

	return allRouters, nil
}

// Cinder API wrappers

// ListVolumes lists all volumes from all projects
// AllTenants is set to true to collect volumes from all projects, not just the authenticated project
func (c *Client) ListVolumes() ([]volumes.Volume, error) {
	opts := volumes.ListOpts{
		AllTenants: true, // List volumes from all projects
	}
	allPages, err := volumes.List(c.cinder, opts).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("cinder", "list_volumes", err, true)
	}

	allVolumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("cinder", "extract_volumes", err, false)
	}

	return allVolumes, nil
}

// Keystone API wrappers

// ListProjects lists all projects
func (c *Client) ListProjects() ([]projects.Project, error) {
	allPages, err := projects.List(c.keystone, projects.ListOpts{}).AllPages()
	if err != nil {
		return nil, errors.NewOpenStackError("keystone", "list_projects", err, true)
	}

	allProjects, err := projects.ExtractProjects(allPages)
	if err != nil {
		return nil, errors.NewOpenStackError("keystone", "extract_projects", err, false)
	}

	return allProjects, nil
}

