package layer2

import (
	"fmt"
	"sync"
	"time"
)

// EntityCatalog is the central registry of all known entities
type EntityCatalog struct {
	entities map[string]*Entity
	mu       sync.RWMutex

	// Indexes for fast lookups
	byType     map[EntityType][]string
	byAdapter  map[string][]string
	byState    map[EntityState][]string
}

// NewEntityCatalog creates a new entity catalog
func NewEntityCatalog() *EntityCatalog {
	return &EntityCatalog{
		entities:  make(map[string]*Entity),
		byType:    make(map[EntityType][]string),
		byAdapter: make(map[string][]string),
		byState:   make(map[EntityState][]string),
	}
}

// Register adds or updates an entity in the catalog
func (c *EntityCatalog) Register(entity *Entity) error {
	if entity.ID == "" {
		return fmt.Errorf("entity ID cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove old indexes if updating
	if existing, exists := c.entities[entity.ID]; exists {
		c.removeFromIndexes(existing)
	}

	// Store entity
	c.entities[entity.ID] = entity

	// Update indexes
	c.addToIndexes(entity)

	return nil
}

// Unregister removes an entity from the catalog
func (c *EntityCatalog) Unregister(entityID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entity, exists := c.entities[entityID]
	if !exists {
		return fmt.Errorf("entity not found: %s", entityID)
	}

	c.removeFromIndexes(entity)
	delete(c.entities, entityID)

	return nil
}

// Get retrieves an entity by ID
func (c *EntityCatalog) Get(entityID string) (*Entity, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entity, exists := c.entities[entityID]
	if !exists {
		return nil, fmt.Errorf("entity not found: %s", entityID)
	}

	return entity, nil
}

// List returns all entities
func (c *EntityCatalog) List() []*Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entities := make([]*Entity, 0, len(c.entities))
	for _, entity := range c.entities {
		entities = append(entities, entity)
	}

	return entities
}

// ListByType returns all entities of a specific type
func (c *EntityCatalog) ListByType(entityType EntityType) []*Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := c.byType[entityType]
	entities := make([]*Entity, 0, len(ids))

	for _, id := range ids {
		if entity, exists := c.entities[id]; exists {
			entities = append(entities, entity)
		}
	}

	return entities
}

// ListByAdapter returns all entities using a specific adapter
func (c *EntityCatalog) ListByAdapter(adapterID string) []*Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := c.byAdapter[adapterID]
	entities := make([]*Entity, 0, len(ids))

	for _, id := range ids {
		if entity, exists := c.entities[id]; exists {
			entities = append(entities, entity)
		}
	}

	return entities
}

// ListByState returns all entities in a specific state
func (c *EntityCatalog) ListByState(state EntityState) []*Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := c.byState[state]
	entities := make([]*Entity, 0, len(ids))

	for _, id := range ids {
		if entity, exists := c.entities[id]; exists {
			entities = append(entities, entity)
		}
	}

	return entities
}

// Search finds entities matching criteria
func (c *EntityCatalog) Search(criteria SearchCriteria) []*Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*Entity

	for _, entity := range c.entities {
		if c.matchesCriteria(entity, criteria) {
			results = append(results, entity)
		}
	}

	return results
}

// SearchCriteria defines search parameters
type SearchCriteria struct {
	Type         *EntityType
	State        *EntityState
	Capability   *Capability
	Tag          string
	TagValue     string
	MinHealthScore float64
}

func (c *EntityCatalog) matchesCriteria(entity *Entity, criteria SearchCriteria) bool {
	if criteria.Type != nil && entity.Type != *criteria.Type {
		return false
	}

	if criteria.State != nil && entity.State != *criteria.State {
		return false
	}

	if criteria.Capability != nil && !entity.HasCapability(*criteria.Capability) {
		return false
	}

	if criteria.Tag != "" {
		if val, exists := entity.Metadata[criteria.Tag]; !exists {
			return false
		} else if criteria.TagValue != "" && fmt.Sprint(val) != criteria.TagValue {
			return false
		}
	}

	if criteria.MinHealthScore > 0 && entity.HealthScore < criteria.MinHealthScore {
		return false
	}

	return true
}

// GetStats returns catalog statistics
func (c *EntityCatalog) GetStats() CatalogStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CatalogStats{
		TotalEntities: len(c.entities),
		ByType:        make(map[EntityType]int),
		ByState:       make(map[EntityState]int),
	}

	for entityType, ids := range c.byType {
		stats.ByType[entityType] = len(ids)
	}

	for state, ids := range c.byState {
		stats.ByState[state] = len(ids)
	}

	return stats
}

// CatalogStats contains catalog statistics
type CatalogStats struct {
	TotalEntities int
	ByType        map[EntityType]int
	ByState       map[EntityState]int
}

// CleanupStale removes entities that haven't been seen recently
func (c *EntityCatalog) CleanupStale(maxAge time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, entity := range c.entities {
		if entity.LastSeenAt.Before(cutoff) {
			c.removeFromIndexes(entity)
			delete(c.entities, id)
			removed++
		}
	}

	return removed
}

// Helper methods for index management

func (c *EntityCatalog) addToIndexes(entity *Entity) {
	// Type index
	c.byType[entity.Type] = append(c.byType[entity.Type], entity.ID)

	// Adapter index
	for _, adapterID := range entity.Adapters {
		c.byAdapter[adapterID] = append(c.byAdapter[adapterID], entity.ID)
	}

	// State index
	c.byState[entity.State] = append(c.byState[entity.State], entity.ID)
}

func (c *EntityCatalog) removeFromIndexes(entity *Entity) {
	// Type index
	c.byType[entity.Type] = removeFromSlice(c.byType[entity.Type], entity.ID)

	// Adapter index
	for _, adapterID := range entity.Adapters {
		c.byAdapter[adapterID] = removeFromSlice(c.byAdapter[adapterID], entity.ID)
	}

	// State index
	c.byState[entity.State] = removeFromSlice(c.byState[entity.State], entity.ID)
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
