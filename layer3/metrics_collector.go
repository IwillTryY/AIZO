package layer3

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector collects and stores time-series metrics
type MetricsCollector struct {
	storage MetricsStorage
	buffer  chan *Metric
	mu      sync.RWMutex
}

// MetricsStorage interface for metric storage backends
type MetricsStorage interface {
	Store(ctx context.Context, metric *Metric) error
	Query(ctx context.Context, req *QueryRequest) ([]Metric, error)
	QuerySeries(ctx context.Context, name, entityID string, start, end time.Time) (*MetricSeries, error)
	Aggregate(ctx context.Context, req *AggregationRequest) (*AggregationResult, error)
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(storage MetricsStorage, bufferSize int) *MetricsCollector {
	return &MetricsCollector{
		storage: storage,
		buffer:  make(chan *Metric, bufferSize),
	}
}

// Start starts the metrics collector
func (c *MetricsCollector) Start(ctx context.Context) {
	go c.processBuffer(ctx)
}

// Collect collects a metric
func (c *MetricsCollector) Collect(metric *Metric) error {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	select {
	case c.buffer <- metric:
		return nil
	default:
		return fmt.Errorf("buffer full, metric dropped")
	}
}

// CollectCounter collects a counter metric
func (c *MetricsCollector) CollectCounter(name, entityID string, value float64, labels map[string]string) error {
	return c.Collect(&Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     value,
		EntityID:  entityID,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

// CollectGauge collects a gauge metric
func (c *MetricsCollector) CollectGauge(name, entityID string, value float64, labels map[string]string) error {
	return c.Collect(&Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		EntityID:  entityID,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

// Query queries metrics
func (c *MetricsCollector) Query(ctx context.Context, req *QueryRequest) ([]Metric, error) {
	return c.storage.Query(ctx, req)
}

// QuerySeries queries a metric series
func (c *MetricsCollector) QuerySeries(ctx context.Context, name, entityID string, start, end time.Time) (*MetricSeries, error) {
	return c.storage.QuerySeries(ctx, name, entityID, start, end)
}

// Aggregate aggregates metrics
func (c *MetricsCollector) Aggregate(ctx context.Context, req *AggregationRequest) (*AggregationResult, error) {
	return c.storage.Aggregate(ctx, req)
}

// processBuffer processes buffered metrics
func (c *MetricsCollector) processBuffer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case metric := <-c.buffer:
			_ = c.storage.Store(ctx, metric)
		}
	}
}

// InMemoryMetricsStorage is an in-memory implementation of MetricsStorage
type InMemoryMetricsStorage struct {
	metrics []Metric
	mu      sync.RWMutex
}

// NewInMemoryMetricsStorage creates a new in-memory metrics storage
func NewInMemoryMetricsStorage() *InMemoryMetricsStorage {
	return &InMemoryMetricsStorage{
		metrics: make([]Metric, 0),
	}
}

// Store stores a metric
func (s *InMemoryMetricsStorage) Store(ctx context.Context, metric *Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics = append(s.metrics, *metric)
	return nil
}

// Query queries metrics
func (s *InMemoryMetricsStorage) Query(ctx context.Context, req *QueryRequest) ([]Metric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]Metric, 0)

	for _, metric := range s.metrics {
		if s.matchesQuery(&metric, req) {
			results = append(results, metric)
		}
	}

	// Apply limit
	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results, nil
}

// QuerySeries queries a metric series
func (s *InMemoryMetricsStorage) QuerySeries(ctx context.Context, name, entityID string, start, end time.Time) (*MetricSeries, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	series := &MetricSeries{
		Name:     name,
		EntityID: entityID,
		Points:   make([]MetricPoint, 0),
	}

	for _, metric := range s.metrics {
		if metric.Name == name && metric.EntityID == entityID &&
			metric.Timestamp.After(start) && metric.Timestamp.Before(end) {
			series.Points = append(series.Points, MetricPoint{
				Timestamp: metric.Timestamp,
				Value:     metric.Value,
			})
			if series.Type == "" {
				series.Type = metric.Type
			}
			if series.Labels == nil && metric.Labels != nil {
				series.Labels = metric.Labels
			}
		}
	}

	return series, nil
}

// Aggregate aggregates metrics
func (s *InMemoryMetricsStorage) Aggregate(ctx context.Context, req *AggregationRequest) (*AggregationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &AggregationResult{
		MetricName: req.MetricName,
		Function:   req.Function,
		Interval:   req.Interval,
		Series:     make([]AggregatedMetricSeries, 0),
	}

	// Group metrics by labels
	groups := make(map[string][]Metric)

	for _, metric := range s.metrics {
		if metric.Name != req.MetricName {
			continue
		}
		if req.EntityID != "" && metric.EntityID != req.EntityID {
			continue
		}
		if metric.Timestamp.Before(req.StartTime) || metric.Timestamp.After(req.EndTime) {
			continue
		}

		// Create group key
		groupKey := ""
		if len(req.GroupBy) > 0 {
			for _, label := range req.GroupBy {
				if val, ok := metric.Labels[label]; ok {
					groupKey += label + "=" + val + ","
				}
			}
		}

		groups[groupKey] = append(groups[groupKey], metric)
	}

	// Aggregate each group
	for _, metrics := range groups {
		series := s.aggregateMetrics(metrics, req)
		result.Series = append(result.Series, series)
	}

	return result, nil
}

func (s *InMemoryMetricsStorage) matchesQuery(metric *Metric, req *QueryRequest) bool {
	if req.EntityID != "" && metric.EntityID != req.EntityID {
		return false
	}

	if !req.StartTime.IsZero() && metric.Timestamp.Before(req.StartTime) {
		return false
	}

	if !req.EndTime.IsZero() && metric.Timestamp.After(req.EndTime) {
		return false
	}

	// Apply filters
	if req.Filters != nil {
		if name, ok := req.Filters["name"].(string); ok && metric.Name != name {
			return false
		}
	}

	return true
}

func (s *InMemoryMetricsStorage) aggregateMetrics(metrics []Metric, req *AggregationRequest) AggregatedMetricSeries {
	series := AggregatedMetricSeries{
		Labels: make(map[string]string),
		Points: make([]MetricPoint, 0),
	}

	if len(metrics) == 0 {
		return series
	}

	// Copy labels from first metric
	if len(req.GroupBy) > 0 {
		for _, label := range req.GroupBy {
			if val, ok := metrics[0].Labels[label]; ok {
				series.Labels[label] = val
			}
		}
	}

	// Group by time intervals
	intervals := make(map[int64][]float64)

	for _, metric := range metrics {
		intervalStart := metric.Timestamp.Truncate(req.Interval).Unix()
		intervals[intervalStart] = append(intervals[intervalStart], metric.Value)
	}

	// Calculate aggregation for each interval
	for timestamp, values := range intervals {
		point := MetricPoint{
			Timestamp: time.Unix(timestamp, 0),
			Value:     s.calculateAggregation(values, req.Function),
		}
		series.Points = append(series.Points, point)
	}

	return series
}

func (s *InMemoryMetricsStorage) calculateAggregation(values []float64, function AggregationFunc) float64 {
	if len(values) == 0 {
		return 0
	}

	switch function {
	case AggFuncSum:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum

	case AggFuncAvg:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))

	case AggFuncMin:
		min := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min

	case AggFuncMax:
		max := values[0]
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max

	case AggFuncCount:
		return float64(len(values))

	default:
		return 0
	}
}
