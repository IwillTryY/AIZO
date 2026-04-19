package policy

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Engine evaluates policies against requests
type Engine struct {
	policies []*Policy
	limiter  *RateLimiter
	mu       sync.RWMutex
}

// NewEngine creates a new policy engine
func NewEngine() *Engine {
	return &Engine{
		policies: make([]*Policy, 0),
		limiter:  NewRateLimiter(),
	}
}

// AddPolicy adds a policy to the engine
func (e *Engine) AddPolicy(p *Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Replace if same ID exists
	for i, existing := range e.policies {
		if existing.ID == p.ID {
			e.policies[i] = p
			e.sortPolicies()
			return
		}
	}

	e.policies = append(e.policies, p)
	e.sortPolicies()
}

// RemovePolicy removes a policy by ID
func (e *Engine) RemovePolicy(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, p := range e.policies {
		if p.ID == id {
			e.policies = append(e.policies[:i], e.policies[i+1:]...)
			return
		}
	}
}

// ListPolicies returns all policies
func (e *Engine) ListPolicies() []*Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Policy, len(e.policies))
	copy(result, e.policies)
	return result
}

// Evaluate evaluates a request against all policies
// Returns allow if any policy explicitly allows, deny if any denies
// Default is deny (deny-by-default)
func (e *Engine) Evaluate(req EvalRequest) EvalResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check rate limits first
	if !e.limiter.Allow(req.Actor, req.Action) {
		return EvalResult{
			Allowed: false,
			Effect:  EffectDeny,
			Reason:  "rate limit exceeded",
		}
	}

	// Evaluate policies in priority order (highest priority first)
	for _, p := range e.policies {
		if !p.Enabled {
			continue
		}

		// Check tenant
		if p.TenantID != "" && req.Context["tenant_id"] != "" && p.TenantID != req.Context["tenant_id"] {
			continue
		}

		for ruleIdx, rule := range p.Rules {
			if matchRule(rule, req) {
				if matchConditions(rule.Conditions, req.Context) {
					return EvalResult{
						Allowed:  rule.Effect == EffectAllow,
						Effect:   rule.Effect,
						PolicyID: p.ID,
						RuleIdx:  ruleIdx,
						Reason:   p.Name + ": rule matched",
					}
				}
			}
		}
	}

	// No policy matched — deny by default
	return EvalResult{
		Allowed: false,
		Effect:  EffectDeny,
		Reason:  "no matching policy (default deny)",
	}
}

// SetRateLimit configures a rate limit
func (e *Engine) SetRateLimit(config RateLimitConfig) {
	e.limiter.SetLimit(config)
}

func (e *Engine) sortPolicies() {
	sort.Slice(e.policies, func(i, j int) bool {
		return e.policies[i].Priority > e.policies[j].Priority
	})
}

func matchRule(rule Rule, req EvalRequest) bool {
	actionMatch := len(rule.Actions) == 0 || matchAny(rule.Actions, req.Action)
	resourceMatch := len(rule.Resources) == 0 || matchAny(rule.Resources, req.Resource)
	actorMatch := len(rule.Actors) == 0 || matchAny(rule.Actors, req.Actor)
	return actionMatch && resourceMatch && actorMatch
}

func matchAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if pattern == "*" {
			return true
		}
		if matched, _ := filepath.Match(pattern, value); matched {
			return true
		}
		if strings.EqualFold(pattern, value) {
			return true
		}
	}
	return false
}

func matchConditions(conditions []Condition, ctx map[string]string) bool {
	for _, cond := range conditions {
		val, ok := ctx[cond.Field]
		if !ok {
			return false
		}
		switch cond.Operator {
		case "eq":
			if val != cond.Value {
				return false
			}
		case "ne":
			if val == cond.Value {
				return false
			}
		case "in":
			found := false
			for _, v := range strings.Split(cond.Value, ",") {
				if strings.TrimSpace(v) == val {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
