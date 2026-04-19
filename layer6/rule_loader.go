package layer6

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultRules returns the built-in default ruleset
func DefaultRules() []*Rule {
	return []*Rule{
		{
			ID:          "memory-cleanup-80",
			Name:        "Memory Cleanup at 80%",
			Description: "Trigger cleanup when memory usage exceeds 80%",
			Conditions:  []Condition{{Metric: "memory_usage", Operator: ">", Value: 80}},
			Action: RuleAction{
				Type:        "cleanup",
				Risk:        "low",
				Reversible:  true,
				AutoApprove: true,
				Reasoning:   "Memory usage exceeded 80% threshold",
			},
			Priority: 50,
			Enabled:  true,
		},
		{
			ID:          "memory-restart-95",
			Name:        "Restart at 95% Memory",
			Description: "Restart service when memory usage is critical",
			Conditions:  []Condition{{Metric: "memory_usage", Operator: ">", Value: 95}},
			Action: RuleAction{
				Type:        "restart",
				Risk:        "medium",
				Reversible:  true,
				AutoApprove: false,
				Reasoning:   "Memory usage critical (>95%), restart required",
			},
			Priority: 90,
			Enabled:  true,
		},
		{
			ID:          "cpu-investigate-90",
			Name:        "Investigate High CPU",
			Description: "Investigate when CPU usage exceeds 90%",
			Conditions:  []Condition{{Metric: "cpu_usage", Operator: ">", Value: 90}},
			Action: RuleAction{
				Type:        "investigate",
				Risk:        "low",
				Reversible:  true,
				AutoApprove: true,
				Reasoning:   "CPU usage exceeded 90%, investigation needed",
			},
			Priority: 40,
			Enabled:  true,
		},
		{
			ID:          "disk-cleanup-85",
			Name:        "Disk Cleanup at 85%",
			Description: "Clean up disk when usage exceeds 85%",
			Conditions:  []Condition{{Metric: "disk_usage", Operator: ">", Value: 85}},
			Action: RuleAction{
				Type:        "cleanup",
				Risk:        "low",
				Reversible:  true,
				AutoApprove: true,
				Reasoning:   "Disk usage exceeded 85% threshold",
			},
			Priority: 50,
			Enabled:  true,
		},
		{
			ID:          "container-crash-restart",
			Name:        "Restart on Container Crash",
			Description: "Restart container when crash event is detected",
			Conditions:  []Condition{{EventType: "container_crash"}},
			Action: RuleAction{
				Type:        "restart",
				Risk:        "medium",
				Reversible:  true,
				AutoApprove: false,
				Reasoning:   "Container crash detected, restart proposed",
			},
			Priority: 80,
			Enabled:  true,
		},
		{
			ID:          "health-check-fail-restart",
			Name:        "Restart on Health Check Failure",
			Description: "Restart service after repeated health check failures",
			Conditions:  []Condition{{EventType: "health_check_fail"}},
			Action: RuleAction{
				Type:        "restart",
				Risk:        "medium",
				Reversible:  true,
				AutoApprove: false,
				Reasoning:   "Health check failed, service may be unresponsive",
			},
			Priority: 70,
			Enabled:  true,
		},
		{
			ID:          "service-down-restart",
			Name:        "Restart on Service Down",
			Description: "Restart when service down event detected",
			Conditions:  []Condition{{EventType: "service_down"}},
			Action: RuleAction{
				Type:        "restart",
				Risk:        "medium",
				Reversible:  true,
				AutoApprove: false,
				Reasoning:   "Service down event detected",
			},
			Priority: 85,
			Enabled:  true,
		},
		{
			ID:          "failed-containers-alert",
			Name:        "Alert on Failed Containers",
			Description: "Alert when any containers are in failed state",
			Conditions:  []Condition{{Metric: "failed_containers", Operator: ">", Value: 0}},
			Action: RuleAction{
				Type:        "investigate",
				Risk:        "low",
				Reversible:  true,
				AutoApprove: true,
				Reasoning:   "One or more containers in failed state",
			},
			Priority: 60,
			Enabled:  true,
		},
	}
}

// LoadRulesFromFile loads rules from a YAML file
func LoadRulesFromFile(path string) ([]*Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rules []*Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// LoadRulesFromDir loads all YAML rule files from a directory
func LoadRulesFromDir(dir string) ([]*Rule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	all := make([]*Rule, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		rules, err := LoadRulesFromFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		all = append(all, rules...)
	}
	return all, nil
}

// SaveRulesToFile saves rules to a YAML file
func SaveRulesToFile(path string, rules []*Rule) error {
	data, err := yaml.Marshal(rules)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
