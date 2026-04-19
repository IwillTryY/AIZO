# Layer 2: Discovery & Registration Layer - Version 1

## Overview
Layer 2 is the Discovery & Registration system for RealityOS - it finds, identifies, and catalogs all managed entities in your infrastructure.

## Architecture

### Core Components

1. **Entity Model** (`entity.go`)
   - Defines the structure of managed entities
   - Entity types: server, api, database, job, pipeline, device, script
   - Capabilities: metrics, logs, traces, commands, health_checks
   - Relationships: depends_on, provides_to, part_of
   - State tracking: unknown, healthy, degraded, unhealthy, offline

2. **Entity Catalog** (`catalog.go`)
   - Central registry of all known entities
   - Thread-safe operations with RWMutex
   - Fast lookups via indexes (by type, adapter, state)
   - Search functionality with flexible criteria
   - Automatic cleanup of stale entities

3. **Discovery Engine** (`discovery.go`)
   - Auto-discovery via multiple methods:
     - Network scanning (port scanning, service detection)
     - Cloud API integration (AWS, Azure, GCP)
     - Container runtime (Docker, Kubernetes)
     - DNS lookup
     - Agent reporting
   - Periodic discovery with configurable intervals
   - Discovery result tracking and error handling

4. **Capability Detector** (`capability_detector.go`)
   - Identifies what each entity can do
   - Pluggable probe system for different capabilities
   - Built-in probes:
     - MetricsProbe: Detects metrics endpoints
     - LogsProbe: Detects log sources
     - CommandProbe: Detects command execution capability
     - HealthCheckProbe: Detects health check endpoints
     - TracesProbe: Detects distributed tracing

5. **Relationship Mapper** (`relationship_mapper.go`)
   - Discovers dependencies between entities
   - Builds dependency graphs with configurable depth
   - Detects circular dependencies
   - Impact analysis (what fails if X fails)
   - Relationship types:
     - DependencyDetector: Finds "depends_on" relationships
     - ProviderDetector: Finds "provides_to" relationships
     - HierarchyDetector: Finds "part_of" relationships

6. **Registration API** (`registration.go`)
   - Manual and programmatic entity registration
   - Validation of registration requests
   - Bulk registration support
   - State management (update health, capabilities)
   - Relationship management
   - Deregistration with dependency checking

7. **Manager** (`manager.go`)
   - High-level orchestration layer
   - Lifecycle management (start, stop)
   - Periodic background tasks:
     - Discovery scanning
     - Relationship mapping
     - Stale entity cleanup
   - System validation
   - Statistics and reporting

## Features Implemented

### ✅ Entity Management
- Register, update, and deregister entities
- Bulk operations
- Search and filtering
- State tracking with health scores

### ✅ Auto-Discovery
- Network scanning for servers and services
- Cloud provider integration
- Container runtime discovery
- Configurable discovery methods and intervals

### ✅ Capability Detection
- Automatic detection of entity capabilities
- Extensible probe system
- Support for metrics, logs, traces, commands, health checks

### ✅ Relationship Mapping
- Automatic dependency discovery
- Dependency graph construction
- Impact analysis
- Circular dependency detection

### ✅ System Validation
- Detect orphaned entities
- Identify entities without adapters
- Validate dependency integrity

## Usage Example

```go
// Create manager
config := &layer2.ManagerConfig{
    DiscoveryConfig: &layer2.DiscoveryConfig{
        Methods:       []layer2.DiscoveryMethod{layer2.DiscoveryNetwork},
        ScanInterval:  5 * time.Minute,
        NetworkRanges: []string{"10.0.0.0/24"},
    },
    EnableAutoDiscovery: true,
    StaleEntityTimeout:  24 * time.Hour,
}

manager := layer2.NewManager(config)
ctx := context.Background()

// Start manager (enables periodic discovery)
manager.Start(ctx)

// Manual entity registration
req := &layer2.RegistrationRequest{
    ID:       "postgres-1",
    Type:     layer2.EntityTypeDatabase,
    Name:     "Production Database",
    Endpoint: "postgres.internal:5432",
    Metadata: map[string]interface{}{
        "version": "14.5",
    },
    AutoDetect:   true,  // Auto-detect capabilities
    MapRelations: true,  // Auto-map relationships
}

response, err := manager.RegisterEntity(ctx, req)

// Search entities
criteria := layer2.SearchCriteria{
    Type:           &layer2.EntityTypeAPI,
    MinHealthScore: 80.0,
}
entities := manager.SearchEntities(criteria)

// Get dependency graph
graph, err := manager.GetDependencyGraph("my-api", 3)

// Get impact analysis
impacted, err := manager.GetImpactedEntities("postgres-1")

// Update entity state
regAPI := manager.GetRegistrationAPI()
regAPI.UpdateState(ctx, "postgres-1", layer2.StateHealthy, 98.5)
```

## Entity Model

```go
Entity {
  id: "unique-identifier"
  type: server | api | database | job | pipeline | device | script
  name: "Human-readable name"
  capabilities: [metrics, logs, traces, commands, health_checks]
  metadata: {tags, labels, custom fields}
  adapters: ["adapter-id-1", "adapter-id-2"]
  relationships: [
    {type: depends_on, target_id: "entity-2"},
    {type: provides_to, target_id: "entity-3"}
  ]
  state: healthy | degraded | unhealthy | offline | unknown
  health_score: 0-100
  endpoint: "connection-string"
  discovered_at: timestamp
  discovered_by: "discovery-method"
  last_seen_at: timestamp
}
```

