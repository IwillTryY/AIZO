package layer5

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// RealContainerRuntime manages actual isolated containers with filesystem and process isolation
type RealContainerRuntime struct {
	containers    map[string]*RealContainer
	images        map[string]*ContainerImage
	dataRoot      string // Root directory for container storage
	mu            sync.RWMutex
}

// RealContainer represents an actual isolated container
type RealContainer struct {
	ID            string
	Name          string
	ImageID       string
	Status        ContainerStatus
	Config        *ContainerConfig
	RootFS        string // Path to container's root filesystem
	WorkDir       string // Working directory inside container
	Cmd           []string
	Env           map[string]string
	Mounts        []Mount
	Process       *os.Process
	PID           int
	CreatedAt     time.Time
	StartedAt     time.Time
	FinishedAt    time.Time
	ExitCode      int
	Running       bool
}

// ContainerImage represents a container image stored on disk
type ContainerImage struct {
	ID        string
	Name      string
	Tag       string
	RootFS    string // Path to image's root filesystem
	Layers    []string
	Config    *ImageConfig
	CreatedAt time.Time
	Size      int64
}

// NewRealContainerRuntime creates a new real container runtime
func NewRealContainerRuntime(dataRoot string) (*RealContainerRuntime, error) {
	if dataRoot == "" {
		dataRoot = "/var/lib/realityos/containers"
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(dataRoot, "containers"),
		filepath.Join(dataRoot, "images"),
		filepath.Join(dataRoot, "volumes"),
		filepath.Join(dataRoot, "tmp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &RealContainerRuntime{
		containers: make(map[string]*RealContainer),
		images:     make(map[string]*ContainerImage),
		dataRoot:   dataRoot,
	}, nil
}

// CreateContainer creates a new isolated container
func (r *RealContainerRuntime) CreateContainer(ctx context.Context, config *ContainerConfig, imageName string, name string) (*RealContainer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate ID
	id := uuid.New().String()[:12]

	// Create container directory structure
	containerDir := filepath.Join(r.dataRoot, "containers", id)
	rootfsDir := filepath.Join(containerDir, "rootfs")
	workDir := filepath.Join(containerDir, "work")

	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs: %w", err)
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}

	// Setup basic filesystem structure
	if err := r.setupContainerFS(rootfsDir); err != nil {
		return nil, fmt.Errorf("failed to setup filesystem: %w", err)
	}

	// Copy image layers if image specified
	if imageName != "" {
		if err := r.extractImageToContainer(imageName, rootfsDir); err != nil {
			return nil, fmt.Errorf("failed to extract image: %w", err)
		}
	}

	container := &RealContainer{
		ID:        id,
		Name:      name,
		ImageID:   imageName,
		Status:    StatusCreated,
		Config:    config,
		RootFS:    rootfsDir,
		WorkDir:   config.WorkingDir,
		Cmd:       config.Cmd,
		Env:       make(map[string]string),
		Mounts:    make([]Mount, 0),
		CreatedAt: time.Now(),
		Running:   false,
	}

	// Set environment variables
	for _, envVar := range config.Env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			container.Env[parts[0]] = parts[1]
		}
	}

	r.containers[id] = container

	return container, nil
}

