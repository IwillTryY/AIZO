package layer3

import (
	"context"
	"sort"
	"time"
)

// CorrelationEngine correlates metrics, logs, traces, and events
type CorrelationEngine struct {
	metricsCollector *MetricsCollector
	logAggregator    *LogAggregator
	traceCollector   *TraceCollector
	eventStream      *EventStream
}

// NewCorrelationEngine creates a new correlation engine
func NewCorrelationEngine(
	metrics *MetricsCollector,
	logs *LogAggregator,
	traces *TraceCollector,
	events *EventStream,
) *CorrelationEngine {
	return &CorrelationEngine{
		metricsCollector: metrics,
		logAggregator:    logs,
		traceCollector:   traces,
		eventStream:      events,
	}
}

// Correlate correlates all telemetry data for an entity
func (e *CorrelationEngine) Correlate(ctx context.Context, req *CorrelationRequest) (*CorrelationResult, error) {
	result := &CorrelationResult{
		EntityID:  req.EntityID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Timeline:  make([]TimelineEntry, 0),
	}

	queryReq := &QueryRequest{
		EntityID:  req.EntityID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	// Collect metrics if requested
	if contains(req.Types, "metrics") && e.metricsCollector != nil {
		metrics, err := e.metricsCollector.Query(ctx, queryReq)
		if err == nil {
			result.Metrics = metrics
			for _, metric := range metrics {
				result.Timeline = append(result.Timeline, TimelineEntry{
					Timestamp: metric.Timestamp,
					Type:      "metric",
					Data:      metric,
				})
			}
		}
	}

	// Collect logs if requested
	if contains(req.Types, "logs") && e.logAggregator != nil {
		logs, err := e.logAggregator.Query(ctx, queryReq)
		if err == nil {
			result.Logs = logs
			for _, log := range logs {
				result.Timeline = append(result.Timeline, TimelineEntry{
					Timestamp: log.Timestamp,
					Type:      "log",
					Data:      log,
				})
			}
		}
	}

	// Collect traces if requested
	if contains(req.Types, "traces") && e.traceCollector != nil {
		traces, err := e.traceCollector.Query(ctx, queryReq)
		if err == nil {
			result.Traces = traces
			for _, trace := range traces {
				result.Timeline = append(result.Timeline, TimelineEntry{
					Timestamp: trace.StartTime,
					Type:      "trace",
					Data:      trace,
				})
			}
		}
	}

	// Collect events if requested
	if contains(req.Types, "events") && e.eventStream != nil {
		events, err := e.eventStream.Query(ctx, queryReq)
		if err == nil {
			result.Events = events
			for _, event := range events {
				result.Timeline = append(result.Timeline, TimelineEntry{
					Timestamp: event.Timestamp,
					Type:      "event",
					Data:      event,
				})
			}
		}
	}

	// Sort timeline by timestamp
	sort.Slice(result.Timeline, func(i, j int) bool {
		return result.Timeline[i].Timestamp.Before(result.Timeline[j].Timestamp)
	})

	return result, nil
}

// FindAnomalies finds anomalies in correlated data
func (e *CorrelationEngine) FindAnomalies(ctx context.Context, entityID string, start, end time.Time) ([]Anomaly, error) {
	// Placeholder for anomaly detection
	// In a real implementation, this would use ML/statistical methods
	anomalies := make([]Anomaly, 0)

	// Get metrics
	metrics, err := e.metricsCollector.Query(ctx, &QueryRequest{
		EntityID:  entityID,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		return anomalies, err
	}

	// Simple threshold-based anomaly detection
	for _, metric := range metrics {
		if metric.Value > 90.0 && metric.Name == "cpu_usage" {
			anomalies = append(anomalies, Anomaly{
				Timestamp: metric.Timestamp,
				Type:      "high_cpu",
				Severity:  "warning",
				Message:   "CPU usage above 90%",
				EntityID:  entityID,
				Data: map[string]interface{}{
					"metric": metric.Name,
					"value":  metric.Value,
				},
			})
		}
	}

	return anomalies, nil
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	EntityID  string                 `json:"entity_id"`
	Data      map[string]interface{} `json:"data"`
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
