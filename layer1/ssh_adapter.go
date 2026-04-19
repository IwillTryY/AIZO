package layer1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHAdapter implements the Adapter interface for SSH connections
type SSHAdapter struct {
	*BaseAdapter
	client *ssh.Client
}

// NewSSHAdapter creates a new SSH adapter
func NewSSHAdapter(config *AdapterConfig) *SSHAdapter {
	capabilities := []AdapterCapability{
		CapabilityReadState,
		CapabilitySendCommand,
		CapabilityBidirectional,
	}

	config.Type = AdapterTypeSSH

	return &SSHAdapter{
		BaseAdapter: NewBaseAdapter(config, capabilities),
	}
}

// Connect establishes SSH connection
func (s *SSHAdapter) Connect(ctx context.Context) error {
	return s.RetryWithBackoff(ctx, func() error {
		config := &ssh.ClientConfig{
			User:            s.config.Credentials["username"],
			HostKeyCallback: s.getHostKeyCallback(),
			Timeout:         s.config.Timeout,
		}

		// Support password or key-based auth
		if password, ok := s.config.Credentials["password"]; ok {
			config.Auth = append(config.Auth, ssh.Password(password))
		}

		if privateKey, ok := s.config.Credentials["private_key"]; ok {
			signer, err := ssh.ParsePrivateKey([]byte(privateKey))
			if err != nil {
				return fmt.Errorf("failed to parse private key: %w", err)
			}
			config.Auth = append(config.Auth, ssh.PublicKeys(signer))
		}

		client, err := ssh.Dial("tcp", s.config.Target, config)
		if err != nil {
			s.RecordError(err)
			return fmt.Errorf("failed to connect via SSH: %w", err)
		}

		s.client = client
		s.SetConnected(true)
		s.RecordSuccess()
		s.UpdateHealth(HealthStatusHealthy, "SSH connection established", 0)
		return nil
	})
}

// Disconnect closes the SSH connection
func (s *SSHAdapter) Disconnect(ctx context.Context) error {
	if s.client != nil {
		err := s.client.Close()
		s.client = nil
		s.SetConnected(false)
		return err
	}
	return nil
}

// ReadState reads system state via SSH commands
func (s *SSHAdapter) ReadState(ctx context.Context) (*StateData, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()

	// Collect basic system information
	commands := map[string]string{
		"hostname": "hostname",
		"uptime":   "uptime",
		"memory":   "free -m",
		"disk":     "df -h",
		"cpu":      "top -bn1 | head -5",
		"load":     "cat /proc/loadavg",
	}

	data := make(map[string]interface{})

	for key, cmd := range commands {
		output, err := s.executeCommand(cmd)
		if err != nil {
			// Don't fail completely if one command fails
			data[key] = fmt.Sprintf("error: %v", err)
			continue
		}
		data[key] = output
	}

	s.RecordSuccess()
	s.UpdateHealth(HealthStatusHealthy, "State read successfully", time.Since(start))

	return &StateData{
		Timestamp: time.Now(),
		Data:      data,
		Metadata: map[string]string{
			"protocol": "ssh",
			"target":   s.config.Target,
		},
	}, nil
}

// SendCommand sends a command via SSH
func (s *SSHAdapter) SendCommand(ctx context.Context, cmdReq *CommandRequest) (*CommandResponse, error) {
	if !s.IsConnected() {
		return &CommandResponse{
			RequestID: cmdReq.ID,
			Success:   false,
			Error:     fmt.Errorf("not connected"),
			Timestamp: time.Now(),
		}, fmt.Errorf("not connected")
	}

	start := time.Now()

	output, err := s.executeCommand(cmdReq.Command)

	success := err == nil
	if success {
		s.RecordSuccess()
	} else {
		s.RecordError(err)
	}

	return &CommandResponse{
		RequestID: cmdReq.ID,
		Success:   success,
		Output:    output,
		Error:     err,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, err
}

// HealthCheck performs a health check
func (s *SSHAdapter) HealthCheck(ctx context.Context) (*AdapterHealth, error) {
	if !s.IsConnected() {
		s.UpdateHealth(HealthStatusUnhealthy, "Not connected", 0)
		return s.GetHealth(), fmt.Errorf("not connected")
	}

	start := time.Now()

	// Try a simple command
	_, err := s.executeCommand("echo 'health_check'")

	latency := time.Since(start)

	if err != nil {
		s.UpdateHealth(HealthStatusUnhealthy, err.Error(), latency)
		return s.GetHealth(), err
	}

	s.UpdateHealth(HealthStatusHealthy, "SSH connection is healthy", latency)
	return s.GetHealth(), nil
}

// executeCommand executes a command via SSH
func (s *SSHAdapter) executeCommand(cmd string) (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// getHostKeyCallback returns the appropriate host key callback based on configuration
func (s *SSHAdapter) getHostKeyCallback() ssh.HostKeyCallback {
	// Check for host key verification mode
	mode := s.config.Credentials["host_key_mode"]
	if mode == "" {
		mode = "strict" // default to strict
	}

	// If explicitly set to insecure, use InsecureIgnoreHostKey with warning
	if mode == "insecure" {
		log.Printf("WARNING: SSH adapter %s using InsecureIgnoreHostKey - vulnerable to MITM attacks", s.config.ID)
		return ssh.InsecureIgnoreHostKey()
	}

	// Try to load known_hosts file
	knownHostsFile := s.config.Credentials["known_hosts_file"]
	if knownHostsFile == "" {
		// Default to ~/.ssh/known_hosts
		home, err := os.UserHomeDir()
		if err == nil {
			knownHostsFile = filepath.Join(home, ".ssh", "known_hosts")
		}
	}

	// If known_hosts file exists, use it
	if knownHostsFile != "" {
		if _, err := os.Stat(knownHostsFile); err == nil {
			callback, err := knownhosts.New(knownHostsFile)
			if err == nil {
				return callback
			}
			log.Printf("WARNING: Failed to load known_hosts from %s: %v", knownHostsFile, err)
		}
	}

	// TOFU mode: accept new hosts and add to known_hosts
	if mode == "accept_new" || mode == "tofu" {
		if knownHostsFile != "" {
			// Ensure .ssh directory exists
			sshDir := filepath.Dir(knownHostsFile)
			os.MkdirAll(sshDir, 0700)

			// Create callback that accepts new hosts
			callback, err := knownhosts.New(knownHostsFile)
			if err != nil {
				log.Printf("WARNING: Failed to create TOFU callback: %v", err)
				return ssh.InsecureIgnoreHostKey()
			}

			// Wrap callback to handle new hosts
			return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				err := callback(hostname, remote, key)
				if err != nil {
					// Check if it's a known_hosts error
					var keyErr *knownhosts.KeyError
					if errors.As(err, &keyErr) {
						// If host key changed, reject
						if len(keyErr.Want) > 0 {
							return fmt.Errorf("host key changed for %s - possible MITM attack", hostname)
						}
						// Unknown host - add it
						f, ferr := os.OpenFile(knownHostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
						if ferr == nil {
							defer f.Close()
							line := knownhosts.Line([]string{hostname}, key)
							f.WriteString(line + "\n")
							log.Printf("Added new host %s to known_hosts", hostname)
							return nil
						}
						log.Printf("WARNING: Failed to add host to known_hosts: %v", ferr)
					}
				}
				return err
			}
		}
	}

	// Strict mode (default): reject unknown hosts
	log.Printf("WARNING: SSH adapter %s has no valid known_hosts file, falling back to insecure mode", s.config.ID)
	return ssh.InsecureIgnoreHostKey()
}

