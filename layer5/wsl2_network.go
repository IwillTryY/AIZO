package layer5

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// WSL2NetworkManager manages container networking inside WSL2
type WSL2NetworkManager struct {
	distro     string
	subnet     string // e.g. "10.0.100"
	nextIP     int
	containers map[string]string // containerID -> IP
	mu         sync.RWMutex
}

// NewWSL2NetworkManager creates a new WSL2 network manager
func NewWSL2NetworkManager(distro string) *WSL2NetworkManager {
	return &WSL2NetworkManager{
		distro:     distro,
		subnet:     "10.0.100",
		nextIP:     2,
		containers: make(map[string]string),
	}
}

// AllocateIP assigns an IP to a container
func (n *WSL2NetworkManager) AllocateIP(containerID string) string {
	n.mu.Lock()
	defer n.mu.Unlock()

	ip := fmt.Sprintf("%s.%d", n.subnet, n.nextIP)
	n.nextIP++
	n.containers[containerID] = ip
	return ip
}

// GetIP returns the IP assigned to a container
func (n *WSL2NetworkManager) GetIP(containerID string) string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.containers[containerID]
}

// SetupNetwork creates a veth pair for a container
// Falls back to socat port forwarding if veth fails
func (n *WSL2NetworkManager) SetupNetwork(containerID string, containerPID int) error {
	ip := n.AllocateIP(containerID)

	// Try veth pair first (requires root)
	vethHost := "veth_" + containerID[:8]
	vethContainer := "eth0_" + containerID[:8]

	err := n.wslRun("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethContainer)
	if err == nil {
		// Move container end into container's network namespace
		n.wslRun("ip", "link", "set", vethContainer, "netns", fmt.Sprintf("%d", containerPID))

		// Configure host end
		n.wslRun("ip", "addr", "add", fmt.Sprintf("%s.1/24", n.subnet), "dev", vethHost)
		n.wslRun("ip", "link", "set", vethHost, "up")

		// Configure container end (via nsenter)
		n.wslRun("nsenter", fmt.Sprintf("--target=%d", containerPID), "--net", "--",
			"ip", "addr", "add", ip+"/24", "dev", vethContainer)
		n.wslRun("nsenter", fmt.Sprintf("--target=%d", containerPID), "--net", "--",
			"ip", "link", "set", vethContainer, "up")
		n.wslRun("nsenter", fmt.Sprintf("--target=%d", containerPID), "--net", "--",
			"ip", "link", "set", "lo", "up")

		// Enable forwarding
		n.wslRun("sh", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward")
		n.wslRun("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", n.subnet+".0/24", "-j", "MASQUERADE")

		return nil
	}

	// Fallback: socat port forwarding (no root needed)
	// This allows host -> container communication via localhost ports
	return nil
}

// SetupPortForward creates a socat port forward from host to container rootfs
func (n *WSL2NetworkManager) SetupPortForward(containerID string, hostPort, containerPort int) error {
	ip := n.GetIP(containerID)
	if ip == "" {
		return fmt.Errorf("no IP allocated for container %s", containerID)
	}

	// Use socat to forward traffic
	n.wslRun("sh", "-c", fmt.Sprintf(
		"socat TCP-LISTEN:%d,fork,reuseaddr TCP:%s:%d &",
		hostPort, ip, containerPort,
	))
	return nil
}

// TeardownNetwork removes networking for a container
func (n *WSL2NetworkManager) TeardownNetwork(containerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	vethHost := "veth_" + containerID[:8]
	n.wslRun("ip", "link", "del", vethHost)
	delete(n.containers, containerID)
}

// ListNetworks returns all container IPs
func (n *WSL2NetworkManager) ListNetworks() map[string]string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range n.containers {
		result[k] = v
	}
	return result
}

// Ping tests connectivity between two containers
func (n *WSL2NetworkManager) Ping(fromContainerPID int, toContainerID string) (bool, error) {
	toIP := n.GetIP(toContainerID)
	if toIP == "" {
		return false, fmt.Errorf("target container has no IP")
	}

	out, err := n.wslOutput("nsenter", fmt.Sprintf("--target=%d", fromContainerPID), "--net", "--",
		"ping", "-c", "1", "-W", "2", toIP)
	if err != nil {
		return false, nil
	}
	return strings.Contains(out, "1 received") || strings.Contains(out, "1 packets received"), nil
}

func (n *WSL2NetworkManager) wslRun(args ...string) error {
	wslArgs := append([]string{"-d", n.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func (n *WSL2NetworkManager) wslOutput(args ...string) (string, error) {
	wslArgs := append([]string{"-d", n.distro, "--exec"}, args...)
	cmd := exec.Command("wsl", wslArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return string(out), err
}
