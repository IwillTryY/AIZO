# Layer 4: State Management Layer - Version 1

## Overview
Layer 4 is the State Management system for RealityOS - it maintains a real-time view of the entire system's state, detects drift, and reconciles differences between desired and actual state.

## Architecture

### Core Components

1. **Types** (`types.go`)
   - SystemState: Complete system state snapshot
   - EntityState: Individual entity state with desired/actual states
   - StateDrift: Drift detection results
   - StateChange: Change tracking
   - ReconciliationRequest/Result: Reconciliation operations
   - Snapshot: Point-in-time state snapshots

2. **State Store** (`state_store.go`)
   - Thread-safe state storage
   - CRUD operations for entity states
   - Query capabilities with filtering
   - Change history tracking
   - Desired vs actual state management
   - System-wide state aggregation

3. **Change Detector** (`change_detector.go`)
   - Drift detection between desired and actual state
   - Severity classification (low, medium, high, critical)
   - Drift scoring (0-100)
   - Change tracking between state versions
   - Field-level difference detection

4. **Reconciliation Engine** (`reconciliation_engine.go`)
   - State reconciliation logic
   - Pluggable reconciler system
   - Dry-run support
   - Automatic reconciliation for drifted entities
   - Change tracking and reporting

5. **State History** (`state_history.go`)
   - Point-in-time snapshots
   - Time-travel queries
   - Snapshot comparison
   - Historical state reconstruction
   - Snapshot management (create, list, delete)

6. **State Projection API** (`state_projection.go`)
   - Current and historical state queries
   - Entity timelines
   - System-wide timelines
   - Drift reports
   - Health reports
   - Advanced filtering and projection

7. **Manager** (`manager.go`)
   - High-level orchestration
   - Periodic reconciliation
   - Automatic snapshots
   - Statistics and reporting
   - Component lifecycle management

## Features Implemented

### ✅ State Management
- Store and retrieve entity states
- Desired vs actual state tracking
- Version management
- Metadata and configuration storage

### ✅ Drift Detection
- Automatic drift detection
- Field-level difference tracking
- Severity classification
- Drift scoring (0-100)
- Configurable thresholds

### ✅ State Reconciliation
- Pluggable reconciler system
- Dry-run mode
- Change tracking
- Automatic reconciliation
- Error handling and reporting

### ✅ State History
- Point-in-time snapshots
- Time-travel queries
- Snapshot comparison
- Change history tracking
- Snapshot tagging

### ✅ Query & Projection
- Flexible state queries
- Historical queries
- Entity timelines
- System timelines
- Drift reports
- Health reports

## Usage Example

```go
// Create Layer 4 manager
config := &layer4.ManagerConfig{
    EnableAutoReconciliation: true,
    ReconciliationInterval:   5 * time.Minute,
    SnapshotInterval:         1 * time.Hour,
    EnableSnapshots:          true,
}

manager := layer4.NewManager(config)
ctx := context.Background()

// Start manager
manager.Start(ctx, config)

// Set entity state
state := &layer4.EntityState{
    EntityID: "web-server-1",
    Type:     "server",
    Status:   layer4.StatusOnline,
    DesiredState: map[string]interface{}{
        "version":   "1.2.0",
        "replicas":  3,
        "cpu_limit": "2000m",
    },
    ActualState: map[string]interface{}{
        "version":   "1.2.0",
        "replicas":  3,
        "cpu_limit": "2000m",
    },
}

err := manager.SetEntityState(ctx, state)

// Detect drift
drift, err := manager.DetectDrift(ctx, "web-server-1")
if drift.HasDrift {
    fmt.Printf("Drift detected! Score: %.1f\n", drift.DriftScore)
    for _, diff := range drift.Differences {
        fmt.Printf("  %s: %v -> %v (severity: %s)\n",
            diff.Field, diff.DesiredValue, diff.ActualValue, diff.Severity)
    }
}

// Reconcile state
reconReq := &layer4.ReconciliationRequest{
    EntityID:     "web-server-1",
    DesiredState: state.DesiredState,
    DryRun:       false,
}

result, err := manager.Reconcile(ctx, reconReq)

// Create snapshot
snapshot, err := manager.CreateSnapshot(ctx, "Production snapshot", []string{"prod"})

// Query states
query := &layer4.StateQuery{
    Status: []layer4.EntityStatus{layer4.StatusOnline},
}

queryResult, err := manager.QueryState(ctx, query)

// Get drift report
projectionAPI := manager.GetProjectionAPI()
driftReport, err := projectionAPI.GetDriftReport(ctx)

// Time-travel query
historicalState, err := projectionAPI.GetHistoricalState(ctx, "web-server-1", timestamp)
```

## Data Models

### EntityState
```go
EntityState {
    entity_id: "unique-id"
    type: "server" | "database" | "api" | ...
    status: online | offline | degraded | maintenance | unknown
    desired_state: {key: value}  // What it should be
    actual_state: {key: value}   // What it actually is
    configuration: {key: value}
    metadata: {key: value}
    last_updated: timestamp
    last_seen: timestamp
    version: int64
    drift: StateDrift
}
```

### StateDrift
```go
StateDrift {
    has_drift: bool
    detected_at: timestamp
    differences: [
        {
            field: "field_name"
            desired_value: value
            actual_value: value
            severity: "low" | "medium" | "high" | "critical"
        }
    ]
    drift_score: 0-100
}
```

### SystemState
```go
SystemState {
    entities: {entity_id: EntityState}
    relationships: RelationshipGraph
    health: {entity_id: HealthState}
    topology: NetworkTopology
    timestamp: timestamp
    version: int64
}
```

