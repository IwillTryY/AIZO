package layer1

import (
	"context"
	"fmt"
	"time"
)

// AdapterFactory creates adapters based on configuration
type AdapterFactory struct {
	registry *AdapterRegistry
}

// NewAdapterFactory creates a new adapter factory
func NewAdapterFactory(registry *AdapterRegistry) *AdapterFactory {
	return &AdapterFactory{
		registry: registry,
	}
}

// Create creates and registers a new adapter based on the config
func (f *AdapterFactory) Create(config *AdapterConfig) (Adapter, error) {
	if config == nil {
		return nil, fmt.Errorf("adapter config cannot be nil")
	}

	if config.ID == "" {
		return nil, fmt.Errorf("adapter ID is required")
	}

	if config.Target == "" {
		return nil, fmt.Errorf("adapter target is required")
	}

	var adapter Adapter

	switch config.Type {
	case AdapterTypeHTTP:
		adapter = NewHTTPAdapter(config)
	case AdapterTypeSSH:
		adapter = NewSSHAdapter(config)
	case AdapterTypeGRPC:
		adapter = NewGRPCAdapter(config)
	case AdapterTypeMQTT:
		adapter = NewMQTTAdapter(config)
	case AdapterTypeMesh:
		adapter = NewMeshAdapter(config)
	default:
		return nil, fmt.Errorf("unsupported adapter type: %s", config.Type)
	}

	// Register the adapter
	if err := f.registry.Register(adapter); err != nil {
		return nil, fmt.Errorf("failed to register adapter: %w", err)
	}

	return adapter, nil
}

// CreateAndConnect creates, registers, and connects an adapter
func (f *AdapterFactory) CreateAndConnect(config *AdapterConfig) (Adapter, error) {
	adapter, err := f.Create(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := adapter.Connect(ctx); err != nil {
		// Unregister on connection failure
		f.registry.Unregister(config.ID)
		return nil, fmt.Errorf("failed to connect adapter: %w", err)
	}

	return adapter, nil
}

// GetRegistry returns the adapter registry
func (f *AdapterFactory) GetRegistry() *AdapterRegistry {
	return f.registry
}
