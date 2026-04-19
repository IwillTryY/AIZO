package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/realityos/aizo/layer1"
)

func main() {
	fmt.Println("=== RealityOS Layer 1 - Adapter Layer Demo ===\n")

	// Create the adapter manager
	manager := layer1.NewManager()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Shutdown(ctx)
	}()

	// Example 1: HTTP Adapter
	fmt.Println("1. Creating HTTP Adapter...")
	httpConfig := &layer1.AdapterConfig{
		ID:     "api-server-1",
		Type:   layer1.AdapterTypeHTTP,
		Target: "https://api.github.com/",
		Credentials: map[string]string{
			// Add credentials if needed
		},
		Timeout:      10 * time.Second,
		RetryAttempts: 3,
	}

	httpAdapter, err := manager.CreateAndConnectAdapter(httpConfig)
	if err != nil {
		log.Printf("Failed to create HTTP adapter: %v\n", err)
	} else {
		fmt.Printf("✓ HTTP Adapter created and connected: %s\n", httpConfig.ID)

		// Read state
		ctx := context.Background()
		state, err := httpAdapter.ReadState(ctx)
		if err != nil {
			log.Printf("Failed to read state: %v\n", err)
		} else {
			fmt.Printf("✓ State read successfully at %s\n", state.Timestamp.Format(time.RFC3339))
		}

		// Health check
		health, err := httpAdapter.HealthCheck(ctx)
		if err != nil {
			log.Printf("Health check failed: %v\n", err)
		} else {
			fmt.Printf("✓ Health: %s (latency: %v)\n", health.Status, health.Latency)
		}
	}

	fmt.Println()

	// Example 2: SSH Adapter (commented out as it requires actual SSH server)
	fmt.Println("2. SSH Adapter Example (configuration only)...")
	sshConfig := &layer1.AdapterConfig{
		ID:     "server-1",
		Type:   layer1.AdapterTypeSSH,
		Target: "192.168.1.100:22",
		Credentials: map[string]string{
			"username": "admin",
			"password": "secret",
		},
		Timeout:      15 * time.Second,
		RetryAttempts: 3,
	}
	fmt.Printf("✓ SSH Adapter config prepared for: %s\n", sshConfig.Target)
	fmt.Println("  (Connection skipped - requires actual SSH server)")

	fmt.Println()

	// Example 3: Registry Operations
	fmt.Println("3. Registry Operations...")
	adapters := manager.ListAdapters()
	fmt.Printf("✓ Total adapters registered: %d\n", len(adapters))

	for _, adapter := range adapters {
		config := adapter.GetConfig()
		fmt.Printf("  - %s (%s): %s\n", config.ID, adapter.GetType(), config.Target)
		fmt.Printf("    Capabilities: %v\n", adapter.GetCapabilities())
	}

	fmt.Println()

	// Example 4: Health Check All
	fmt.Println("4. Health Check All Adapters...")
	ctx := context.Background()
	healthResults := manager.HealthCheckAll(ctx)
	for id, health := range healthResults {
		fmt.Printf("  - %s: %s (success rate: %.2f%%)\n",
			id, health.Status, health.SuccessRate*100)
	}

	fmt.Println()

	// Example 5: Statistics
	fmt.Println("5. Adapter Layer Statistics...")
	stats := manager.GetStats()
	fmt.Printf("✓ Total Adapters: %d\n", stats.TotalAdapters)
	fmt.Printf("✓ Average Success Rate: %.2f%%\n", stats.AverageSuccessRate*100)
	fmt.Println("  By Type:")
	for adapterType, count := range stats.ByType {
		fmt.Printf("    - %s: %d\n", adapterType, count)
	}
	fmt.Println("  By Health Status:")
	for status, count := range stats.ByStatus {
		fmt.Printf("    - %s: %d\n", status, count)
	}

	fmt.Println()

	// Example 6: Send Command
	fmt.Println("6. Sending Command Example...")
	if httpAdapter != nil {
		cmdReq := &layer1.CommandRequest{
			ID:      "cmd-1",
			Command: "zen",
			Args: map[string]interface{}{
				"method": "GET",
			},
			Timeout: 5 * time.Second,
		}

		resp, err := manager.SendCommand(ctx, "api-server-1", cmdReq)
		if err != nil {
			log.Printf("Command failed: %v\n", err)
		} else {
			fmt.Printf("✓ Command executed: success=%v, duration=%v\n",
				resp.Success, resp.Duration)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
}
