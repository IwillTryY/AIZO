package layer2

import (
	"context"
	"fmt"
	"time"
)

// Manager orchestrates all Layer 2 components
type Manager struct {
	catalog            *EntityCatalog
	discoveryEngine    *DiscoveryEngine
	capabilityDetector *CapabilityDetector
	relationshipMapper *RelationshipMapper
	registrationAPI    *RegistrationAPI

	// Background tasks
	discoveryCtx    context.Context
	discoveryCancel context.CancelFunc
}

// ManagerConfig configures the Layer 2 manager
type ManagerConfig struct {
	DiscoveryConfig     *DiscoveryConfig
	EnableAutoDiscovery bool
	StaleEntityTimeout  time.Duration
}

// NewManager creates a new Layer 2 manager
func NewManager(config *ManagerConfig) *Manager {
	catalog := NewEntityCatalog()
	capDetector := NewCapabilityDetector()
	relMapper := NewRelationshipMapper(catalog)
	discoveryEngine := NewDiscoveryEngine(config.DiscoveryConfig, catalog)
	registrationAPI := NewRegistrationAPI(catalog, capDetector, relMapper)

	return &Manager{
		catalog:            catalog,
		discoveryEngine:    discoveryEngine,
		capabilityDetector: capDetector,
		relationshipMapper: relMapper,
		registrationAPI:    registrationAPI,
	}
}

// Start starts the Layer 2 manager
func (m *Manager) Start(ctx context.Context) error {
	// Start periodic discovery if enabled
	if m.discoveryEngine != nil {
		m.discoveryCtx, m.discoveryCancel = context.WithCancel(ctx)
		go m.discoveryEngine.StartPeriodicDiscovery(m.discoveryCtx)
	}

	// Start periodic relationship mapping
	go m.periodicRelationshipMapping(ctx)

	// Start periodic cleanup
	go m.periodicCleanup(ctx)

	return nil
}

// Stop stops the Layer 2 manager
func (m *Manager) Stop() error {
	if m.discoveryCancel != nil {
		m.discoveryCancel()
	}
	return nil
}

// GetCatalog returns the entity catalog
func (m *Manager) GetCatalog() *EntityCatalog {
	return m.catalog
}

// GetRegistrationAPI returns the registration API
func (m *Manager) GetRegistrationAPI() *RegistrationAPI {
	return m.registrationAPI
}

// GetRelationshipMapper returns the relationship mapper
func (m *Manager) GetRelationshipMapper() *RelationshipMapper {
	return m.relationshipMapper
}

// Discover runs discovery manually
func (m *Manager) Discover(ctx context.Context) (*DiscoveryResult, error) {
	result, err := m.discoveryEngine.Discover(ctx)
	if err != nil {
		return nil, err
	}

	// Detect capabilities for newly discovered entities
	for _, entity := range result.Entities {
		if err := m.capabilityDetector.DetectCapabilities(ctx, entity); err != nil {
			// Log error but continue
			continue
		}
		_ = m.catalog.Register(entity)
	}

	// Map relationships for new entities
	for _, entity := range result.Entities {
		if err := m.relationshipMapper.MapRelationships(ctx, entity.ID); err != nil {
			// Log error but continue
			continue
		}
	}

	return result, nil
}

// RegisterEntity registers a new entity
func (m *Manager) RegisterEntity(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error) {
	return m.registrationAPI.Register(ctx, req)
}

// GetEntity retrieves an entity by ID
func (m *Manager) GetEntity(entityID string) (*Entity, error) {
	return m.catalog.Get(entityID)
}

// ListEntities returns all entities
func (m *Manager) ListEntities() []*Entity {
	return m.catalog.List()
}

// ListEntitiesByType returns entities of a specific type
func (m *Manager) ListEntitiesByType(entityType EntityType) []*Entity {
	return m.catalog.ListByType(entityType)
}

// SearchEntities searches for entities matching criteria
func (m *Manager) SearchEntities(criteria SearchCriteria) []*Entity {
	return m.catalog.Search(criteria)
}

// GetDependencyGraph builds a dependency graph for an entity
func (m *Manager) GetDependencyGraph(entityID string, maxDepth int) (*DependencyGraph, error) {
	return m.relationshipMapper.GetDependencyGraph(entityID, maxDepth)
}

// GetImpactedEntities returns entities that would be affected if the given entity fails
func (m *Manager) GetImpactedEntities(entityID string) ([]*Entity, error) {
	return m.relationshipMapper.GetImpactedEntities(entityID)
}

// GetStats returns Layer 2 statistics
func (m *Manager) GetStats() ManagerStats {
	catalogStats := m.catalog.GetStats()
	regStats := m.registrationAPI.GetRegistrationStats()

	return ManagerStats{
		TotalEntities:     catalogStats.TotalEntities,
		ByType:            catalogStats.ByType,
		ByState:           catalogStats.ByState,
		HealthyEntities:   regStats.HealthyEntities,
		UnhealthyEntities: regStats.UnhealthyEntities,
	}
}

// ManagerStats contains Layer 2 statistics
type ManagerStats struct {
	TotalEntities     int
	ByType            map[EntityType]int
	ByState           map[EntityState]int
	HealthyEntities   int
	UnhealthyEntities int
}

// periodicRelationshipMapping runs relationship mapping periodically
func (m *Manager) periodicRelationshipMapping(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.relationshipMapper.MapAllRelationships(ctx)
		}
	}
}

// periodicCleanup removes stale entities periodically
func (m *Manager) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Remove entities not seen in 24 hours
			removed := m.catalog.CleanupStale(24 * time.Hour)
			if removed > 0 {
				fmt.Printf("Cleaned up %d stale entities\n", removed)
			}
		}
	}
}

// ValidateSystem validates the entire system for issues
func (m *Manager) ValidateSystem() []ValidationError {
	errors := make([]ValidationError, 0)

	// Check for circular dependencies
	if err := m.relationshipMapper.ValidateDependencies(); err != nil {
		errors = append(errors, ValidationError{
			Type:    "circular_dependency",
			Message: err.Error(),
		})
	}

	// Check for orphaned entities (no relationships)
	entities := m.catalog.List()
	for _, entity := range entities {
		if len(entity.Relationships) == 0 && entity.Type != EntityTypeServer {
			errors = append(errors, ValidationError{
				Type:     "orphaned_entity",
				EntityID: entity.ID,
				Message:  fmt.Sprintf("entity %s has no relationships", entity.ID),
			})
		}
	}

	// Check for entities with no adapters
	for _, entity := range entities {
		if len(entity.Adapters) == 0 {
			errors = append(errors, ValidationError{
				Type:     "no_adapters",
				EntityID: entity.ID,
				Message:  fmt.Sprintf("entity %s has no adapters configured", entity.ID),
			})
		}
	}

	return errors
}

// ValidationError represents a system validation error
type ValidationError struct {
	Type     string
	EntityID string
	Message  string
}
