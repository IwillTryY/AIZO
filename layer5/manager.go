package layer5

import (
	"context"
	"time"
)

// Manager orchestrates all Layer 5 components
type Manager struct {
	runtime        *ContainerRuntime
	imageManager   *ImageManager
	networkManager *NetworkManager
	volumeManager  *VolumeManager
}

// ManagerConfig configures the Layer 5 manager
type ManagerConfig struct {
	DataRoot          string
	EnableMetrics     bool
	MetricsInterval   time.Duration
	DefaultNetwork    string
	DefaultRuntime    string
}

// NewManager creates a new Layer 5 manager
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = &ManagerConfig{
			DataRoot:        "/var/lib/realityos",
			EnableMetrics:   true,
			MetricsInterval: 30 * time.Second,
			DefaultNetwork:  "bridge",
			DefaultRuntime:  "realityos",
		}
	}

	return &Manager{
		runtime:        NewContainerRuntime(),
		imageManager:   NewImageManager(),
		networkManager: NewNetworkManager(),
		volumeManager:  NewVolumeManager(),
	}
}

// Start starts the Layer 5 manager
func (m *Manager) Start(ctx context.Context) error {
	// In a real implementation, this would:
	// 1. Initialize storage
	// 2. Load existing containers
	// 3. Start monitoring
	// 4. Set up networking
	return nil
}

// GetRuntime returns the container runtime
func (m *Manager) GetRuntime() *ContainerRuntime {
	return m.runtime
}

// GetImageManager returns the image manager
func (m *Manager) GetImageManager() *ImageManager {
	return m.imageManager
}

// GetNetworkManager returns the network manager
func (m *Manager) GetNetworkManager() *NetworkManager {
	return m.networkManager
}

// GetVolumeManager returns the volume manager
func (m *Manager) GetVolumeManager() *VolumeManager {
	return m.volumeManager
}

// CreateContainer creates a new container
func (m *Manager) CreateContainer(ctx context.Context, config *ContainerConfig, name string) (*Container, error) {
	return m.runtime.CreateContainer(ctx, config, name)
}

// StartContainer starts a container
func (m *Manager) StartContainer(ctx context.Context, containerID string) error {
	return m.runtime.StartContainer(ctx, containerID)
}

// StopContainer stops a container
func (m *Manager) StopContainer(ctx context.Context, containerID string, timeout int) error {
	return m.runtime.StopContainer(ctx, containerID, timeout)
}

// RemoveContainer removes a container
func (m *Manager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return m.runtime.RemoveContainer(ctx, containerID, force)
}

// ListContainers lists all containers
func (m *Manager) ListContainers(ctx context.Context, all bool) ([]*Container, error) {
	return m.runtime.ListContainers(ctx, all)
}

// GetContainer retrieves a container
func (m *Manager) GetContainer(ctx context.Context, containerID string) (*Container, error) {
	return m.runtime.GetContainer(ctx, containerID)
}

// PullImage pulls an image
func (m *Manager) PullImage(ctx context.Context, imageName string) (*Image, error) {
	return m.imageManager.PullImage(ctx, imageName)
}

// BuildImage builds an image
func (m *Manager) BuildImage(ctx context.Context, dockerfile string, tags []string) (*Image, error) {
	return m.imageManager.BuildImage(ctx, dockerfile, tags)
}

// ListImages lists all images
func (m *Manager) ListImages(ctx context.Context) ([]*Image, error) {
	return m.imageManager.ListImages(ctx)
}

// RemoveImage removes an image
func (m *Manager) RemoveImage(ctx context.Context, imageID string, force bool) error {
	return m.imageManager.RemoveImage(ctx, imageID, force)
}

// CreateNetwork creates a network
func (m *Manager) CreateNetwork(ctx context.Context, name string, driver string, internal bool, labels map[string]string) (*Network, error) {
	return m.networkManager.CreateNetwork(ctx, name, driver, internal, labels)
}

// ListNetworks lists all networks
func (m *Manager) ListNetworks(ctx context.Context) ([]*Network, error) {
	return m.networkManager.ListNetworks(ctx)
}

// RemoveNetwork removes a network
func (m *Manager) RemoveNetwork(ctx context.Context, networkID string) error {
	return m.networkManager.RemoveNetwork(ctx, networkID)
}

// CreateVolume creates a volume
func (m *Manager) CreateVolume(ctx context.Context, name string, driver string, labels map[string]string) (*Volume, error) {
	return m.volumeManager.CreateVolume(ctx, name, driver, labels)
}

// ListVolumes lists all volumes
func (m *Manager) ListVolumes(ctx context.Context) ([]*Volume, error) {
	return m.volumeManager.ListVolumes(ctx)
}

