package storage

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditEntry represents a single audit trail entry
type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	TenantID  string    `json:"tenant_id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail"`
	Layer     string    `json:"layer"`
}

// AuditFilter filters audit trail queries
type AuditFilter struct {
	TenantID  string
	Actor     string
	Action    string
	Resource  string
	Layer     string
	Since     time.Time
	Until     time.Time
	Limit     int
	Offset    int
}

// AuditStore manages the centralized audit trail
type AuditStore struct {
	db *DB
}

// NewAuditStore creates a new audit store
func NewAuditStore(db *DB) *AuditStore {
	return &AuditStore{db: db}
}

// Record writes an audit entry
func (s *AuditStore) Record(entry AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.TenantID == "" {
		entry.TenantID = "default"
	}

	_, err := s.db.Exec(
		`INSERT INTO audit_trail (id, timestamp, tenant_id, actor, action, resource, detail, layer)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp, entry.TenantID, entry.Actor, entry.Action, entry.Resource, entry.Detail, entry.Layer,
	)
	return err
}

// Query retrieves audit entries matching the filter
func (s *AuditStore) Query(filter AuditFilter) ([]AuditEntry, error) {
	query := "SELECT id, timestamp, tenant_id, actor, action, resource, detail, layer FROM audit_trail WHERE 1=1"
	args := make([]interface{}, 0)

	if filter.TenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, filter.TenantID)
	}
	if filter.Actor != "" {
		query += " AND actor = ?"
		args = append(args, filter.Actor)
	}
	if filter.Action != "" {
		query += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.Resource != "" {
		query += " AND resource LIKE ?"
		args = append(args, "%"+filter.Resource+"%")
	}
	if filter.Layer != "" {
		query += " AND layer = ?"
		args = append(args, filter.Layer)
	}
	if !filter.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}
	if !filter.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.Until)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 100"
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]AuditEntry, 0)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.TenantID, &e.Actor, &e.Action, &e.Resource, &e.Detail, &e.Layer); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Prune deletes audit entries older than the given duration
func (s *AuditStore) Prune(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Exec("DELETE FROM audit_trail WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Count returns the total number of audit entries
func (s *AuditStore) Count(tenantID string) (int64, error) {
	query := "SELECT COUNT(*) FROM audit_trail"
	args := make([]interface{}, 0)
	if tenantID != "" {
		query += " WHERE tenant_id = ?"
		args = append(args, tenantID)
	}
	var count int64
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// Export returns all audit entries as JSON bytes
func (s *AuditStore) Export(filter AuditFilter) ([]byte, error) {
	entries, err := s.Query(filter)
	if err != nil {
		return nil, err
	}
	return json.Marshal(entries)
}
