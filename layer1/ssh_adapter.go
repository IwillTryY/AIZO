package layer1

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
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
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
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
