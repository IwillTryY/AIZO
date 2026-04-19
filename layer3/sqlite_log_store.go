package layer3

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// SQLiteLogStorage implements LogStorage backed by SQLite
type SQLiteLogStorage struct {
	db *sql.DB
}

// NewSQLiteLogStorage creates a new SQLite-backed log storage
func NewSQLiteLogStorage(db *sql.DB) *SQLiteLogStorage {
	return &SQLiteLogStorage{db: db}
}

// Store stores a log entry in SQLite
func (s *SQLiteLogStorage) Store(ctx context.Context, entry *LogEntry) error {
	fields, _ := json.Marshal(entry.Fields)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO logs (id, timestamp, level, message, source, entity_id, fields) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp, string(entry.Level), entry.Message, entry.Source, entry.EntityID, string(fields),
	)
	return err
}

// Query queries logs from SQLite
func (s *SQLiteLogStorage) Query(ctx context.Context, req *QueryRequest) ([]LogEntry, error) {
	query := "SELECT id, timestamp, level, message, source, entity_id, fields FROM logs WHERE 1=1"
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
	if level, ok := req.Filters["level"]; ok {
		query += " AND level = ?"
		args = append(args, level)
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

	return scanLogs(rows)
}

// Search searches logs by message content
func (s *SQLiteLogStorage) Search(ctx context.Context, queryStr string, entityID string, start, end time.Time, limit int) ([]LogEntry, error) {
	query := "SELECT id, timestamp, level, message, source, entity_id, fields FROM logs WHERE message LIKE ?"
	args := []interface{}{"%" + queryStr + "%"}

	if entityID != "" {
		query += " AND entity_id = ?"
		args = append(args, entityID)
	}
	if !start.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, start)
	}
	if !end.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, end)
	}

	query += " ORDER BY timestamp DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanLogs(rows)
}

func scanLogs(rows *sql.Rows) ([]LogEntry, error) {
	logs := make([]LogEntry, 0)
	for rows.Next() {
		var e LogEntry
		var level string
		var fieldsJSON sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &level, &e.Message, &e.Source, &e.EntityID, &fieldsJSON); err != nil {
			continue
		}
		e.Level = LogLevel(level)
		if fieldsJSON.Valid {
			json.Unmarshal([]byte(fieldsJSON.String), &e.Fields)
		}
		logs = append(logs, e)
	}
	return logs, nil
}
