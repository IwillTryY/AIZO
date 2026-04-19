package layer4

import (
	"time"
)

// SystemState represents the complete state of the system
type SystemState struct {
	Entities      map[string]*EntityState `json:"entities"`
	Relationships *RelationshipGraph      `json:"relationships"`
	Health        map[string]*HealthState `json:"health"`
	Topology      *NetworkTopology        `json:"topology"`
	Timestamp     time.Time               `json:"timestamp"`
	Version       int64                   `json:"version"`
}

// EntityState represents the state of a single entity
type EntityState struct {
	EntityID       string                 `json:"entity_id"`
	Type           string                 `json:"type"`
	Status         EntityStatus           `json:"status"`
	DesiredState   map[string]interface{} `json:"desired_state"`
	ActualState    map[string]interface{} `json:"actual_state"`
	Configuration  map[string]interface{} `json:"configuration"`
	Metadata       map[string]interface{} `json:"metadata"`
	LastUpdated    time.Time              `json:"last_updated"`
	LastSeen       time.Time              `json:"last_seen"`
	Version        int64                  `json:"version"`
	Drift          *StateDrift            `json:"drift,omitempty"`
}

// EntityStatus represents the operational status
type EntityStatus string

const (
	StatusOnline    EntityStatus = "online"
	StatusOffline   EntityStatus = "offline"
	StatusDegraded  EntityStatus = "degraded"
	StatusMaintenance EntityStatus = "maintenance"
	StatusUnknown   EntityStatus = "unknown"
)

// StateDrift represents differences between desired and actual state
type StateDrift struct {
	HasDrift      bool                   `json:"has_drift"`
	DetectedAt    time.Time              `json:"detected_at"`
	Differences   []StateDifference      `json:"differences"`
	DriftScore    float64                `json:"drift_score"` // 0-100
}

// StateDifference represents a single difference
type StateDifference struct {
	Field        string      `json:"field"`
	DesiredValue interface{} `json:"desired_value"`
	ActualValue  interface{} `json:"actual_value"`
	Severity     string      `json:"severity"` // low, medium, high, critical
}

// HealthState represents the health status of an entity
type HealthState struct {
	EntityID      string             `json:"entity_id"`
	Status        string             `json:"status"`
	Score         float64            `json:"score"` // 0-100
	Checks        []HealthCheck      `json:"checks"`
	LastCheck     time.Time          `json:"last_check"`
	Issues        []HealthIssue      `json:"issues"`
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthIssue represents a health problem
type HealthIssue struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	DetectedAt  time.Time              `json:"detected_at"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// RelationshipGraph represents entity relationships
type RelationshipGraph struct {
	Nodes map[string]*GraphNode `json:"nodes"`
	Edges []*GraphEdge          `json:"edges"`
}

// GraphNode represents a node in the relationship graph
type GraphNode struct {
	EntityID   string                 `json:"entity_id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// GraphEdge represents an edge in the relationship graph
type GraphEdge struct {
	From         string                 `json:"from"`
	To           string                 `json:"to"`
	Type         string                 `json:"type"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// NetworkTopology represents the network topology
type NetworkTopology struct {
	Clusters  map[string]*Cluster  `json:"clusters"`
	Zones     map[string]*Zone     `json:"zones"`
	Networks  map[string]*Network  `json:"networks"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// Cluster represents a cluster of entities
type Cluster struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Entities []string `json:"entities"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Zone represents a logical or physical zone
type Zone struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"` // availability_zone, region, datacenter
	Entities []string `json:"entities"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Network represents a network segment
type Network struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	CIDR     string   `json:"cidr"`
	Entities []string `json:"entities"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StateChange represents a change in state
type StateChange struct {
	ID           string                 `json:"id"`
	EntityID     string                 `json:"entity_id"`
	Timestamp    time.Time              `json:"timestamp"`
	ChangeType   ChangeType             `json:"change_type"`
	Field        string                 `json:"field,omitempty"`
	OldValue     interface{}            `json:"old_value,omitempty"`
	NewValue     interface{}            `json:"new_value,omitempty"`
	Source       string                 `json:"source"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ChangeType represents the type of state change
type ChangeType string

const (
	ChangeTypeCreate       ChangeType = "create"
	ChangeTypeUpdate       ChangeType = "update"
	ChangeTypeDelete       ChangeType = "delete"
	ChangeTypeStatusChange ChangeType = "status_change"
	ChangeTypeDrift        ChangeType = "drift"
	ChangeTypeReconcile    ChangeType = "reconcile"
)

// StateQuery represents a query for state data
type StateQuery struct {
	EntityIDs    []string               `json:"entity_ids,omitempty"`
	EntityTypes  []string               `json:"entity_types,omitempty"`
	Status       []EntityStatus         `json:"status,omitempty"`
	HasDrift     *bool                  `json:"has_drift,omitempty"`
	Timestamp    *time.Time             `json:"timestamp,omitempty"` // For time-travel
	Filters      map[string]interface{} `json:"filters,omitempty"`
}

// StateQueryResult represents the result of a state query
type StateQueryResult struct {
	States    []*EntityState `json:"states"`
	Count     int            `json:"count"`
	Timestamp time.Time      `json:"timestamp"`
}

// ReconciliationRequest represents a request to reconcile state
type ReconciliationRequest struct {
	EntityID     string                 `json:"entity_id"`
	DesiredState map[string]interface{} `json:"desired_state"`
	Force        bool                   `json:"force"`
	DryRun       bool                   `json:"dry_run"`
}

// ReconciliationResult represents the result of reconciliation
type ReconciliationResult struct {
	EntityID      string                 `json:"entity_id"`
	Success       bool                   `json:"success"`
	Changes       []ReconciliationChange `json:"changes"`
	Errors        []string               `json:"errors,omitempty"`
	DryRun        bool                   `json:"dry_run"`
	Duration      time.Duration          `json:"duration"`
}

// ReconciliationChange represents a change made during reconciliation
type ReconciliationChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Applied  bool        `json:"applied"`
}

// Snapshot represents a point-in-time snapshot of system state
type Snapshot struct {
	ID          string       `json:"id"`
	Timestamp   time.Time    `json:"timestamp"`
	State       *SystemState `json:"state"`
	Description string       `json:"description"`
	Tags        []string     `json:"tags,omitempty"`
}
