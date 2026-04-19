package security

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Role defines what a node is allowed to do
type Role string

const (
	RoleAdmin    Role = "admin"    // full access: read, write, execute
	RoleOperator Role = "operator" // read + execute commands
	RoleReader   Role = "reader"   // read state only
)

// NodeEntry represents a registered node with its role and shared secret
type NodeEntry struct {
	ID     string `yaml:"id"`
	Role   Role   `yaml:"role"`
	Secret string `yaml:"key"` // base64 shared secret
}

// NodesConfig is the full nodes.yaml structure
type NodesConfig struct {
	Nodes []NodeEntry `yaml:"nodes"`
}

// RBAC enforces role-based access control for mesh operations
type RBAC struct {
	nodes map[string]*NodeEntry
	mu    sync.RWMutex
}

// NewRBAC creates a new RBAC instance
func NewRBAC() *RBAC {
	return &RBAC{nodes: make(map[string]*NodeEntry)}
}

// LoadFromFile loads node config from a YAML file
func (r *RBAC) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read nodes config: %w", err)
	}

	var config NodesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse nodes config: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range config.Nodes {
		r.nodes[config.Nodes[i].ID] = &config.Nodes[i]
	}
	return nil
}

// SaveToFile saves node config to a YAML file
func (r *RBAC) SaveToFile(path string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config := NodesConfig{Nodes: make([]NodeEntry, 0, len(r.nodes))}
	for _, n := range r.nodes {
		config.Nodes = append(config.Nodes, *n)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// RegisterNode adds or updates a node
func (r *RBAC) RegisterNode(entry NodeEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[entry.ID] = &entry
}

// GetNode returns a node entry by ID
func (r *RBAC) GetNode(nodeID string) (*NodeEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[nodeID]
	return n, ok
}

// GetSecret returns the shared secret for a node
func (r *RBAC) GetSecret(nodeID string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n, ok := r.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", nodeID)
	}
	return []byte(n.Secret), nil
}

// CanExecute checks if a node has permission to perform an action
func (r *RBAC) CanExecute(nodeID string, action string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, ok := r.nodes[nodeID]
	if !ok {
		return false // unknown nodes denied by default
	}

	switch node.Role {
	case RoleAdmin:
		return true // admin can do everything
	case RoleOperator:
		// operators can read state and execute commands, but not manage nodes
		switch action {
		case "read_state", "send_command", "health_check", "file_transfer":
			return true
		}
		return false
	case RoleReader:
		// readers can only read state and health
		switch action {
		case "read_state", "health_check":
			return true
		}
		return false
	}

	return false
}

// CanRead checks if a node can read state
func (r *RBAC) CanRead(nodeID string) bool {
	return r.CanExecute(nodeID, "read_state")
}

// CanCommand checks if a node can send commands
func (r *RBAC) CanCommand(nodeID string) bool {
	return r.CanExecute(nodeID, "send_command")
}
