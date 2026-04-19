package layer2

import (
	"context"
	"fmt"
	"time"
)

// RegistrationAPI provides manual and programmatic entity registration
type RegistrationAPI struct {
	catalog           *EntityCatalog
	capabilityDetector *CapabilityDetector
	relationshipMapper *RelationshipMapper
}

// NewRegistrationAPI creates a new registration API
func NewRegistrationAPI(catalog *EntityCatalog, capDetector *CapabilityDetector, relMapper *RelationshipMapper) *RegistrationAPI {
	return &RegistrationAPI{
		catalog:           catalog,
		capabilityDetector: capDetector,
		relationshipMapper: relMapper,
	}
}

// RegistrationRequest contains entity registration details
type RegistrationRequest struct {
	ID            string                 `json:"id"`
	Type          EntityType             `json:"type"`
	Name          string                 `json:"name"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Adapters      []string               `json:"adapters,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	AutoDetect    bool                   `json:"auto_detect"` // Auto-detect capabilities
	MapRelations  bool                   `json:"map_relations"` // Auto-map relationships
}

// RegistrationResponse contains registration result
type RegistrationResponse struct {
	Entity       *Entity   `json:"entity"`
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	Warnings     []string  `json:"warnings,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Register registers a new entity
func (r *RegistrationAPI) Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error) {
	response := &RegistrationResponse{
		RegisteredAt: time.Now(),
		Warnings:     make([]string, 0),
	}

	// Validate request
	if err := r.validateRequest(req); err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("validation failed: %v", err)
		return response, err
	}

	// Create entity
	entity := &Entity{
		ID:            req.ID,
		Type:          req.Type,
		Name:          req.Name,
		Endpoint:      req.Endpoint,
		Metadata:      req.Metadata,
		Adapters:      req.Adapters,
		Config:        req.Config,
		State:         StateUnknown,
		DiscoveredBy:  "manual_registration",
		DiscoveredAt:  time.Now(),
		LastSeenAt:    time.Now(),
		Capabilities:  make([]Capability, 0),
		Relationships: make([]Relationship, 0),
	}

	// Auto-detect capabilities if requested
	if req.AutoDetect {
		if err := r.capabilityDetector.DetectCapabilities(ctx, entity); err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("capability detection failed: %v", err))
		}
	}

	// Register in catalog
	if err := r.catalog.Register(entity); err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("registration failed: %v", err)
		return response, err
	}

	// Map relationships if requested
	if req.MapRelations {
		if err := r.relationshipMapper.MapRelationships(ctx, entity.ID); err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("relationship mapping failed: %v", err))
		}
	}

	response.Entity = entity
	response.Success = true
	response.Message = "entity registered successfully"

	return response, nil
}

// Update updates an existing entity
func (r *RegistrationAPI) Update(ctx context.Context, entityID string, updates map[string]interface{}) error {
	entity, err := r.catalog.Get(entityID)
	if err != nil {
		return err
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		entity.Name = name
	}

	if endpoint, ok := updates["endpoint"].(string); ok {
		entity.Endpoint = endpoint
	}

	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		if entity.Metadata == nil {
			entity.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			entity.Metadata[k] = v
		}
	}

	if adapters, ok := updates["adapters"].([]string); ok {
		entity.Adapters = adapters
	}

	entity.LastSeenAt = time.Now()

	return r.catalog.Register(entity)
}

// Deregister removes an entity
func (r *RegistrationAPI) Deregister(ctx context.Context, entityID string) error {
	// Check if other entities depend on this one
	impacted, err := r.relationshipMapper.GetImpactedEntities(entityID)
	if err != nil {
		return err
	}

	if len(impacted) > 0 {
		return fmt.Errorf("cannot deregister: %d entities depend on this entity", len(impacted))
	}

	return r.catalog.Unregister(entityID)
}

// ForceDeregister removes an entity and updates dependent entities
func (r *RegistrationAPI) ForceDeregister(ctx context.Context, entityID string) error {
	// Get impacted entities
	impacted, err := r.relationshipMapper.GetImpactedEntities(entityID)
	if err != nil {
		return err
	}

	// Remove relationships pointing to this entity
	for _, entity := range impacted {
		newRelationships := make([]Relationship, 0)
		for _, rel := range entity.Relationships {
			if rel.TargetID != entityID {
				newRelationships = append(newRelationships, rel)
			}
		}
		entity.Relationships = newRelationships
		_ = r.catalog.Register(entity)
	}

	return r.catalog.Unregister(entityID)
}

// BulkRegister registers multiple entities
func (r *RegistrationAPI) BulkRegister(ctx context.Context, requests []*RegistrationRequest) ([]*RegistrationResponse, error) {
	responses := make([]*RegistrationResponse, 0, len(requests))

	for _, req := range requests {
		response, err := r.Register(ctx, req)
		if err != nil {
			// Continue with other registrations
			responses = append(responses, response)
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// UpdateState updates an entity's operational state
func (r *RegistrationAPI) UpdateState(ctx context.Context, entityID string, state EntityState, healthScore float64) error {
	entity, err := r.catalog.Get(entityID)
	if err != nil {
		return err
	}

	entity.UpdateState(state, healthScore)

	return r.catalog.Register(entity)
}

// AddCapability adds a capability to an entity
func (r *RegistrationAPI) AddCapability(ctx context.Context, entityID string, capability Capability) error {
	entity, err := r.catalog.Get(entityID)
	if err != nil {
		return err
	}

	entity.AddCapability(capability)

	return r.catalog.Register(entity)
}

// AddRelationship adds a relationship to an entity
func (r *RegistrationAPI) AddRelationship(ctx context.Context, entityID string, relType RelationType, targetID string, metadata map[string]interface{}) error {
	entity, err := r.catalog.Get(entityID)
	if err != nil {
		return err
	}

	// Verify target exists
	if _, err := r.catalog.Get(targetID); err != nil {
		return fmt.Errorf("target entity not found: %s", targetID)
	}

	entity.AddRelationship(relType, targetID, metadata)

	return r.catalog.Register(entity)
}

// validateRequest validates a registration request
func (r *RegistrationAPI) validateRequest(req *RegistrationRequest) error {
	if req.ID == "" {
		return fmt.Errorf("entity ID is required")
	}

	if req.Type == "" {
		return fmt.Errorf("entity type is required")
	}

	if req.Name == "" {
		return fmt.Errorf("entity name is required")
	}

	// Validate entity type
	validTypes := []EntityType{
		EntityTypeServer,
		EntityTypeAPI,
		EntityTypeDatabase,
		EntityTypeJob,
		EntityTypePipeline,
		EntityTypeDevice,
		EntityTypeScript,
	}

	valid := false
	for _, t := range validTypes {
		if req.Type == t {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid entity type: %s", req.Type)
	}

	return nil
}

// GetRegistrationStats returns registration statistics
func (r *RegistrationAPI) GetRegistrationStats() RegistrationStats {
	stats := r.catalog.GetStats()

	return RegistrationStats{
		TotalEntities:    stats.TotalEntities,
		ByType:           stats.ByType,
		ByState:          stats.ByState,
		HealthyEntities:  len(r.catalog.ListByState(StateHealthy)),
		UnhealthyEntities: len(r.catalog.ListByState(StateUnhealthy)),
	}
}

// RegistrationStats contains registration statistics
type RegistrationStats struct {
	TotalEntities     int
	ByType            map[EntityType]int
	ByState           map[EntityState]int
	HealthyEntities   int
	UnhealthyEntities int
}
