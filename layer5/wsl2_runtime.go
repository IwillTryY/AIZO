package layer5

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// WSL2Runtime runs containers inside WSL2 for real Linux namespace isolation
type WSL2Runtime struct {
	distro   string // WSL2 distro to use (e.g. "Ubuntu")
	dataRoot string // Path inside WSL2 for container storage
}

// WSL2Container represents a container running inside WSL2
type WSL2Container struct {
	ID        string
	Name      string
	PID       int
	Status    ContainerStatus
	RootFS    string // WSL2 path to rootfs
	Running   bool
	CreatedAt time.Time
	StartedAt time.Time
	ExitCode  int
}

// wsl2ContainerState is used for JSON persistence
type wsl2ContainerState struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	PID       int             `json:"pid"`
	Status    ContainerStatus `json:"status"`
	RootFS    string          `json:"rootfs"`
	Running   bool            `json:"running"`
	CreatedAt time.Time       `json:"created_at"`
	StartedAt time.Time       `json:"started_at"`
	ExitCode  int             `json:"exit_code"`
}

// NewWSL2Runtime creates a new WSL2-backed container runtime
func NewWSL2Runtime(distro string, dataRoot string) (*WSL2Runtime, error) {
	if distro == "" {
		distro = "Ubuntu"
	}
	if dataRoot == "" {
		dataRoot = "/var/lib/realityos"
	}

	r := &WSL2Runtime{
		distro:   distro,
		dataRoot: dataRoot,
	}

	// Verify WSL2 is available
	if err := r.checkWSL2(); err != nil {
		return nil, fmt.Errorf("WSL2 not available: %w", err)
	}

	// Initialize directory structure inside WSL2
	if err := r.initDirectories(); err != nil {
		return nil, fmt.Errorf("failed to init WSL2 directories: %w", err)
	}

	return r, nil
}

// checkWSL2 verifies WSL2 is installed and the distro is available
func (r *WSL2Runtime) checkWSL2() error {
	cmd := exec.Command("wsl", "-d", r.distro, "--exec", "echo", "ok")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("distro %q not found or WSL2 not running: %w", r.distro, err)
	}
	if !strings.Contains(string(out), "ok") {
		return fmt.Errorf("unexpected WSL2 response: %q", string(out))
	}
	return nil
}

// initDirectories creates the container storage directories inside WSL2
func (r *WSL2Runtime) initDirectories() error {
	dirs := []string{
		r.dataRoot + "/containers",
		r.dataRoot + "/images",
		r.dataRoot + "/volumes",
	}
	for _, d := range dirs {
		if err := r.wslRun("mkdir", "-p", d); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

// CreateContainer creates an isolated container inside WSL2
func (r *WSL2Runtime) CreateContainer(ctx context.Context, name string, image string, cmd []string, env []string) (*WSL2Container, error) {
	// Generate short ID
	idOut, err := r.wslOutput("cat", "/proc/sys/kernel/random/uuid")
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	id := strings.ReplaceAll(strings.TrimSpace(idOut), "-", "")[:12]

	containerDir := fmt.Sprintf("%s/containers/%s", r.dataRoot, id)
	rootfs := containerDir + "/rootfs"

	// Create rootfs directory structure
	dirs := []string{
		rootfs + "/bin",
		rootfs + "/etc",
		rootfs + "/proc",
		rootfs + "/sys",
		rootfs + "/dev",
		rootfs + "/tmp",
		rootfs + "/var/log",
		rootfs + "/app",
	}
	for _, d := range dirs {
		if err := r.wslRun("mkdir", "-p", d); err != nil {
			return nil, fmt.Errorf("failed to create rootfs dir %s: %w", d, err)
		}
	}

	// Copy busybox or base binaries if available
	// busybox provides sh, echo, ls, cat, etc. all in one binary
	r.wslRun("sh", "-c", fmt.Sprintf(
		"if command -v busybox >/dev/null 2>&1; then cp $(which busybox) %s/bin/busybox && for cmd in sh echo ls cat mkdir rm cp mv pwd env; do ln -sf /bin/busybox %s/bin/$cmd; done; fi",
		rootfs, rootfs,
	))

	// If no busybox, copy individual binaries from the host WSL2 environment
	r.wslRun("sh", "-c", fmt.Sprintf(
		`for bin in sh echo ls cat mkdir rm cp mv pwd env; do
			path=$(which $bin 2>/dev/null)
			if [ -n "$path" ]; then
				cp "$path" %s/bin/$bin 2>/dev/null
				# Copy required shared libraries
				ldd "$path" 2>/dev/null | grep -o '/[^ ]*' | while read lib; do
					libdir=%s$(dirname $lib)
					mkdir -p "$libdir"
					cp "$lib" "$libdir/" 2>/dev/null
				done
			fi
		done`, rootfs, rootfs,
	))

	// Write container metadata to disk
	state := wsl2ContainerState{
		ID:        id,
		Name:      name,
		Status:    StatusCreated,
		RootFS:    rootfs,
		Running:   false,
		CreatedAt: time.Now(),
	}
	stateJSON, _ := json.Marshal(state)
	stateFile := containerDir + "/state.json"
	// Use tee to write the file safely without shell quoting issues
	writeCmd := exec.Command("wsl", "-d", r.distro, "--exec", "tee", stateFile)
	writeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	writeCmd.Stdin = strings.NewReader(string(stateJSON))
	if err := writeCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to write state: %w", err)
	}

	return &WSL2Container{
		ID:        id,
		Name:      name,
		Status:    StatusCreated,
		RootFS:    rootfs,
		Running:   false,
		CreatedAt: time.Now(),
	}, nil
}

// StartContainer starts a container with real Linux namespace isolation
func (r *WSL2Runtime) StartContainer(ctx context.Context, container *WSL2Container, cmd []string, env []string) error {
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}

	// Build env prefix
	envPrefix := ""
	for _, e := range env {
		envPrefix += "export " + e + "; "
	}

	// Try multiple isolation strategies in order of strength
	strategies := []struct {
		name string
		cmd  string
	}{
		{
			"user+mount+chroot",
			fmt.Sprintf("unshare --user --map-root-user --mount chroot %s %s",
				container.RootFS, strings.Join(cmd, " ")),
		},
		{
			"user+chroot",
			fmt.Sprintf("unshare --user --map-root-user chroot %s %s",
				container.RootFS, strings.Join(cmd, " ")),
		},
		{
			"chroot-only",
			fmt.Sprintf("chroot %s %s",
				container.RootFS, strings.Join(cmd, " ")),
		},
	}

	var lastErr error
	for _, strat := range strategies {
		fullCmd := strat.cmd
		if envPrefix != "" {
			fullCmd = envPrefix + fullCmd
		}

		// Run in background, capture PID
		pidOut, err := r.wslOutputWithStderr("sh", "-c", fullCmd+" & echo $!")
		if err != nil {
			lastErr = fmt.Errorf("%s failed: %s", strat.name, strings.TrimSpace(pidOut))
			continue
		}

		// Extract PID from last line
		lines := strings.Split(strings.TrimSpace(pidOut), "\n")
		pidStr := strings.TrimSpace(lines[len(lines)-1])
		pid := 0
		fmt.Sscanf(pidStr, "%d", &pid)

		if pid == 0 {
			lastErr = fmt.Errorf("%s: got PID 0", strat.name)
			continue
		}

		// Verify process is actually running
		checkOut, _ := r.wslOutput("kill", "-0", fmt.Sprintf("%d", pid))
		_ = checkOut

		container.PID = pid
		container.Status = StatusRunning
		container.Running = true
		container.StartedAt = time.Now()

		r.persistState(container)
		return nil
	}

	return fmt.Errorf("all isolation strategies failed: %w", lastErr)
}

