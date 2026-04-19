package layer6

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RuleEngine evaluates rules against events and system summaries
type RuleEngine struct {
	rules []*Rule
	store *SQLiteStore
	mu    sync.RWMutex
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(store *SQLiteStore) *RuleEngine {
	return &RuleEngine{
		rules: make([]*Rule, 0),
		store: store,
	}
}

// AddRule adds a rule to the engine
func (e *RuleEngine) AddRule(rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Replace if same ID
	for i, r := range e.rules {
		if r.ID == rule.ID {
			e.rules[i] = rule
			return
		}
	}
	e.rules = append(e.rules, rule)
}

// RemoveRule removes a rule by ID
func (e *RuleEngine) RemoveRule(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return
		}
	}
}

// ListRules returns all rules
func (e *RuleEngine) ListRules() []*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Rule, len(e.rules))
	copy(result, e.rules)
	return result
}

// GetRule returns a rule by ID
func (e *RuleEngine) GetRule(id string) *Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, r := range e.rules {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Evaluate evaluates all rules against a SystemEvent
// Returns the highest-priority matching proposal, or nil
func (e *RuleEngine) Evaluate(event *SystemEvent) (*ActionProposal, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var best *Rule
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if matchesEvent(rule, event) {
			if best == nil || rule.Priority > best.Priority {
				best = rule
			}
		}
	}

	if best == nil {
		return nil, nil
	}

	return e.buildProposal(best, event.EntityID, event), nil
}

// EvaluateSummary evaluates all rules against a SystemSummary
// Returns all matching proposals
func (e *RuleEngine) EvaluateSummary(summary *SystemSummary) ([]*ActionProposal, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	proposals := make([]*ActionProposal, 0)
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if matchesSummary(rule, summary) {
			entityID := rule.Action.EntityID
			if entityID == "" {
				entityID = "system"
			}
			proposals = append(proposals, e.buildProposal(rule, entityID, nil))
		}
	}

	return proposals, nil
}

// UpdateStats updates a rule's success/failure counts
func (e *RuleEngine) UpdateStats(ruleID string, succeeded bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, r := range e.rules {
		if r.ID == ruleID {
			if succeeded {
				r.SuccessCount++
			} else {
				r.FailureCount++
			}
			return
		}
	}
}

// buildProposal creates an ActionProposal from a matched rule
func (e *RuleEngine) buildProposal(rule *Rule, entityID string, event *SystemEvent) *ActionProposal {
	// Determine if auto-approve based on rule setting and success rate
	requiresApproval := !rule.Action.AutoApprove
	if rule.Action.AutoApprove && rule.SuccessRate() < 0.7 && (rule.SuccessCount+rule.FailureCount) > 5 {
		// Demote to requiring approval if success rate dropped
		requiresApproval = true
	}

	// Confidence based on success rate
	confidence := rule.SuccessRate()
	if rule.SuccessCount+rule.FailureCount == 0 {
		confidence = 0.75 // default for new rules
	}

	reasoning := rule.Action.Reasoning
	if reasoning == "" {
		reasoning = fmt.Sprintf("Rule '%s' matched. Success rate: %.0f%% (%d/%d)",
			rule.Name, confidence*100, rule.SuccessCount, rule.SuccessCount+rule.FailureCount)
	}
	if event != nil {
		reasoning += fmt.Sprintf(" | Event: %s on %s", event.Type, event.EntityID)
	}

	params := make(map[string]interface{})
	for k, v := range rule.Action.Parameters {
		params[k] = v
	}
	if event != nil {
		params["endpoint"] = "http://localhost:8080"
		params["event_id"] = event.ID
	}

	rule.LastFired = time.Now()

	return &ActionProposal{
		ID:               uuid.New().String(),
		Timestamp:        time.Now(),
		Source:           "rule:" + rule.ID,
		RuleID:           rule.ID,
		Action:           rule.Action.Type,
		EntityID:         entityID,
		Priority:         rule.Priority,
		Risk:             rule.Action.Risk,
		Confidence:       confidence,
		Reasoning:        reasoning,
		Parameters:       params,
		RequiresApproval: requiresApproval,
		Status:           ProposalPending,
	}
}

// --- Matching logic ---

func matchesEvent(rule *Rule, event *SystemEvent) bool {
	for _, cond := range rule.Conditions {
		if cond.EventType != "" {
			if string(event.Type) != cond.EventType {
				return false
			}
		}
		if cond.Contains != "" {
			if !strings.Contains(strings.ToLower(event.Description), strings.ToLower(cond.Contains)) &&
				!strings.Contains(strings.ToLower(string(event.Type)), strings.ToLower(cond.Contains)) {
				return false
			}
		}
		if cond.Metric != "" {
			val, ok := event.Metrics[cond.Metric]
			if !ok {
				return false
			}
			if !evalOperator(val, cond.Operator, cond.Value) {
				return false
			}
		}
	}
	return len(rule.Conditions) > 0
}

func matchesSummary(rule *Rule, summary *SystemSummary) bool {
	for _, cond := range rule.Conditions {
		if cond.EventType != "" {
			return false // event-only condition, skip for summary
		}
		if cond.Metric == "" {
			continue
		}

		var val float64
		switch cond.Metric {
		case "memory_usage":
			val = summary.MemoryUsage
		case "cpu_usage":
			val = summary.CPUUsage
		case "disk_usage":
			val = summary.DiskUsage
		case "failed_containers":
			val = float64(summary.FailedContainers)
		case "unhealthy_entities":
			val = float64(summary.UnhealthyEntities)
		case "average_load":
			val = summary.AverageLoad
		default:
			// Check custom metrics map
			if v, ok := summary.Metrics[cond.Metric]; ok {
				val = v
			} else {
				return false
			}
		}

		if !evalOperator(val, cond.Operator, cond.Value) {
			return false
		}
	}
	return len(rule.Conditions) > 0
}

func evalOperator(val float64, op string, threshold float64) bool {
	switch op {
	case ">":
		return val > threshold
	case ">=":
		return val >= threshold
	case "<":
		return val < threshold
	case "<=":
		return val <= threshold
	case "==":
		return val == threshold
	case "!=":
		return val != threshold
	}
	return false
}
