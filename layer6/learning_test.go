package layer6

import (
	"testing"
)

func TestLearningTuneThresholds(t *testing.T) {
	engine := NewRuleEngine(nil)
	learning := NewLearningEngine(nil, engine)

	// Add a rule with poor success rate
	engine.AddRule(&Rule{
		ID:   "tune-test",
		Name: "Tune Test",
		Conditions: []Condition{
			{Metric: "memory_usage", Operator: ">", Value: 80.0},
		},
		Action:       RuleAction{Type: "cleanup", AutoApprove: false},
		Priority:     50,
		Enabled:      true,
		SuccessCount: 5,
		FailureCount: 5, // 50% success rate — should tighten
	})

	changes := learning.TuneThresholds()

	rule := engine.GetRule("tune-test")
	if rule.Conditions[0].Value >= 80.0 {
		t.Errorf("expected threshold to tighten below 80, got %.1f", rule.Conditions[0].Value)
	}
	if len(changes) == 0 {
		t.Error("expected at least one change reported")
	}
}

func TestLearningAutoApprovePromotion(t *testing.T) {
	engine := NewRuleEngine(nil)
	learning := NewLearningEngine(nil, engine)

	engine.AddRule(&Rule{
		ID:   "promote-test",
		Name: "Promote Test",
		Conditions: []Condition{
			{Metric: "cpu_usage", Operator: ">", Value: 90},
		},
		Action:       RuleAction{Type: "investigate", AutoApprove: false},
		Priority:     50,
		Enabled:      true,
		SuccessCount: 10,
		FailureCount: 0, // 100% success, 10 in a row
	})

	learning.TuneThresholds()

	rule := engine.GetRule("promote-test")
	if !rule.Action.AutoApprove {
		t.Error("expected rule to be promoted to auto-approve after 10 consecutive successes")
	}
}

func TestLearningNoChangeWithInsufficientData(t *testing.T) {
	engine := NewRuleEngine(nil)
	learning := NewLearningEngine(nil, engine)

	engine.AddRule(&Rule{
		ID:   "insufficient",
		Name: "Not Enough Data",
		Conditions: []Condition{
			{Metric: "memory_usage", Operator: ">", Value: 80},
		},
		Action:       RuleAction{Type: "cleanup"},
		Priority:     50,
		Enabled:      true,
		SuccessCount: 2,
		FailureCount: 1, // only 3 total — not enough
	})

	changes := learning.TuneThresholds()
	if len(changes) != 0 {
		t.Errorf("expected no changes with insufficient data, got %d", len(changes))
	}
}

func TestLearningRelaxThreshold(t *testing.T) {
	engine := NewRuleEngine(nil)
	learning := NewLearningEngine(nil, engine)

	engine.AddRule(&Rule{
		ID:   "relax-test",
		Name: "Relax Test",
		Conditions: []Condition{
			{Metric: "disk_usage", Operator: ">", Value: 85.0},
		},
		Action:       RuleAction{Type: "cleanup"},
		Priority:     50,
		Enabled:      true,
		SuccessCount: 19,
		FailureCount: 1, // 95% success over 20 — should relax
	})

	learning.TuneThresholds()

	rule := engine.GetRule("relax-test")
	if rule.Conditions[0].Value <= 85.0 {
		t.Errorf("expected threshold to relax above 85, got %.1f", rule.Conditions[0].Value)
	}
}

func TestDefaultRulesLoaded(t *testing.T) {
	rules := DefaultRules()
	if len(rules) < 7 {
		t.Errorf("expected at least 7 default rules, got %d", len(rules))
	}

	// Check all have IDs and are enabled
	for _, r := range rules {
		if r.ID == "" {
			t.Error("rule has empty ID")
		}
		if !r.Enabled {
			t.Errorf("default rule %s should be enabled", r.ID)
		}
		if len(r.Conditions) == 0 {
			t.Errorf("rule %s has no conditions", r.ID)
		}
	}
}
