# Layer 3: Telemetry & Observability Layer - Version 1

## Overview
Layer 3 is the Telemetry & Observability system for RealityOS - it collects, normalizes, and stores all operational data from your infrastructure.

## Architecture

### Core Components

1. **Types** (`types.go`)
   - Metric types: counter, gauge, histogram, summary
   - Log levels: debug, info, warning, error, fatal
   - Trace and span structures
   - Event types: state_change, health_change, alert, deployment, configuration, error
   - Query and aggregation models

2. **Metrics Collector** (`metrics_collector.go`)
   - Time-series metric collection
   - Buffered ingestion for high throughput
   - Query and aggregation support
   - Multiple aggregation functions: sum, avg, min, max, count, percentiles
   - In-memory storage with pluggable backend interface

3. **Log Aggregator** (`log_aggregator.go`)
   - Structured and unstructured log collection
   - Log levels and filtering
   - Full-text search capability
   - Buffered ingestion
   - In-memory storage with pluggable backend interface

4. **Trace Collector** (`trace_collector.go`)
   - Distributed tracing support
   - Span recording and trace reconstruction
   - Parent-child span relationships
   - Span logs and tags
   - Trace querying

5. **Event Stream** (`event_stream.go`)
   - Real-time event bus
   - Pub/sub pattern for event distribution
   - Event types for different system changes
   - Subscriber management
   - Event persistence and querying

6. **Correlation Engine** (`correlation_engine.go`)
   - Links metrics, logs, traces, and events
   - Timeline construction across all telemetry types
   - Anomaly detection (basic threshold-based)
   - Cross-entity correlation

7. **Manager** (`manager.go`)
   - High-level orchestration layer
   - Unified query interface
   - Component lifecycle management
   - Statistics and reporting

## Features Implemented

### ✅ Metrics Collection
- Counter and gauge metrics
- Time-series storage
- Aggregation (sum, avg, min, max, count)
- Label-based grouping
- Time-based windowing

### ✅ Log Aggregation
- Structured log entries
- Log levels and filtering
- Full-text search
- Entity-based filtering
- Time-range queries

### ✅ Distributed Tracing
- Span recording
- Trace reconstruction
- Parent-child relationships
- Span logs and tags
- Trace querying

### ✅ Event Streaming
- Real-time event publishing
- Pub/sub subscriptions
- Event persistence
- Event type filtering
- Entity-based routing

### ✅ Data Correlation
- Cross-telemetry correlation
- Timeline construction
- Entity-centric views
- Basic anomaly detection

## Usage Example

```go
// Create Layer 3 manager
config := &layer3.ManagerConfig{
    MetricsBufferSize: 10000,
    LogsBufferSize:    10000,
    TracesBufferSize:  5000,
    EventsBufferSize:  5000,
}

manager := layer3.NewManager(config)
ctx := context.Background()

// Start manager
manager.Start(ctx)

// Collect metrics
metric := &layer3.Metric{
    Name:     "cpu_usage",
    Type:     layer3.MetricTypeGauge,
    Value:    75.5,
    EntityID: "server-1",
    Labels: map[string]string{
        "host": "prod-web-01",
    },
}
manager.CollectMetric(metric)

// Log entries
logEntry := &layer3.LogEntry{
    Level:    layer3.LogLevelInfo,
    Message:  "Request processed successfully",
    EntityID: "api-1",
    Source:   "http-handler",
    Fields: map[string]interface{}{
        "method":      "GET",
        "path":        "/users",
        "status_code": 200,
        "duration_ms": 45,
    },
}
manager.Log(logEntry)

// Record traces
span := manager.GetTraceCollector().StartSpan("process_request", "api-1", "", "")
span.WithTag("http.method", "GET")
span.WithTag("http.url", "/users")
span.Log(map[string]interface{}{
    "event": "database_query",
    "query": "SELECT * FROM users",
})
span.Finish()

// Publish events
event := &layer3.Event{
    Type:     layer3.EventTypeStateChange,
    EntityID: "server-1",
    Source:   "health_monitor",
    Severity: "info",
    Message:  "Server state changed to healthy",
    Data: map[string]interface{}{
        "old_state": "degraded",
        "new_state": "healthy",
    },
}
manager.PublishEvent(event)

// Query metrics
queryReq := &layer3.QueryRequest{
    Type:      "metrics",
    EntityID:  "server-1",
    StartTime: time.Now().Add(-1 * time.Hour),
    EndTime:   time.Now(),
}
result, err := manager.Query(ctx, queryReq)

// Aggregate metrics
aggReq := &layer3.AggregationRequest{
    MetricName: "cpu_usage",
    EntityID:   "server-1",
    StartTime:  time.Now().Add(-1 * time.Hour),
    EndTime:    time.Now(),
    Interval:   5 * time.Minute,
    Function:   layer3.AggFuncAvg,
}
aggResult, err := manager.GetMetricsCollector().Aggregate(ctx, aggReq)

// Correlate data
corrReq := &layer3.CorrelationRequest{
    EntityID:  "api-1",
    StartTime: time.Now().Add(-30 * time.Minute),
    EndTime:   time.Now(),
    Types:     []string{"metrics", "logs", "traces", "events"},
}
corrResult, err := manager.Correlate(ctx, corrReq)

// Subscribe to events
eventChan := manager.GetEventStream().Subscribe("server-1")
go func() {
    for event := range eventChan {
        fmt.Printf("Received event: %s\n", event.Message)
    }
}()
```

## Data Models