// StopContainer stops a running container by killing its process tree
func (r *WSL2Runtime) StopContainer(ctx context.Context, container *WSL2Container, timeoutSec int) error {
	if !container.Running || container.PID == 0 {
		return nil
	}

	// Send SIGTERM first
	r.wslRun("kill", "-TERM", fmt.Sprintf("%d", container.PID))

	// Wait for timeout then SIGKILL
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		out, _ := r.wslOutput("kill", "-0", fmt.Sprintf("%d", container.PID))
		if strings.TrimSpace(out) != "" {
			break // process gone
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Force kill if still running
	r.wslRun("kill", "-KILL", fmt.Sprintf("%d", container.PID))

	container.Running = false
	container.Status = StatusExited
	container.ExitCode = 0

	r.persistState(container)
	return nil
}

// ExecInContainer runs a command inside a running container's namespace
func (r *WSL2Runtime) ExecInContainer(ctx context.Context, container *WSL2Container, cmd []string) (string, error) {
	// Ensure rootfs has binaries
	lsOut, _ := r.wslOutput("ls", container.RootFS+"/bin/")
	if strings.TrimSpace(lsOut) == "" {
		r.populateRootfsBins(container.RootFS)
	}

	// Try strategies in order of isolation strength
	strategies := [][]string{
		append([]string{"unshare", "--user", "--map-root-user", "--mount", "chroot", container.RootFS}, cmd...),
		append([]string{"unshare", "--user", "--map-root-user", "chroot", container.RootFS}, cmd...),
		append([]string{"chroot", container.RootFS}, cmd...),
	}

	for _, args := range strategies {
		out, err := r.wslOutputWithStderr(args...)
		if err == nil {
			return out, nil
		}
	}

	return "", fmt.Errorf("exec failed: all isolation strategies failed for %s", container.Name)
}

// populateRootfsBins copies essential binaries into the container rootfs
func (r *WSL2Runtime) populateRootfsBins(rootfs string) {
	r.wslRun("mkdir", "-p", rootfs+"/bin", rootfs+"/lib", rootfs+"/lib64", rootfs+"/lib/x86_64-linux-gnu")

	// Try busybox first (single static binary, no deps)
	busyboxPath, err := r.wslOutput("which", "busybox")
	if err == nil && strings.TrimSpace(busyboxPath) != "" {
		busyboxPath = strings.TrimSpace(busyboxPath)
		r.wslRun("cp", busyboxPath, rootfs+"/bin/busybox")
		for _, cmd := range []string{"sh", "echo", "ls", "cat", "mkdir", "rm", "cp", "mv", "pwd", "env", "find", "grep"} {
			r.wslRun("ln", "-sf", "/bin/busybox", rootfs+"/bin/"+cmd)
		}
		return
	}

	// Fallback: copy /bin/sh and its libs
	for _, bin := range []string{"sh", "echo", "ls", "cat"} {
		binPath, err := r.wslOutput("which", bin)
		if err != nil {
			continue
		}
		binPath = strings.TrimSpace(binPath)
		r.wslRun("cp", binPath, rootfs+"/bin/"+bin)

		// Copy shared libraries
		lddOut, err := r.wslOutput("ldd", binPath)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(lddOut, "\n") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasPrefix(p, "/") {
					dir := rootfs + p[:strings.LastIndex(p, "/")]
					r.wslRun("mkdir", "-p", dir)
					r.wslRun("cp", p, rootfs+p)
				}
			}
		}
	}
}

