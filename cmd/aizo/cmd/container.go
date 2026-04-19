package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer5"
)

var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage containers",
}

var containerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return fmt.Errorf("WSL2 runtime not available: %w", err)
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

var containerCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image, _ := cmd.Flags().GetString("image")

		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}

		container, err := runtime.CreateContainer(context.Background(), args[0], image, []string{"/bin/sh"}, nil)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Container created: %s (%s)\n", container.Name, container.ID)
		fmt.Printf("  RootFS: %s\n", container.RootFS)
		return nil
	},
}

var containerStartCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}

		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.ID == args[0] || c.Name == args[0] {
				err := runtime.StartContainer(context.Background(), c, []string{"/bin/sh"}, nil)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Container %s started (PID: %d)\n", c.Name, c.PID)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

var containerStopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "Stop a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}

		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.ID == args[0] || c.Name == args[0] {
				err := runtime.StopContainer(context.Background(), c, 10)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Container %s stopped\n", c.Name)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

var containerExecCmd = &cobra.Command{
	Use:   "exec [id] -- [command...]",
	Short: "Execute a command in a container",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}

		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.ID == args[0] || c.Name == args[0] {
				out, err := runtime.ExecInContainer(context.Background(), c, args[1:])
				if err != nil {
					return err
				}
				fmt.Print(out)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

var containerRemoveCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
		if err != nil {
			return err
		}

		containers, _ := runtime.ListContainers(context.Background())
		for _, c := range containers {
			if c.ID == args[0] || c.Name == args[0] {
				err := runtime.RemoveContainer(context.Background(), c)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Container %s removed\n", c.Name)
				return nil
			}
		}
		return fmt.Errorf("container not found: %s", args[0])
	},
}

func init() {
	containerCreateCmd.Flags().String("image", "", "image name")

	containerCmd.AddCommand(containerListCmd)
	containerCmd.AddCommand(containerCreateCmd)
	containerCmd.AddCommand(containerStartCmd)
	containerCmd.AddCommand(containerStopCmd)
	containerCmd.AddCommand(containerExecCmd)
	containerCmd.AddCommand(containerRemoveCmd)
}
