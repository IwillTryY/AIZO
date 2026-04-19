package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage tenants",
}

var tenantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tenants",
	RunE: func(cmd *cobra.Command, args []string) error {
		tenants, err := tenantMgr.List()
		if err != nil {
			return err
		}
		current := tenantMgr.Current()
		for _, t := range tenants {
			marker := " "
			if t.ID == current {
				marker = "*"
			}
			fmt.Printf("  %s [%s] %s\n", marker, t.ID, t.Name)
		}
		return nil
	},
}

var tenantCreateCmd = &cobra.Command{
	Use:   "create [id] [name]",
	Short: "Create a new tenant",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := tenantMgr.Create(args[0], args[1], nil)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tenant created: %s (%s)\n", t.ID, t.Name)
		return nil
	},
}

var tenantSwitchCmd = &cobra.Command{
	Use:   "switch [id]",
	Short: "Switch active tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := tenantMgr.Switch(args[0]); err != nil {
			return err
		}
		fmt.Printf("✓ Switched to tenant: %s\n", args[0])
		return nil
	},
}

func init() {
	tenantCmd.AddCommand(tenantListCmd)
	tenantCmd.AddCommand(tenantCreateCmd)
	tenantCmd.AddCommand(tenantSwitchCmd)
}
