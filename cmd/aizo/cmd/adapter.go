package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer1"
	"github.com/realityos/aizo/storage"
)

var adapterCmd = &cobra.Command{
	Use:   "adapter",
	Short: "Manage adapters (HTTP, SSH, gRPC, MQTT)",
}

var adapterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all adapters",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer1.NewManager()
		adapters := mgr.ListAdapters()
		if len(adapters) == 0 {
			fmt.Println("No adapters registered")
			return nil
		}
		for _, a := range adapters {
			cfg := a.GetConfig()
			h := a.GetHealth()
			fmt.Printf("  [%s] %s (%s) - %s\n", cfg.ID, cfg.Target, cfg.Type, h.Status)
		}
		return nil
	},
}

var adapterAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add and connect an adapter",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		adapterType, _ := cmd.Flags().GetString("type")
		target, _ := cmd.Flags().GetString("target")

		if id == "" || adapterType == "" || target == "" {
			return fmt.Errorf("--id, --type, and --target are required")
		}

		mgr := layer1.NewManager()
		config := &layer1.AdapterConfig{
			ID:     id,
			Type:   layer1.AdapterType(adapterType),
			Target: target,
			Timeout: 10 * time.Second,
		}

		adapter, err := mgr.CreateAdapter(config)
		if err != nil {
			return fmt.Errorf("failed to create adapter: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := adapter.Connect(ctx); err != nil {
			fmt.Printf("⚠ Adapter created but connection failed: %v\n", err)
		} else {
			fmt.Printf("✓ Adapter %s connected to %s\n", id, target)
		}

		audit.Record(storage.AuditEntry{
			Actor: "cli", Action: "adapter.add", Resource: id,
			Detail: fmt.Sprintf("type=%s target=%s", adapterType, target), Layer: "layer1",
		})

		return nil
	},
}

var adapterHealthCmd = &cobra.Command{
	Use:   "health [id]",
	Short: "Check adapter health",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer1.NewManager()
		adapter, err := mgr.GetAdapter(args[0])
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		health, err := adapter.HealthCheck(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("Adapter: %s\n", args[0])
		fmt.Printf("  Status:  %s\n", health.Status)
		fmt.Printf("  Latency: %v\n", health.Latency)
		fmt.Printf("  Message: %s\n", health.Message)
		fmt.Printf("  Errors:  %d\n", health.ErrorCount)
		fmt.Printf("  Success: %.1f%%\n", health.SuccessRate*100)
		return nil
	},
}

func init() {
	adapterAddCmd.Flags().String("id", "", "adapter ID")
	adapterAddCmd.Flags().String("type", "", "adapter type (http, ssh, grpc, mqtt)")
	adapterAddCmd.Flags().String("target", "", "target endpoint")

	adapterCmd.AddCommand(adapterListCmd)
	adapterCmd.AddCommand(adapterAddCmd)
	adapterCmd.AddCommand(adapterHealthCmd)
}
