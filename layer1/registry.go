package layer1

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AdapterRegistry manages all available adapters
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
	metadata map[string]*AdapterMetadata
}

// AdapterMetadata contains information about a registered adapter
type AdapterMetadata struct {
	ID           string
	Type         AdapterType
	Capabilities []AdapterCapability
	Target       string
	RegisteredAt time.Time
	LastHealthCheck time.Time
	Tags         map[string]string
}

// NewAdapterRegistry creates a new adapter registry
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]Adapter),
		metadata: make(map[string]*AdapterMetadata),
	}
}

// Register adds an adapter to the registry
func (r *AdapterRegistry) Register(adapter Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	config := adapter.GetConfig()
	if config.ID == "" {
		return fmt.Errorf("adapter ID cannot be empty")
	}

	if _, exists := r.adapters[config.ID]; exists {
		return fmt.Errorf("adapter with ID %s already registered", config.ID)
	}

	r.adapters[config.ID] = adapter
	r.metadata[config.ID] = &AdapterMetadata{
		ID:           config.ID,
		Type:         adapter.GetType(),
		Capabilities: adapter.GetCapabilities(),
		Target:       config.Target,
		RegisteredAt: time.Now(),
		Tags:         make(map[string]string),
	}

	return nil
}

// Unregister removes an adapter from the registry
func (r *AdapterRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	adapter, exists := r.adapters[id]
	if !exists {
		return fmt.Errorf("adapter with ID %s not found", id)
	}

	// Disconnect before removing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	adapter.Disconnect(ctx)

	delete(r.adapters, id)
	delete(r.metadata, id)

	return nil
}

// Get retrieves an adapter by ID
func (r *AdapterRegistry) Get(id string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[id]
	if !exists {
		return nil, fmt.Errorf("adapter with ID %s not found", id)
	}

	return adapter, nil
}

// List returns all registered adapters
func (r *AdapterRegistry) List() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters := make([]Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}

	return adapters
}

// ListByType returns all adapters of a specific type
func (r *AdapterRegistry) ListByType(adapterType AdapterType) []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters := make([]Adapter, 0)
	for _, adapter := range r.adapters {
		if adapter.GetType() == adapterType {
			adapters = append(adapters, adapter)
		}
	}

	return adapters
}

// GetMetadata returns metadata for an adapter
func (r *AdapterRegistry) GetMetadata(id string) (*AdapterMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[id]
	if !exists {
		return nil, fmt.Errorf("metadata for adapter %s not found", id)
	}

	return metadata, nil
}

// ListMetadata returns metadata for all adapters
func (r *AdapterRegistry) ListMetadata() []*AdapterMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make([]*AdapterMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		metadata = append(metadata, meta)
	}

	return metadata
}

// HealthCheckAll performs health checks on all adapters
func (r *AdapterRegistry) HealthCheckAll(ctx context.Context) map[string]*AdapterHealth {
	r.mu.RLock()
	adapters := make(map[string]Adapter, len(r.adapters))
	for id, adapter := range r.adapters {
		adapters[id] = adapter
	}
	r.mu.RUnlock()

	results := make(map[string]*AdapterHealth)
	var wg sync.WaitGroup

	for id, adapter := range adapters {
		wg.Add(1)
		go func(id string, adapter Adapter) {
			defer wg.Done()

			health, _ := adapter.HealthCheck(ctx)
			results[id] = health

			// Update last health check time
			r.mu.Lock()
			if meta, exists := r.metadata[id]; exists {
				meta.LastHealthCheck = time.Now()
			}
			r.mu.Unlock()
		}(id, adapter)
	}

	wg.Wait()
	return results
}

// Count returns the number of registered adapters
func (r *AdapterRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}

// Clear removes all adapters from the registry
func (r *AdapterRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, adapter := range r.adapters {
		adapter.Disconnect(ctx)
	}

	r.adapters = make(map[string]Adapter)
	r.metadata = make(map[string]*AdapterMetadata)
}
