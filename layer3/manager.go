package layer3

import (
	"context"
	"database/sql"
	"time"
)

// Manager orchestrates all Layer 3 components
type Manager struct {
	metricsCollector  *MetricsCollector
	logAggregator     *LogAggregator
	traceCollector    *TraceCollector
	eventStream       *EventStream
	correlationEngine *CorrelationEngine
}

// ManagerConfig configures the Layer 3 manager
type ManagerConfig struct {
	MetricsBufferSize int
	LogsBufferSize    int
	TracesBufferSize  int
	EventsBufferSize  int
	DB                *sql.DB // Optional: if provided, use SQLite storage instead of in-memory
}

// NewManager creates a new Layer 3 manager
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = &ManagerConfig{
			MetricsBufferSize: 10000,
			LogsBufferSize:    10000,
			TracesBufferSize:  5000,
			EventsBufferSize:  5000,
		}
	}

	// Create storage backends - use SQLite if DB provided, otherwise in-memory
	var metricsStorage MetricsStorage
	var logsStorage LogStorage
	var tracesStorage TraceStorage
	var eventsStorage EventStorage

	if config.DB != nil {
		// Use SQLite storage
		metricsStorage = NewSQLiteMetricsStorage(config.DB)
		logsStorage = NewSQLiteLogStorage(config.DB)
		tracesStorage = NewSQLiteTraceStorage(config.DB)
		eventsStorage = NewSQLiteEventStorage(config.DB)
	} else {
		// Use in-memory storage
		metricsStorage = NewInMemoryMetricsStorage()
		logsStorage = NewInMemoryLogStorage()
		tracesStorage = NewInMemoryTraceStorage()
		eventsStorage = NewInMemoryEventStorage()
	}

	// Create collectors
	metricsCollector := NewMetricsCollector(metricsStorage, config.MetricsBufferSize)
	logAggregator := NewLogAggregator(logsStorage, config.LogsBufferSize)
	traceCollector := NewTraceCollector(tracesStorage, config.TracesBufferSize)
	eventStream := NewEventStream(eventsStorage, config.EventsBufferSize)

	// Create correlation engine
	correlationEngine := NewCorrelationEngine(
		metricsCollector,
		logAggregator,
		traceCollector,
		eventStream,
	)

	return &Manager{
		metricsCollector:  metricsCollector,
		logAggregator:     logAggregator,
		traceCollector:    traceCollector,
		eventStream:       eventStream,
		correlationEngine: correlationEngine,
	}
}

// Start starts the Layer 3 manager
func (m *Manager) Start(ctx context.Context) error {
	m.metricsCollector.Start(ctx)
	m.logAggregator.Start(ctx)
	m.traceCollector.Start(ctx)
	m.eventStream.Start(ctx)

	return nil
}

// GetMetricsCollector returns the metrics collector
func (m *Manager) GetMetricsCollector() *MetricsCollector {
	return m.metricsCollector
}

// GetLogAggregator returns the log aggregator
func (m *Manager) GetLogAggregator() *LogAggregator {
	return m.logAggregator
}

// GetTraceCollector returns the trace collector
func (m *Manager) GetTraceCollector() *TraceCollector {
	return m.traceCollector
}

// GetEventStream returns the event stream
func (m *Manager) GetEventStream() *EventStream {
	return m.eventStream
}

// GetCorrelationEngine returns the correlation engine
func (m *Manager) GetCorrelationEngine() *CorrelationEngine {
	return m.correlationEngine
}

// CollectMetric collects a metric
func (m *Manager) CollectMetric(metric *Metric) error {
	return m.metricsCollector.Collect(metric)
}

// Log logs an entry
func (m *Manager) Log(entry *LogEntry) error {
	return m.logAggregator.Log(entry)
}

// RecordSpan records a span
func (m *Manager) RecordSpan(span *Span) error {
	return m.traceCollector.RecordSpan(span)
}

// PublishEvent publishes an event
func (m *Manager) PublishEvent(event *Event) error {
	return m.eventStream.Publish(event)
}

// Query queries telemetry data
func (m *Manager) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	startTime := time.Now()
	result := &QueryResult{
		Type: req.Type,
	}

	switch req.Type {
	case "metrics":
		metrics, err := m.metricsCollector.Query(ctx, req)
		if err != nil {
			return nil, err
		}
		result.Metrics = metrics
		result.Count = len(metrics)
		result.TotalCount = len(metrics)

	case "logs":
		logs, err := m.logAggregator.Query(ctx, req)
		if err != nil {
			return nil, err
		}
		result.Logs = logs
		result.Count = len(logs)
		result.TotalCount = len(logs)

	case "traces":
		traces, err := m.traceCollector.Query(ctx, req)
		if err != nil {
			return nil, err
		}
		result.Traces = traces
		result.Count = len(traces)
		result.TotalCount = len(traces)

	case "events":
		events, err := m.eventStream.Query(ctx, req)
		if err != nil {
			return nil, err
		}
		result.Events = events
		result.Count = len(events)
		result.TotalCount = len(events)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// Correlate correlates telemetry data
func (m *Manager) Correlate(ctx context.Context, req *CorrelationRequest) (*CorrelationResult, error) {
	return m.correlationEngine.Correlate(ctx, req)
}

// GetStats returns Layer 3 statistics
func (m *Manager) GetStats() ManagerStats {
	return ManagerStats{
		// In a real implementation, would track actual stats
		MetricsCollected: 0,
		LogsCollected:    0,
		TracesCollected:  0,
		EventsPublished:  0,
	}
}

// ManagerStats contains Layer 3 statistics
type ManagerStats struct {
	MetricsCollected int64
	LogsCollected    int64
	TracesCollected  int64
	EventsPublished  int64
}
