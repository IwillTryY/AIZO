package layer4

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StateStore manages the storage and retrieval of entity states
type StateStore struct {
	states  map[string]*EntityState
	history map[string][]*StateChange
	mu      sync.RWMutex
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		states:  make(map[string]*EntityState),
		history: make(map[string][]*StateChange),
	}
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

	// Store state
	s.states[state.EntityID] = state

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
