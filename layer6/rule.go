package layer6

import "time"

// Rule represents a single detection and response rule
type Rule struct {
	ID           string      `json:"id" yaml:"id"`
	Name         string      `json:"name" yaml:"name"`
	Description  string      `json:"description" yaml:"description"`
	Conditions   []Condition `json:"conditions" yaml:"conditions"`
	Action       RuleAction  `json:"action" yaml:"action"`
	Priority     int         `json:"priority" yaml:"priority"`
	Enabled      bool        `json:"enabled" yaml:"enabled"`
	// Self-tuning stats
	SuccessCount int       `json:"success_count" yaml:"-"`
	FailureCount int       `json:"failure_count" yaml:"-"`
	LastFired    time.Time `json:"last_fired" yaml:"-"`
}

// SuccessRate returns the rule's historical success rate (0-1)
func (r *Rule) SuccessRate() float64 {
	total := r.SuccessCount + r.FailureCount
	if total == 0 {
		return 1.0 // assume good until proven otherwise
	}
	return float64(r.SuccessCount) / float64(total)
}

// Condition represents a single match condition
type Condition struct {
	// For metric-based conditions
	Metric   string  `json:"metric" yaml:"metric"`     // "memory_usage", "cpu_usage", "disk_usage"
	Operator string  `json:"operator" yaml:"operator"` // ">", "<", ">=", "<=", "=="
	Value    float64 `json:"value" yaml:"value"`

	// For event-based conditions
	EventType string `json:"event_type" yaml:"event_type"` // matches SystemEvent.Type
	Field     string `json:"field" yaml:"field"`           // matches SystemEvent fields
	Contains  string `json:"contains" yaml:"contains"`     // substring match
}

// RuleAction defines what to do when a rule matches
type RuleAction struct {
	Type        string                 `json:"type" yaml:"type"`               // "restart", "cleanup", "scale", "investigate", "alert"
	EntityID    string                 `json:"entity_id" yaml:"entity_id"`     // "" means use event's entity
	Risk        string                 `json:"risk" yaml:"risk"`               // "low", "medium", "high"
	Reversible  bool                   `json:"reversible" yaml:"reversible"`
	AutoApprove bool                   `json:"auto_approve" yaml:"auto_approve"`
	Parameters  map[string]interface{} `json:"parameters" yaml:"parameters"`
	Reasoning   string                 `json:"reasoning" yaml:"reasoning"`
}

// SuggestedRule is a rule mined from incident patterns
type SuggestedRule struct {
	Rule        *Rule   `json:"rule"`
	Confidence  float64 `json:"confidence"`
	BasedOn     int     `json:"based_on"` // number of incidents this is based on
	Description string  `json:"description"`
}