## Running the Demo

```bash
cd examples
go run layer2_demo.go
```

The demo demonstrates:
1. Manual entity registration
2. Entity catalog operations
3. Relationship mapping and dependency graphs
4. Capability detection
5. State management
6. Discovery simulation
7. System statistics
8. System validation
9. Bulk registration

## Integration with Layer 1

Layer 2 works with Layer 1 (Adapter Layer) by:
- Storing adapter IDs in entity records
- Using adapters for capability detection
- Coordinating with adapters for state updates

Example integration:
```go
// Layer 1: Create adapter
adapter, err := layer1Manager.CreateAndConnectAdapter(adapterConfig)

// Layer 2: Register entity with adapter
entityReq := &layer2.RegistrationRequest{
    ID:       "my-server",
    Type:     layer2.EntityTypeServer,
    Adapters: []string{adapter.GetID()},
}
manager.RegisterEntity(ctx, entityReq)
```

## What's Next (Future Versions)

### Enhanced Discovery
- Active service fingerprinting
- Credential-based discovery (SSH, WMI)
- API-based discovery for more cloud providers
- Kubernetes operator for automatic pod discovery
- Service mesh integration (Istio, Linkerd)

### Advanced Relationship Mapping
- Machine learning for relationship inference
- Network traffic analysis for dependency detection
- Log correlation for relationship discovery
- Performance impact relationships

### Capability Detection
- Dynamic capability probing
- Version-specific capability detection
- Custom capability definitions
- Capability scoring (how well does it support X)

### Entity Lifecycle
- Entity versioning and history
- Entity templates and cloning
- Entity groups and collections
- Entity tagging and labeling system

### Integration
- Layer 3 integration (telemetry collection)
- Layer 4 integration (state synchronization)
- Layer 5 integration (command routing)
- Export to external CMDBs

## Design Decisions

1. **Go Language**: Performance, concurrency, strong typing
2. **Interface-Based**: Extensible scanner and detector system
3. **Thread-Safe**: All catalog operations use proper locking
4. **Context-Aware**: All operations support cancellation
5. **Index-Based**: Fast lookups via multiple indexes
6. **Pluggable**: Easy to add new discovery methods and capability probes

## Configuration

### Discovery Configuration
```go
DiscoveryConfig {
    Methods:           []DiscoveryMethod  // Which discovery methods to use
    ScanInterval:      time.Duration      // How often to scan
    NetworkRanges:     []string           // CIDR ranges for network scanning
    CloudProviders:    []string           // AWS, Azure, GCP, etc.
    ContainerRuntimes: []string           // Docker, Kubernetes, etc.
    Tags:              map[string]string  // Tags to apply to discovered entities
}
```

### Manager Configuration
```go
ManagerConfig {
    DiscoveryConfig:     *DiscoveryConfig
    EnableAutoDiscovery: bool          // Enable periodic discovery
    StaleEntityTimeout:  time.Duration // When to remove unseen entities
}
```

## API Reference

### Manager
- `Start(ctx)` - Start background tasks
- `Stop()` - Stop background tasks
- `RegisterEntity(ctx, req)` - Register new entity
- `GetEntity(id)` - Get entity by ID
- `ListEntities()` - List all entities
- `ListEntitiesByType(type)` - List entities of specific type
- `SearchEntities(criteria)` - Search with criteria
- `GetDependencyGraph(id, depth)` - Build dependency graph
- `GetImpactedEntities(id)` - Get impact analysis
- `GetStats()` - Get system statistics
- `ValidateSystem()` - Validate system integrity

### Registration API
- `Register(ctx, req)` - Register entity
- `Update(ctx, id, updates)` - Update entity
- `Deregister(ctx, id)` - Remove entity (with dependency check)
- `ForceDeregister(ctx, id)` - Force remove entity
- `BulkRegister(ctx, requests)` - Register multiple entities
- `UpdateState(ctx, id, state, score)` - Update entity state
- `AddCapability(ctx, id, capability)` - Add capability
- `AddRelationship(ctx, id, type, target)` - Add relationship

### Entity Catalog
- `Register(entity)` - Add/update entity
- `Unregister(id)` - Remove entity
- `Get(id)` - Get entity
- `List()` - List all
- `ListByType(type)` - List by type
- `ListByAdapter(adapter)` - List by adapter
- `ListByState(state)` - List by state
- `Search(criteria)` - Search entities
- `GetStats()` - Get statistics
- `CleanupStale(maxAge)` - Remove stale entities

## Limitations (V1)

- Discovery scanners are placeholder implementations
- Capability detection uses simple heuristics
- Relationship inference is basic
- No persistence layer (in-memory only)
- No authentication/authorization
- No multi-tenancy support
- Limited cloud provider support

## Contributing

To add a new discovery method:
1. Implement the `Scanner` interface
2. Register with `DiscoveryEngine.RegisterScanner()`

To add a new capability probe:
1. Implement the `CapabilityProbe` interface
2. Register with `CapabilityDetector.RegisterProbe()`

To add a new relationship detector:
1. Implement the `RelationshipDetector` interface
2. Register with `RelationshipMapper.RegisterDetector()`
