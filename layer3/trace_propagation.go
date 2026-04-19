package layer3

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
)

// TraceContext carries distributed trace information across service boundaries
type TraceContext struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id"`
	Baggage      map[string]string `json:"baggage"`
}

// NewTraceContext creates a new root trace context
func NewTraceContext() *TraceContext {
	return &TraceContext{
		TraceID: uuid.New().String(),
		SpanID:  uuid.New().String()[:16],
		Baggage: make(map[string]string),
	}
}

// ChildSpan creates a child span context
func (tc *TraceContext) ChildSpan() *TraceContext {
	return &TraceContext{
		TraceID:      tc.TraceID,
		SpanID:       uuid.New().String()[:16],
		ParentSpanID: tc.SpanID,
		Baggage:      tc.Baggage,
	}
}

// InjectHeaders returns trace context as a string map (for mesh messages)
func (tc *TraceContext) InjectHeaders() map[string]string {
	headers := map[string]string{
		"x-trace-id":  tc.TraceID,
		"x-span-id":   tc.SpanID,
		"x-parent-id": tc.ParentSpanID,
	}
	for k, v := range tc.Baggage {
		headers["x-baggage-"+k] = v
	}
	return headers
}

// ExtractTrace extracts a trace context from headers
func ExtractTrace(headers map[string]string) *TraceContext {
	tc := &TraceContext{
		TraceID:      headers["x-trace-id"],
		SpanID:       headers["x-span-id"],
		ParentSpanID: headers["x-parent-id"],
		Baggage:      make(map[string]string),
	}
	for k, v := range headers {
		if len(k) > 10 && k[:10] == "x-baggage-" {
			tc.Baggage[k[10:]] = v
		}
	}
	if tc.TraceID == "" {
		return NewTraceContext()
	}
	return tc
}

// traceContextKey is the context key for trace propagation
type traceContextKey struct{}

// WithTrace adds trace context to a Go context
func WithTrace(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey{}, tc)
}

// TraceFromContext extracts trace context from a Go context
func TraceFromContext(ctx context.Context) *TraceContext {
	tc, ok := ctx.Value(traceContextKey{}).(*TraceContext)
	if !ok {
		return NewTraceContext()
	}
	return tc
}

// AnomalyResult represents a detected anomaly
type AnomalyResult struct {
	EntityID  string    `json:"entity_id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Mean      float64   `json:"mean"`
	StdDev    float64   `json:"std_dev"`
	ZScore    float64   `json:"z_score"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"` // low, medium, high, critical
}

// DetectAnomalies performs z-score anomaly detection on recent metrics
// for a given entity. Returns anomalies where |z-score| > threshold.
func DetectAnomalies(storage MetricsStorage, entityID string, window time.Duration, threshold float64) ([]AnomalyResult, error) {
	if threshold == 0 {
		threshold = 2.5 // default: 2.5 standard deviations
	}

	end := time.Now()
	start := end.Add(-window)

	metrics, err := storage.Query(context.Background(), &QueryRequest{
		EntityID:  entityID,
		StartTime: start,
		EndTime:   end,
		Limit:     10000,
		Filters:   make(map[string]interface{}),
	})
	if err != nil {
		return nil, err
	}

	// Group by metric name
	groups := make(map[string][]float64)
	latest := make(map[string]Metric)
	for _, m := range metrics {
		groups[m.Name] = append(groups[m.Name], m.Value)
		if existing, ok := latest[m.Name]; !ok || m.Timestamp.After(existing.Timestamp) {
			latest[m.Name] = m
		}
	}

	anomalies := make([]AnomalyResult, 0)
	for name, values := range groups {
		if len(values) < 10 {
			continue // need enough data points
		}

		mean, stddev := meanStdDev(values)
		if stddev == 0 {
			continue
		}

		latestVal := latest[name].Value
		zScore := (latestVal - mean) / stddev

		if math.Abs(zScore) > threshold {
			severity := "low"
			if math.Abs(zScore) > 4 {
				severity = "critical"
			} else if math.Abs(zScore) > 3.5 {
				severity = "high"
			} else if math.Abs(zScore) > 3 {
				severity = "medium"
			}

			anomalies = append(anomalies, AnomalyResult{
				EntityID:  entityID,
				Metric:    name,
				Value:     latestVal,
				Mean:      mean,
				StdDev:    stddev,
				ZScore:    zScore,
				Timestamp: latest[name].Timestamp,
				Severity:  severity,
			})
		}
	}

	return anomalies, nil
}

func meanStdDev(values []float64) (float64, float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / n

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= n

	return mean, math.Sqrt(variance)
}
