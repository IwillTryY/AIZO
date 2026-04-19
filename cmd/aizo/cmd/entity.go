package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer2"
)

var entityCmd = &cobra.Command{
	Use:   "entity",
	Short: "Manage entities",
}

var entityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all entities",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer2.NewManager(nil)
		entities := mgr.ListEntities()
		if len(entities) == 0 {
			fmt.Println("No entities registered")
			return nil
		}
		for _, e := range entities {
			fmt.Printf("  [%s] %s (%s) - %s\n", e.ID, e.Name, e.Type, e.State)
		}
		return nil
	},
}

var entityRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new entity",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		entityType, _ := cmd.Flags().GetString("type")
		endpoint, _ := cmd.Flags().GetString("endpoint")

		if id == "" || name == "" {
			return fmt.Errorf("--id and --name are required")
		}

		mgr := layer2.NewManager(nil)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := &layer2.RegistrationRequest{
			ID:       id,
			Type:     layer2.EntityType(entityType),
			Name:     name,
			Endpoint: endpoint,
		}

		resp, err := mgr.RegisterEntity(ctx, req)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Entity registered: %s (%s)\n", resp.Entity.ID, resp.Entity.Name)
		return nil
	},
}

var entityInspectCmd = &cobra.Command{
	Use:   "inspect [id]",
	Short: "Inspect an entity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := layer2.NewManager(nil)
		entity, err := mgr.GetEntity(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Entity: %s\n", entity.ID)
		fmt.Printf("  Name:     %s\n", entity.Name)
		fmt.Printf("  Type:     %s\n", entity.Type)
		fmt.Printf("  Endpoint: %s\n", entity.Endpoint)
		fmt.Printf("  State:    %s\n", entity.State)
		if len(entity.Capabilities) > 0 {
			fmt.Printf("  Capabilities: %v\n", entity.Capabilities)
		}
		return nil
	},
}

func init() {
	entityRegisterCmd.Flags().String("id", "", "entity ID")
	entityRegisterCmd.Flags().String("name", "", "entity name")
	entityRegisterCmd.Flags().String("type", "server", "entity type")
	entityRegisterCmd.Flags().String("endpoint", "", "entity endpoint")

	entityCmd.AddCommand(entityListCmd)
	entityCmd.AddCommand(entityRegisterCmd)
	entityCmd.AddCommand(entityInspectCmd)
}
