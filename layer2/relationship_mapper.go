package layer2

import (
	"context"
	"fmt"
	"sync"
)

// RelationshipMapper discovers dependencies between entities
type RelationshipMapper struct {
	catalog *EntityCatalog
	mappers map[RelationType]RelationshipDetector
	mu      sync.RWMutex
}

// RelationshipDetector detects relationships between entities
type RelationshipDetector interface {
	Detect(ctx context.Context, source *Entity, catalog *EntityCatalog) ([]Relationship, error)
	RelationType() RelationType
}

// NewRelationshipMapper creates a new relationship mapper
func NewRelationshipMapper(catalog *EntityCatalog) *RelationshipMapper {
	mapper := &RelationshipMapper{
		catalog: catalog,
		mappers: make(map[RelationType]RelationshipDetector),
	}

	// Register default detectors
	mapper.RegisterDetector(&DependencyDetector{})
	mapper.RegisterDetector(&ProviderDetector{})
	mapper.RegisterDetector(&HierarchyDetector{})

	return mapper
}

// RegisterDetector adds a relationship detector
func (m *RelationshipMapper) RegisterDetector(detector RelationshipDetector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mappers[detector.RelationType()] = detector
}

// MapRelationships discovers all relationships for an entity
func (m *RelationshipMapper) MapRelationships(ctx context.Context, entityID string) error {
	entity, err := m.catalog.Get(entityID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	detectors := make([]RelationshipDetector, 0, len(m.mappers))
	for _, detector := range m.mappers {
		detectors = append(detectors, detector)
	}
	m.mu.RUnlock()

	// Clear existing relationships
	entity.Relationships = make([]Relationship, 0)

	// Run all detectors
	for _, detector := range detectors {
		relationships, err := detector.Detect(ctx, entity, m.catalog)
		if err != nil {
			// Log error but continue with other detectors
			continue
		}

		entity.Relationships = append(entity.Relationships, relationships...)
	}

	return m.catalog.Register(entity)
}

// MapAllRelationships discovers relationships for all entities
func (m *RelationshipMapper) MapAllRelationships(ctx context.Context) error {
	entities := m.catalog.List()

	for _, entity := range entities {
		if err := m.MapRelationships(ctx, entity.ID); err != nil {
			// Log error but continue with other entities
			continue
		}
	}

	return nil
}

// GetDependencyGraph builds a dependency graph for an entity
func (m *RelationshipMapper) GetDependencyGraph(entityID string, maxDepth int) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*Entity),
		Edges: make([]GraphEdge, 0),
	}

	visited := make(map[string]bool)
	if err := m.buildGraph(entityID, 0, maxDepth, graph, visited); err != nil {
		return nil, err
	}

	return graph, nil
}

