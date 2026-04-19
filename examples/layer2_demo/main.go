package main

import (
	"context"
	"fmt"
	"time"

	"github.com/realityos/aizo/layer2"
)

func main() {
	fmt.Println("=== RealityOS Layer 2 Demo ===")
	fmt.Println("Discovery & Registration Layer\n")

	// Create manager configuration
	config := &layer2.ManagerConfig{
		DiscoveryConfig: &layer2.DiscoveryConfig{
			Methods:           []layer2.DiscoveryMethod{layer2.DiscoveryNetwork, layer2.DiscoveryCloud},
			ScanInterval:      5 * time.Minute,
			NetworkRanges:     []string{"10.0.0.0/24"},
			CloudProviders:    []string{"AWS", "Azure"},
			ContainerRuntimes: []string{"Docker", "Kubernetes"},
		},
		EnableAutoDiscovery: false, // Manual for demo
		StaleEntityTimeout:  24 * time.Hour,
	}

	// Create Layer 2 manager
	manager := layer2.NewManager(config)
	ctx := context.Background()

	fmt.Println("1. Manual Entity Registration")
	fmt.Println("------------------------------")

	// Register a database
	dbReq := &layer2.RegistrationRequest{
		ID:       "postgres-prod-1",
		Type:     layer2.EntityTypeDatabase,
		Name:     "Production PostgreSQL",
		Endpoint: "postgres.prod.internal:5432",
		Metadata: map[string]interface{}{
			"version":     "14.5",
			"environment": "production",
			"host":        "postgres.prod.internal",
		},
		Adapters:     []string{"ssh-adapter-1", "postgres-adapter-1"},
		AutoDetect:   true,
		MapRelations: false,
	}

	dbResp, err := manager.RegisterEntity(ctx, dbReq)
	if err != nil {
		fmt.Printf("Error registering database: %v\n", err)
	} else {
		fmt.Printf("✓ Registered: %s (%s)\n", dbResp.Entity.Name, dbResp.Entity.ID)
		fmt.Printf("  Capabilities: %v\n", dbResp.Entity.Capabilities)
	}

	// Register an API service
	apiReq := &layer2.RegistrationRequest{
		ID:       "user-api-1",
		Type:     layer2.EntityTypeAPI,
		Name:     "User Management API",
		Endpoint: "https://api.example.com/users",
		Metadata: map[string]interface{}{
			"version":       "2.1.0",
			"environment":   "production",
			"database_host": "postgres.prod.internal",
			"depends_on":    []string{"postgres-prod-1"},
		},
		Adapters:     []string{"http-adapter-1"},
		AutoDetect:   true,
		MapRelations: true,
	}

	apiResp, err := manager.RegisterEntity(ctx, apiReq)
	if err != nil {
		fmt.Printf("Error registering API: %v\n", err)
	} else {
		fmt.Printf("✓ Registered: %s (%s)\n", apiResp.Entity.Name, apiResp.Entity.ID)
		fmt.Printf("  Capabilities: %v\n", apiResp.Entity.Capabilities)
		fmt.Printf("  Relationships: %d\n", len(apiResp.Entity.Relationships))
	}

	// Register a background job
	jobReq := &layer2.RegistrationRequest{
		ID:       "email-job-1",
		Type:     layer2.EntityTypeJob,
		Name:     "Email Notification Job",
		Endpoint: "job-runner.internal:8080",
		Metadata: map[string]interface{}{
			"schedule":    "*/5 * * * *",
			"environment": "production",
			"depends_on":  []string{"user-api-1"},
		},
		Adapters:     []string{"http-adapter-2"},
		AutoDetect:   true,
		MapRelations: true,
	}

	jobResp, err := manager.RegisterEntity(ctx, jobReq)
	if err != nil {
		fmt.Printf("Error registering job: %v\n", err)
	} else {
		fmt.Printf("✓ Registered: %s (%s)\n", jobResp.Entity.Name, jobResp.Entity.ID)
		fmt.Printf("  Capabilities: %v\n", jobResp.Entity.Capabilities)
	}

	fmt.Println("\n2. Entity Catalog Operations")
	fmt.Println("-----------------------------")

	// List all entities
	allEntities := manager.ListEntities()
	fmt.Printf("Total entities: %d\n", len(allEntities))

	// List by type
	apis := manager.ListEntitiesByType(layer2.EntityTypeAPI)
	fmt.Printf("APIs: %d\n", len(apis))

	databases := manager.ListEntitiesByType(layer2.EntityTypeDatabase)
	fmt.Printf("Databases: %d\n", len(databases))

	// Search entities
	criteria := layer2.SearchCriteria{
		Tag:      "environment",
		TagValue: "production",
	}
	prodEntities := manager.SearchEntities(criteria)
	fmt.Printf("Production entities: %d\n", len(prodEntities))

	fmt.Println("\n3. Relationship Mapping")
	fmt.Println("-----------------------")

	// Get dependency graph for the job
	graph, err := manager.GetDependencyGraph("email-job-1", 3)
	if err != nil {
		fmt.Printf("Error building graph: %v\n", err)
	} else {
		fmt.Printf("Dependency graph for 'email-job-1':\n")
		fmt.Printf("  Nodes: %d\n", len(graph.Nodes))
		fmt.Printf("  Edges: %d\n", len(graph.Edges))
		for _, edge := range graph.Edges {
			fromEntity, _ := manager.GetEntity(edge.From)
			toEntity, _ := manager.GetEntity(edge.To)
			fmt.Printf("  %s (%s) -> %s (%s)\n",
				fromEntity.Name, edge.Relationship,
				toEntity.Name, edge.To)
		}
	}

	// Get impacted entities if database fails
	impacted, err := manager.GetImpactedEntities("postgres-prod-1")
	if err != nil {
		fmt.Printf("Error getting impacted entities: %v\n", err)
	} else {
		fmt.Printf("\nIf 'postgres-prod-1' fails, %d entities would be impacted:\n", len(impacted))
		for _, entity := range impacted {
			fmt.Printf("  - %s (%s)\n", entity.Name, entity.ID)
		}
	}

	fmt.Println("\n4. Capability Detection")
	fmt.Println("-----------------------")

	for _, entity := range allEntities {
		fmt.Printf("%s (%s):\n", entity.Name, entity.Type)
		if len(entity.Capabilities) > 0 {
			for _, cap := range entity.Capabilities {
				fmt.Printf("  ✓ %s\n", cap)
			}
		} else {
			fmt.Println("  (no capabilities detected)")
		}
	}

	fmt.Println("\n5. State Management")
	fmt.Println("-------------------")

	// Update entity states
	regAPI := manager.GetRegistrationAPI()

	_ = regAPI.UpdateState(ctx, "postgres-prod-1", layer2.StateHealthy, 98.5)
	fmt.Println("✓ Updated postgres-prod-1: healthy (98.5)")

	_ = regAPI.UpdateState(ctx, "user-api-1", layer2.StateHealthy, 95.0)
	fmt.Println("✓ Updated user-api-1: healthy (95.0)")

	_ = regAPI.UpdateState(ctx, "email-job-1", layer2.StateDegraded, 75.0)
	fmt.Println("✓ Updated email-job-1: degraded (75.0)")

	// List entities by state
	healthyEntities := manager.GetCatalog().ListByState(layer2.StateHealthy)
	fmt.Printf("\nHealthy entities: %d\n", len(healthyEntities))

	degradedEntities := manager.GetCatalog().ListByState(layer2.StateDegraded)
	fmt.Printf("Degraded entities: %d\n", len(degradedEntities))

	fmt.Println("\n6. Discovery Simulation")
	fmt.Println("-----------------------")

	// Run manual discovery
	result, err := manager.Discover(ctx)
	if err != nil {
		fmt.Printf("Discovery error: %v\n", err)
	} else {
		fmt.Printf("Discovery completed in %v\n", result.Duration)
		fmt.Printf("  Entities found: %d\n", result.EntitiesFound)
		fmt.Printf("  Errors: %d\n", len(result.Errors))

		for _, entity := range result.Entities {
			fmt.Printf("  - Discovered: %s via %s\n", entity.Name, entity.DiscoveredBy)
		}
	}

	fmt.Println("\n7. System Statistics")
	fmt.Println("--------------------")

	stats := manager.GetStats()
	fmt.Printf("Total entities: %d\n", stats.TotalEntities)
	fmt.Printf("Healthy: %d\n", stats.HealthyEntities)
	fmt.Printf("Unhealthy: %d\n", stats.UnhealthyEntities)

	fmt.Println("\nBy type:")
	for entityType, count := range stats.ByType {
		fmt.Printf("  %s: %d\n", entityType, count)
	}

	fmt.Println("\nBy state:")
	for state, count := range stats.ByState {
		fmt.Printf("  %s: %d\n", state, count)
	}

	fmt.Println("\n8. System Validation")
	fmt.Println("--------------------")

	validationErrors := manager.ValidateSystem()
	if len(validationErrors) == 0 {
		fmt.Println("✓ System validation passed - no issues found")
	} else {
		fmt.Printf("⚠ Found %d validation issues:\n", len(validationErrors))
		for _, err := range validationErrors {
			fmt.Printf("  - [%s] %s: %s\n", err.Type, err.EntityID, err.Message)
		}
	}

	fmt.Println("\n9. Bulk Registration")
	fmt.Println("--------------------")

	bulkRequests := []*layer2.RegistrationRequest{
		{
			ID:       "cache-redis-1",
			Type:     layer2.EntityTypeDatabase,
			Name:     "Redis Cache",
			Endpoint: "redis.internal:6379",
			Metadata: map[string]interface{}{
				"type": "cache",
			},
			AutoDetect: true,
		},
		{
			ID:       "queue-rabbitmq-1",
			Type:     layer2.EntityTypePipeline,
			Name:     "RabbitMQ Queue",
			Endpoint: "rabbitmq.internal:5672",
			AutoDetect: true,
		},
	}

	bulkResponses, err := regAPI.BulkRegister(ctx, bulkRequests)
	if err != nil {
		fmt.Printf("Bulk registration error: %v\n", err)
	}

	successCount := 0
	for _, resp := range bulkResponses {
		if resp.Success {
			successCount++
			fmt.Printf("✓ %s\n", resp.Entity.Name)
		} else {
			fmt.Printf("✗ %s: %s\n", resp.Message, resp.Message)
		}
	}
	fmt.Printf("Successfully registered %d/%d entities\n", successCount, len(bulkRequests))

	fmt.Println("\n=== Demo Complete ===")
}
