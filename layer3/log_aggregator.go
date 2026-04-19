package layer3

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LogAggregator collects and stores logs
type LogAggregator struct {
	storage LogStorage
	buffer  chan *LogEntry
	mu      sync.RWMutex
}

// LogStorage interface for log storage backends
type LogStorage interface {
	Store(ctx context.Context, entry *LogEntry) error
	Query(ctx context.Context, req *QueryRequest) ([]LogEntry, error)
	Search(ctx context.Context, query string, entityID string, start, end time.Time, limit int) ([]LogEntry, error)
}

// NewLogAggregator creates a new log aggregator
func NewLogAggregator(storage LogStorage, bufferSize int) *LogAggregator {
	return &LogAggregator{
		storage: storage,
		buffer:  make(chan *LogEntry, bufferSize),
	}
}

// Start starts the log aggregator
func (a *LogAggregator) Start(ctx context.Context) {
	go a.processBuffer(ctx)
}

// Log logs an entry
func (a *LogAggregator) Log(entry *LogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	select {
	case a.buffer <- entry:
		return nil
	default:
		return fmt.Errorf("buffer full, log dropped")
	}
}

// LogDebug logs a debug message
func (a *LogAggregator) LogDebug(entityID, source, message string, fields map[string]interface{}) error {
	return a.Log(&LogEntry{
		Level:    LogLevelDebug,
		EntityID: entityID,
		Source:   source,
		Message:  message,
		Fields:   fields,
	})
}

// LogInfo logs an info message
func (a *LogAggregator) LogInfo(entityID, source, message string, fields map[string]interface{}) error {
	return a.Log(&LogEntry{
		Level:    LogLevelInfo,
		EntityID: entityID,
		Source:   source,
		Message:  message,
		Fields:   fields,
	})
}

// LogWarning logs a warning message
func (a *LogAggregator) LogWarning(entityID, source, message string, fields map[string]interface{}) error {
	return a.Log(&LogEntry{
		Level:    LogLevelWarning,
		EntityID: entityID,
		Source:   source,
		Message:  message,
		Fields:   fields,
	})
}

// LogError logs an error message
func (a *LogAggregator) LogError(entityID, source, message string, fields map[string]interface{}) error {
	return a.Log(&LogEntry{
		Level:    LogLevelError,
		EntityID: entityID,
		Source:   source,
		Message:  message,
		Fields:   fields,
	})
}

// Query queries logs
func (a *LogAggregator) Query(ctx context.Context, req *QueryRequest) ([]LogEntry, error) {
	return a.storage.Query(ctx, req)
}

// Search searches logs
func (a *LogAggregator) Search(ctx context.Context, query string, entityID string, start, end time.Time, limit int) ([]LogEntry, error) {
	return a.storage.Search(ctx, query, entityID, start, end, limit)
}

// processBuffer processes buffered log entries
func (a *LogAggregator) processBuffer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-a.buffer:
			_ = a.storage.Store(ctx, entry)
		}
	}
}

// InMemoryLogStorage is an in-memory implementation of LogStorage
type InMemoryLogStorage struct {
	logs []LogEntry
	mu   sync.RWMutex
}

// NewInMemoryLogStorage creates a new in-memory log storage
func NewInMemoryLogStorage() *InMemoryLogStorage {
	return &InMemoryLogStorage{
		logs: make([]LogEntry, 0),
	}
}

// Store stores a log entry
func (s *InMemoryLogStorage) Store(ctx context.Context, entry *LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, *entry)
	return nil
}

// Query queries logs
func (s *InMemoryLogStorage) Query(ctx context.Context, req *QueryRequest) ([]LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]LogEntry, 0)

	for _, log := range s.logs {
		if s.matchesQuery(&log, req) {
			results = append(results, log)
		}
	}

	// Apply limit
	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results, nil
}

// Search searches logs
func (s *InMemoryLogStorage) Search(ctx context.Context, query string, entityID string, start, end time.Time, limit int) ([]LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]LogEntry, 0)

	for _, log := range s.logs {
		if entityID != "" && log.EntityID != entityID {
			continue
		}

		if !start.IsZero() && log.Timestamp.Before(start) {
			continue
		}

		if !end.IsZero() && log.Timestamp.After(end) {
			continue
		}

		// Simple text search in message
		if query != "" {
			// In a real implementation, this would be more sophisticated
			if !containsString(log.Message, query) {
				continue
			}
		}

		results = append(results, log)

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

func (s *InMemoryLogStorage) matchesQuery(log *LogEntry, req *QueryRequest) bool {
	if req.EntityID != "" && log.EntityID != req.EntityID {
		return false
	}

	if !req.StartTime.IsZero() && log.Timestamp.Before(req.StartTime) {
		return false
	}

	if !req.EndTime.IsZero() && log.Timestamp.After(req.EndTime) {
		return false
	}

	// Apply filters
	if req.Filters != nil {
		if level, ok := req.Filters["level"].(string); ok && string(log.Level) != level {
			return false
		}
		if source, ok := req.Filters["source"].(string); ok && log.Source != source {
			return false
		}
	}

	return true
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
