package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer5"
)

// --- aizo create ---

var createCmd = &cobra.Command{
	Use:   "create <type> [count] [names...]",
	Short: "Create resources",
	Long:  "Create containers, entities, or adapters.\n\nExamples:\n  aizo create container 3 web api worker\n  aizo create container myapp\n  aizo create entity --id web-1 --name WebServer --type api",
}

var createContainerCmd = &cobra.Command{
	Use:   "container [count] [names...]",
	Short: "Create one or more containers",
	Long:  "Create containers by name. Optionally specify a count followed by names.\n\nExamples:\n  aizo create container web\n  aizo create container 3 web api worker",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image, _ := cmd.Flags().GetString("image")

		// Parse args: if first arg is a number, it's the count
		var names []string
		if count, err := strconv.Atoi(args[0]); err == nil {
			// First arg is a count
			names = args[1:]
			if len(names) != count {
				return fmt.Errorf("expected %d names, got %d", count, len(names))
			}
		} else {
			// First arg is a name, all args are names
			names = args
		}

		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return fmt.Errorf("WSL2 runtime: %w", err)
		}

		for _, name := range names {
			c, err := runtime.CreateContainer(context.Background(), name, image, []string{"/bin/sh"}, nil)
			if err != nil {
				fmt.Printf("  ✗ %s: %v\n", name, err)
				continue
			}
			fmt.Printf("  ✓ %s (%s)\n", c.Name, c.ID)
		}

		return nil
	},
}

// --- aizo start ---

var startCmd = &cobra.Command{
	Use:   "start <type> <name>",
	Short: "Start a resource",
}

var startContainerCmd = &cobra.Command{
	Use:   "container <name>",
	Short: "Start a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}
		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.Name == args[0] || c.ID == args[0] {
				if err := runtime.StartContainer(context.Background(), c, []string{"/bin/sh"}, nil); err != nil {
					return err
				}
				fmt.Printf("  ✓ Started %s (PID: %d)\n", c.Name, c.PID)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

// --- aizo stop ---

var stopCmd = &cobra.Command{
	Use:   "stop <type> <name>",
	Short: "Stop a resource",
}

var stopContainerCmd = &cobra.Command{
	Use:   "container <name>",
	Short: "Stop a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}
		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.Name == args[0] || c.ID == args[0] {
				if err := runtime.StopContainer(context.Background(), c, 10); err != nil {
					return err
				}
				fmt.Printf("  ✓ Stopped %s\n", c.Name)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

// --- aizo remove ---

var removeCmd = &cobra.Command{
	Use:   "remove <type> <name>",
	Short: "Remove a resource",
}

var removeContainerCmd = &cobra.Command{
	Use:   "container <name>",
	Short: "Remove a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}
		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.Name == args[0] || c.ID == args[0] {
				if err := runtime.RemoveContainer(context.Background(), c); err != nil {
					return err
				}
				fmt.Printf("  ✓ Removed %s\n", c.Name)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

// --- aizo exec ---

var execCmd = &cobra.Command{
	Use:   "exec container <name> [command...]",
	Short: "Execute a command in a container",
}

var execContainerCmd = &cobra.Command{
	Use:   "container <name> [command...]",
	Short: "Execute a command in a container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}

		name := args[0]
		shellCmd := []string{"sh", "-c", "echo 'connected'"}
		if len(args) > 1 {
			shellCmd = args[1:]
		}

		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.Name == name || c.ID == name {
				out, err := runtime.ExecInContainer(context.Background(), c, shellCmd)
				if err != nil {
					return err
				}
				fmt.Print(out)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", name)
	},
}

// --- aizo list ---

var listCmd = &cobra.Command{
	Use:   "list <type>",
	Short: "List resources",
}

var listContainersCmd = &cobra.Command{
	Use:     "containers",
	Aliases: []string{"container"},
	Short:   "List all containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return fmt.Errorf("WSL2 runtime: %w", err)
		}
		containers, err := runtime.ListContainers(context.Background())
		if err != nil {
			return err
		}
		if len(containers) == 0 {
			fmt.Println("No containers")
			return nil
		}
		fmt.Printf("  %-14s %-20s %-10s %-8s\n", "ID", "NAME", "STATUS", "PID")
		for _, c := range containers {
			fmt.Printf("  %-14s %-20s %-10s %-8d\n", c.ID, c.Name, c.Status, c.PID)
		}
		return nil
	},
}

var listRulesCmd2 = &cobra.Command{
	Use:     "rules",
	Aliases: []string{"rule"},
	Short:   "List all rules",
	RunE:    rulesListCmd.RunE,
}

var listProposalsCmd = &cobra.Command{
	Use:     "proposals",
	Aliases: []string{"proposal"},
	Short:   "List proposals",
	RunE:    proposalsListCmd.RunE,
}

var listIncidentsCmd = &cobra.Command{
	Use:     "incidents",
	Aliases: []string{"incident"},
	Short:   "List incidents",
	RunE:    incidentsListCmd.RunE,
}

func init() {
	// create
	createContainerCmd.Flags().String("image", "", "image name")
	createCmd.AddCommand(createContainerCmd)

	// start / stop / remove
	startCmd.AddCommand(startContainerCmd)
	stopCmd.AddCommand(stopContainerCmd)
	removeCmd.AddCommand(removeContainerCmd)

	// exec
	execCmd.AddCommand(execContainerCmd)

	// list
	listCmd.AddCommand(listContainersCmd)
	listCmd.AddCommand(listRulesCmd2)
	listCmd.AddCommand(listProposalsCmd)
	listCmd.AddCommand(listIncidentsCmd)
}
