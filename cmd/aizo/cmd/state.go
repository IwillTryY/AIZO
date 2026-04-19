package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer4"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage entity state",
}

var stateGetCmd = &cobra.Command{
	Use:   "get [entity-id]",
	Short: "Get entity state",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer4.NewManager(nil)
		ctx := context.Background()

		state, err := mgr.GetEntityState(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Entity: %s\n", state.EntityID)
		fmt.Printf("  Type:   %s\n", state.Type)
		fmt.Printf("  Status: %s\n", state.Status)
		if len(state.DesiredState) > 0 {
			fmt.Println("  Desired State:")
			for k, v := range state.DesiredState {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
		if len(state.ActualState) > 0 {
			fmt.Println("  Actual State:")
			for k, v := range state.ActualState {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
		return nil
	},
}

var stateDriftCmd = &cobra.Command{
	Use:   "drift [entity-id]",
	Short: "Detect state drift",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer4.NewManager(nil)
		ctx := context.Background()
		cfg := &layer4.ManagerConfig{
			EnableAutoReconciliation: false,
			ReconciliationInterval:   5 * time.Minute,
		}
		mgr.Start(ctx, cfg)

		drift, err := mgr.DetectDrift(ctx, args[0])
		if err != nil {
			return err
		}

		if !drift.HasDrift {
			fmt.Printf("✓ No drift detected for %s\n", args[0])
			return nil
		}

		fmt.Printf("⚠ Drift detected for %s (score: %.1f)\n", args[0], drift.DriftScore)
		for _, d := range drift.Differences {
			fmt.Printf("  %s: expected=%v actual=%v\n", d.Field, d.DesiredValue, d.ActualValue)
		}
		return nil
	},
}

func init() {
	stateCmd.AddCommand(stateGetCmd)
	stateCmd.AddCommand(stateDriftCmd)
}
