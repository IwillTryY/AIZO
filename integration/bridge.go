package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/realityos/aizo/layer1"
	"github.com/realityos/aizo/layer2"
)

// Bridge connects Layer 1 (Adapters) with Layer 2 (Discovery & Registration)
type Bridge struct {
	layer1Manager *layer1.Manager
	layer2Manager *layer2.Manager
}

// NewBridge creates a new integration bridge
func NewBridge(l1Manager *layer1.Manager, l2Manager *layer2.Manager) *Bridge {
	return &Bridge{
		layer1Manager: l1Manager,
		layer2Manager: l2Manager,
	}
}

// DiscoverAndRegister discovers entities via adapters and registers them
func (b *Bridge) DiscoverAndRegister(ctx context.Context, adapterID string) error {
	// Get adapter from Layer 1
	adapter, err := b.layer1Manager.GetAdapter(adapterID)
	if err != nil {
		return fmt.Errorf("adapter not found: %w", err)
	}

	// Read state from adapter
	state, err := adapter.ReadState(ctx)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	// Convert adapter state to entity
	entity := b.stateToEntity(adapterID, state)

	// Register entity in Layer 2
	req := &layer2.RegistrationRequest{
		ID:           entity.ID,
		Type:         entity.Type,
		Name:         entity.Name,
		Endpoint:     entity.Endpoint,
		Metadata:     entity.Metadata,
		Adapters:     []string{adapterID},
		AutoDetect:   true,
		MapRelations: true,
	}

	_, err = b.layer2Manager.RegisterEntity(ctx, req)
	return err
}

// SyncAdapterHealth syncs adapter health to entity state
func (b *Bridge) SyncAdapterHealth(ctx context.Context, adapterID string) error {
	// Get adapter health
	adapter, err := b.layer1Manager.GetAdapter(adapterID)
	if err != nil {
		return err
	}

	health, err := adapter.HealthCheck(ctx)
	if err != nil {
		return err
	}

	// Find entities using this adapter
	entities := b.layer2Manager.GetCatalog().ListByAdapter(adapterID)

	// Update entity states based on adapter health
	regAPI := b.layer2Manager.GetRegistrationAPI()
	for _, entity := range entities {
		state := b.healthToState(health.Status)
		healthScore := b.calculateHealthScore(health)

		_ = regAPI.UpdateState(ctx, entity.ID, state, healthScore)
	}

	return nil
}

// CreateAdapterForEntity creates and connects an adapter for an entity
func (b *Bridge) CreateAdapterForEntity(ctx context.Context, entityID string, adapterConfig *layer1.AdapterConfig) error {
	// Create adapter in Layer 1
	_, err := b.layer1Manager.CreateAndConnectAdapter(adapterConfig)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Update entity with adapter ID
	entity, err := b.layer2Manager.GetEntity(entityID)
	if err != nil {
		return err
	}

	entity.Adapters = append(entity.Adapters, adapterConfig.ID)
	return b.layer2Manager.GetCatalog().Register(entity)
}

// ExecuteCommand executes a command on an entity via its adapter
func (b *Bridge) ExecuteCommand(ctx context.Context, entityID string, command *layer1.CommandRequest) (*layer1.CommandResponse, error) {
	// Get entity
	entity, err := b.layer2Manager.GetEntity(entityID)
	if err != nil {
		return nil, err
	}

	if len(entity.Adapters) == 0 {
		return nil, fmt.Errorf("entity has no adapters")
	}

	// Use first adapter
	adapter, err := b.layer1Manager.GetAdapter(entity.Adapters[0])
	if err != nil {
		return nil, err
	}

	// Execute command
	return adapter.SendCommand(ctx, command)
}

// SyncAllAdapterHealth syncs health for all adapters
func (b *Bridge) SyncAllAdapterHealth(ctx context.Context) error {
	adapters := b.layer1Manager.ListAdapters()

	for _, adapter := range adapters {
		adapterID := adapter.GetConfig().ID
		_ = b.SyncAdapterHealth(ctx, adapterID)
	}

	return nil
}

// StartPeriodicSync starts periodic health synchronization
func (b *Bridge) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = b.SyncAllAdapterHealth(ctx)
		}
	}
}

// Helper functions

func (b *Bridge) stateToEntity(adapterID string, state *layer1.StateData) *layer2.Entity {
	// Infer entity type from state data
	entityType := layer2.EntityTypeServer
	if state.Data != nil {
		if _, ok := state.Data["api_version"]; ok {
			entityType = layer2.EntityTypeAPI
		}
		if _, ok := state.Data["database_type"]; ok {
			entityType = layer2.EntityTypeDatabase
		}
	}

	// Extract endpoint from metadata if available
	endpoint := ""
	if state.Metadata != nil {
		if ep, ok := state.Metadata["endpoint"]; ok {
			endpoint = ep
		}
	}

	return &layer2.Entity{
		ID:       fmt.Sprintf("entity-%s", adapterID),
		Type:     entityType,
		Name:     fmt.Sprintf("Entity from %s", adapterID),
		Endpoint: endpoint,
		Metadata: state.Data,
		State:    layer2.StateUnknown,
	}
}

func (b *Bridge) healthToState(status layer1.HealthStatus) layer2.EntityState {
	switch status {
	case layer1.HealthStatusHealthy:
		return layer2.StateHealthy
	case layer1.HealthStatusDegraded:
		return layer2.StateDegraded
	case layer1.HealthStatusUnhealthy:
		return layer2.StateUnhealthy
	default:
		return layer2.StateUnknown
	}
}

func (b *Bridge) calculateHealthScore(health *layer1.AdapterHealth) float64 {
	if health.Status == layer1.HealthStatusHealthy {
		return 100.0
	} else if health.Status == layer1.HealthStatusDegraded {
		return 60.0
	} else if health.Status == layer1.HealthStatusUnhealthy {
		return 20.0
	}
	return 0.0
}

// GetEntityState gets current state of an entity via its adapter
func (b *Bridge) GetEntityState(ctx context.Context, entityID string) (*layer1.StateData, error) {
	entity, err := b.layer2Manager.GetEntity(entityID)
	if err != nil {
		return nil, err
	}

	if len(entity.Adapters) == 0 {
		return nil, fmt.Errorf("entity has no adapters")
	}

	adapter, err := b.layer1Manager.GetAdapter(entity.Adapters[0])
	if err != nil {
		return nil, err
	}

	return adapter.ReadState(ctx)
}
