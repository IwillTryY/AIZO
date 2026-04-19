package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/policy"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policies",
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		policies := policyEng.ListPolicies()
		if len(policies) == 0 {
			fmt.Println("No policies loaded")
			return nil
		}
		for _, p := range policies {
			enabled := "✓"
			if !p.Enabled {
				enabled = "✗"
			}
			fmt.Printf("  %s [%s] %s (priority: %d, rules: %d)\n",
				enabled, p.ID, p.Name, p.Priority, len(p.Rules))
		}
		return nil
	},
}

var policyEvalCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate a policy",
	RunE: func(cmd *cobra.Command, args []string) error {
		actor, _ := cmd.Flags().GetString("actor")
		action, _ := cmd.Flags().GetString("action")
		resource, _ := cmd.Flags().GetString("resource")

		result := policyEng.Evaluate(policy.EvalRequest{
			Actor:    actor,
			Action:   action,
			Resource: resource,
			Context:  map[string]string{"tenant_id": tenantMgr.Current()},
		})

		if result.Allowed {
			fmt.Printf("✓ ALLOWED: %s\n", result.Reason)
		} else {
			fmt.Printf("✗ DENIED: %s\n", result.Reason)
		}
		if result.PolicyID != "" {
			fmt.Printf("  Policy: %s (rule #%d)\n", result.PolicyID, result.RuleIdx)
		}
		return nil
	},
}

func init() {
	policyEvalCmd.Flags().String("actor", "cli", "actor performing the action")
	policyEvalCmd.Flags().String("action", "", "action to evaluate")
	policyEvalCmd.Flags().String("resource", "", "target resource")

	policyCmd.AddCommand(policyListCmd)
	policyCmd.AddCommand(policyEvalCmd)
}
