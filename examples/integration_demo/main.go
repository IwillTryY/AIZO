package main

import (
	"context"
	"fmt"
	"time"

	"github.com/realityos/aizo/integration"
	"github.com/realityos/aizo/layer1"
	"github.com/realityos/aizo/layer2"
)

func main() {
	fmt.Println("=== RealityOS Integration Demo ===")
	fmt.Println("Layer 1 + Layer 2 Integration\n")

	ctx := context.Background()

	// Initialize Layer 1
	fmt.Println("1. Initializing Layer 1 (Adapter Layer)")
	fmt.Println("----------------------------------------")
	layer1Manager := layer1.NewManager()

	// Create HTTP adapter
	httpConfig := &layer1.AdapterConfig{
		ID:            "http-api-1",
		Type:          layer1.AdapterTypeHTTP,
		Target:        "https://api.github.com",
		Timeout:       10 * time.Second,
		RetryAttempts: 3,
	}

	_, err := layer1Manager.CreateAndConnectAdapter(httpConfig)
	if err != nil {
		fmt.Printf("Error creating HTTP adapter: %v\n", err)
		return
	}
	fmt.Printf("✓ Created HTTP adapter: %s\n", httpConfig.ID)

	// Initialize Layer 2
	fmt.Println("\n2. Initializing Layer 2 (Discovery & Registration)")
	fmt.Println("---------------------------------------------------")
	layer2Config := &layer2.ManagerConfig{
		DiscoveryConfig: &layer2.DiscoveryConfig{
			Methods:      []layer2.DiscoveryMethod{layer2.DiscoveryManual},
			ScanInterval: 5 * time.Minute,
		},
		EnableAutoDiscovery: false,
		StaleEntityTimeout:  24 * time.Hour,
	}

	layer2Manager := layer2.NewManager(layer2Config)
	fmt.Println("✓ Layer 2 manager initialized")

	// Create integration bridge
	fmt.Println("\n3. Creating Integration Bridge")
	fmt.Println("-------------------------------")
	bridge := integration.NewBridge(layer1Manager, layer2Manager)
	fmt.Println("✓ Bridge created")

	// Register entity with adapter
	fmt.Println("\n4. Registering Entity with Adapter")
	fmt.Println("-----------------------------------")
	entityReq := &layer2.RegistrationRequest{
		ID:       "github-api",
		Type:     layer2.EntityTypeAPI,
		Name:     "GitHub API",
		Endpoint: "https://api.github.com",
		Metadata: map[string]interface{}{
			"provider": "github",
			"version":  "v3",
		},
		Adapters:     []string{httpConfig.ID},
		AutoDetect:   true,
		MapRelations: false,
	}

	resp, err := layer2Manager.RegisterEntity(ctx, entityReq)
	if err != nil {
		fmt.Printf("Error registering entity: %v\n", err)
		return
	}
	fmt.Printf("✓ Registered entity: %s\n", resp.Entity.Name)
	fmt.Printf("  ID: %s\n", resp.Entity.ID)
	fmt.Printf("  Type: %s\n", resp.Entity.Type)
	fmt.Printf("  Adapters: %v\n", resp.Entity.Adapters)
	fmt.Printf("  Capabilities: %v\n", resp.Entity.Capabilities)

	// Sync adapter health to entity
	fmt.Println("\n5. Syncing Adapter Health to Entity")
	fmt.Println("------------------------------------")
	err = bridge.SyncAdapterHealth(ctx, httpConfig.ID)
	if err != nil {
		fmt.Printf("Error syncing health: %v\n", err)
	} else {
		fmt.Println("✓ Health synced")

		entity, _ := layer2Manager.GetEntity("github-api")
		fmt.Printf("  Entity state: %s\n", entity.State)
		fmt.Printf("  Health score: %.1f\n", entity.HealthScore)
	}

	// Get entity state via adapter
	fmt.Println("\n6. Reading Entity State via Adapter")
	fmt.Println("------------------------------------")
	state, err := bridge.GetEntityState(ctx, "github-api")
	if err != nil {
		fmt.Printf("Error reading state: %v\n", err)
	} else {
		fmt.Printf("✓ State retrieved\n")
		fmt.Printf("  Timestamp: %s\n", state.Timestamp.Format(time.RFC3339))
		if len(state.Data) > 0 {
			fmt.Println("  Data:")
			for k, v := range state.Data {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
	}

	// Create another entity with new adapter
	fmt.Println("\n7. Creating Entity with New SSH Adapter")
	fmt.Println("----------------------------------------")

	// Register entity first
	serverReq := &layer2.RegistrationRequest{
		ID:       "prod-server-1",
		Type:     layer2.EntityTypeServer,
		Name:     "Production Server",
		Endpoint: "10.0.1.100:22",
		Metadata: map[string]interface{}{
			"environment": "production",
			"role":        "web-server",
		},
		AutoDetect:   true,
		MapRelations: false,
	}

	serverResp, err := layer2Manager.RegisterEntity(ctx, serverReq)
	if err != nil {
		fmt.Printf("Error registering server: %v\n", err)
	} else {
		fmt.Printf("✓ Registered entity: %s\n", serverResp.Entity.Name)
	}

	// Create adapter for the entity
	sshConfig := &layer1.AdapterConfig{
		ID:     "ssh-server-1",
		Type:   layer1.AdapterTypeSSH,
		Target: "10.0.1.100:22",
		Credentials: map[string]string{
			"username": "admin",
			"password": "password",
		},
		Timeout:       15 * time.Second,
		RetryAttempts: 2,
	}

	err = bridge.CreateAdapterForEntity(ctx, "prod-server-1", sshConfig)
	if err != nil {
		fmt.Printf("Error creating adapter: %v\n", err)
	} else {
		fmt.Println("✓ SSH adapter created and linked to entity")

		entity, _ := layer2Manager.GetEntity("prod-server-1")
		fmt.Printf("  Entity adapters: %v\n", entity.Adapters)
	}

	// Show integrated statistics
	fmt.Println("\n8. Integrated System Statistics")
	fmt.Println("--------------------------------")

	layer1Stats := layer1Manager.GetStats()
	fmt.Printf("Layer 1:\n")
	fmt.Printf("  Total adapters: %d\n", layer1Stats.TotalAdapters)
	fmt.Printf("  By type:\n")
	for adapterType, count := range layer1Stats.ByType {
		fmt.Printf("    %s: %d\n", adapterType, count)
	}
	fmt.Printf("  By status:\n")
	for status, count := range layer1Stats.ByStatus {
		fmt.Printf("    %s: %d\n", status, count)
	}

	layer2Stats := layer2Manager.GetStats()
	fmt.Printf("\nLayer 2:\n")
	fmt.Printf("  Total entities: %d\n", layer2Stats.TotalEntities)
	fmt.Printf("  Healthy: %d\n", layer2Stats.HealthyEntities)
	fmt.Printf("  By type:\n")
	for entityType, count := range layer2Stats.ByType {
		fmt.Printf("    %s: %d\n", entityType, count)
	}

	// Demonstrate command execution via bridge
	fmt.Println("\n9. Execute Command via Bridge")
	fmt.Println("------------------------------")

	cmd := &layer1.CommandRequest{
		ID:      "test-cmd-1",
		Command: "status",
		Args: map[string]interface{}{
			"verbose": true,
		},
	}

	cmdResp, err := bridge.ExecuteCommand(ctx, "github-api", cmd)
	if err != nil {
		fmt.Printf("Command execution: %v\n", err)
	} else {
		fmt.Printf("✓ Command executed: %s\n", cmdResp.RequestID)
		fmt.Printf("  Success: %v\n", cmdResp.Success)
		fmt.Printf("  Duration: %v\n", cmdResp.Duration)
	}

	fmt.Println("\n=== Integration Demo Complete ===")
	fmt.Println("\nThe bridge successfully connects:")
	fmt.Println("  • Layer 1 adapters with Layer 2 entities")
	fmt.Println("  • Adapter health with entity state")
	fmt.Println("  • Command execution through entity abstraction")
	fmt.Println("  • State reading via entity interface")
}
