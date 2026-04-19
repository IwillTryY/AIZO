package layer1

import (
	"context"
	"fmt"
	"time"
)

// Manager orchestrates adapter lifecycle and operations
type Manager struct {
	registry *AdapterRegistry
	factory  *AdapterFactory
	stopCh   chan struct{}
}

// NewManager creates a new adapter manager
func NewManager() *Manager {
	registry := NewAdapterRegistry()
	factory := NewAdapterFactory(registry)

	return &Manager{
		registry: registry,
		factory:  factory,
		stopCh:   make(chan struct{}),
	}
}

// CreateAdapter creates a new adapter
func (m *Manager) CreateAdapter(config *AdapterConfig) (Adapter, error) {
	return m.factory.Create(config)
}

// CreateAndConnectAdapter creates and connects a new adapter
func (m *Manager) CreateAndConnectAdapter(config *AdapterConfig) (Adapter, error) {
	return m.factory.CreateAndConnect(config)
}

// GetAdapter retrieves an adapter by ID
func (m *Manager) GetAdapter(id string) (Adapter, error) {
	return m.registry.Get(id)
}

// ListAdapters returns all registered adapters
func (m *Manager) ListAdapters() []Adapter {
	return m.registry.List()
}

// RemoveAdapter removes an adapter
func (m *Manager) RemoveAdapter(id string) error {
	return m.registry.Unregister(id)
}

// ReadState reads state from a specific adapter
func (m *Manager) ReadState(ctx context.Context, adapterID string) (*StateData, error) {
	adapter, err := m.registry.Get(adapterID)
	if err != nil {
		return nil, err
	}

	return adapter.ReadState(ctx)
}

// SendCommand sends a command to a specific adapter
func (m *Manager) SendCommand(ctx context.Context, adapterID string, req *CommandRequest) (*CommandResponse, error) {
	adapter, err := m.registry.Get(adapterID)
	if err != nil {
		return nil, err
	}

	return adapter.SendCommand(ctx, req)
}

// HealthCheckAll performs health checks on all adapters
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]*AdapterHealth {
	return m.registry.HealthCheckAll(ctx)
}

// StartHealthMonitoring starts periodic health checks for all adapters
func (m *Manager) StartHealthMonitoring(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				m.HealthCheckAll(ctx)
				cancel()
			case <-m.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// StopHealthMonitoring stops the health monitoring
func (m *Manager) StopHealthMonitoring() {
	close(m.stopCh)
}

// GetStats returns statistics about the adapter layer
func (m *Manager) GetStats() *Stats {
	adapters := m.registry.List()
	metadata := m.registry.ListMetadata()

	stats := &Stats{
		TotalAdapters: len(adapters),
		ByType:        make(map[AdapterType]int),
		ByStatus:      make(map[HealthStatus]int),
	}

	for _, adapter := range adapters {
		stats.ByType[adapter.GetType()]++

		health := adapter.GetHealth()
		stats.ByStatus[health.Status]++
	}

	// Calculate average success rate
	var totalSuccessRate float64
	for _, adapter := range adapters {
		health := adapter.GetHealth()
		totalSuccessRate += health.SuccessRate
	}
	if len(adapters) > 0 {
		stats.AverageSuccessRate = totalSuccessRate / float64(len(adapters))
	}

	// Find oldest registration
	for _, meta := range metadata {
		if stats.OldestRegistration.IsZero() || meta.RegisteredAt.Before(stats.OldestRegistration) {
			stats.OldestRegistration = meta.RegisteredAt
		}
	}

	return stats
}

// Shutdown gracefully shuts down all adapters
func (m *Manager) Shutdown(ctx context.Context) error {
	m.StopHealthMonitoring()

	adapters := m.registry.List()
	errCh := make(chan error, len(adapters))

	for _, adapter := range adapters {
		go func(a Adapter) {
			errCh <- a.Disconnect(ctx)
		}(adapter)
	}

	var errors []error
	for i := 0; i < len(adapters); i++ {
		if err := <-errCh; err != nil {
			errors = append(errors, err)
		}
	}

	m.registry.Clear()

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors", len(errors))
	}

	return nil
}

// Stats represents statistics about the adapter layer
type Stats struct {
	TotalAdapters       int
	ByType              map[AdapterType]int
	ByStatus            map[HealthStatus]int
	AverageSuccessRate  float64
	OldestRegistration  time.Time
}
