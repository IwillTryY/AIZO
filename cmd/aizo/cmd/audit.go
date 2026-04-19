package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/storage"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit trail",
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent audit entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		last, _ := cmd.Flags().GetString("last")
		layer, _ := cmd.Flags().GetString("layer")
		actor, _ := cmd.Flags().GetString("actor")
		limit, _ := cmd.Flags().GetInt("limit")

		duration, err := time.ParseDuration(last)
		if err != nil {
			duration = 24 * time.Hour
		}

		entries, err := audit.Query(storage.AuditFilter{
			TenantID: tenantMgr.Current(),
			Actor:    actor,
			Layer:    layer,
			Since:    time.Now().Add(-duration),
			Limit:    limit,
		})
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("No audit entries")
			return nil
		}

		for _, e := range entries {
			fmt.Printf("  %s  %-8s  %-15s  %-20s  %s\n",
				e.Timestamp.Format("15:04:05"), e.Actor, e.Action, e.Resource, e.Detail)
		}
		fmt.Printf("\n%d entries\n", len(entries))
		return nil
	},
}

func init() {
	auditListCmd.Flags().String("last", "24h", "time range")
	auditListCmd.Flags().String("layer", "", "filter by layer")
	auditListCmd.Flags().String("actor", "", "filter by actor")
	auditListCmd.Flags().Int("limit", 50, "max results")

	auditCmd.AddCommand(auditListCmd)
}
