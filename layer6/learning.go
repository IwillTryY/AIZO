package layer6

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// LearningEngine tunes rules based on outcomes and mines new patterns
type LearningEngine struct {
	store  *SQLiteStore
	engine *RuleEngine
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine(store *SQLiteStore, engine *RuleEngine) *LearningEngine {
	return &LearningEngine{store: store, engine: engine}
}

// RecordOutcome records the outcome of a proposal and updates rule stats
func (l *LearningEngine) RecordOutcome(proposal *ActionProposal, succeeded bool, duration time.Duration) {
	// Update rule stats
	if proposal.RuleID != "" {
		l.engine.UpdateStats(proposal.RuleID, succeeded)
	}

	// Persist incident
	incident := HistoricalIncident{
		ID:              uuid.New().String(),
		Timestamp:       proposal.Timestamp,
		Type:            EventType(proposal.Action),
		EntityID:        proposal.EntityID,
		Description:     proposal.Reasoning,
		ActionTaken:     proposal.Action,
		ActionSucceeded: succeeded,
		Resolution:      proposal.Result,
		Duration:        duration,
	}
	l.store.StoreIncident(incident)

	// Update learning data
	pattern := fmt.Sprintf("%s:%s", proposal.RuleID, proposal.Action)
	existing, err := l.store.LoadLearningData(pattern)
	if err != nil {
		existing = &LearningData{
			IncidentPattern: pattern,
			SuccessfulFixes: make([]string, 0),
			FailedFixes:     make([]string, 0),
			Context:         make(map[string]interface{}),
		}
	}

	existing.Frequency++
	existing.LastOccurrence = time.Now()
	if succeeded {
		existing.SuccessfulFixes = append(existing.SuccessfulFixes, proposal.Action)
	} else {
		existing.FailedFixes = append(existing.FailedFixes, proposal.Action)
	}

	l.store.StoreLearningData(existing)
}

// TuneThresholds adjusts rule thresholds based on success rates
// - If success rate < 70% over last 10 fires: tighten threshold by 5%
// - If success rate > 85% over last 20 fires: relax threshold by 2%
func (l *LearningEngine) TuneThresholds() []string {
	changes := make([]string, 0)

	for _, rule := range l.engine.ListRules() {
		total := rule.SuccessCount + rule.FailureCount
		if total < 5 {
			continue // not enough data
		}

		rate := rule.SuccessRate()

		for i, cond := range rule.Conditions {
			if cond.Metric == "" || cond.Value == 0 {
				continue
			}

			if rate < 0.70 && total >= 10 {
				// Tighten: fire earlier (lower threshold for > conditions)
				oldVal := cond.Value
				if cond.Operator == ">" || cond.Operator == ">=" {
					rule.Conditions[i].Value = cond.Value * 0.95
				} else {
					rule.Conditions[i].Value = cond.Value * 1.05
				}
				changes = append(changes, fmt.Sprintf(
					"Rule '%s': tightened %s threshold %.1f → %.1f (success rate: %.0f%%)",
					rule.Name, cond.Metric, oldVal, rule.Conditions[i].Value, rate*100,
				))
			} else if rate > 0.85 && total >= 20 {
				// Relax: fire a bit later (higher threshold for > conditions)
				oldVal := cond.Value
				if cond.Operator == ">" || cond.Operator == ">=" {
					rule.Conditions[i].Value = cond.Value * 1.02
				} else {
					rule.Conditions[i].Value = cond.Value * 0.98
				}
				changes = append(changes, fmt.Sprintf(
					"Rule '%s': relaxed %s threshold %.1f → %.1f (success rate: %.0f%%)",
					rule.Name, cond.Metric, oldVal, rule.Conditions[i].Value, rate*100,
				))
			}
		}

		// Auto-approve promotion: if requires approval but succeeded 10 times in a row
		if rule.Action.AutoApprove == false && rule.SuccessCount >= 10 && rule.FailureCount == 0 {
			rule.Action.AutoApprove = true
			changes = append(changes, fmt.Sprintf(
				"Rule '%s': promoted to auto-approve (10 consecutive successes)",
				rule.Name,
			))
		}
	}

	return changes
}

// MinePatterns scans incident history for recurring patterns not covered by existing rules
func (l *LearningEngine) MinePatterns() []*SuggestedRule {
	incidents, err := l.store.LoadIncidents(200)
	if err != nil || len(incidents) < 5 {
		return nil
	}

	// Count event type frequencies
	typeCounts := make(map[string]int)
	typeSuccesses := make(map[string]int)

	for _, inc := range incidents {
		key := string(inc.Type)
		typeCounts[key]++
		if inc.ActionSucceeded {
			typeSuccesses[key]++
		}
	}

	// Find patterns that fire frequently but have no matching rule
	suggestions := make([]*SuggestedRule, 0)
	existingRules := l.engine.ListRules()

	for eventType, count := range typeCounts {
		if count < 3 {
			continue // not frequent enough
		}

		// Check if already covered by a rule
		covered := false
		for _, rule := range existingRules {
			for _, cond := range rule.Conditions {
				if cond.EventType == eventType {
					covered = true
					break
				}
			}
		}
		if covered {
			continue
		}

		successRate := float64(typeSuccesses[eventType]) / float64(count)
		action := "investigate"
		risk := "low"
		autoApprove := true

		// Suggest more aggressive action for high-frequency events
		if count >= 10 {
			action = "restart"
			risk = "medium"
			autoApprove = false
		}

		suggestions = append(suggestions, &SuggestedRule{
			Rule: &Rule{
				ID:          "suggested-" + eventType,
				Name:        fmt.Sprintf("Auto: Handle %s", eventType),
				Description: fmt.Sprintf("Suggested based on %d incidents", count),
				Conditions:  []Condition{{EventType: eventType}},
				Action: RuleAction{
					Type:        action,
					Risk:        risk,
					Reversible:  true,
					AutoApprove: autoApprove,
					Reasoning:   fmt.Sprintf("Pattern detected: %s occurred %d times", eventType, count),
				},
				Priority: 50,
				Enabled:  false, // disabled until user approves
			},
			Confidence:  successRate,
			BasedOn:     count,
			Description: fmt.Sprintf("Seen %d times, %.0f%% success rate when actioned", count, successRate*100),
		})
	}

	// Sort by frequency (most common first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].BasedOn > suggestions[j].BasedOn
	})

	return suggestions
}