## Running the Demo

```bash
cd examples
go run layer4_demo.go
```

The demo demonstrates:
1. Setting entity states
2. Detecting state drift
3. Querying entity states
4. State reconciliation (dry-run and actual)
5. Creating snapshots
6. State history tracking
7. System state overview
8. Drift reports
9. Health reports
10. Time-travel queries and snapshot comparison

## Integration with Previous Layers

### With Layer 1 (Adapters)
- Adapters report actual state from systems
- State changes trigger adapter commands

### With Layer 2 (Discovery & Registration)
- Entity catalog syncs with state store
- Entity metadata flows into state

### With Layer 3 (Telemetry)
- State changes generate events
- Drift detection creates alerts
- Reconciliation actions logged

Example integration:
```go
// Layer 2 entity registered
entity := layer2Manager.GetEntity("server-1")

// Layer 4 tracks its state
state := &layer4.EntityState{
    EntityID: entity.ID,
    Type:     string(entity.Type),
    Status:   layer4.StatusOnline,
}
layer4Manager.SetEntityState(ctx, state)

// Layer 1 adapter reports actual state
actualState, _ := adapter.ReadState(ctx)
layer4Manager.UpdateActualState(ctx, entity.ID, actualState.Data)

// Layer 3 logs state change
layer3Manager.PublishEvent(&layer3.Event{
    Type:     layer3.EventTypeStateChange,
    EntityID: entity.ID,
    Message:  "State updated",
})
```

## What's Next (Future Versions)

### Advanced Drift Detection
- Machine learning-based drift prediction
- Anomaly detection in state changes
- Drift trend analysis
- Automatic drift remediation

### Enhanced Reconciliation
- Multi-step reconciliation workflows
- Rollback on failure
- Dependency-aware reconciliation
- Parallel reconciliation

### State Persistence
- Database backend (PostgreSQL, MongoDB)
- Distributed state store (etcd, Consul)
- State replication
- Backup and restore

### Advanced Queries
- GraphQL API for state queries
- Complex filtering and aggregation
- Real-time state subscriptions
- State diff API

### Compliance & Audit
- State change audit trail
- Compliance checking
- Policy enforcement
- Approval workflows

## Design Decisions

1. **In-Memory Storage**: Fast development and testing (use external backends for production)
2. **Desired vs Actual**: Clear separation for drift detection
3. **Pluggable Reconcilers**: Entity-specific reconciliation logic
4. **Snapshot-Based History**: Efficient time-travel queries
5. **Thread-Safe**: All operations use proper locking
6. **Context-Aware**: All operations support cancellation

## API Reference

### Manager
- `Start(ctx, config)` - Start background tasks
- `SetEntityState(ctx, state)` - Set entity state
- `GetEntityState(ctx, entityID)` - Get entity state
- `UpdateDesiredState(ctx, entityID, state)` - Update desired state
- `UpdateActualState(ctx, entityID, state)` - Update actual state
- `DetectDrift(ctx, entityID)` - Detect drift
- `Reconcile(ctx, req)` - Reconcile state
- `CreateSnapshot(ctx, description, tags)` - Create snapshot
- `QueryState(ctx, query)` - Query states
- `GetSystemState(ctx)` - Get complete system state
- `GetStats(ctx)` - Get statistics

### State Store
- `Set(ctx, state)` - Store state
- `Get(ctx, entityID)` - Get state
- `Delete(ctx, entityID)` - Delete state
- `List(ctx)` - List all states
- `Query(ctx, query)` - Query states
- `GetHistory(ctx, entityID, limit)` - Get change history
- `GetSystemState(ctx)` - Get system state

### Change Detector
- `DetectDrift(desired, actual)` - Detect drift
- `DetectChanges(oldState, newState)` - Detect changes
- `CalculateDriftScore(drift)` - Calculate drift score

### Reconciliation Engine
- `Reconcile(ctx, req)` - Reconcile entity
- `ReconcileAll(ctx)` - Reconcile all drifted entities
- `CheckDrift(ctx, entityID)` - Check for drift
- `RegisterReconciler(entityType, reconciler)` - Register reconciler

### State History
- `CreateSnapshot(ctx, state, description, tags)` - Create snapshot
- `GetSnapshot(ctx, snapshotID)` - Get snapshot
- `ListSnapshots(ctx)` - List snapshots
- `GetStateAtTime(ctx, timestamp)` - Time-travel query
- `DeleteSnapshot(ctx, snapshotID)` - Delete snapshot
- `CompareSnapshots(ctx, id1, id2)` - Compare snapshots

### State Projection API
- `GetCurrentState(ctx, entityID)` - Get current state
- `GetHistoricalState(ctx, entityID, timestamp)` - Get historical state
- `QueryCurrentState(ctx, query)` - Query current states
- `QueryHistoricalState(ctx, query, timestamp)` - Query historical states
- `GetEntityTimeline(ctx, entityID, start, end)` - Get entity timeline
- `GetSystemTimeline(ctx, start, end)` - Get system timeline
- `GetDriftReport(ctx)` - Generate drift report
- `GetHealthReport(ctx)` - Generate health report

## Limitations (V1)

- In-memory storage only (no persistence)
- Basic reconciliation logic
- No distributed state management
- No state replication
- Limited query optimization
- No authentication/authorization
- No multi-tenancy support

## Performance Considerations

- In-memory storage is fast but not persistent
- Large state histories consume memory
- Snapshot comparison can be expensive for large systems
- Query performance degrades with many entities
- Consider external storage for production use