// RemoveVolume removes a volume
func (m *Manager) RemoveVolume(ctx context.Context, name string, force bool) error {
	return m.volumeManager.RemoveVolume(ctx, name, force)
}

// GetStats returns Layer 5 statistics
func (m *Manager) GetStats(ctx context.Context) (*ManagerStats, error) {
	containers, _ := m.runtime.ListContainers(ctx, true)
	images, _ := m.imageManager.ListImages(ctx)
	networks, _ := m.networkManager.ListNetworks(ctx)
	volumes, _ := m.volumeManager.ListVolumes(ctx)

	stats := &ManagerStats{
		TotalContainers: len(containers),
		RunningContainers: 0,
		TotalImages:     len(images),
		TotalNetworks:   len(networks),
		TotalVolumes:    len(volumes),
		ByStatus:        make(map[ContainerStatus]int),
	}

	for _, container := range containers {
		stats.ByStatus[container.Status]++
		if container.State.Running {
			stats.RunningContainers++
		}
	}

	return stats, nil
}

// ManagerStats contains Layer 5 statistics
type ManagerStats struct {
	TotalContainers   int
	RunningContainers int
	TotalImages       int
	TotalNetworks     int
	TotalVolumes      int
	ByStatus          map[ContainerStatus]int
}

// RunContainer is a convenience method to create and start a container
func (m *Manager) RunContainer(ctx context.Context, config *ContainerConfig, name string) (*Container, error) {
	// Create container
	container, err := m.CreateContainer(ctx, config, name)
	if err != nil {
		return nil, err
	}

	// Start container
	if err := m.StartContainer(ctx, container.ID); err != nil {
		return nil, err
	}

	return container, nil
}

// GetContainerStats gets resource usage statistics
func (m *Manager) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	return m.runtime.GetContainerStats(ctx, containerID)
}

// ExecInContainer executes a command in a container
func (m *Manager) ExecInContainer(ctx context.Context, config *ExecConfig) error {
	return m.runtime.ExecInContainer(ctx, config)
}

// GetContainerLogs retrieves container logs
func (m *Manager) GetContainerLogs(ctx context.Context, containerID string, config *ContainerLogs) ([]string, error) {
	return m.runtime.GetContainerLogs(ctx, containerID, config)
}

// PauseContainer pauses a container
func (m *Manager) PauseContainer(ctx context.Context, containerID string) error {
	return m.runtime.PauseContainer(ctx, containerID)
}

// UnpauseContainer unpauses a container
func (m *Manager) UnpauseContainer(ctx context.Context, containerID string) error {
	return m.runtime.UnpauseContainer(ctx, containerID)
}

// RestartContainer restarts a container
func (m *Manager) RestartContainer(ctx context.Context, containerID string, timeout int) error {
	return m.runtime.RestartContainer(ctx, containerID, timeout)
}

// ConnectContainerToNetwork connects a container to a network
func (m *Manager) ConnectContainerToNetwork(ctx context.Context, networkID string, containerID string, config *EndpointConfig) error {
	return m.networkManager.ConnectContainer(ctx, networkID, containerID, config)
}

// DisconnectContainerFromNetwork disconnects a container from a network
func (m *Manager) DisconnectContainerFromNetwork(ctx context.Context, networkID string, containerID string, force bool) error {
	return m.networkManager.DisconnectContainer(ctx, networkID, containerID, force)
}

// Prune removes unused resources
func (m *Manager) Prune(ctx context.Context) (*PruneResult, error) {
	result := &PruneResult{
		ContainersRemoved: make([]string, 0),
		ImagesRemoved:     make([]string, 0),
		NetworksRemoved:   make([]string, 0),
		VolumesRemoved:    make([]string, 0),
	}

	// Prune stopped containers
	containers, _ := m.ListContainers(ctx, true)
	for _, container := range containers {
		if !container.State.Running {
			if err := m.RemoveContainer(ctx, container.ID, false); err == nil {
				result.ContainersRemoved = append(result.ContainersRemoved, container.ID)
			}
		}
	}

	// Prune unused images
	removedImages, _ := m.imageManager.PruneImages(ctx)
	result.ImagesRemoved = removedImages

	// Prune unused networks
	removedNetworks, _ := m.networkManager.PruneNetworks(ctx)
	result.NetworksRemoved = removedNetworks

	// Prune unused volumes
	removedVolumes, _ := m.volumeManager.PruneVolumes(ctx)
	result.VolumesRemoved = removedVolumes

	return result, nil
}

// PruneResult contains the results of a prune operation
type PruneResult struct {
	ContainersRemoved []string `json:"containers_removed"`
	ImagesRemoved     []string `json:"images_removed"`
	NetworksRemoved   []string `json:"networks_removed"`
	VolumesRemoved    []string `json:"volumes_removed"`
	SpaceReclaimed    int64    `json:"space_reclaimed"`
}
