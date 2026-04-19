package layer3

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TraceCollector collects and stores distributed traces
type TraceCollector struct {
	storage TraceStorage
	buffer  chan *Span
	mu      sync.RWMutex
}

// TraceStorage interface for trace storage backends
type TraceStorage interface {
	StoreSpan(ctx context.Context, span *Span) error
	GetTrace(ctx context.Context, traceID string) (*Trace, error)
	Query(ctx context.Context, req *QueryRequest) ([]Trace, error)
}

// NewTraceCollector creates a new trace collector
func NewTraceCollector(storage TraceStorage, bufferSize int) *TraceCollector {
	return &TraceCollector{
		storage: storage,
		buffer:  make(chan *Span, bufferSize),
	}
}

// Start starts the trace collector
func (c *TraceCollector) Start(ctx context.Context) {
	go c.processBuffer(ctx)
}

// RecordSpan records a span
func (c *TraceCollector) RecordSpan(span *Span) error {
	if span.SpanID == "" {
		span.SpanID = uuid.New().String()
	}
	if span.TraceID == "" {
		span.TraceID = uuid.New().String()
	}
	if span.StartTime.IsZero() {
		span.StartTime = time.Now()
	}
	if span.EndTime.IsZero() {
		span.EndTime = time.Now()
	}
	span.Duration = span.EndTime.Sub(span.StartTime)

	select {
	case c.buffer <- span:
		return nil
	default:
		return fmt.Errorf("buffer full, span dropped")
	}
}

// StartSpan starts a new span
func (c *TraceCollector) StartSpan(name, entityID string, traceID, parentID string) *SpanBuilder {
	if traceID == "" {
		traceID = uuid.New().String()
	}

	return &SpanBuilder{
		collector: c,
		span: &Span{
			SpanID:    uuid.New().String(),
			TraceID:   traceID,
			ParentID:  parentID,
			Name:      name,
			EntityID:  entityID,
			StartTime: time.Now(),
			Tags:      make(map[string]string),
			Logs:      make([]SpanLog, 0),
		},
	}
}

// GetTrace retrieves a complete trace
func (c *TraceCollector) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	return c.storage.GetTrace(ctx, traceID)
}

// Query queries traces
func (c *TraceCollector) Query(ctx context.Context, req *QueryRequest) ([]Trace, error) {
	return c.storage.Query(ctx, req)
}

// processBuffer processes buffered spans
func (c *TraceCollector) processBuffer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case span := <-c.buffer:
			_ = c.storage.StoreSpan(ctx, span)
		}
	}
}

// SpanBuilder helps build spans
type SpanBuilder struct {
	collector *TraceCollector
	span      *Span
}

// WithTag adds a tag to the span
func (b *SpanBuilder) WithTag(key, value string) *SpanBuilder {
	b.span.Tags[key] = value
	return b
}

// Log adds a log entry to the span
func (b *SpanBuilder) Log(fields map[string]interface{}) *SpanBuilder {
	b.span.Logs = append(b.span.Logs, SpanLog{
		Timestamp: time.Now(),
		Fields:    fields,
	})
	return b
}

// Finish finishes the span and records it
func (b *SpanBuilder) Finish() error {
	b.span.EndTime = time.Now()
	b.span.Duration = b.span.EndTime.Sub(b.span.StartTime)
	return b.collector.RecordSpan(b.span)
}

// GetSpan returns the span
func (b *SpanBuilder) GetSpan() *Span {
	return b.span
}

// InMemoryTraceStorage is an in-memory implementation of TraceStorage
type InMemoryTraceStorage struct {
	spans  map[string][]*Span // traceID -> spans
	traces map[string]*Trace  // traceID -> trace
	mu     sync.RWMutex
}

// NewInMemoryTraceStorage creates a new in-memory trace storage
func NewInMemoryTraceStorage() *InMemoryTraceStorage {
	return &InMemoryTraceStorage{
		spans:  make(map[string][]*Span),
		traces: make(map[string]*Trace),
	}
}

// StoreSpan stores a span
func (s *InMemoryTraceStorage) StoreSpan(ctx context.Context, span *Span) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.spans[span.TraceID] = append(s.spans[span.TraceID], span)

	// Rebuild trace
	s.rebuildTrace(span.TraceID)

	return nil
}

// GetTrace retrieves a complete trace
func (s *InMemoryTraceStorage) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trace, exists := s.traces[traceID]
	if !exists {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	return trace, nil
}

// Query queries traces
func (s *InMemoryTraceStorage) Query(ctx context.Context, req *QueryRequest) ([]Trace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]Trace, 0)

	for _, trace := range s.traces {
		if s.matchesQuery(trace, req) {
			results = append(results, *trace)
		}
	}

	// Apply limit
	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results, nil
}

func (s *InMemoryTraceStorage) rebuildTrace(traceID string) {
	spans := s.spans[traceID]
	if len(spans) == 0 {
		return
	}

	trace := &Trace{
		TraceID: traceID,
		Spans:   make([]Span, len(spans)),
		Tags:    make(map[string]string),
	}

	// Copy spans
	for i, span := range spans {
		trace.Spans[i] = *span

		// Update trace timing
		if trace.StartTime.IsZero() || span.StartTime.Before(trace.StartTime) {
			trace.StartTime = span.StartTime
		}
		if trace.EndTime.IsZero() || span.EndTime.After(trace.EndTime) {
			trace.EndTime = span.EndTime
		}
	}

	trace.Duration = trace.EndTime.Sub(trace.StartTime)

	s.traces[traceID] = trace
}

func (s *InMemoryTraceStorage) matchesQuery(trace *Trace, req *QueryRequest) bool {
	if req.EntityID != "" {
		found := false
		for _, span := range trace.Spans {
			if span.EntityID == req.EntityID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if !req.StartTime.IsZero() && trace.StartTime.Before(req.StartTime) {
		return false
	}

	if !req.EndTime.IsZero() && trace.EndTime.After(req.EndTime) {
		return false
	}

	return true
}
