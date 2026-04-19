package main

import (
	"context"
	"fmt"
	"time"

	"github.com/realityos/aizo/layer3"
)

func main() {
	fmt.Println("=== RealityOS Layer 3 Demo ===")
	fmt.Println("Telemetry & Observability Layer\n")

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
	fmt.Println("✓ Layer 3 manager started\n")

	// 1. Metrics Collection
	fmt.Println("1. Metrics Collection")
	fmt.Println("---------------------")

	// Collect CPU metrics
	_ = manager.CollectMetric(&layer3.Metric{
		Name:     "cpu_usage",
		Type:     layer3.MetricTypeGauge,
		Value:    75.5,
		EntityID: "server-1",
		Labels: map[string]string{
			"host": "prod-web-01",
			"env":  "production",
		},
	})
	fmt.Println("✓ Collected CPU metric: 75.5%")

	_ = manager.CollectMetric(&layer3.Metric{
		Name:     "memory_usage",
		Type:     layer3.MetricTypeGauge,
		Value:    8192.0,
		EntityID: "server-1",
		Labels: map[string]string{
			"host": "prod-web-01",
			"unit": "MB",
		},
	})
	fmt.Println("✓ Collected memory metric: 8192 MB")

	// Collect request counter
	_ = manager.CollectMetric(&layer3.Metric{
		Name:     "http_requests_total",
		Type:     layer3.MetricTypeCounter,
		Value:    1523.0,
		EntityID: "api-1",
		Labels: map[string]string{
			"method": "GET",
			"path":   "/users",
			"status": "200",
		},
	})
	fmt.Println("✓ Collected request counter: 1523")

	// 2. Log Aggregation
	fmt.Println("\n2. Log Aggregation")
	fmt.Println("------------------")

	_ = manager.Log(&layer3.LogEntry{
		Level:    layer3.LogLevelInfo,
		Message:  "Server started successfully",
		EntityID: "server-1",
		Source:   "main",
		Fields: map[string]interface{}{
			"port":    8080,
			"version": "1.0.0",
		},
	})
	fmt.Println("✓ Logged INFO: Server started")

	_ = manager.Log(&layer3.LogEntry{
		Level:    layer3.LogLevelWarning,
		Message:  "High memory usage detected",
		EntityID: "server-1",
		Source:   "monitor",
		Fields: map[string]interface{}{
			"memory_mb": 8192,
			"threshold": 8000,
		},
	})
	fmt.Println("✓ Logged WARNING: High memory usage")

	_ = manager.Log(&layer3.LogEntry{
		Level:    layer3.LogLevelError,
		Message:  "Database connection failed",
		EntityID: "api-1",
		Source:   "database",
		Fields: map[string]interface{}{
			"error":   "connection timeout",
			"retries": 3,
		},
	})
	fmt.Println("✓ Logged ERROR: Database connection failed")

	// 3. Distributed Tracing
	fmt.Println("\n3. Distributed Tracing")
	fmt.Println("----------------------")

	traceCollector := manager.GetTraceCollector()

	// Start root span
	rootSpan := traceCollector.StartSpan("handle_request", "api-1", "", "")
	rootSpan.WithTag("http.method", "GET")
	rootSpan.WithTag("http.url", "/users/123")
	rootSpan.Log(map[string]interface{}{
		"event": "request_received",
	})

	traceID := rootSpan.GetSpan().TraceID
	fmt.Printf("✓ Started trace: %s\n", traceID[:8])

	// Child span - database query
	time.Sleep(10 * time.Millisecond)
	dbSpan := traceCollector.StartSpan("database_query", "database-1", traceID, rootSpan.GetSpan().SpanID)
	dbSpan.WithTag("db.type", "postgres")
	dbSpan.WithTag("db.statement", "SELECT * FROM users WHERE id = 123")
	time.Sleep(25 * time.Millisecond)
	dbSpan.Finish()
	fmt.Println("  ✓ Span: database_query (25ms)")

	// Child span - cache check
	time.Sleep(5 * time.Millisecond)
	cacheSpan := traceCollector.StartSpan("cache_check", "cache-1", traceID, rootSpan.GetSpan().SpanID)
	cacheSpan.WithTag("cache.hit", "false")
	time.Sleep(5 * time.Millisecond)
	cacheSpan.Finish()
	fmt.Println("  ✓ Span: cache_check (5ms)")

	// Finish root span
	time.Sleep(10 * time.Millisecond)
	rootSpan.Log(map[string]interface{}{
		"event": "response_sent",
	})
	rootSpan.Finish()
	fmt.Println("  ✓ Span: handle_request (50ms)")

	// 4. Event Streaming
	fmt.Println("\n4. Event Streaming")
	fmt.Println("------------------")

	eventStream := manager.GetEventStream()

	// Subscribe to events
	eventChan := eventStream.Subscribe("server-1")
	receivedEvents := make([]string, 0)

	go func() {
		for event := range eventChan {
			receivedEvents = append(receivedEvents, event.Message)
		}
	}()

	// Publish events
	_ = manager.PublishEvent(&layer3.Event{
		Type:     layer3.EventTypeStateChange,
		EntityID: "server-1",
		Source:   "health_monitor",
		Severity: "info",
		Message:  "Server state changed to healthy",
		Data: map[string]interface{}{
			"old_state": "degraded",
			"new_state": "healthy",
		},
	})
	fmt.Println("✓ Published: State change event")

	_ = manager.PublishEvent(&layer3.Event{
		Type:     layer3.EventTypeAlert,
		EntityID: "server-1",
		Source:   "monitor",
		Severity: "warning",
		Message:  "CPU usage above threshold",
		Data: map[string]interface{}{
			"cpu_usage": 85.0,
			"threshold": 80.0,
		},
	})
	fmt.Println("✓ Published: Alert event")

	_ = manager.PublishEvent(&layer3.Event{
		Type:     layer3.EventTypeDeployment,
		EntityID: "server-1",
		Source:   "ci_cd",
		Severity: "info",
		Message:  "New version deployed",
		Data: map[string]interface{}{
			"version": "1.2.0",
			"commit":  "abc123",
		},
	})
	fmt.Println("✓ Published: Deployment event")

	time.Sleep(100 * time.Millisecond)
	fmt.Printf("✓ Received %d events via subscription\n", len(receivedEvents))

	// 5. Querying Telemetry Data
	fmt.Println("\n5. Querying Telemetry Data")
	fmt.Println("--------------------------")

	// Query metrics
	metricsQuery := &layer3.QueryRequest{
		Type:      "metrics",
		EntityID:  "server-1",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	metricsResult, _ := manager.Query(ctx, metricsQuery)
	fmt.Printf("✓ Metrics query: %d results in %v\n", metricsResult.Count, metricsResult.Duration)

	// Query logs
	logsQuery := &layer3.QueryRequest{
		Type:      "logs",
		EntityID:  "server-1",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	logsResult, _ := manager.Query(ctx, logsQuery)
	fmt.Printf("✓ Logs query: %d results in %v\n", logsResult.Count, logsResult.Duration)

	// Query traces
	tracesQuery := &layer3.QueryRequest{
		Type:      "traces",
		EntityID:  "api-1",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	tracesResult, _ := manager.Query(ctx, tracesQuery)
	fmt.Printf("✓ Traces query: %d results in %v\n", tracesResult.Count, tracesResult.Duration)

	// Query events
	eventsQuery := &layer3.QueryRequest{
		Type:      "events",
		EntityID:  "server-1",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	eventsResult, _ := manager.Query(ctx, eventsQuery)
	fmt.Printf("✓ Events query: %d results in %v\n", eventsResult.Count, eventsResult.Duration)

	// 6. Metric Aggregation
	fmt.Println("\n6. Metric Aggregation")
	fmt.Println("---------------------")

	// Collect more metrics for aggregation
	for i := 0; i < 10; i++ {
		_ = manager.CollectMetric(&layer3.Metric{
			Name:     "response_time",
			Type:     layer3.MetricTypeGauge,
			Value:    float64(50 + i*5),
			EntityID: "api-1",
			Labels: map[string]string{
				"endpoint": "/users",
			},
		})
		time.Sleep(1 * time.Millisecond)
	}

	aggReq := &layer3.AggregationRequest{
		MetricName: "response_time",
		EntityID:   "api-1",
		StartTime:  time.Now().Add(-1 * time.Minute),
		EndTime:    time.Now(),
		Interval:   10 * time.Second,
		Function:   layer3.AggFuncAvg,
	}

	aggResult, _ := manager.GetMetricsCollector().Aggregate(ctx, aggReq)
	fmt.Printf("✓ Aggregated response_time (avg): %d series\n", len(aggResult.Series))
	for _, series := range aggResult.Series {
		fmt.Printf("  Series with %d points\n", len(series.Points))
	}

	// 7. Data Correlation
	fmt.Println("\n7. Data Correlation")
	fmt.Println("-------------------")

	corrReq := &layer3.CorrelationRequest{
		EntityID:  "server-1",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Types:     []string{"metrics", "logs", "events"},
	}

	corrResult, _ := manager.Correlate(ctx, corrReq)
	fmt.Printf("✓ Correlated data for server-1:\n")
	fmt.Printf("  Metrics: %d\n", len(corrResult.Metrics))
	fmt.Printf("  Logs: %d\n", len(corrResult.Logs))
	fmt.Printf("  Events: %d\n", len(corrResult.Events))
	fmt.Printf("  Timeline entries: %d\n", len(corrResult.Timeline))

	// 8. Timeline View
	fmt.Println("\n8. Timeline View")
	fmt.Println("----------------")

	if len(corrResult.Timeline) > 0 {
		fmt.Println("Recent timeline entries:")
		count := 5
		if len(corrResult.Timeline) < count {
			count = len(corrResult.Timeline)
		}
		for i := 0; i < count; i++ {
			entry := corrResult.Timeline[i]
			fmt.Printf("  [%s] %s\n",
				entry.Timestamp.Format("15:04:05"),
				entry.Type)
		}
	}

	// 9. Search Logs
	fmt.Println("\n9. Search Logs")
	fmt.Println("--------------")

	searchResults, _ := manager.GetLogAggregator().Search(
		ctx,
		"connection",
		"",
		time.Now().Add(-1*time.Hour),
		time.Now(),
		10,
	)
	fmt.Printf("✓ Search for 'connection': %d results\n", len(searchResults))
	for _, log := range searchResults {
		fmt.Printf("  [%s] %s: %s\n", log.Level, log.EntityID, log.Message)
	}

	// 10. Get Complete Trace
	fmt.Println("\n10. Get Complete Trace")
	fmt.Println("----------------------")

	trace, err := traceCollector.GetTrace(ctx, traceID)
	if err == nil {
		fmt.Printf("✓ Retrieved trace: %s\n", traceID[:8])
		fmt.Printf("  Duration: %v\n", trace.Duration)
		fmt.Printf("  Spans: %d\n", len(trace.Spans))
		for _, span := range trace.Spans {
			fmt.Printf("    - %s (%v)\n", span.Name, span.Duration)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nLayer 3 provides:")
	fmt.Println("  • Metrics collection and aggregation")
	fmt.Println("  • Log aggregation and search")
	fmt.Println("  • Distributed tracing")
	fmt.Println("  • Real-time event streaming")
	fmt.Println("  • Cross-telemetry correlation")
	fmt.Println("  • Timeline reconstruction")
}
