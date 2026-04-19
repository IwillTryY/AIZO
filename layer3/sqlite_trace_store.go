package layer3

import (
	"context"
	"database/sql"
	"encoding/json"
)

// SQLiteTraceStorage implements TraceStorage backed by SQLite
type SQLiteTraceStorage struct {
	db *sql.DB
}

// NewSQLiteTraceStorage creates a new SQLite-backed trace storage
func NewSQLiteTraceStorage(db *sql.DB) *SQLiteTraceStorage {
	return &SQLiteTraceStorage{db: db}
}

// StoreSpan stores a span in SQLite
func (s *SQLiteTraceStorage) StoreSpan(ctx context.Context, span *Span) error {
	attrs, _ := json.Marshal(span.Tags)

	// Ensure trace exists
	s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO traces (id, name, start_time, entity_id) VALUES (?, ?, ?, ?)`,
		span.TraceID, span.Name, span.StartTime, span.EntityID,
	)

	// Update trace end time
	s.db.ExecContext(ctx,
		`UPDATE traces SET end_time = ?, duration_ms = ? WHERE id = ? AND (end_time IS NULL OR end_time < ?)`,
		span.EndTime, float64(span.Duration.Milliseconds()), span.TraceID, span.EndTime,
	)

	// Insert span
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO spans (id, trace_id, parent_id, name, start_time, end_time, duration_ms, attributes) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		span.SpanID, span.TraceID, span.ParentID, span.Name, span.StartTime, span.EndTime, float64(span.Duration.Milliseconds()), string(attrs),
	)
	return err
}

// GetTrace retrieves a complete trace with all spans
func (s *SQLiteTraceStorage) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	// Get trace
	var trace Trace
	var durationMs sql.NullFloat64
	var metadata sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, start_time, end_time, duration_ms, metadata FROM traces WHERE id = ?`, traceID,
	).Scan(&trace.TraceID, &trace.Tags, &trace.StartTime, &trace.EndTime, &durationMs, &metadata)
	if err != nil {
		return nil, err
	}

	// Get spans
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, trace_id, parent_id, name, start_time, end_time, duration_ms, attributes FROM spans WHERE trace_id = ? ORDER BY start_time ASC`, traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trace.Spans = make([]Span, 0)
	for rows.Next() {
		var span Span
		var parentID sql.NullString
		var durMs sql.NullFloat64
		var attrsJSON sql.NullString
		if err := rows.Scan(&span.SpanID, &span.TraceID, &parentID, &span.Name, &span.StartTime, &span.EndTime, &durMs, &attrsJSON); err != nil {
			continue
		}
		if parentID.Valid {
			span.ParentID = parentID.String
		}
		if attrsJSON.Valid {
			json.Unmarshal([]byte(attrsJSON.String), &span.Tags)
		}
		trace.Spans = append(trace.Spans, span)
	}

	return &trace, nil
}

// Query queries traces from SQLite
func (s *SQLiteTraceStorage) Query(ctx context.Context, req *QueryRequest) ([]Trace, error) {
	query := "SELECT id, name, start_time, end_time, duration_ms, entity_id FROM traces WHERE 1=1"
	args := make([]interface{}, 0)

	if req.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, req.EntityID)
	}
	if !req.StartTime.IsZero() {
		query += " AND start_time >= ?"
		args = append(args, req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query += " AND start_time <= ?"
		args = append(args, req.EndTime)
	}

	query += " ORDER BY start_time DESC"
	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	traces := make([]Trace, 0)
	for rows.Next() {
		var t Trace
		var name string
		var durMs sql.NullFloat64
		var entityID sql.NullString
		if err := rows.Scan(&t.TraceID, &name, &t.StartTime, &t.EndTime, &durMs, &entityID); err != nil {
			continue
		}
		t.Tags = map[string]string{"name": name}
		traces = append(traces, t)
	}
	return traces, nil
}
