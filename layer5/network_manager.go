package layer5

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NetworkManager manages container networks
type NetworkManager struct {
	networks map[string]*Network
	mu       sync.RWMutex
}

// NewNetworkManager creates a new network manager
func NewNetworkManager() *NetworkManager {
	manager := &NetworkManager{
		networks: make(map[string]*Network),
	}

	// Create default networks
	manager.createDefaultNetworks()

	return manager
}

// createDefaultNetworks creates default networks
func (m *NetworkManager) createDefaultNetworks() {
	// Bridge network
	bridge := &Network{
		ID:         "bridge",
		Name:       "bridge",
		Driver:     "bridge",
		Scope:      "local",
		Internal:   false,
		Attachable: true,
		IPAM: &IPAMConfig{
			Driver: "default",
			Config: []IPAMPool{
				{
					Subnet:  "172.17.0.0/16",
					Gateway: "172.17.0.1",
				},
			},
		},
		Containers: make(map[string]*EndpointConfig),
		Options:    make(map[string]string),
		Labels:     make(map[string]string),
		CreatedAt:  time.Now(),
	}
	m.networks["bridge"] = bridge

	// Host network
	host := &Network{
		ID:         "host",
		Name:       "host",
		Driver:     "host",
		Scope:      "local",
		Internal:   false,
		Attachable: false,
		Containers: make(map[string]*EndpointConfig),
		Options:    make(map[string]string),
		Labels:     make(map[string]string),
		CreatedAt:  time.Now(),
	}
	m.networks["host"] = host

	// None network
	none := &Network{
		ID:         "none",
		Name:       "none",
		Driver:     "null",
		Scope:      "local",
		Internal:   false,
		Attachable: false,
		Containers: make(map[string]*EndpointConfig),
		Options:    make(map[string]string),
		Labels:     make(map[string]string),
		CreatedAt:  time.Now(),
	}
	m.networks["none"] = none
}

// CreateNetwork creates a new network
func (m *NetworkManager) CreateNetwork(ctx context.Context, name string, driver string, internal bool, labels map[string]string) (*Network, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if network already exists
	for _, net := range m.networks {
		if net.Name == name {
			return nil, fmt.Errorf("network already exists: %s", name)
		}
	}

	if driver == "" {
		driver = "bridge"
	}

	id := uuid.New().String()

	network := &Network{
		ID:         id,
		Name:       name,
		Driver:     driver,
		Scope:      "local",
		Internal:   internal,
		Attachable: true,
		IPAM: &IPAMConfig{
			Driver: "default",
			Config: []IPAMPool{
				{
					Subnet:  "172.18.0.0/16",
					Gateway: "172.18.0.1",
				},
			},
		},
		Containers: make(map[string]*EndpointConfig),
		Options:    make(map[string]string),
		Labels:     labels,
		CreatedAt:  time.Now(),
	}

	m.networks[id] = network

	return network, nil
}

// GetNetwork retrieves a network
func (m *NetworkManager) GetNetwork(ctx context.Context, networkID string) (*Network, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	network, exists := m.networks[networkID]
	if !exists {
		return nil, fmt.Errorf("network not found: %s", networkID)
	}

	return network, nil
}

// ListNetworks lists all networks
func (m *NetworkManager) ListNetworks(ctx context.Context) ([]*Network, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	networks := make([]*Network, 0, len(m.networks))
	for _, network := range m.networks {
		networks = append(networks, network)
	}

	return networks, nil
}

// RemoveNetwork removes a network
func (m *NetworkManager) RemoveNetwork(ctx context.Context, networkID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	network, exists := m.networks[networkID]
	if !exists {
		return fmt.Errorf("network not found: %s", networkID)
	}

	// Cannot remove default networks
	if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
		return fmt.Errorf("cannot remove default network: %s", network.Name)
	}

	// Check if network has containers
	if len(network.Containers) > 0 {
		return fmt.Errorf("network has active endpoints: %s", networkID)
	}

	delete(m.networks, networkID)

	return nil
}

// ConnectContainer connects a container to a network
func (m *NetworkManager) ConnectContainer(ctx context.Context, networkID string, containerID string, config *EndpointConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	network, exists := m.networks[networkID]
	if !exists {
		return fmt.Errorf("network not found: %s", networkID)
	}

	// Check if container already connected
	if _, exists := network.Containers[containerID]; exists {
		return fmt.Errorf("container already connected to network: %s", containerID)
	}

	if config == nil {
		config = &EndpointConfig{
			EndpointID: uuid.New().String(),
		}
	}

	// Assign IP address if not provided
	if config.IPAddress == "" && network.IPAM != nil && len(network.IPAM.Config) > 0 {
		// Simple IP allocation (in real implementation, use proper IPAM)
		config.IPAddress = fmt.Sprintf("172.18.0.%d", len(network.Containers)+2)
		config.Gateway = network.IPAM.Config[0].Gateway
		config.IPPrefixLen = 16
	}

	network.Containers[containerID] = config

	return nil
}

// DisconnectContainer disconnects a container from a network
func (m *NetworkManager) DisconnectContainer(ctx context.Context, networkID string, containerID string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	network, exists := m.networks[networkID]
	if !exists {
		return fmt.Errorf("network not found: %s", networkID)
	}

	if _, exists := network.Containers[containerID]; !exists {
		return fmt.Errorf("container not connected to network: %s", containerID)
	}

	delete(network.Containers, containerID)

	return nil
}

// PruneNetworks removes unused networks
func (m *NetworkManager) PruneNetworks(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := make([]string, 0)

	for id, network := range m.networks {
		// Skip default networks
		if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
			continue
		}

		// Remove if no containers
		if len(network.Containers) == 0 {
			delete(m.networks, id)
			removed = append(removed, id)
		}
	}

	return removed, nil
}