func (m *RelationshipMapper) buildGraph(entityID string, depth, maxDepth int, graph *DependencyGraph, visited map[string]bool) error {
	if depth > maxDepth || visited[entityID] {
		return nil
	}

	entity, err := m.catalog.Get(entityID)
	if err != nil {
		return err
	}

	visited[entityID] = true
	graph.Nodes[entityID] = entity

	// Add edges for dependencies
	for _, rel := range entity.Relationships {
		edge := GraphEdge{
			From:         entityID,
			To:           rel.TargetID,
			Relationship: rel.Type,
		}
		graph.Edges = append(graph.Edges, edge)

		// Recursively build graph for dependencies
		if err := m.buildGraph(rel.TargetID, depth+1, maxDepth, graph, visited); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// DependencyGraph represents entity dependencies
type DependencyGraph struct {
	Nodes map[string]*Entity
	Edges []GraphEdge
}

// GraphEdge represents a connection in the dependency graph
type GraphEdge struct {
	From         string
	To           string
	Relationship RelationType
}

// DependencyDetector detects "depends_on" relationships
type DependencyDetector struct{}

func (d *DependencyDetector) RelationType() RelationType {
	return RelationDependsOn
}

func (d *DependencyDetector) Detect(ctx context.Context, source *Entity, catalog *EntityCatalog) ([]Relationship, error) {
	relationships := make([]Relationship, 0)

	// Check metadata for explicit dependencies
	if deps, exists := source.Metadata["depends_on"]; exists {
		if depList, ok := deps.([]string); ok {
			for _, depID := range depList {
				relationships = append(relationships, Relationship{
					Type:     RelationDependsOn,
					TargetID: depID,
					Metadata: map[string]interface{}{
						"explicit": true,
					},
				})
			}
		}
	}

	// Infer dependencies based on entity type
	switch source.Type {
	case EntityTypeAPI:
		// APIs typically depend on databases
		databases := catalog.ListByType(EntityTypeDatabase)
		for _, db := range databases {
			// Check if API references this database
			if d.referencesEntity(source, db) {
				relationships = append(relationships, Relationship{
					Type:     RelationDependsOn,
					TargetID: db.ID,
					Metadata: map[string]interface{}{
						"inferred": true,
						"reason":   "database_connection",
					},
				})
			}
		}

	case EntityTypeJob:
		// Jobs might depend on APIs or databases
		apis := catalog.ListByType(EntityTypeAPI)
		for _, api := range apis {
			if d.referencesEntity(source, api) {
				relationships = append(relationships, Relationship{
					Type:     RelationDependsOn,
					TargetID: api.ID,
					Metadata: map[string]interface{}{
						"inferred": true,
						"reason":   "api_call",
					},
				})
			}
		}
	}

	return relationships, nil
}

func (d *DependencyDetector) referencesEntity(source, target *Entity) bool {
	// Check if source's endpoint or config references target
	if source.Endpoint != "" && target.Endpoint != "" {
		// Simple check - in real implementation would be more sophisticated
		return false
	}

	// Check metadata for references
	if dbHost, exists := source.Metadata["database_host"]; exists {
		if targetHost, ok := target.Metadata["host"]; ok {
			return dbHost == targetHost
		}
	}

	return false
}

// ProviderDetector detects "provides_to" relationships
type ProviderDetector struct{}

func (p *ProviderDetector) RelationType() RelationType {
	return RelationProvidesTo
}

func (p *ProviderDetector) Detect(ctx context.Context, source *Entity, catalog *EntityCatalog) ([]Relationship, error) {
	relationships := make([]Relationship, 0)

	// Find entities that depend on this one
	allEntities := catalog.List()
	for _, entity := range allEntities {
		for _, rel := range entity.Relationships {
			if rel.Type == RelationDependsOn && rel.TargetID == source.ID {
				relationships = append(relationships, Relationship{
					Type:     RelationProvidesTo,
					TargetID: entity.ID,
					Metadata: map[string]interface{}{
						"inferred": true,
					},
				})
			}
		}
	}

	return relationships, nil
}

// HierarchyDetector detects "part_of" relationships
type HierarchyDetector struct{}

func (h *HierarchyDetector) RelationType() RelationType {
	return RelationPartOf
}

func (h *HierarchyDetector) Detect(ctx context.Context, source *Entity, catalog *EntityCatalog) ([]Relationship, error) {
	relationships := make([]Relationship, 0)

	// Check metadata for parent/cluster information
	if parentID, exists := source.Metadata["parent_id"]; exists {
		if parentIDStr, ok := parentID.(string); ok {
			relationships = append(relationships, Relationship{
				Type:     RelationPartOf,
				TargetID: parentIDStr,
				Metadata: map[string]interface{}{
					"explicit": true,
				},
			})
		}
	}

	// Check for cluster membership
	if clusterID, exists := source.Metadata["cluster_id"]; exists {
		if clusterIDStr, ok := clusterID.(string); ok {
			relationships = append(relationships, Relationship{
				Type:     RelationPartOf,
				TargetID: clusterIDStr,
				Metadata: map[string]interface{}{
					"type": "cluster",
				},
			})
		}
	}

	return relationships, nil
}

// GetImpactedEntities returns all entities that would be affected if the given entity fails
func (m *RelationshipMapper) GetImpactedEntities(entityID string) ([]*Entity, error) {
	entity, err := m.catalog.Get(entityID)
	if err != nil {
		return nil, err
	}

	impacted := make(map[string]*Entity)
	m.collectImpactedEntities(entity, impacted)

	result := make([]*Entity, 0, len(impacted))
	for _, e := range impacted {
		result = append(result, e)
	}

	return result, nil
}

func (m *RelationshipMapper) collectImpactedEntities(entity *Entity, impacted map[string]*Entity) {
	// Find entities that depend on this one
	allEntities := m.catalog.List()
	for _, e := range allEntities {
		for _, rel := range e.Relationships {
			if rel.Type == RelationDependsOn && rel.TargetID == entity.ID {
				if _, exists := impacted[e.ID]; !exists {
					impacted[e.ID] = e
					// Recursively find entities that depend on this one
					m.collectImpactedEntities(e, impacted)
				}
			}
		}
	}
}

// ValidateDependencies checks for circular dependencies
func (m *RelationshipMapper) ValidateDependencies() error {
	entities := m.catalog.List()

	for _, entity := range entities {
		visited := make(map[string]bool)
		if err := m.detectCycle(entity.ID, visited, make(map[string]bool)); err != nil {
			return err
		}
	}

	return nil
}

func (m *RelationshipMapper) detectCycle(entityID string, visited, recStack map[string]bool) error {
	visited[entityID] = true
	recStack[entityID] = true

	entity, err := m.catalog.Get(entityID)
	if err != nil {
		return nil // Entity not found, skip
	}

	for _, rel := range entity.GetRelationships(RelationDependsOn) {
		if !visited[rel.TargetID] {
			if err := m.detectCycle(rel.TargetID, visited, recStack); err != nil {
				return err
			}
		} else if recStack[rel.TargetID] {
			return fmt.Errorf("circular dependency detected: %s -> %s", entityID, rel.TargetID)
		}
	}

	recStack[entityID] = false
	return nil
}