### Metric
```go
Metric {
    name: "metric_name"
    type: counter | gauge | histogram | summary
    value: 123.45
    timestamp: time
    entity_id: "entity-id"
    labels: {key: value}
    metadata: {custom fields}
}
```

### Log Entry
```go
LogEntry {
    id: "unique-id"
    timestamp: time
    level: debug | info | warning | error | fatal
    message: "log message"
    entity_id: "entity-id"
    source: "component-name"
    fields: {structured data}
    tags: [tag1, tag2]
}
```

### Trace & Span
```go
Trace {
    trace_id: "trace-id"
    start_time: time
    end_time: time
    duration: duration
    spans: [span1, span2, ...]
    tags: {key: value}
}

Span {
    span_id: "span-id"
    trace_id: "trace-id"
    parent_id: "parent-span-id"
    name: "operation-name"
    entity_id: "entity-id"
    start_time: time
    end_time: time
    duration: duration
    tags: {key: value}
    logs: [{timestamp, fields}]
}
```

### Event
```go
Event {
    id: "event-id"
    type: state_change | health_change | alert | deployment | configuration | error
    timestamp: time
    entity_id: "entity-id"
    source: "source-component"
    severity: "info" | "warning" | "error"
    message: "event message"
    data: {event-specific data}
    tags: [tag1, tag2]
}
```

## Running the Demo

```bash
cd examples
go run layer3_demo.go
```

The demo demonstrates:
1. Metrics collection (counters and gauges)
2. Log aggregation with different levels
3. Distributed tracing
4. Event streaming and subscriptions
5. Querying telemetry data
6. Metric aggregation
7. Data correlation
8. Timeline construction

## Integration with Previous Layers

### With Layer 1 (Adapters)
- Adapters report metrics about their health and performance
- Adapter operations generate logs
- Adapter commands create traces

### With Layer 2 (Discovery & Registration)
- Entity state changes generate events
- Entity health scores derived from metrics
- Entity discovery creates audit logs

Example integration:
```go
// Layer 1 adapter reports metrics
adapter.HealthCheck(ctx) // generates metrics

// Layer 3 collects them
layer3Manager.CollectMetric(&layer3.Metric{
    Name:     "adapter_health",
    EntityID: entity.ID,
    Value:    healthScore,
})

// Layer 2 entity state change generates event
layer3Manager.PublishEvent(&layer3.Event{
    Type:     layer3.EventTypeStateChange,
    EntityID: entity.ID,
    Message:  "Entity state changed",
})
```

## What's Next (Future Versions)

### Storage Backends
- Prometheus for metrics
- Elasticsearch for logs
- Jaeger for traces
- Kafka for event streaming

### Advanced Features
- Metric downsampling and retention policies
- Log parsing and field extraction
- Trace sampling strategies
- Event replay and time-travel debugging

### Processing
- Stream processing for real-time aggregation
- Anomaly detection with ML models
- Automatic baseline learning
- Predictive analytics

### Query Language
- PromQL-compatible metric queries
- Lucene-style log queries
- Trace query DSL
- Cross-telemetry joins

### Visualization
- Real-time dashboards
- Metric graphs and heatmaps
- Log viewers with syntax highlighting
- Trace flamegraphs
- Event timelines

## Design Decisions

1. **Buffered Ingestion**: High-throughput collection with async processing
2. **Pluggable Storage**: Interface-based storage for easy backend swapping
3. **In-Memory Default**: Fast development and testing
4. **Correlation-First**: Built-in correlation across all telemetry types
5. **Entity-Centric**: All data tied to entities from Layer 2
6. **Real-Time Events**: Pub/sub for immediate notification

## Performance Considerations

- Buffer sizes control memory usage vs throughput
- In-memory storage has no persistence (use external backends for production)
- Query performance degrades with large datasets
- Aggregation is computed on-demand (consider pre-aggregation)

## Limitations (V1)

- In-memory storage only (no persistence)
- No data retention policies
- Basic aggregation functions
- Simple anomaly detection
- No authentication/authorization
- No multi-tenancy
- Limited query optimization

## API Reference

### Manager
- `Start(ctx)` - Start all collectors
- `CollectMetric(metric)` - Collect a metric
- `Log(entry)` - Log an entry
- `RecordSpan(span)` - Record a span
- `PublishEvent(event)` - Publish an event
- `Query(ctx, req)` - Query telemetry data
- `Correlate(ctx, req)` - Correlate data

### Metrics Collector
- `Collect(metric)` - Collect metric
- `CollectCounter(name, entityID, value, labels)` - Collect counter
- `CollectGauge(name, entityID, value, labels)` - Collect gauge
- `Query(ctx, req)` - Query metrics
- `QuerySeries(ctx, name, entityID, start, end)` - Query series
- `Aggregate(ctx, req)` - Aggregate metrics

### Log Aggregator
- `Log(entry)` - Log entry
- `LogDebug/Info/Warning/Error(...)` - Convenience methods
- `Query(ctx, req)` - Query logs
- `Search(ctx, query, entityID, start, end, limit)` - Search logs

### Trace Collector
- `RecordSpan(span)` - Record span
- `StartSpan(name, entityID, traceID, parentID)` - Start span builder
- `GetTrace(ctx, traceID)` - Get complete trace
- `Query(ctx, req)` - Query traces

### Event Stream
- `Publish(event)` - Publish event
- `PublishStateChange(...)` - Publish state change
- `PublishAlert(...)` - Publish alert
- `Subscribe(entityID)` - Subscribe to events
- `Unsubscribe(entityID, ch)` - Unsubscribe
- `Query(ctx, req)` - Query events
