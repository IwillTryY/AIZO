package layer4

import (
	"fmt"
	"time"
)

// ChangeDetector detects changes and drift in entity states
type ChangeDetector struct {
	thresholds map[string]float64
}

// NewChangeDetector creates a new change detector
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{
		thresholds: map[string]float64{
			"low":      10.0,
			"medium":   30.0,
			"high":     60.0,
			"critical": 90.0,
		},
	}
}

// DetectDrift detects drift between desired and actual state
func (d *ChangeDetector) DetectDrift(desired, actual map[string]interface{}) *StateDrift {
	drift := &StateDrift{
		HasDrift:    false,
		DetectedAt:  time.Now(),
		Differences: make([]StateDifference, 0),
		DriftScore:  0.0,
	}

	if desired == nil || actual == nil {
		return drift
	}

	totalFields := len(desired)
	driftingFields := 0

	// Check each desired field
	for key, desiredValue := range desired {
		actualValue, exists := actual[key]

		if !exists {
			drift.Differences = append(drift.Differences, StateDifference{
				Field:        key,
				DesiredValue: desiredValue,
				ActualValue:  nil,
				Severity:     "high",
			})
			drift.HasDrift = true
			driftingFields++
		} else if !d.valuesEqual(desiredValue, actualValue) {
			severity := d.calculateSeverity(key, desiredValue, actualValue)
			drift.Differences = append(drift.Differences, StateDifference{
				Field:        key,
				DesiredValue: desiredValue,
				ActualValue:  actualValue,
				Severity:     severity,
			})
			drift.HasDrift = true
			driftingFields++
		}
	}

	// Check for extra fields in actual state
	for key, actualValue := range actual {
		if _, exists := desired[key]; !exists {
			drift.Differences = append(drift.Differences, StateDifference{
				Field:        key,
				DesiredValue: nil,
				ActualValue:  actualValue,
				Severity:     "low",
			})
			drift.HasDrift = true
			driftingFields++
		}
	}

	// Calculate drift score
	if totalFields > 0 {
		drift.DriftScore = (float64(driftingFields) / float64(totalFields)) * 100.0
	}

	return drift
}

// DetectChanges detects changes between two states
func (d *ChangeDetector) DetectChanges(oldState, newState *EntityState) []*StateChange {
	changes := make([]*StateChange, 0)

	if oldState == nil || newState == nil {
		return changes
	}

	// Check status change
	if oldState.Status != newState.Status {
		changes = append(changes, &StateChange{
			ID:         fmt.Sprintf("change-%d", time.Now().UnixNano()),
			EntityID:   newState.EntityID,
			Timestamp:  time.Now(),
			ChangeType: ChangeTypeStatusChange,
			Field:      "status",
			OldValue:   oldState.Status,
			NewValue:   newState.Status,
			Source:     "change_detector",
		})
	}

	// Check actual state changes
	for key, newValue := range newState.ActualState {
		oldValue, exists := oldState.ActualState[key]
		if !exists || !d.valuesEqual(oldValue, newValue) {
			changes = append(changes, &StateChange{
				ID:         fmt.Sprintf("change-%d-%s", time.Now().UnixNano(), key),
				EntityID:   newState.EntityID,
				Timestamp:  time.Now(),
				ChangeType: ChangeTypeUpdate,
				Field:      key,
				OldValue:   oldValue,
				NewValue:   newValue,
				Source:     "change_detector",
			})
		}
	}

	return changes
}

// valuesEqual compares two values for equality
func (d *ChangeDetector) valuesEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// calculateSeverity calculates the severity of a drift
func (d *ChangeDetector) calculateSeverity(field string, desired, actual interface{}) string {
	// Critical fields
	criticalFields := []string{"version", "security_group", "firewall_rules", "encryption"}
	for _, cf := range criticalFields {
		if field == cf {
			return "critical"
		}
	}

	// High priority fields
	highFields := []string{"port", "protocol", "endpoint", "credentials"}
	for _, hf := range highFields {
		if field == hf {
			return "high"
		}
	}

	// Medium priority fields
	mediumFields := []string{"timeout", "retry_count", "buffer_size"}
	for _, mf := range mediumFields {
		if field == mf {
			return "medium"
		}
	}

	return "low"
}

// CalculateDriftScore calculates an overall drift score
func (d *ChangeDetector) CalculateDriftScore(drift *StateDrift) float64 {
	if drift == nil || !drift.HasDrift {
		return 0.0
	}

	score := 0.0
	weights := map[string]float64{
		"low":      1.0,
		"medium":   2.5,
		"high":     5.0,
		"critical": 10.0,
	}

	for _, diff := range drift.Differences {
		score += weights[diff.Severity]
	}

	// Normalize to 0-100
	maxScore := float64(len(drift.Differences)) * weights["critical"]
	if maxScore > 0 {
		return (score / maxScore) * 100.0
	}

	return 0.0
}
