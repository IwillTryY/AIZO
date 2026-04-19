package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/realityos/aizo/layer6"
)

func main() {
	fmt.Println("=== AIZO Layer 6: Rule Engine Demo ===\n")

	ctx := context.Background()
	_ = ctx

	// Initialize manager with default rules (no DB for demo)
	mgr := layer6.NewManager(nil, layer6.NewDefaultActionExecutor())

	// --- Show loaded rules ---
	fmt.Println("📋 Loaded Rules:")
	fmt.Println(strings.Repeat("─", 60))
	for _, r := range mgr.ListRules() {
		autoApprove := "requires approval"
		if r.Action.AutoApprove {
			autoApprove = "auto-approve"
		}
		fmt.Printf("  [%s] %s\n    → %s (%s, %s)\n",
			r.ID, r.Name, r.Action.Type, r.Action.Risk, autoApprove)
	}

	// --- Scenario 1: Memory leak event ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("🔥 Scenario 1: Memory Leak Detected")
	fmt.Println(strings.Repeat("─", 60))

	memoryEvent := &layer6.SystemEvent{
		ID:          "evt-001",
		Timestamp:   time.Now(),
		Type:        layer6.EventHighMemory,
		Severity:    "high",
		EntityID:    "web-server-1",
		EntityType:  "api",
		Description: "Memory usage at 87%",
		Metrics:     map[string]float64{"memory_usage": 87.0},
	}

	proposal, err := mgr.ProcessEvent(memoryEvent)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if proposal != nil {
		fmt.Printf("  ✓ Rule fired: %s\n", proposal.RuleID)
		fmt.Printf("  Action: %s on %s\n", proposal.Action, proposal.EntityID)
		fmt.Printf("  Risk: %s | Auto-approve: %v\n", proposal.Risk, !proposal.RequiresApproval)
		fmt.Printf("  Reasoning: %s\n", proposal.Reasoning)
	}

	// --- Scenario 2: Container crash event ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("💥 Scenario 2: Container Crash")
	fmt.Println(strings.Repeat("─", 60))

	crashEvent := &layer6.SystemEvent{
		ID:          "evt-002",
		Timestamp:   time.Now(),
		Type:        layer6.EventContainerCrash,
		Severity:    "critical",
		EntityID:    "worker-container-3",
		EntityType:  "container",
		Description: "Container exited with code 137 (OOM killed)",
	}

	proposal2, err := mgr.ProcessEvent(crashEvent)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if proposal2 != nil {
		fmt.Printf("  ✓ Rule fired: %s\n", proposal2.RuleID)
		fmt.Printf("  Action: %s on %s\n", proposal2.Action, proposal2.EntityID)
		fmt.Printf("  Risk: %s | Requires approval: %v\n", proposal2.Risk, proposal2.RequiresApproval)
	}

	// --- Scenario 3: System summary evaluation ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("📊 Scenario 3: System Summary Evaluation")
	fmt.Println(strings.Repeat("─", 60))

	summary := &layer6.SystemSummary{
		Timestamp:         time.Now(),
		TotalEntities:     12,
		HealthyEntities:   9,
		DegradedEntities:  2,
		UnhealthyEntities: 1,
		RunningContainers: 8,
		FailedContainers:  1,
		MemoryUsage:       91.0,
		CPUUsage:          45.0,
		DiskUsage:         72.0,
	}

	proposals, err := mgr.ProcessSummary(summary)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  %d rules fired from summary:\n", len(proposals))
		for _, p := range proposals {
			fmt.Printf("  → [%s] %s on %s (risk: %s)\n",
				p.RuleID, p.Action, p.EntityID, p.Risk)
		}
	}

	// --- Pending proposals ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("📥 Pending Proposals (require approval):")
	fmt.Println(strings.Repeat("─", 60))

	pending := mgr.GetPendingProposals()
	for _, p := range pending {
		fmt.Printf("  [%s] %s → %s on %s\n", p.ID[:8], p.RuleID, p.Action, p.EntityID)
	}
	fmt.Printf("  Total pending: %d\n", len(pending))

	// --- Approve one ---
	if len(pending) > 0 {
		fmt.Println("\n✅ Approving first proposal...")
		if err := mgr.ApproveProposal(pending[0].ID, "demo-user"); err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  Approved and executing: %s\n", pending[0].ID[:8])
		}
		time.Sleep(500 * time.Millisecond)
	}

	// --- Stats ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("📈 Stats:")
	stats := mgr.GetStats()
	fmt.Printf("  Total proposals: %d\n", stats.TotalProposals)
	fmt.Printf("  Pending: %d\n", stats.PendingProposals)
	fmt.Printf("  Total rules: %d\n", stats.TotalRules)

	// --- Learning: tune thresholds ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("🧠 Threshold Tuning (needs more data):")
	changes := mgr.TuneThresholds()
	if len(changes) == 0 {
		fmt.Println("  No changes yet — need at least 5 incidents per rule")
	} else {
		for _, c := range changes {
			fmt.Println("  " + c)
		}
	}

	// --- Pattern mining ---
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("🔍 Pattern Mining Suggestions:")
	suggestions := mgr.SuggestRules()
	if len(suggestions) == 0 {
		fmt.Println("  No suggestions yet — need more incident history")
	} else {
		for _, s := range suggestions {
			fmt.Printf("  [%.0f%%] %s: %s\n", s.Confidence*100, s.Rule.Name, s.Description)
		}
	}

	fmt.Println("\n✅ Demo complete!")
	fmt.Println("\nRun 'aizo rules list' to see rules in the CLI")
	fmt.Println("Run 'aizo proposals list' to see pending proposals")
}
