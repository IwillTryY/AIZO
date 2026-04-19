package main

import (
	"context"
	"fmt"
	"time"

	"github.com/realityos/aizo/layer4"
)

func main() {
	fmt.Println("=== RealityOS Layer 4 Demo ===")
	fmt.Println("State Management Layer\n")

	// Create Layer 4 manager
	config := &layer4.ManagerConfig{
		EnableAutoReconciliation: false,
		ReconciliationInterval:   5 * time.Minute,
		SnapshotInterval:         1 * time.Hour,
		EnableSnapshots:          true,
	}

	manager := layer4.NewManager(config)
	ctx := context.Background()

	// Start manager
	manager.Start(ctx, config)
	fmt.Println("✓ Layer 4 manager started\n")

	// 1. Set Entity States
	fmt.Println("1. Setting Entity States")
	fmt.Println("------------------------")

	// Create state for a web server
	webServerState := &layer4.EntityState{
		EntityID: "web-server-1",
		Type:     "server",
		Status:   layer4.StatusOnline,
		DesiredState: map[string]interface{}{
			"version":    "1.2.0",
			"replicas":   3,
			"cpu_limit":  "2000m",
			"mem_limit":  "4Gi",
			"port":       8080,
		},
		ActualState: map[string]interface{}{
			"version":    "1.2.0",
			"replicas":   3,
			"cpu_limit":  "2000m",
			"mem_limit":  "4Gi",
			"port":       8080,
		},
		Configuration: map[string]interface{}{
			"environment": "production",
			"region":      "us-east-1",
		},
		Metadata: map[string]interface{}{
			"owner": "platform-team",
			"cost_center": "engineering",
		},
	}

	err := manager.SetEntityState(ctx, webServerState)
	if err != nil {
		fmt.Printf("Error setting state: %v\n", err)
	} else {
		fmt.Printf("✓ Set state for %s\n", webServerState.EntityID)
		fmt.Printf("  Status: %s\n", webServerState.Status)
		fmt.Printf("  Desired replicas: %v\n", webServerState.DesiredState["replicas"])
	}

	// Create state for a database
	dbState := &layer4.EntityState{
		EntityID: "postgres-db-1",
		Type:     "database",
		Status:   layer4.StatusOnline,
		DesiredState: map[string]interface{}{
			"version":       "14.5",
			"storage_size":  "100Gi",
			"backup_enabled": true,
			"max_connections": 100,
		},
		ActualState: map[string]interface{}{
			"version":       "14.5",
			"storage_size":  "100Gi",
			"backup_enabled": true,
			"max_connections": 100,
		},
		Configuration: map[string]interface{}{
			"environment": "production",
		},
	}

	_ = manager.SetEntityState(ctx, dbState)
	fmt.Printf("✓ Set state for %s\n", dbState.EntityID)

	// 2. Detect Drift
	fmt.Println("\n2. Detecting State Drift")
	fmt.Println("------------------------")

	// Simulate drift by changing actual state
	_ = manager.UpdateActualState(ctx, "web-server-1", map[string]interface{}{
		"version":    "1.1.0", // Drifted from desired 1.2.0
		"replicas":   2,       // Drifted from desired 3
		"cpu_limit":  "2000m",
		"mem_limit":  "4Gi",
		"port":       8080,
	})

	drift, err := manager.DetectDrift(ctx, "web-server-1")
	if err != nil {
		fmt.Printf("Error detecting drift: %v\n", err)
	} else {
		fmt.Printf("✓ Drift detection for web-server-1:\n")
		fmt.Printf("  Has drift: %v\n", drift.HasDrift)
		fmt.Printf("  Drift score: %.1f\n", drift.DriftScore)
		fmt.Printf("  Differences found: %d\n", len(drift.Differences))
		for _, diff := range drift.Differences {
			fmt.Printf("    - %s: desired=%v, actual=%v (severity: %s)\n",
				diff.Field, diff.DesiredValue, diff.ActualValue, diff.Severity)
		}
	}

	// 3. Query States
	fmt.Println("\n3. Querying Entity States")
	fmt.Println("-------------------------")

	// Query all online entities
	query := &layer4.StateQuery{
		Status: []layer4.EntityStatus{layer4.StatusOnline},
	}

	result, err := manager.QueryState(ctx, query)
	if err != nil {
		fmt.Printf("Error querying states: %v\n", err)
	} else {
		fmt.Printf("✓ Query results: %d entities online\n", result.Count)
		for _, state := range result.States {
			fmt.Printf("  - %s (%s)\n", state.EntityID, state.Type)
		}
	}

	// Query entities with drift
	hasDrift := true
	driftQuery := &layer4.StateQuery{
		HasDrift: &hasDrift,
	}

	driftResult, _ := manager.QueryState(ctx, driftQuery)
	fmt.Printf("✓ Entities with drift: %d\n", driftResult.Count)

	// 4. Reconciliation
	fmt.Println("\n4. State Reconciliation")
	fmt.Println("-----------------------")

	// Register a default reconciler
	reconciler := layer4.NewDefaultReconciler("server")
	manager.RegisterReconciler("server", reconciler)

	// Perform reconciliation (dry run)
	reconReq := &layer4.ReconciliationRequest{
		EntityID: "web-server-1",
		DesiredState: map[string]interface{}{
			"version":    "1.2.0",
			"replicas":   3,
			"cpu_limit":  "2000m",
			"mem_limit":  "4Gi",
			"port":       8080,
		},
		DryRun: true,
	}

	reconResult, err := manager.Reconcile(ctx, reconReq)
	if err != nil {
		fmt.Printf("Error reconciling: %v\n", err)
	} else {
		fmt.Printf("✓ Reconciliation (dry run) for %s:\n", reconResult.EntityID)
		fmt.Printf("  Success: %v\n", reconResult.Success)
		fmt.Printf("  Changes to apply: %d\n", len(reconResult.Changes))
		for _, change := range reconResult.Changes {
			fmt.Printf("    - %s: %v -> %v\n", change.Field, change.OldValue, change.NewValue)
		}
		fmt.Printf("  Duration: %v\n", reconResult.Duration)
	}

	// 5. State History & Snapshots
	fmt.Println("\n5. State History & Snapshots")
	fmt.Println("----------------------------")

	// Create a snapshot
	snapshot, err := manager.CreateSnapshot(ctx, "Initial state snapshot", []string{"demo", "initial"})
	if err != nil {
		fmt.Printf("Error creating snapshot: %v\n", err)
	} else {
		fmt.Printf("✓ Created snapshot: %s\n", snapshot.ID)
		fmt.Printf("  Timestamp: %s\n", snapshot.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Description: %s\n", snapshot.Description)
		fmt.Printf("  Entities captured: %d\n", len(snapshot.State.Entities))
	}

	// Get state history
	history, err := manager.GetEntityHistory(ctx, "web-server-1", 10)
	if err != nil {
		fmt.Printf("Error getting history: %v\n", err)
	} else {
		fmt.Printf("\n✓ State history for web-server-1: %d changes\n", len(history))
		for _, change := range history {
			fmt.Printf("  [%s] %s: %s\n",
				change.Timestamp.Format("15:04:05"),
				change.ChangeType,
				change.Field)
		}
	}

	// 6. System State
	fmt.Println("\n6. System State Overview")
	fmt.Println("------------------------")

	systemState, err := manager.GetSystemState(ctx)
	if err != nil {
		fmt.Printf("Error getting system state: %v\n", err)
	} else {
		fmt.Printf("✓ System state:\n")
		fmt.Printf("  Total entities: %d\n", len(systemState.Entities))
		fmt.Printf("  Version: %d\n", systemState.Version)
		fmt.Printf("  Timestamp: %s\n", systemState.Timestamp.Format(time.RFC3339))
	}

	// 7. Drift Report
	fmt.Println("\n7. Drift Report")
	fmt.Println("---------------")

	projectionAPI := manager.GetProjectionAPI()
	driftReport, err := projectionAPI.GetDriftReport(ctx)
	if err != nil {
		fmt.Printf("Error generating drift report: %v\n", err)
	} else {
		fmt.Printf("✓ Drift Report:\n")
		fmt.Printf("  Total entities: %d\n", driftReport.TotalEntities)
		fmt.Printf("  Entities with drift: %d\n", driftReport.EntitiesWithDrift)
		fmt.Printf("  Drift percentage: %.1f%%\n",
			float64(driftReport.EntitiesWithDrift)/float64(driftReport.TotalEntities)*100)

		if len(driftReport.DriftDetails) > 0 {
			fmt.Println("\n  Drift details:")
			for _, detail := range driftReport.DriftDetails {
				fmt.Printf("    - %s (%s): score=%.1f, differences=%d\n",
					detail.EntityID, detail.EntityType, detail.DriftScore, len(detail.Differences))
			}
		}
	}

	// 8. Health Report
	fmt.Println("\n8. Health Report")
	fmt.Println("----------------")

	healthReport, err := projectionAPI.GetHealthReport(ctx)
	if err != nil {
		fmt.Printf("Error generating health report: %v\n", err)
	} else {
		fmt.Printf("✓ Health Report:\n")
		fmt.Printf("  Total entities: %d\n", healthReport.TotalEntities)
		fmt.Printf("  By status:\n")
		for status, count := range healthReport.ByStatus {
			fmt.Printf("    %s: %d\n", status, count)
		}

		if len(healthReport.Issues) > 0 {
			fmt.Printf("\n  Issues found: %d\n", len(healthReport.Issues))
			for _, issue := range healthReport.Issues {
				fmt.Printf("    - %s: %s (last seen: %s)\n",
					issue.EntityID, issue.Status, issue.LastSeen.Format(time.RFC3339))
			}
		}
	}

	// 9. Statistics
	fmt.Println("\n9. Layer 4 Statistics")
	fmt.Println("---------------------")

	stats, err := manager.GetStats(ctx)
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
	} else {
		fmt.Printf("✓ Statistics:\n")
		fmt.Printf("  Total entities: %d\n", stats.TotalEntities)
		fmt.Printf("  Entities with drift: %d\n", stats.EntitiesWithDrift)
		fmt.Printf("  Total snapshots: %d\n", stats.TotalSnapshots)
		fmt.Printf("  By status:\n")
		for status, count := range stats.ByStatus {
			fmt.Printf("    %s: %d\n", status, count)
		}
	}

	// 10. Time-Travel Query
	fmt.Println("\n10. Time-Travel Query")
	fmt.Println("---------------------")

	// Wait a moment and create another snapshot
	time.Sleep(100 * time.Millisecond)

	// Update state
	_ = manager.UpdateActualState(ctx, "web-server-1", map[string]interface{}{
		"version":    "1.2.0",
		"replicas":   3,
		"cpu_limit":  "2000m",
		"mem_limit":  "4Gi",
		"port":       8080,
	})

	snapshot2, _ := manager.CreateSnapshot(ctx, "After reconciliation", []string{"demo", "reconciled"})
	fmt.Printf("✓ Created second snapshot: %s\n", snapshot2.ID)

	// Compare snapshots
	stateHistory := manager.GetStateHistory()
	comparison, err := stateHistory.CompareSnapshots(ctx, snapshot.ID, snapshot2.ID)
	if err != nil {
		fmt.Printf("Error comparing snapshots: %v\n", err)
	} else {
		fmt.Printf("\n✓ Snapshot comparison:\n")
		fmt.Printf("  Snapshot 1: %s\n", snapshot.Timestamp.Format("15:04:05"))
		fmt.Printf("  Snapshot 2: %s\n", snapshot2.Timestamp.Format("15:04:05"))
		fmt.Printf("  Differences: %d\n", len(comparison.Differences))
		for _, diff := range comparison.Differences {
			fmt.Printf("    - %s: %s\n", diff.EntityID, diff.ChangeType)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nLayer 4 provides:")
	fmt.Println("  • Real-time state tracking")
	fmt.Println("  • Drift detection and scoring")
	fmt.Println("  • State reconciliation")
	fmt.Println("  • Historical snapshots")
	fmt.Println("  • Time-travel queries")
	fmt.Println("  • Comprehensive reporting")
}