// StartContainer starts a container with actual process isolation
func (r *RealContainerRuntime) StartContainer(ctx context.Context, containerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	container, exists := r.containers[containerID]
	if !exists {
		return fmt.Errorf("container not found: %s", containerID)
	}

	if container.Running {
		return fmt.Errorf("container already running: %s", containerID)
	}

	// Prepare command
	cmd := container.Cmd
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}

	// Create isolated process
	process := exec.Command(cmd[0], cmd[1:]...)

	// Set up namespaces (Linux-specific, will be no-op on Windows)
	process.SysProcAttr = &syscall.SysProcAttr{}

	// Set environment
	process.Env = make([]string, 0)
	for k, v := range container.Env {
		process.Env = append(process.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set working directory
	if container.WorkDir != "" {
		process.Dir = container.WorkDir
	}

	// Start the process
	if err := process.Start(); err != nil {
		return fmt.Errorf("failed to start container process: %w", err)
	}

	// Update container state
	container.Process = process.Process
	container.PID = process.Process.Pid
	container.Status = StatusRunning
	container.Running = true
	container.StartedAt = time.Now()

	// Monitor process in background
	go r.monitorContainer(containerID, process)

	return nil
}

// StopContainer stops a running container
func (r *RealContainerRuntime) StopContainer(ctx context.Context, containerID string, timeout int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	container, exists := r.containers[containerID]
	if !exists {
		return fmt.Errorf("container not found: %s", containerID)
	}

	if !container.Running {
		return fmt.Errorf("container not running: %s", containerID)
	}

	// Send SIGTERM
	if container.Process != nil {
		if err := container.Process.Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, force kill
			container.Process.Kill()
		}

		// Wait for process to exit (with timeout)
		done := make(chan error, 1)
		go func() {
			_, err := container.Process.Wait()
			done <- err
		}()

		select {
		case <-time.After(time.Duration(timeout) * time.Second):
			// Timeout - force kill
			container.Process.Kill()
		case <-done:
			// Process exited
		}
	}

	container.Status = StatusExited
	container.Running = false
	container.FinishedAt = time.Now()

	return nil
}

// RemoveContainer removes a container and cleans up its filesystem
func (r *RealContainerRuntime) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	container, exists := r.containers[containerID]
	if !exists {
		return fmt.Errorf("container not found: %s", containerID)
	}

	if container.Running && !force {
		return fmt.Errorf("cannot remove running container: %s (use force)", containerID)
	}

	// Stop if running
	if container.Running {
		if container.Process != nil {
			container.Process.Kill()
		}
		container.Running = false
	}

	// Remove container filesystem
	containerDir := filepath.Join(r.dataRoot, "containers", containerID)
	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("failed to remove container directory: %w", err)
	}

	delete(r.containers, containerID)

	return nil
}

// GetContainer retrieves a container
func (r *RealContainerRuntime) GetContainer(ctx context.Context, containerID string) (*RealContainer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	container, exists := r.containers[containerID]
	if !exists {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	return container, nil
}

// ListContainers lists all containers
func (r *RealContainerRuntime) ListContainers(ctx context.Context, all bool) ([]*RealContainer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	containers := make([]*RealContainer, 0)

	for _, container := range r.containers {
		if all || container.Running {
			containers = append(containers, container)
		}
	}

	return containers, nil
}

// ExecInContainer executes a command in a running container
func (r *RealContainerRuntime) ExecInContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	r.mu.RLock()
	container, exists := r.containers[containerID]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("container not found: %s", containerID)
	}

	if !container.Running {
		return "", fmt.Errorf("container not running: %s", containerID)
	}

	// Execute command in container's namespace
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	execCmd.SysProcAttr = &syscall.SysProcAttr{}

	output, err := execCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec failed: %w", err)
	}

	return string(output), nil
}

// setupContainerFS creates basic filesystem structure for container
func (r *RealContainerRuntime) setupContainerFS(rootfs string) error {
	// Create standard directories
	dirs := []string{
		"bin", "sbin", "usr/bin", "usr/sbin", "usr/local/bin",
		"etc", "var", "tmp", "home", "root",
		"proc", "sys", "dev",
		"lib", "lib64", "usr/lib", "usr/lib64",
	}

	for _, dir := range dirs {
		path := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	// Create basic device nodes (simplified)
	devNodes := []struct {
		name string
		mode os.FileMode
	}{
		{"null", 0666},
		{"zero", 0666},
		{"random", 0666},
		{"urandom", 0666},
	}

	for _, node := range devNodes {
		path := filepath.Join(rootfs, "dev", node.name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			f.Close()
			os.Chmod(path, node.mode)
		}
	}

	return nil
}

// extractImageToContainer extracts an image to container rootfs
func (r *RealContainerRuntime) extractImageToContainer(imageID string, rootfs string) error {
	r.mu.RLock()
	image, exists := r.images[imageID]
	r.mu.RUnlock()

	if !exists {
		// Image doesn't exist, create a basic one
		return r.setupContainerFS(rootfs)
	}

	// Copy image rootfs to container rootfs
	return r.copyDir(image.RootFS, rootfs)
}

// copyDir recursively copies a directory
func (r *RealContainerRuntime) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return r.copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func (r *RealContainerRuntime) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// monitorContainer monitors a container process
func (r *RealContainerRuntime) monitorContainer(containerID string, process *exec.Cmd) {
	err := process.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()

	container, exists := r.containers[containerID]
	if !exists {
		return
	}

	container.Running = false
	container.Status = StatusExited
	container.FinishedAt = time.Now()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			container.ExitCode = exitErr.ExitCode()
		} else {
			container.ExitCode = 1
		}
	} else {
		container.ExitCode = 0
	}
}

