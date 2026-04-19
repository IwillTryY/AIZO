# Layer 1: Adapter Layer - Version 1

## Overview
This is the first implementation of Layer 1 (Adapter Layer) for RealityOS - the universal connector that bridges RealityOS with any external system.

## Architecture

### Core Components

1. **Adapter Interface** (`adapter.go`)
   - Defines the contract all adapters must implement
   - Core operations: Connect, Disconnect, ReadState, SendCommand, HealthCheck
   - BaseAdapter provides common functionality (retry logic, health tracking, metrics)

2. **Protocol Adapters**
   - **HTTP Adapter** (`http_adapter.go`): REST/HTTP endpoints
     - Supports GET/POST/PUT/DELETE methods
     - Multiple auth methods (Bearer token, API key, Basic auth)
     - Automatic retry with exponential backoff

   - **SSH Adapter** (`ssh_adapter.go`): Remote server management
     - Password and key-based authentication
     - Command execution and state collection
     - System metrics gathering (CPU, memory, disk, uptime)

3. **Adapter Registry** (`registry.go`)
   - Central catalog of all adapters
   - Thread-safe operations
   - Metadata tracking (registration time, health checks, tags)
   - Bulk operations (health check all, list by type)

4. **Adapter Factory** (`factory.go`)
   - Creates adapters from configuration
   - Automatic registration
   - Connection management

5. **Manager** (`manager.go`)
   - High-level orchestration layer
   - Lifecycle management (create, connect, disconnect, shutdown)
   - Periodic health monitoring
   - Statistics and reporting

## Features Implemented

### ✅ Bidirectional Communication
- Read state from systems
- Send commands to systems
- Full request/response tracking

### ✅ Health Monitoring
- Per-adapter health checks
- Status tracking (healthy, degraded, unhealthy)
- Latency measurement
- Success rate calculation

### ✅ Automatic Reconnection
- Configurable retry attempts
- Exponential backoff
- Error tracking and reporting

### ✅ Protocol Translation
- HTTP/REST normalization
- SSH command execution
- Consistent state data format

## Usage Example

```go
// Create manager
manager := layer1.NewManager()

// Create HTTP adapter
config := &layer1.AdapterConfig{
    ID:     "api-server-1",
    Type:   layer1.AdapterTypeHTTP,
    Target: "https://api.example.com",
    Credentials: map[string]string{
        "bearer_token": "your-token",
    },
    Timeout:      10 * time.Second,
    RetryAttempts: 3,
}

adapter, err := manager.CreateAndConnectAdapter(config)

// Read state
state, err := adapter.ReadState(context.Background())

// Send command
cmd := &layer1.CommandRequest{
    ID:      "cmd-1",
    Command: "restart",
    Args:    map[string]interface{}{"force": true},
}
response, err := adapter.SendCommand(context.Background(), cmd)

// Health check
health, err := adapter.HealthCheck(context.Background())
```

## Configuration

### Adapter Config Structure
```go
type AdapterConfig struct {
    ID              string                 // Unique identifier
    Type            AdapterType            // http, ssh, etc.
    Target          string                 // Connection target (URL, IP:port)
    Credentials     map[string]string      // Auth credentials
    Timeout         time.Duration          // Operation timeout
    RetryAttempts   int                    // Number of retry attempts
    RetryBackoff    time.Duration          // Backoff between retries
    HealthCheckInterval time.Duration      // Health check frequency
    Metadata        map[string]interface{} // Custom metadata
}
```

## Running the Demo

```bash
cd examples
go run layer1_demo.go
```

The demo shows:
1. Creating and connecting HTTP adapter
2. Reading state from endpoints
3. Health checking
4. Registry operations
5. Statistics gathering
6. Command execution

## What's Next (Future Versions)

### Additional Protocol Adapters
- gRPC adapter
- WebSocket adapter
- MQTT adapter
- SNMP adapter
- GraphQL adapter

### Enhanced Features
- Custom adapter SDK for building new adapters
- Adapter plugin system
- Advanced metrics collection
- Event streaming support
- Connection pooling
- Circuit breaker pattern
- Rate limiting

### Integration
- Layer 2 integration (Discovery & Registration)
- Telemetry export to Layer 3
- State synchronization with Layer 4

## Design Decisions

1. **Go Language**: Chosen for performance, concurrency, and strong typing
2. **Interface-Based**: Allows easy extension with new adapter types
3. **Thread-Safe**: All registry operations use proper locking
4. **Context-Aware**: All operations support context for cancellation/timeout
5. **Health-First**: Built-in health monitoring from the start
6. **Retry Logic**: Automatic retry with backoff for resilience

## Testing

To test with real systems:

```go
// HTTP endpoint
config := &layer1.AdapterConfig{
    ID:     "test-api",
    Type:   layer1.AdapterTypeHTTP,
    Target: "https://httpbin.org/get",
}

// SSH server
config := &layer1.AdapterConfig{
    ID:     "test-server",
    Type:   layer1.AdapterTypeSSH,
    Target: "your-server:22",
    Credentials: map[string]string{
        "username": "user",
        "password": "pass",
    },
}
```

## Limitations (V1)

- SSH adapter uses `InsecureIgnoreHostKey` (needs proper host key verification)
- No TLS certificate validation options for HTTP
- Limited error context in some failure scenarios
- No connection pooling yet
- No rate limiting
- No circuit breaker implementation

## Contributing

To add a new adapter type:

1. Implement the `Adapter` interface
2. Embed `BaseAdapter` for common functionality
3. Add the adapter type constant
4. Update the factory to create your adapter
5. Add tests and documentation
