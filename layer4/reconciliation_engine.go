package layer4

import (
	"context"
	"fmt"
	"time"
)

// ReconciliationEngine compares desired vs actual state and reconciles differences
type ReconciliationEngine struct {
	stateStore      *StateStore
	changeDetector  *ChangeDetector
	reconcilers     map[string]Reconciler
}

// Reconciler interface for entity-specific reconciliation logic
type Reconciler interface {
	Reconcile(ctx context.Context, entityID string, desired, actual map[string]interface{}) (*ReconciliationResult, error)
	CanReconcile(entityType string) bool
}

// NewReconciliationEngine creates a new reconciliation engine
func NewReconciliationEngine(stateStore *StateStore, changeDetector *ChangeDetector) *ReconciliationEngine {
	return &ReconciliationEngine{
		stateStore:     stateStore,
		changeDetector: changeDetector,
		reconcilers:    make(map[string]Reconciler),
	}
}

// RegisterReconciler registers a reconciler for an entity type
func (e *ReconciliationEngine) RegisterReconciler(entityType string, reconciler Reconciler) {
	e.reconcilers[entityType] = reconciler
}

// Reconcile reconciles the state of an entity
func (e *ReconciliationEngine) Reconcile(ctx context.Context, req *ReconciliationRequest) (*ReconciliationResult, error) {
	startTime := time.Now()

	// Get current state
	state, err := e.stateStore.Get(ctx, req.EntityID)
	if err != nil {
		return nil, err
	}

	// Find appropriate reconciler
	reconciler, exists := e.reconcilers[state.Type]
	if !exists {
		return nil, fmt.Errorf("no reconciler found for entity type: %s", state.Type)
	}

	// Perform reconciliation
	result, err := reconciler.Reconcile(ctx, req.EntityID, req.DesiredState, state.ActualState)
	if err != nil {
		return nil, err
	}

	result.Duration = time.Since(startTime)
	result.DryRun = req.DryRun

	// Update desired state if not dry run
	if !req.DryRun && result.Success {
		err = e.stateStore.UpdateDesiredState(ctx, req.EntityID, req.DesiredState)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// ReconcileAll reconciles all entities with drift
func (e *ReconciliationEngine) ReconcileAll(ctx context.Context) ([]*ReconciliationResult, error) {
	// Query for entities with drift
	hasDrift := true
	query := &StateQuery{
		HasDrift: &hasDrift,
	}

	queryResult, err := e.stateStore.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	results := make([]*ReconciliationResult, 0)

	for _, state := range queryResult.States {
		req := &ReconciliationRequest{
			EntityID:     state.EntityID,
			DesiredState: state.DesiredState,
			Force:        false,
			DryRun:       false,
		}

		result, err := e.Reconcile(ctx, req)
		if err != nil {
			// Log error but continue with other entities
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// CheckDrift checks for drift in an entity's state
func (e *ReconciliationEngine) CheckDrift(ctx context.Context, entityID string) (*StateDrift, error) {
	state, err := e.stateStore.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}

	return e.changeDetector.DetectDrift(state.DesiredState, state.ActualState), nil
}

// DefaultReconciler is a basic reconciler implementation
type DefaultReconciler struct {
	entityType string
}

// NewDefaultReconciler creates a new default reconciler
func NewDefaultReconciler(entityType string) *DefaultReconciler {
	return &DefaultReconciler{
		entityType: entityType,
	}
}

// CanReconcile checks if this reconciler can handle the entity type
func (r *DefaultReconciler) CanReconcile(entityType string) bool {
	return r.entityType == entityType
}

// Reconcile performs basic reconciliation
func (r *DefaultReconciler) Reconcile(ctx context.Context, entityID string, desired, actual map[string]interface{}) (*ReconciliationResult, error) {
	result := &ReconciliationResult{
		EntityID: entityID,
		Success:  true,
		Changes:  make([]ReconciliationChange, 0),
		Errors:   make([]string, 0),
	}

	// Compare desired and actual
	for key, desiredValue := range desired {
		actualValue, exists := actual[key]

		if !exists || !equal(desiredValue, actualValue) {
			change := ReconciliationChange{
				Field:    key,
				OldValue: actualValue,
				NewValue: desiredValue,
				Applied:  true,
			}
			result.Changes = append(result.Changes, change)
		}
	}

	return result, nil
}

func equal(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
