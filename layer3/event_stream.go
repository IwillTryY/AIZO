package layer3

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventStream manages real-time event streaming
type EventStream struct {
	storage     EventStorage
	buffer      chan *Event
	subscribers map[string][]chan *Event
	mu          sync.RWMutex
}

// EventStorage interface for event storage backends
type EventStorage interface {
	Store(ctx context.Context, event *Event) error
	Query(ctx context.Context, req *QueryRequest) ([]Event, error)
}

// NewEventStream creates a new event stream
func NewEventStream(storage EventStorage, bufferSize int) *EventStream {
	return &EventStream{
		storage:     storage,
		buffer:      make(chan *Event, bufferSize),
		subscribers: make(map[string][]chan *Event),
	}
}

// Start starts the event stream
func (s *EventStream) Start(ctx context.Context) {
	go s.processBuffer(ctx)
}

// Publish publishes an event
func (s *EventStream) Publish(event *Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case s.buffer <- event:
		return nil
	default:
		return fmt.Errorf("buffer full, event dropped")
	}
}

// PublishStateChange publishes a state change event
func (s *EventStream) PublishStateChange(entityID, source, message string, data map[string]interface{}) error {
	return s.Publish(&Event{
		Type:     EventTypeStateChange,
		EntityID: entityID,
		Source:   source,
		Severity: "info",
		Message:  message,
		Data:     data,
	})
}

// PublishAlert publishes an alert event
func (s *EventStream) PublishAlert(entityID, source, severity, message string, data map[string]interface{}) error {
	return s.Publish(&Event{
		Type:     EventTypeAlert,
		EntityID: entityID,
		Source:   source,
		Severity: severity,
		Message:  message,
		Data:     data,
	})
}

// Subscribe subscribes to events for an entity
func (s *EventStream) Subscribe(entityID string) chan *Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *Event, 100)
	s.subscribers[entityID] = append(s.subscribers[entityID], ch)

	return ch
}

// Unsubscribe unsubscribes from events
func (s *EventStream) Unsubscribe(entityID string, ch chan *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[entityID]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[entityID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// Query queries events
func (s *EventStream) Query(ctx context.Context, req *QueryRequest) ([]Event, error) {
	return s.storage.Query(ctx, req)
}

// processBuffer processes buffered events
func (s *EventStream) processBuffer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.buffer:
			// Store event
			_ = s.storage.Store(ctx, event)

			// Notify subscribers
			s.notifySubscribers(event)
		}
	}
}

// notifySubscribers notifies all subscribers of an event
func (s *EventStream) notifySubscribers(event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Notify entity-specific subscribers
	if subs, exists := s.subscribers[event.EntityID]; exists {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
				// Subscriber buffer full, skip
			}
		}
	}

	// Notify wildcard subscribers
	if subs, exists := s.subscribers["*"]; exists {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// InMemoryEventStorage is an in-memory implementation of EventStorage
type InMemoryEventStorage struct {
	events []Event
	mu     sync.RWMutex
}

// NewInMemoryEventStorage creates a new in-memory event storage
func NewInMemoryEventStorage() *InMemoryEventStorage {
	return &InMemoryEventStorage{
		events: make([]Event, 0),
	}
}

// Store stores an event
func (s *InMemoryEventStorage) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, *event)
	return nil
}

// Query queries events
func (s *InMemoryEventStorage) Query(ctx context.Context, req *QueryRequest) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]Event, 0)

	for _, event := range s.events {
		if s.matchesQuery(&event, req) {
			results = append(results, event)
		}
	}

	// Apply limit
	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results, nil
}

func (s *InMemoryEventStorage) matchesQuery(event *Event, req *QueryRequest) bool {
	if req.EntityID != "" && event.EntityID != req.EntityID {
		return false
	}

	if !req.StartTime.IsZero() && event.Timestamp.Before(req.StartTime) {
		return false
	}

	if !req.EndTime.IsZero() && event.Timestamp.After(req.EndTime) {
		return false
	}

	// Apply filters
	if req.Filters != nil {
		if eventType, ok := req.Filters["type"].(string); ok && string(event.Type) != eventType {
			return false
		}
		if severity, ok := req.Filters["severity"].(string); ok && event.Severity != severity {
			return false
		}
	}

	return true
}
