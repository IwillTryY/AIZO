package layer4

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StateHistory manages historical state data
type StateHistory struct {
	snapshots map[string]*Snapshot
	mu        sync.RWMutex
}

// NewStateHistory creates a new state history manager
func NewStateHistory() *StateHistory {
	return &StateHistory{
		snapshots: make(map[string]*Snapshot),
	}
}

// CreateSnapshot creates a snapshot of the current system state
func (h *StateHistory) CreateSnapshot(ctx context.Context, state *SystemState, description string, tags []string) (*Snapshot, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	snapshot := &Snapshot{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		State:       state,
		Description: description,
		Tags:        tags,
	}

	h.snapshots[snapshot.ID] = snapshot

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot by ID
func (h *StateHistory) GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot, exists := h.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	return snapshot, nil
}

// ListSnapshots lists all snapshots
func (h *StateHistory) ListSnapshots(ctx context.Context) ([]*Snapshot, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshots := make([]*Snapshot, 0, len(h.snapshots))
	for _, snapshot := range h.snapshots {
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetStateAtTime retrieves the state at a specific point in time (time-travel)
func (h *StateHistory) GetStateAtTime(ctx context.Context, timestamp time.Time) (*SystemState, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Find the closest snapshot before the requested time
	var closestSnapshot *Snapshot
	var closestDiff time.Duration

	for _, snapshot := range h.snapshots {
		if snapshot.Timestamp.Before(timestamp) || snapshot.Timestamp.Equal(timestamp) {
			diff := timestamp.Sub(snapshot.Timestamp)
			if closestSnapshot == nil || diff < closestDiff {
				closestSnapshot = snapshot
				closestDiff = diff
			}
		}
	}

	if closestSnapshot == nil {
		return nil, fmt.Errorf("no snapshot found before time: %s", timestamp)
	}

	return closestSnapshot.State, nil
}

// DeleteSnapshot deletes a snapshot
func (h *StateHistory) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.snapshots[snapshotID]; !exists {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	delete(h.snapshots, snapshotID)
	return nil
}

// CompareSnapshots compares two snapshots and returns the differences
func (h *StateHistory) CompareSnapshots(ctx context.Context, snapshot1ID, snapshot2ID string) (*SnapshotComparison, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snap1, exists := h.snapshots[snapshot1ID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshot1ID)
	}

	snap2, exists := h.snapshots[snapshot2ID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshot2ID)
	}

	comparison := &SnapshotComparison{
		Snapshot1:   snap1,
		Snapshot2:   snap2,
		Differences: make([]EntityDifference, 0),
	}

	// Compare entities
	for entityID, state1 := range snap1.State.Entities {
		state2, exists := snap2.State.Entities[entityID]
		if !exists {
			comparison.Differences = append(comparison.Differences, EntityDifference{
				EntityID:   entityID,
				ChangeType: "removed",
			})
		} else {
			// Compare states
			if state1.Status != state2.Status {
				comparison.Differences = append(comparison.Differences, EntityDifference{
					EntityID:   entityID,
					ChangeType: "modified",
					Field:      "status",
					OldValue:   state1.Status,
					NewValue:   state2.Status,
				})
			}
		}
	}

	// Check for new entities
	for entityID := range snap2.State.Entities {
		if _, exists := snap1.State.Entities[entityID]; !exists {
			comparison.Differences = append(comparison.Differences, EntityDifference{
				EntityID:   entityID,
				ChangeType: "added",
			})
		}
	}

	return comparison, nil
}

// SnapshotComparison represents a comparison between two snapshots
type SnapshotComparison struct {
	Snapshot1   *Snapshot          `json:"snapshot1"`
	Snapshot2   *Snapshot          `json:"snapshot2"`
	Differences []EntityDifference `json:"differences"`
}

// EntityDifference represents a difference in an entity between snapshots
type EntityDifference struct {
	EntityID   string      `json:"entity_id"`
	ChangeType string      `json:"change_type"` // added, removed, modified
	Field      string      `json:"field,omitempty"`
	OldValue   interface{} `json:"old_value,omitempty"`
	NewValue   interface{} `json:"new_value,omitempty"`
}
