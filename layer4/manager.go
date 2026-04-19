package layer4

import (
	"context"
	"database/sql"
	"time"
)

// Manager orchestrates all Layer 4 components
type Manager struct {
	stateStore           *StateStore
	changeDetector       *ChangeDetector
	reconciliationEngine *ReconciliationEngine
	stateHistory         *StateHistory
	projectionAPI        *StateProjectionAPI
}

// ManagerConfig configures the Layer 4 manager
type ManagerConfig struct {
	EnableAutoReconciliation bool
	ReconciliationInterval   time.Duration
	SnapshotInterval         time.Duration
	EnableSnapshots          bool
	DB                       *sql.DB // Optional: if provided, persist state to database
}

// NewManager creates a new Layer 4 manager
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = &ManagerConfig{
			EnableAutoReconciliation: false,
			ReconciliationInterval:   5 * time.Minute,
			SnapshotInterval:         1 * time.Hour,
			EnableSnapshots:          true,
		}
	}

	stateStore := NewStateStore(config.DB)
	changeDetector := NewChangeDetector()
	reconciliationEngine := NewReconciliationEngine(stateStore, changeDetector)
	stateHistory := NewStateHistory()
	projectionAPI := NewStateProjectionAPI(stateStore, stateHistory)

	return &Manager{
		stateStore:           stateStore,
		changeDetector:       changeDetector,
		reconciliationEngine: reconciliationEngine,
		stateHistory:         stateHistory,
		projectionAPI:        projectionAPI,
	}
}

// Start starts the Layer 4 manager
func (m *Manager) Start(ctx context.Context, config *ManagerConfig) error {
	if config.EnableAutoReconciliation {
		go m.periodicReconciliation(ctx, config.ReconciliationInterval)
	}

	if config.EnableSnapshots {
		go m.periodicSnapshots(ctx, config.SnapshotInterval)
	}

	return nil
}

// GetStateStore returns the state store
func (m *Manager) GetStateStore() *StateStore {
	return m.stateStore
}

// GetChangeDetector returns the change detector
func (m *Manager) GetChangeDetector() *ChangeDetector {
	return m.changeDetector
}

// GetReconciliationEngine returns the reconciliation engine
func (m *Manager) GetReconciliationEngine() *ReconciliationEngine {
	return m.reconciliationEngine
}

// GetStateHistory returns the state history
func (m *Manager) GetStateHistory() *StateHistory {
	return m.stateHistory
}

// GetProjectionAPI returns the projection API
func (m *Manager) GetProjectionAPI() *StateProjectionAPI {
	return m.projectionAPI
}

// SetEntityState sets the state for an entity
func (m *Manager) SetEntityState(ctx context.Context, state *EntityState) error {
	return m.stateStore.Set(ctx, state)
}

// GetEntityState gets the state for an entity
func (m *Manager) GetEntityState(ctx context.Context, entityID string) (*EntityState, error) {
	return m.stateStore.Get(ctx, entityID)
}

// UpdateDesiredState updates the desired state for an entity
func (m *Manager) UpdateDesiredState(ctx context.Context, entityID string, desiredState map[string]interface{}) error {
	return m.stateStore.UpdateDesiredState(ctx, entityID, desiredState)
}

// UpdateActualState updates the actual state for an entity
func (m *Manager) UpdateActualState(ctx context.Context, entityID string, actualState map[string]interface{}) error {
	return m.stateStore.UpdateActualState(ctx, entityID, actualState)
}

// DetectDrift detects drift for an entity
func (m *Manager) DetectDrift(ctx context.Context, entityID string) (*StateDrift, error) {
	state, err := m.stateStore.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}

	drift := m.changeDetector.DetectDrift(state.DesiredState, state.ActualState)

	// Update state with drift information
	state.Drift = drift
	_ = m.stateStore.Set(ctx, state)

	return drift, nil
}

// Reconcile reconciles an entity's state
func (m *Manager) Reconcile(ctx context.Context, req *ReconciliationRequest) (*ReconciliationResult, error) {
	return m.reconciliationEngine.Reconcile(ctx, req)
}

// GetSystemState gets the complete system state
func (m *Manager) GetSystemState(ctx context.Context) (*SystemState, error) {
	return m.stateStore.GetSystemState(ctx)
}

// CreateSnapshot creates a snapshot of the current state
func (m *Manager) CreateSnapshot(ctx context.Context, description string, tags []string) (*Snapshot, error) {
	systemState, err := m.stateStore.GetSystemState(ctx)
	if err != nil {
		return nil, err
	}

	return m.stateHistory.CreateSnapshot(ctx, systemState, description, tags)
}

// QueryState queries entity states
func (m *Manager) QueryState(ctx context.Context, query *StateQuery) (*StateQueryResult, error) {
	return m.stateStore.Query(ctx, query)
}

// GetEntityHistory gets the change history for an entity
func (m *Manager) GetEntityHistory(ctx context.Context, entityID string, limit int) ([]*StateChange, error) {
	return m.stateStore.GetHistory(ctx, entityID, limit)
}

// GetStats returns Layer 4 statistics
func (m *Manager) GetStats(ctx context.Context) (*ManagerStats, error) {
	states, err := m.stateStore.List(ctx)
	if err != nil {
		return nil, err
	}

	stats := &ManagerStats{
		TotalEntities:    len(states),
		ByStatus:         make(map[EntityStatus]int),
		EntitiesWithDrift: 0,
	}

	for _, state := range states {
		stats.ByStatus[state.Status]++
		if state.Drift != nil && state.Drift.HasDrift {
			stats.EntitiesWithDrift++
		}
	}

	snapshots, _ := m.stateHistory.ListSnapshots(ctx)
	stats.TotalSnapshots = len(snapshots)

	return stats, nil
}

// ManagerStats contains Layer 4 statistics
type ManagerStats struct {
	TotalEntities     int
	ByStatus          map[EntityStatus]int
	EntitiesWithDrift int
	TotalSnapshots    int
}

// periodicReconciliation runs reconciliation periodically
func (m *Manager) periodicReconciliation(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = m.reconciliationEngine.ReconcileAll(ctx)
		}
	}
}

// periodicSnapshots creates snapshots periodically
func (m *Manager) periodicSnapshots(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = m.CreateSnapshot(ctx, "Automatic snapshot", []string{"auto"})
		}
	}
}

// RegisterReconciler registers a reconciler for an entity type
func (m *Manager) RegisterReconciler(entityType string, reconciler Reconciler) {
	m.reconciliationEngine.RegisterReconciler(entityType, reconciler)
}

// DetectAllDrift detects drift for all entities
func (m *Manager) DetectAllDrift(ctx context.Context) (map[string]*StateDrift, error) {
	states, err := m.stateStore.List(ctx)
	if err != nil {
		return nil, err
	}

	drifts := make(map[string]*StateDrift)

	for _, state := range states {
		drift := m.changeDetector.DetectDrift(state.DesiredState, state.ActualState)
		if drift.HasDrift {
			drifts[state.EntityID] = drift

			// Update state with drift
			state.Drift = drift
			_ = m.stateStore.Set(ctx, state)
		}
	}

	return drifts, nil
}
