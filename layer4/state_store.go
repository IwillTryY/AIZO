package layer4

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// StateStore manages the storage and retrieval of entity states
type StateStore struct {
	states  map[string]*EntityState
	history map[string][]*StateChange
	db      *sql.DB // Optional: if provided, persist to database
	mu      sync.RWMutex
}

// NewStateStore creates a new state store
func NewStateStore(db *sql.DB) *StateStore {
	s := &StateStore{
		states:  make(map[string]*EntityState),
		history: make(map[string][]*StateChange),
		db:      db,
	}

	// If DB provided, load existing state from database
	if db != nil {
		s.loadFromDB()
	}

	return s
}

// Set stores or updates an entity state
func (s *StateStore) Set(ctx context.Context, state *EntityState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state.EntityID == "" {
		return fmt.Errorf("entity ID cannot be empty")
	}

	// Get old state for change tracking
	oldState, exists := s.states[state.EntityID]

	// Update version
	if exists {
		state.Version = oldState.Version + 1
	} else {
		state.Version = 1
	}

	state.LastUpdated = time.Now()

	// Store state in memory
	s.states[state.EntityID] = state

	// Persist to database if available
	if s.db != nil {
		if err := s.persistToDB(state); err != nil {
			// Log error but don't fail the operation
			// In-memory state is still updated
			fmt.Printf("Warning: failed to persist state to DB: %v\n", err)
		}
	}

	// Record change
	if exists {
		change := &StateChange{
			ID:         fmt.Sprintf("change-%d", time.Now().UnixNano()),
			EntityID:   state.EntityID,
			Timestamp:  time.Now(),
			ChangeType: ChangeTypeUpdate,
			Source:     "state_store",
		}
		s.history[state.EntityID] = append(s.history[state.EntityID], change)
	} else {
		change := &StateChange{
			ID:         fmt.Sprintf("change-%d", time.Now().UnixNano()),
			EntityID:   state.EntityID,
			Timestamp:  time.Now(),
			ChangeType: ChangeTypeCreate,
			Source:     "state_store",
		}
		s.history[state.EntityID] = append(s.history[state.EntityID], change)
	}

	return nil
}

// Get retrieves an entity state
func (s *StateStore) Get(ctx context.Context, entityID string) (*EntityState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.states[entityID]
	if !exists {
		return nil, fmt.Errorf("entity state not found: %s", entityID)
	}

	return state, nil
}

// Delete removes an entity state
func (s *StateStore) Delete(ctx context.Context, entityID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.states[entityID]; !exists {
		return fmt.Errorf("entity state not found: %s", entityID)
	}

	delete(s.states, entityID)

	// Delete from database if available
	if s.db != nil {
		if err := s.deleteFromDB(entityID); err != nil {
			fmt.Printf("Warning: failed to delete state from DB: %v\n", err)
		}
	}

	// Record deletion
	change := &StateChange{
		ID:         fmt.Sprintf("change-%d", time.Now().UnixNano()),
		EntityID:   entityID,
		Timestamp:  time.Now(),
		ChangeType: ChangeTypeDelete,
		Source:     "state_store",
	}
	s.history[entityID] = append(s.history[entityID], change)

	return nil
}

// List returns all entity states
func (s *StateStore) List(ctx context.Context) ([]*EntityState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]*EntityState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, state)
	}

	return states, nil
}

// Query queries entity states based on criteria
func (s *StateStore) Query(ctx context.Context, query *StateQuery) (*StateQueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*EntityState, 0)

	for _, state := range s.states {
		if s.matchesQuery(state, query) {
			results = append(results, state)
		}
	}

	return &StateQueryResult{
		States:    results,
		Count:     len(results),
		Timestamp: time.Now(),
	}, nil
}

// GetHistory returns the change history for an entity
func (s *StateStore) GetHistory(ctx context.Context, entityID string, limit int) ([]*StateChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.history[entityID]
	if !exists {
		return []*StateChange{}, nil
	}

	// Return most recent changes
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:], nil
	}

	return history, nil
}

// GetSystemState returns the complete system state
func (s *StateStore) GetSystemState(ctx context.Context) (*SystemState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	systemState := &SystemState{
		Entities:  make(map[string]*EntityState),
		Timestamp: time.Now(),
		Version:   time.Now().Unix(),
	}

	for id, state := range s.states {
		systemState.Entities[id] = state
	}

	return systemState, nil
}

