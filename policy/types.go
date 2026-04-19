package policy

import "time"

// Effect represents the result of a policy evaluation
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Policy represents a set of rules
type Policy struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Rules       []Rule    `json:"rules" yaml:"rules"`
	Effect      Effect    `json:"effect" yaml:"effect"` // default effect if no rules match
	Priority    int       `json:"priority" yaml:"priority"`
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	TenantID    string    `json:"tenant_id" yaml:"tenant_id"`
	CreatedAt   time.Time `json:"created_at" yaml:"-"`
}

// Rule represents a single policy rule
type Rule struct {
	Actions    []string    `json:"actions" yaml:"actions"`       // e.g. ["container.start", "container.stop"]
	Resources  []string    `json:"resources" yaml:"resources"`   // e.g. ["web-server-*", "*"]
	Actors     []string    `json:"actors" yaml:"actors"`         // e.g. ["admin", "operator"]
	Effect     Effect      `json:"effect" yaml:"effect"`
	Conditions []Condition `json:"conditions" yaml:"conditions"`
}

// Condition represents an additional constraint on a rule
type Condition struct {
	Field    string `json:"field" yaml:"field"`       // e.g. "risk", "time"
	Operator string `json:"operator" yaml:"operator"` // e.g. "eq", "ne", "gt", "lt", "in"
	Value    string `json:"value" yaml:"value"`
}

// EvalRequest represents a request to evaluate a policy
type EvalRequest struct {
	Actor    string            `json:"actor"`
	Action   string            `json:"action"`
	Resource string            `json:"resource"`
	Context  map[string]string `json:"context"` // extra fields for conditions
}

// EvalResult represents the result of a policy evaluation
type EvalResult struct {
	Allowed  bool   `json:"allowed"`
	Effect   Effect `json:"effect"`
	PolicyID string `json:"policy_id"`
	RuleIdx  int    `json:"rule_idx"`
	Reason   string `json:"reason"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Action     string `json:"action" yaml:"action"`
	MaxPerMin  int    `json:"max_per_min" yaml:"max_per_min"`
	MaxPerHour int    `json:"max_per_hour" yaml:"max_per_hour"`
}