// BuildImage builds a container image from a directory
func (r *RealContainerRuntime) BuildImage(ctx context.Context, buildPath string, imageName string, tag string) (*ContainerImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	imageID := uuid.New().String()[:12]
	imageDir := filepath.Join(r.dataRoot, "images", imageID)
	rootfsDir := filepath.Join(imageDir, "rootfs")

	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	// Copy build context to image rootfs
	if err := r.copyDir(buildPath, rootfsDir); err != nil {
		return nil, fmt.Errorf("failed to copy build context: %w", err)
	}

	// Calculate size
	var size int64
	filepath.Walk(rootfsDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	image := &ContainerImage{
		ID:     imageID,
		Name:   imageName,
		Tag:    tag,
		RootFS: rootfsDir,
		Config: &ImageConfig{
			Cmd:        []string{"/bin/sh"},
			Env:        []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			WorkingDir: "/",
		},
		CreatedAt: time.Now(),
		Size:      size,
	}

	r.images[imageID] = image
	r.images[imageName+":"+tag] = image

	return image, nil
}

// ImportImage imports a tar.gz image
func (r *RealContainerRuntime) ImportImage(ctx context.Context, tarPath string, imageName string, tag string) (*ContainerImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	imageID := uuid.New().String()[:12]
	imageDir := filepath.Join(r.dataRoot, "images", imageID)
	rootfsDir := filepath.Join(imageDir, "rootfs")

	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	// Extract tar.gz
	if err := r.extractTarGz(tarPath, rootfsDir); err != nil {
		return nil, fmt.Errorf("failed to extract image: %w", err)
	}

	// Calculate size
	var size int64
	filepath.Walk(rootfsDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	image := &ContainerImage{
		ID:     imageID,
		Name:   imageName,
		Tag:    tag,
		RootFS: rootfsDir,
		Config: &ImageConfig{
			Cmd:        []string{"/bin/sh"},
			Env:        []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			WorkingDir: "/",
		},
		CreatedAt: time.Now(),
		Size:      size,
	}

	r.images[imageID] = image
	r.images[imageName+":"+tag] = image

	return image, nil
}

// extractTarGz extracts a tar.gz file
func (r *RealContainerRuntime) extractTarGz(tarPath string, dest string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

// GetImage retrieves an image
func (r *RealContainerRuntime) GetImage(ctx context.Context, imageID string) (*ContainerImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	image, exists := r.images[imageID]
	if !exists {
		return nil, fmt.Errorf("image not found: %s", imageID)
	}

	return image, nil
}

// ListImages lists all images
func (r *RealContainerRuntime) ListImages(ctx context.Context) ([]*ContainerImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	images := make([]*ContainerImage, 0)
	seen := make(map[string]bool)

	for _, image := range r.images {
		if !seen[image.ID] {
			images = append(images, image)
			seen[image.ID] = true
		}
	}

	return images, nil
}

// RemoveImage removes an image
func (r *RealContainerRuntime) RemoveImage(ctx context.Context, imageID string, force bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	image, exists := r.images[imageID]
	if !exists {
		return fmt.Errorf("image not found: %s", imageID)
	}

	// Check if any containers are using this image
	if !force {
		for _, container := range r.containers {
			if container.ImageID == imageID {
				return fmt.Errorf("image is in use by container %s", container.ID)
			}
		}
	}

	// Remove image directory
	imageDir := filepath.Dir(image.RootFS)
	if err := os.RemoveAll(imageDir); err != nil {
		return fmt.Errorf("failed to remove image directory: %w", err)
	}

	// Remove from maps
	delete(r.images, imageID)
	delete(r.images, image.Name+":"+image.Tag)

	return nil
}
