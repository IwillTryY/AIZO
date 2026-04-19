package layer5

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// WSL2VolumeManager manages persistent volumes for WSL2 containers
type WSL2VolumeManager struct {
	distro   string
	dataRoot string
	volumes  map[string]*WSL2Volume
	mu       sync.RWMutex
}

// WSL2Volume represents a persistent volume
type WSL2Volume struct {
	Name       string
	Path       string // Path inside WSL2
	Size       int64  // Bytes used
	MountedBy  []string // Container IDs
	CreatedAt  string
}

// NewWSL2VolumeManager creates a new volume manager
func NewWSL2VolumeManager(distro, dataRoot string) *WSL2VolumeManager {
	vm := &WSL2VolumeManager{
		distro:   distro,
		dataRoot: dataRoot + "/volumes",
		volumes:  make(map[string]*WSL2Volume),
	}
	// Ensure volumes directory exists
	vm.wslRun("mkdir", "-p", vm.dataRoot)
	return vm
}

// CreateVolume creates a new persistent volume
func (vm *WSL2VolumeManager) CreateVolume(name string) (*WSL2Volume, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.volumes[name]; exists {
		return nil, fmt.Errorf("volume %s already exists", name)
	}

	path := vm.dataRoot + "/" + name
	if err := vm.wslRun("mkdir", "-p", path); err != nil {
		return nil, fmt.Errorf("failed to create volume directory: %w", err)
	}

	// Get creation time
	createdAt, _ := vm.wslOutput("date", "+%Y-%m-%dT%H:%M:%S")

	vol := &WSL2Volume{
		Name:      name,
		Path:      path,
		MountedBy: make([]string, 0),
		CreatedAt: strings.TrimSpace(createdAt),
	}
	vm.volumes[name] = vol
	return vol, nil
}

// GetVolume returns a volume by name
func (vm *WSL2VolumeManager) GetVolume(name string) (*WSL2Volume, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	vol, exists := vm.volumes[name]
	if !exists {
		return nil, fmt.Errorf("volume %s not found", name)
	}
	return vol, nil
}

// ListVolumes returns all volumes
func (vm *WSL2VolumeManager) ListVolumes() []*WSL2Volume {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	vols := make([]*WSL2Volume, 0, len(vm.volumes))
	for _, v := range vm.volumes {
		vols = append(vols, v)
	}
	return vols
}

// MountVolume bind-mounts a volume into a container's rootfs
func (vm *WSL2VolumeManager) MountVolume(volumeName, containerID, rootfs, mountPoint string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vol, exists := vm.volumes[volumeName]
	if !exists {
		return fmt.Errorf("volume %s not found", volumeName)
	}

	// Create mount point inside container rootfs
	destPath := rootfs + mountPoint
	if err := vm.wslRun("mkdir", "-p", destPath); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Bind mount the volume into the container rootfs
	err := vm.wslRun("mount", "--bind", vol.Path, destPath)
	if err != nil {
		// Fallback: copy files instead of bind mount (works without root)
		vm.wslRun("cp", "-a", vol.Path+"/.", destPath+"/")
	}

	vol.MountedBy = append(vol.MountedBy, containerID)
	return nil
}

// UnmountVolume unmounts a volume from a container
func (vm *WSL2VolumeManager) UnmountVolume(volumeName, containerID, rootfs, mountPoint string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vol, exists := vm.volumes[volumeName]
	if !exists {
		return fmt.Errorf("volume %s not found", volumeName)
	}

	destPath := rootfs + mountPoint

	// Try to unmount
	vm.wslRun("umount", destPath)

	// Sync data back to volume (for copy-based fallback)
	vm.wslRun("cp", "-a", destPath+"/.", vol.Path+"/")

	// Remove container from mounted list
	newMounted := make([]string, 0)
	for _, id := range vol.MountedBy {
		if id != containerID {
			newMounted = append(newMounted, id)
		}
	}
	vol.MountedBy = newMounted

	return nil
}

// RemoveVolume removes a volume (must not be mounted)
func (vm *WSL2VolumeManager) RemoveVolume(name string, force bool) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vol, exists := vm.volumes[name]
	if !exists {
		return fmt.Errorf("volume %s not found", name)
	}

	if len(vol.MountedBy) > 0 && !force {
		return fmt.Errorf("volume %s is mounted by %d containers", name, len(vol.MountedBy))
	}

	if err := vm.wslRun("rm", "-rf", vol.Path); err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}

	delete(vm.volumes, name)
	return nil
}

// GetVolumeSize returns the size of a volume in bytes
func (vm *WSL2VolumeManager) GetVolumeSize(name string) (int64, error) {
	vol, exists := vm.volumes[name]
	if !exists {
		return 0, fmt.Errorf("volume %s not found", name)
	}

	out, err := vm.wslOutput("du", "-sb", vol.Path)
	if err != nil {
		return 0, err
	}

	var size int64
	fmt.Sscanf(strings.TrimSpace(out), "%d", &size)
	return size, nil
}

func (vm *WSL2VolumeManager) wslRun(args ...string) error {
	wslArgs := append([]string{"-d", vm.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func (vm *WSL2VolumeManager) wslOutput(args ...string) (string, error) {
	wslArgs := append([]string{"-d", vm.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return string(out), err
}
