package layer3

import (
	"context"
	"database/sql"
	"encoding/json"
)

// SQLiteEventStorage implements EventStorage backed by SQLite
type SQLiteEventStorage struct {
	db *sql.DB
}

// NewSQLiteEventStorage creates a new SQLite-backed event storage
func NewSQLiteEventStorage(db *sql.DB) *SQLiteEventStorage {
	return &SQLiteEventStorage{db: db}
}

// Store stores an event in SQLite
func (s *SQLiteEventStorage) Store(ctx context.Context, event *Event) error {
	data, _ := json.Marshal(event.Data)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, timestamp, type, source, entity_id, severity, data) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.Timestamp, string(event.Type), event.Source, event.EntityID, event.Severity, string(data),
	)
	return err
}

// Query queries events from SQLite
func (s *SQLiteEventStorage) Query(ctx context.Context, req *QueryRequest) ([]Event, error) {
	query := "SELECT id, timestamp, type, source, entity_id, severity, data FROM events WHERE 1=1"
	args := make([]interface{}, 0)

	if req.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, req.EntityID)
	}
	if !req.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, req.EndTime)
	}
	if eventType, ok := req.Filters["type"]; ok {
		query += " AND type = ?"
		args = append(args, eventType)
	}
	if severity, ok := req.Filters["severity"]; ok {
		query += " AND severity = ?"
		args = append(args, severity)
	}

	query += " ORDER BY timestamp DESC"
	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	} else {
		query += " LIMIT 1000"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var e Event
		var etype string
		var dataJSON sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &etype, &e.Source, &e.EntityID, &e.Severity, &dataJSON); err != nil {
			continue
		}
		e.Type = EventType(etype)
		if dataJSON.Valid {
			json.Unmarshal([]byte(dataJSON.String), &e.Data)
		}
		events = append(events, e)
	}
	return events, nil
}
