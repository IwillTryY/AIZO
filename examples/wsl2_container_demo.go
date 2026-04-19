package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/realityos/aizo/layer5"
)

func main() {
	fmt.Println("=== RealityOS WSL2 Container Demo ===")
	fmt.Println("Real Linux namespace isolation via WSL2\n")

	ctx := context.Background()

	// Create WSL2 runtime - use home directory to avoid permission issues
	fmt.Println("🔧 Initializing WSL2 Container Runtime...")
	runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
	if err != nil {
		fmt.Printf("❌ Failed to initialize WSL2 runtime: %v\n", err)
		fmt.Println("\nMake sure WSL2 is installed and Ubuntu distro is available:")
		fmt.Println("  wsl --install -d Ubuntu")
		fmt.Println("  wsl --set-default Ubuntu")
		return
	}
	fmt.Println("✓ WSL2 runtime ready")
	fmt.Println("  Distro: Ubuntu")
	fmt.Println("  Data root: ~/realityos")
	fmt.Println("  Isolation: PID + Mount + UTS + IPC namespaces\n")

	// Create a container
	fmt.Println("📦 Creating isolated container...")
	container, err := runtime.CreateContainer(
		ctx,
		"demo-container",
		"ubuntu:latest",
		[]string{"/bin/sh"},
		[]string{"APP_ENV=production", "APP_PORT=8080"},
	)
	if err != nil {
		fmt.Printf("❌ Failed to create container: %v\n", err)
		return
	}
	fmt.Printf("✓ Container created\n")
	fmt.Printf("  ID: %s\n", container.ID)
	fmt.Printf("  Name: %s\n", container.Name)
	fmt.Printf("  RootFS: %s\n", container.RootFS)
	fmt.Printf("  Status: %s\n\n", container.Status)

	// Start the container
	fmt.Println("▶️  Starting container with namespace isolation...")
	err = runtime.StartContainer(
		ctx,
		container,
		[]string{"/bin/sh", "-c", "echo 'Hello from isolated container!'; sleep 5"},
		[]string{"APP_ENV=production"},
	)
	if err != nil {
		fmt.Printf("❌ Failed to start container: %v\n", err)
		return
	}
	fmt.Printf("✓ Container started\n")
	fmt.Printf("  PID: %d\n", container.PID)
	fmt.Printf("  Namespaces: PID, Mount, UTS, IPC\n")
	fmt.Printf("  RootFS: %s\n\n", container.RootFS)

	time.Sleep(1 * time.Second)

	// Execute command inside container
	fmt.Println("⚡ Executing command inside container...")
	output, err := runtime.ExecInContainer(ctx, container, []string{"sh", "-c", "echo 'exec works!'; ls /"})
	if err != nil {
		fmt.Printf("❌ Exec failed: %v\n", err)
	} else {
		fmt.Printf("✓ Output:\n%s\n", output)
	}

	// List containers
	fmt.Println("📋 Listing containers...")
	containers, err := runtime.ListContainers(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to list: %v\n", err)
	} else {
		for _, c := range containers {
			fmt.Printf("  [%s] %s - %s (PID: %d)\n", c.ID, c.Name, c.Status, c.PID)
		}
	}
	fmt.Println()

	// Interactive shell
	fmt.Println("💬 Interactive shell inside container (type 'exit' to quit):")
	fmt.Println(strings.Repeat("-", 50))
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("container# ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "exit" || input == "quit" {
			break
		}
		if input == "" {
			continue
		}
		out, err := runtime.ExecInContainer(ctx, container, []string{"sh", "-c", input})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Print(out)
		}
	}

	// Stop container
	fmt.Println("\n⏹️  Stopping container...")
	err = runtime.StopContainer(ctx, container, 5)
	if err != nil {
		fmt.Printf("❌ Stop failed: %v\n", err)
	} else {
		fmt.Printf("✓ Container stopped (exit code: %d)\n", container.ExitCode)
	}

	// Remove container
	fmt.Println("🧹 Removing container...")
	err = runtime.RemoveContainer(ctx, container)
	if err != nil {
		fmt.Printf("❌ Remove failed: %v\n", err)
	} else {
		fmt.Println("✓ Container removed (filesystem cleaned up)")
	}

	fmt.Println("\n✅ Demo complete!")
	fmt.Println("Real Linux namespace isolation via WSL2 is working.")
}
