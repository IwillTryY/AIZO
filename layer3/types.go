package layer3

import (
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric data point
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	EntityID  string                 `json:"entity_id"`
	Labels    map[string]string      `json:"labels"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricSeries represents a time series of metrics
type MetricSeries struct {
	Name     string            `json:"name"`
	Type     MetricType        `json:"type"`
	EntityID string            `json:"entity_id"`
	Labels   map[string]string `json:"labels"`
	Points   []MetricPoint     `json:"points"`
}

// MetricPoint represents a single point in a time series
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	EntityID  string                 `json:"entity_id"`
	Source    string                 `json:"source"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// Trace represents a distributed trace
type Trace struct {
	TraceID   string      `json:"trace_id"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Spans     []Span      `json:"spans"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// Span represents a single span in a trace
type Span struct {
	SpanID    string                 `json:"span_id"`
	TraceID   string                 `json:"trace_id"`
	ParentID  string                 `json:"parent_id,omitempty"`
	Name      string                 `json:"name"`
	EntityID  string                 `json:"entity_id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Logs      []SpanLog              `json:"logs,omitempty"`
}

// SpanLog represents a log entry within a span
type SpanLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// EventType represents the type of event
type EventType string

const (
	EventTypeStateChange   EventType = "state_change"
	EventTypeHealthChange  EventType = "health_change"
	EventTypeAlert         EventType = "alert"
	EventTypeDeployment    EventType = "deployment"
	EventTypeConfiguration EventType = "configuration"
	EventTypeError         EventType = "error"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	EntityID  string                 `json:"entity_id"`
	Source    string                 `json:"source"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// QueryRequest represents a query for telemetry data
type QueryRequest struct {
	Type      string                 `json:"type"` // metrics, logs, traces, events
	EntityID  string                 `json:"entity_id,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
}

// QueryResult represents the result of a query
type QueryResult struct {
	Type       string        `json:"type"`
	Count      int           `json:"count"`
	Metrics    []Metric      `json:"metrics,omitempty"`
	Logs       []LogEntry    `json:"logs,omitempty"`
	Traces     []Trace       `json:"traces,omitempty"`
	Events     []Event       `json:"events,omitempty"`
	Duration   time.Duration `json:"duration"`
	TotalCount int           `json:"total_count"`
}

// AggregationRequest represents a request for aggregated data
type AggregationRequest struct {
	MetricName string            `json:"metric_name"`
	EntityID   string            `json:"entity_id,omitempty"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Interval   time.Duration     `json:"interval"`
	Function   AggregationFunc   `json:"function"`
	GroupBy    []string          `json:"group_by,omitempty"`
	Filters    map[string]string `json:"filters,omitempty"`
}

// AggregationFunc represents an aggregation function
type AggregationFunc string

const (
	AggFuncSum   AggregationFunc = "sum"
	AggFuncAvg   AggregationFunc = "avg"
	AggFuncMin   AggregationFunc = "min"
	AggFuncMax   AggregationFunc = "max"
	AggFuncCount AggregationFunc = "count"
	AggFuncP50   AggregationFunc = "p50"
	AggFuncP95   AggregationFunc = "p95"
	AggFuncP99   AggregationFunc = "p99"
)

// AggregationResult represents aggregated metric data
type AggregationResult struct {
	MetricName string                   `json:"metric_name"`
	Function   AggregationFunc          `json:"function"`
	Interval   time.Duration            `json:"interval"`
	Series     []AggregatedMetricSeries `json:"series"`
}

// AggregatedMetricSeries represents a single aggregated series
type AggregatedMetricSeries struct {
	Labels map[string]string `json:"labels"`
	Points []MetricPoint     `json:"points"`
}

// CorrelationRequest represents a request to correlate data
type CorrelationRequest struct {
	EntityID  string    `json:"entity_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Types     []string  `json:"types"` // metrics, logs, traces, events
}

// CorrelationResult represents correlated telemetry data
type CorrelationResult struct {
	EntityID  string     `json:"entity_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	Metrics   []Metric   `json:"metrics,omitempty"`
	Logs      []LogEntry `json:"logs,omitempty"`
	Traces    []Trace    `json:"traces,omitempty"`
	Events    []Event    `json:"events,omitempty"`
	Timeline  []TimelineEntry `json:"timeline"`
}

// TimelineEntry represents a single entry in a correlated timeline
type TimelineEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Type      string      `json:"type"` // metric, log, trace, event
	Data      interface{} `json:"data"`
}
