package layer4

import (
	"context"
	"fmt"
	"time"
)

// StateProjectionAPI provides query and projection capabilities for state data
type StateProjectionAPI struct {
	stateStore   *StateStore
	stateHistory *StateHistory
}

// NewStateProjectionAPI creates a new state projection API
func NewStateProjectionAPI(stateStore *StateStore, stateHistory *StateHistory) *StateProjectionAPI {
	return &StateProjectionAPI{
		stateStore:   stateStore,
		stateHistory: stateHistory,
	}
}

// GetCurrentState gets the current state of an entity
func (api *StateProjectionAPI) GetCurrentState(ctx context.Context, entityID string) (*EntityState, error) {
	return api.stateStore.Get(ctx, entityID)
}

// GetHistoricalState gets the state of an entity at a specific time
func (api *StateProjectionAPI) GetHistoricalState(ctx context.Context, entityID string, timestamp time.Time) (*EntityState, error) {
	systemState, err := api.stateHistory.GetStateAtTime(ctx, timestamp)
	if err != nil {
		return nil, err
	}

	state, exists := systemState.Entities[entityID]
	if !exists {
		return nil, fmt.Errorf("entity not found in historical state: %s", entityID)
	}

	return state, nil
}

// QueryCurrentState queries current entity states
func (api *StateProjectionAPI) QueryCurrentState(ctx context.Context, query *StateQuery) (*StateQueryResult, error) {
	return api.stateStore.Query(ctx, query)
}

// QueryHistoricalState queries historical entity states
func (api *StateProjectionAPI) QueryHistoricalState(ctx context.Context, query *StateQuery, timestamp time.Time) (*StateQueryResult, error) {
	systemState, err := api.stateHistory.GetStateAtTime(ctx, timestamp)
	if err != nil {
		return nil, err
	}

	// Filter entities based on query
	results := make([]*EntityState, 0)
	for _, state := range systemState.Entities {
		if api.matchesQuery(state, query) {
			results = append(results, state)
		}
	}

	return &StateQueryResult{
		States:    results,
		Count:     len(results),
		Timestamp: timestamp,
	}, nil
}

// GetEntityTimeline gets the timeline of changes for an entity
func (api *StateProjectionAPI) GetEntityTimeline(ctx context.Context, entityID string, start, end time.Time) (*EntityTimeline, error) {
	history, err := api.stateStore.GetHistory(ctx, entityID, 0)
	if err != nil {
		return nil, err
	}

	timeline := &EntityTimeline{
		EntityID: entityID,
		Start:    start,
		End:      end,
		Changes:  make([]*StateChange, 0),
	}

	for _, change := range history {
		if change.Timestamp.After(start) && change.Timestamp.Before(end) {
			timeline.Changes = append(timeline.Changes, change)
		}
	}

	return timeline, nil
}

// GetSystemTimeline gets the timeline of all changes in the system
func (api *StateProjectionAPI) GetSystemTimeline(ctx context.Context, start, end time.Time) (*SystemTimeline, error) {
	states, err := api.stateStore.List(ctx)
	if err != nil {
		return nil, err
	}

	timeline := &SystemTimeline{
		Start:   start,
		End:     end,
		Changes: make([]*StateChange, 0),
	}

	for _, state := range states {
		history, err := api.stateStore.GetHistory(ctx, state.EntityID, 0)
		if err != nil {
			continue
		}

		for _, change := range history {
			if change.Timestamp.After(start) && change.Timestamp.Before(end) {
				timeline.Changes = append(timeline.Changes, change)
			}
		}
	}

	return timeline, nil
}

// GetDriftReport generates a drift report for all entities
func (api *StateProjectionAPI) GetDriftReport(ctx context.Context) (*DriftReport, error) {
	hasDrift := true
	query := &StateQuery{
		HasDrift: &hasDrift,
	}

	result, err := api.stateStore.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	report := &DriftReport{
		Timestamp:         time.Now(),
		TotalEntities:     0,
		EntitiesWithDrift: len(result.States),
		DriftDetails:      make([]DriftDetail, 0),
	}

	// Get total entities
	allStates, _ := api.stateStore.List(ctx)
	report.TotalEntities = len(allStates)

	for _, state := range result.States {
		if state.Drift != nil {
			detail := DriftDetail{
				EntityID:   state.EntityID,
				EntityType: state.Type,
				DriftScore: state.Drift.DriftScore,
				Differences: state.Drift.Differences,
				DetectedAt: state.Drift.DetectedAt,
			}
			report.DriftDetails = append(report.DriftDetails, detail)
		}
	}

	return report, nil
}

// GetHealthReport generates a health report for all entities
func (api *StateProjectionAPI) GetHealthReport(ctx context.Context) (*HealthReport, error) {
	states, err := api.stateStore.List(ctx)
	if err != nil {
		return nil, err
	}

	report := &HealthReport{
		Timestamp:     time.Now(),
		TotalEntities: len(states),
		ByStatus:      make(map[EntityStatus]int),
		Issues:        make([]EntityHealthIssue, 0),
	}

	for _, state := range states {
		report.ByStatus[state.Status]++

		if state.Status == StatusOffline || state.Status == StatusDegraded {
			issue := EntityHealthIssue{
				EntityID:   state.EntityID,
				EntityType: state.Type,
				Status:     state.Status,
				LastSeen:   state.LastSeen,
			}
			report.Issues = append(report.Issues, issue)
		}
	}

	return report, nil
}

// matchesQuery checks if a state matches the query
func (api *StateProjectionAPI) matchesQuery(state *EntityState, query *StateQuery) bool {
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

	return true
}

// EntityTimeline represents the timeline of changes for an entity
type EntityTimeline struct {
	EntityID string         `json:"entity_id"`
	Start    time.Time      `json:"start"`
	End      time.Time      `json:"end"`
	Changes  []*StateChange `json:"changes"`
}

// SystemTimeline represents the timeline of all changes in the system
type SystemTimeline struct {
	Start   time.Time      `json:"start"`
	End     time.Time      `json:"end"`
	Changes []*StateChange `json:"changes"`
}

// DriftReport represents a report of all drift in the system
type DriftReport struct {
	Timestamp         time.Time     `json:"timestamp"`
	TotalEntities     int           `json:"total_entities"`
	EntitiesWithDrift int           `json:"entities_with_drift"`
	DriftDetails      []DriftDetail `json:"drift_details"`
}

// DriftDetail represents drift details for a single entity
type DriftDetail struct {
	EntityID    string             `json:"entity_id"`
	EntityType  string             `json:"entity_type"`
	DriftScore  float64            `json:"drift_score"`
	Differences []StateDifference  `json:"differences"`
	DetectedAt  time.Time          `json:"detected_at"`
}

// HealthReport represents a health report for all entities
type HealthReport struct {
	Timestamp     time.Time            `json:"timestamp"`
	TotalEntities int                  `json:"total_entities"`
	ByStatus      map[EntityStatus]int `json:"by_status"`
	Issues        []EntityHealthIssue  `json:"issues"`
}

// EntityHealthIssue represents a health issue for an entity
type EntityHealthIssue struct {
	EntityID   string       `json:"entity_id"`
	EntityType string       `json:"entity_type"`
	Status     EntityStatus `json:"status"`
	LastSeen   time.Time    `json:"last_seen"`
}