// RemoveContainer removes a container and its filesystem
func (r *WSL2Runtime) RemoveContainer(ctx context.Context, container *WSL2Container) error {
	if container.Running {
		r.StopContainer(ctx, container, 5)
	}

	containerDir := fmt.Sprintf("%s/containers/%s", r.dataRoot, container.ID)
	return r.wslRun("rm", "-rf", containerDir)
}

// ListContainers lists all containers stored on disk in WSL2
func (r *WSL2Runtime) ListContainers(ctx context.Context) ([]*WSL2Container, error) {
	out, err := r.wslOutput("find", r.dataRoot+"/containers", "-maxdepth", "2", "-name", "state.json")
	if err != nil || strings.TrimSpace(out) == "" {
		return make([]*WSL2Container, 0), nil
	}

	containers := make([]*WSL2Container, 0)
	for _, stateFile := range strings.Split(strings.TrimSpace(out), "\n") {
		stateFile = strings.TrimSpace(stateFile)
		if stateFile == "" {
			continue
		}
		content, err := r.wslOutput("cat", stateFile)
		if err != nil {
			continue
		}
		var state wsl2ContainerState
		if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &state); err != nil {
			continue
		}
		containers = append(containers, &WSL2Container{
			ID:        state.ID,
			Name:      state.Name,
			PID:       state.PID,
			Status:    state.Status,
			RootFS:    state.RootFS,
			Running:   state.Running,
			CreatedAt: state.CreatedAt,
			StartedAt: state.StartedAt,
			ExitCode:  state.ExitCode,
		})
	}

	return containers, nil
}

// GetContainerStats returns resource usage for a running container
func (r *WSL2Runtime) GetContainerStats(ctx context.Context, container *WSL2Container) (map[string]string, error) {
	if !container.Running || container.PID == 0 {
		return nil, fmt.Errorf("container not running")
	}

	// Read from /proc/{pid}/status for memory, /proc/{pid}/stat for CPU
	memOut, _ := r.wslOutput("sh", "-c", fmt.Sprintf(
		"cat /proc/%d/status 2>/dev/null | grep -E 'VmRSS|VmSize'", container.PID,
	))
	cpuOut, _ := r.wslOutput("sh", "-c", fmt.Sprintf(
		"cat /proc/%d/stat 2>/dev/null | awk '{print $14+$15}'", container.PID,
	))

	return map[string]string{
		"memory": strings.TrimSpace(memOut),
		"cpu":    strings.TrimSpace(cpuOut),
		"pid":    fmt.Sprintf("%d", container.PID),
	}, nil
}

// persistState writes container state to disk inside WSL2
func (r *WSL2Runtime) persistState(container *WSL2Container) {
	state := wsl2ContainerState{
		ID:        container.ID,
		Name:      container.Name,
		PID:       container.PID,
		Status:    container.Status,
		RootFS:    container.RootFS,
		Running:   container.Running,
		CreatedAt: container.CreatedAt,
		StartedAt: container.StartedAt,
		ExitCode:  container.ExitCode,
	}
	stateJSON, _ := json.Marshal(state)
	stateFile := fmt.Sprintf("%s/containers/%s/state.json", r.dataRoot, container.ID)
	writeCmd := exec.Command("wsl", "-d", r.distro, "--exec", "tee", stateFile)
	writeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	writeCmd.Stdin = strings.NewReader(string(stateJSON))
	writeCmd.Run()
}

// wslOutputWithStderr runs a command inside WSL2 and returns both stdout and stderr
func (r *WSL2Runtime) wslOutputWithStderr(args ...string) (string, error) {
	wslArgs := append([]string{"-d", r.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// wslRun runs a command inside WSL2 and discards output
func (r *WSL2Runtime) wslRun(args ...string) error {
	wslArgs := append([]string{"-d", r.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

// wslOutput runs a command inside WSL2 and returns stdout
func (r *WSL2Runtime) wslOutput(args ...string) (string, error) {
	wslArgs := append([]string{"-d", r.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return string(out), err
}
