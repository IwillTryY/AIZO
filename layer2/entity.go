package layer2

import (
	"time"
)

// EntityType represents the type of entity
type EntityType string

const (
	EntityTypeServer   EntityType = "server"
	EntityTypeAPI      EntityType = "api"
	EntityTypeDatabase EntityType = "database"
	EntityTypeJob      EntityType = "job"
	EntityTypePipeline EntityType = "pipeline"
	EntityTypeDevice   EntityType = "device"
	EntityTypeScript   EntityType = "script"
)

// Capability represents what an entity can do
type Capability string

const (
	CapabilityMetrics      Capability = "metrics"
	CapabilityLogs         Capability = "logs"
	CapabilityTraces       Capability = "traces"
	CapabilityCommands     Capability = "commands"
	CapabilityHealthChecks Capability = "health_checks"
)

// RelationType represents the type of relationship between entities
type RelationType string

const (
	RelationDependsOn RelationType = "depends_on"
	RelationProvidesTo RelationType = "provides_to"
	RelationPartOf     RelationType = "part_of"
)

// EntityState represents the current operational state
type EntityState string

const (
	StateUnknown   EntityState = "unknown"
	StateHealthy   EntityState = "healthy"
	StateDegraded  EntityState = "degraded"
	StateUnhealthy EntityState = "unhealthy"
	StateOffline   EntityState = "offline"
)

// Relationship represents a connection between entities
type Relationship struct {
	Type     RelationType `json:"type"`
	TargetID string       `json:"target_id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Entity represents a managed entity in the system
type Entity struct {
	ID            string                 `json:"id"`
	Type          EntityType             `json:"type"`
	Name          string                 `json:"name"`
	Capabilities  []Capability           `json:"capabilities"`
	Metadata      map[string]interface{} `json:"metadata"`
	Adapters      []string               `json:"adapters"` // List of adapter IDs
	Relationships []Relationship         `json:"relationships"`
	State         EntityState            `json:"state"`

	// Discovery information
	DiscoveredAt time.Time `json:"discovered_at"`
	DiscoveredBy string    `json:"discovered_by"` // Discovery method
	LastSeenAt   time.Time `json:"last_seen_at"`

	// Connection details
	Endpoint string                 `json:"endpoint,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`

	// Health tracking
	HealthScore    float64   `json:"health_score"` // 0-100
	LastHealthCheck time.Time `json:"last_health_check"`
}

// AddCapability adds a capability to the entity
func (e *Entity) AddCapability(cap Capability) {
	for _, existing := range e.Capabilities {
		if existing == cap {
			return
		}
	}
	e.Capabilities = append(e.Capabilities, cap)
}

// HasCapability checks if entity has a specific capability
func (e *Entity) HasCapability(cap Capability) bool {
	for _, c := range e.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// AddRelationship adds a relationship to another entity
func (e *Entity) AddRelationship(relType RelationType, targetID string, metadata map[string]interface{}) {
	e.Relationships = append(e.Relationships, Relationship{
		Type:     relType,
		TargetID: targetID,
		Metadata: metadata,
	})
}

// GetRelationships returns all relationships of a specific type
func (e *Entity) GetRelationships(relType RelationType) []Relationship {
	var results []Relationship
	for _, rel := range e.Relationships {
		if rel.Type == relType {
			results = append(results, rel)
		}
	}
	return results
}

// UpdateState updates the entity's state and health score
func (e *Entity) UpdateState(state EntityState, healthScore float64) {
	e.State = state
	e.HealthScore = healthScore
	e.LastHealthCheck = time.Now()
	e.LastSeenAt = time.Now()
}
