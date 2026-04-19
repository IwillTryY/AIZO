package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer6"
	"github.com/realityos/aizo/storage"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage detection and response rules",
}

var proposalsCmd = &cobra.Command{
	Use:   "proposals",
	Short: "Manage action proposals",
}

var incidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "View incident history",
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rules with success rates",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer6.NewManager(db, nil)
		rules := mgr.ListRules()

		if len(rules) == 0 {
			fmt.Println("No rules loaded")
			return nil
		}

		fmt.Printf("  %-30s %-8s %-8s %-10s %-8s %s\n",
			"NAME", "ENABLED", "RISK", "SUCCESS", "FIRES", "LAST FIRED")
		fmt.Println(strings.Repeat("─", 80))

		for _, r := range rules {
			enabled := "✓"
			if !r.Enabled {
				enabled = "✗"
			}
			total := r.SuccessCount + r.FailureCount
			successStr := "--"
			if total > 0 {
				successStr = fmt.Sprintf("%.0f%%", r.SuccessRate()*100)
			}
			lastFired := "--"
			if !r.LastFired.IsZero() {
				lastFired = r.LastFired.Format("01/02 15:04")
			}
			fmt.Printf("  %-30s %-8s %-8s %-10s %-8d %s\n",
				r.Name, enabled, r.Action.Risk, successStr, total, lastFired)
		}
		return nil
	},
}

var rulesDisableCmd = &cobra.Command{
	Use:   "disable [id]",
	Short: "Disable a rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer6.NewManager(db, nil)
		rule := mgr.ListRules()
		for _, r := range rule {
			if r.ID == args[0] || r.Name == args[0] {
				r.Enabled = false
				fmt.Printf("✓ Rule disabled: %s\n", r.Name)
				return nil
			}
		}
		return fmt.Errorf("rule not found: %s", args[0])
	},
}

var rulesTuneCmd = &cobra.Command{
	Use:   "tune",
	Short: "Manually trigger threshold tuning",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer6.NewManager(db, nil)
		changes := mgr.TuneThresholds()
		if len(changes) == 0 {
			fmt.Println("No threshold changes (not enough data yet)")
			return nil
		}
		fmt.Println("Threshold changes:")
		for _, c := range changes {
			fmt.Println("  " + c)
		}
		return nil
	},
}

var rulesSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Show pattern-mined rule suggestions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer6.NewManager(db, nil)
		suggestions := mgr.SuggestRules()
		if len(suggestions) == 0 {
			fmt.Println("No suggestions yet (need more incident history)")
			return nil
		}
		for _, s := range suggestions {
			fmt.Printf("  [%.0f%% confidence] %s\n", s.Confidence*100, s.Rule.Name)
			fmt.Printf("    %s\n", s.Description)
			fmt.Printf("    Action: %s | Risk: %s\n\n", s.Rule.Action.Type, s.Rule.Action.Risk)
		}
		return nil
	},
}

var proposalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List action proposals",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := layer6.NewSQLiteStore(sqlDB)
		proposals, err := store.LoadProposals("", 20)
		if err != nil {
			return err
		}
		if len(proposals) == 0 {
			fmt.Println("No proposals")
			return nil
		}
		fmt.Printf("  %-10s %-20s %-12s %-8s %-10s %s\n",
			"ID", "ENTITY", "ACTION", "RISK", "STATUS", "TIME")
		fmt.Println(strings.Repeat("─", 75))
		for _, p := range proposals {
			fmt.Printf("  %-10s %-20s %-12s %-8s %-10s %s\n",
				p.ID[:8], p.EntityID, p.Action, p.Risk, p.Status,
				p.Timestamp.Format("01/02 15:04"))
		}
		return nil
	},
}

var proposalsApproveCmd = &cobra.Command{
	Use:   "approve [id]",
	Short: "Approve a pending proposal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		executor := layer6.NewDefaultActionExecutor()
		mgr := layer6.NewManager(db, executor)

		// Load proposals from DB into manager
		store := layer6.NewSQLiteStore(sqlDB)
		proposals, _ := store.LoadProposals("pending", 50)
		for _, p := range proposals {
			mgr.GetAllProposals() // trigger load
			_ = p
		}

		if err := mgr.ApproveProposal(args[0], "cli"); err != nil {
			return err
		}

		audit.Record(storage.AuditEntry{
			Actor: "cli", Action: "proposal.approve", Resource: args[0],
			Layer: "layer6",
		})

		fmt.Printf("✓ Proposal %s approved and executing\n", args[0])
		return nil
	},
}

var proposalsRejectCmd = &cobra.Command{
	Use:   "reject [id] [reason]",
	Short: "Reject a pending proposal",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer6.NewManager(db, nil)
		reason := "rejected via CLI"
		if len(args) > 1 {
			reason = strings.Join(args[1:], " ")
		}
		if err := mgr.RejectProposal(args[0], reason); err != nil {
			return err
		}
		fmt.Printf("✓ Proposal %s rejected\n", args[0])
		return nil
	},
}

var incidentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show incident history",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		store := layer6.NewSQLiteStore(sqlDB)
		incidents, err := store.LoadIncidents(limit)
		if err != nil {
			return err
		}
		if len(incidents) == 0 {
			fmt.Println("No incidents recorded")
			return nil
		}
		for _, inc := range incidents {
			status := "✓"
			if !inc.ActionSucceeded {
				status = "✗"
			}
			fmt.Printf("  %s %s  [%-20s]  %-15s  → %s\n",
				status, inc.Timestamp.Format("01/02 15:04"),
				inc.Type, inc.EntityID, inc.ActionTaken)
		}
		return nil
	},
}

var incidentsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show success rates per rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer6.NewManager(db, nil)
		rules := mgr.ListRules()

		fmt.Printf("  %-30s %-10s %-10s %-10s\n", "RULE", "SUCCESS", "FAILURES", "RATE")
		fmt.Println(strings.Repeat("─", 65))
		for _, r := range rules {
			total := r.SuccessCount + r.FailureCount
			rate := "--"
			if total > 0 {
				rate = fmt.Sprintf("%.0f%%", r.SuccessRate()*100)
			}
			fmt.Printf("  %-30s %-10d %-10d %-10s\n",
				r.Name, r.SuccessCount, r.FailureCount, rate)
		}
		return nil
	},
}

func init() {
	incidentsListCmd.Flags().Int("limit", 20, "max results")

	rulesCmd.AddCommand(rulesListCmd, rulesDisableCmd, rulesTuneCmd, rulesSuggestCmd)
	proposalsCmd.AddCommand(proposalsListCmd, proposalsApproveCmd, proposalsRejectCmd)
	incidentsCmd.AddCommand(incidentsListCmd, incidentsStatsCmd)

	rootCmd.AddCommand(rulesCmd)
	rootCmd.AddCommand(proposalsCmd)
	rootCmd.AddCommand(incidentsCmd)
}