// UpdateDesiredState updates the desired state for an entity
func (s *StateStore) UpdateDesiredState(ctx context.Context, entityID string, desiredState map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.states[entityID]
	if !exists {
		return fmt.Errorf("entity state not found: %s", entityID)
	}

	state.DesiredState = desiredState
	state.LastUpdated = time.Now()
	state.Version++

	// Persist to database if available
	if s.db != nil {
		if err := s.persistToDB(state); err != nil {
			fmt.Printf("Warning: failed to persist desired state to DB: %v\n", err)
		}
	}

	return nil
}

// UpdateActualState updates the actual state for an entity
func (s *StateStore) UpdateActualState(ctx context.Context, entityID string, actualState map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.states[entityID]
	if !exists {
		return fmt.Errorf("entity state not found: %s", entityID)
	}

	state.ActualState = actualState
	state.LastSeen = time.Now()
	state.LastUpdated = time.Now()
	state.Version++

	// Persist to database if available
	if s.db != nil {
		if err := s.persistToDB(state); err != nil {
			fmt.Printf("Warning: failed to persist actual state to DB: %v\n", err)
		}
	}

	return nil
}

// matchesQuery checks if a state matches the query criteria
func (s *StateStore) matchesQuery(state *EntityState, query *StateQuery) bool {
	// Filter by entity IDs
	if len(query.EntityIDs) > 0 {
		found := false
		for _, id := range query.EntityIDs {
			if state.EntityID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by entity types
	if len(query.EntityTypes) > 0 {
		found := false
		for _, t := range query.EntityTypes {
			if state.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by status
	if len(query.Status) > 0 {
		found := false
		for _, status := range query.Status {
			if state.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by drift
	if query.HasDrift != nil {
		hasDrift := state.Drift != nil && state.Drift.HasDrift
		if *query.HasDrift != hasDrift {
			return false
		}
	}

	return true
}

// loadFromDB loads all entity states from the database into memory
func (s *StateStore) loadFromDB() {
	if s.db == nil {
		return
	}

	rows, err := s.db.Query(`
		SELECT entity_id, type, status, desired_state, actual_state, metadata, version, updated_at
		FROM entity_state_persistent
	`)
	if err != nil {
		fmt.Printf("Warning: failed to load state from DB: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var state EntityState
		var desiredJSON, actualJSON, metadataJSON sql.NullString
		var updatedAt time.Time

		err := rows.Scan(
			&state.EntityID,
			&state.Type,
			&state.Status,
			&desiredJSON,
			&actualJSON,
			&metadataJSON,
			&state.Version,
			&updatedAt,
		)
		if err != nil {
			continue
		}

		state.LastUpdated = updatedAt

		// Unmarshal JSON fields
		if desiredJSON.Valid && desiredJSON.String != "" {
			json.Unmarshal([]byte(desiredJSON.String), &state.DesiredState)
		}
		if actualJSON.Valid && actualJSON.String != "" {
			json.Unmarshal([]byte(actualJSON.String), &state.ActualState)
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			json.Unmarshal([]byte(metadataJSON.String), &state.Metadata)
		}

		s.states[state.EntityID] = &state
	}
}

// persistToDB writes an entity state to the database
func (s *StateStore) persistToDB(state *EntityState) error {
	if s.db == nil {
		return nil
	}

	// Marshal JSON fields
	desiredJSON, _ := json.Marshal(state.DesiredState)
	actualJSON, _ := json.Marshal(state.ActualState)
	metadataJSON, _ := json.Marshal(state.Metadata)

	_, err := s.db.Exec(`
		INSERT INTO entity_state_persistent (entity_id, type, status, desired_state, actual_state, metadata, version, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id) DO UPDATE SET
			type = excluded.type,
			status = excluded.status,
			desired_state = excluded.desired_state,
			actual_state = excluded.actual_state,
			metadata = excluded.metadata,
			version = excluded.version,
			updated_at = excluded.updated_at
	`,
		state.EntityID,
		state.Type,
		state.Status,
		string(desiredJSON),
		string(actualJSON),
		string(metadataJSON),
		state.Version,
		state.LastUpdated,
	)

	return err
}

// deleteFromDB removes an entity state from the database
func (s *StateStore) deleteFromDB(entityID string) error {
	if s.db == nil {
		return nil
	}

	_, err := s.db.Exec(`DELETE FROM entity_state_persistent WHERE entity_id = ?`, entityID)
	return err
}
