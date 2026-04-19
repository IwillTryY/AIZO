package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EventRecord represents a single immutable event in the log
type EventRecord struct {
	Seq          int64     `json:"seq"`
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"` // state_change, command, proposal, mesh_msg, incident
	EntityID     string    `json:"entity_id"`
	NodeID       string    `json:"node_id"`
	Payload      string    `json:"payload"`
	Checksum     string    `json:"checksum"`
	PrevChecksum string    `json:"prev_checksum"`
}

// EventLog is an append-only, tamper-evident event log backed by SQLite
type EventLog struct {
	db           *DB
	lastChecksum string
}

// NewEventLog creates a new event log
func NewEventLog(db *DB) (*EventLog, error) {
	el := &EventLog{db: db}

	// Get the last checksum for chain integrity
	var checksum sql.NullString
	db.QueryRow("SELECT checksum FROM event_log ORDER BY seq DESC LIMIT 1").Scan(&checksum)
	if checksum.Valid {
		el.lastChecksum = checksum.String
	}

	return el, nil
}

// Append adds an event to the log. Events are immutable once written.
func (el *EventLog) Append(eventType, entityID, nodeID string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Compute checksum: SHA256(prev_checksum + payload)
	checksumInput := el.lastChecksum + string(payloadJSON)
	hash := sha256.Sum256([]byte(checksumInput))
	checksum := hex.EncodeToString(hash[:])

	_, err = el.db.Exec(
		`INSERT INTO event_log (id, timestamp, type, entity_id, node_id, payload, checksum, prev_checksum)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), time.Now(), eventType, entityID, nodeID,
		string(payloadJSON), checksum, el.lastChecksum,
	)
	if err != nil {
		return fmt.Errorf("append event: %w", err)
	}

	el.lastChecksum = checksum
	return nil
}

// Replay replays all events since a given time, calling handler for each
func (el *EventLog) Replay(since time.Time, handler func(EventRecord)) error {
	rows, err := el.db.Query(
		`SELECT seq, timestamp, type, entity_id, node_id, payload, checksum, prev_checksum
		 FROM event_log WHERE timestamp >= ? ORDER BY seq ASC`, since,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var e EventRecord
		var entityID, nodeID sql.NullString
		if err := rows.Scan(&e.Seq, &e.Timestamp, &e.Type, &entityID, &nodeID,
			&e.Payload, &e.Checksum, &e.PrevChecksum); err != nil {
			continue
		}
		if entityID.Valid {
			e.EntityID = entityID.String
		}
		if nodeID.Valid {
			e.NodeID = nodeID.String
		}
		handler(e)
	}
	return nil
}

// ReplayEntity replays events for a specific entity
func (el *EventLog) ReplayEntity(entityID string, since time.Time, handler func(EventRecord)) error {
	rows, err := el.db.Query(
		`SELECT seq, timestamp, type, entity_id, node_id, payload, checksum, prev_checksum
		 FROM event_log WHERE entity_id = ? AND timestamp >= ? ORDER BY seq ASC`, entityID, since,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var e EventRecord
		var eid, nid sql.NullString
		if err := rows.Scan(&e.Seq, &e.Timestamp, &e.Type, &eid, &nid,
			&e.Payload, &e.Checksum, &e.PrevChecksum); err != nil {
			continue
		}
		if eid.Valid {
			e.EntityID = eid.String
		}
		if nid.Valid {
			e.NodeID = nid.String
		}
		handler(e)
	}
	return nil
}

// VerifyIntegrity walks the entire chain and verifies checksums
// Returns the number of verified events and any integrity violations
func (el *EventLog) VerifyIntegrity() (verified int, violations []string, err error) {
	rows, err := el.db.Query(
		`SELECT seq, payload, checksum, prev_checksum FROM event_log ORDER BY seq ASC`,
	)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	prevChecksum := ""
	for rows.Next() {
		var seq int64
		var payload, checksum, prevCS string
		var prevCSNull sql.NullString
		if err := rows.Scan(&seq, &payload, &checksum, &prevCSNull); err != nil {
			continue
		}
		if prevCSNull.Valid {
			prevCS = prevCSNull.String
		}

		// Verify chain link
		if prevCS != prevChecksum {
			violations = append(violations, fmt.Sprintf("seq %d: prev_checksum mismatch (expected %s, got %s)", seq, prevChecksum, prevCS))
		}

		// Verify checksum
		checksumInput := prevCS + payload
		hash := sha256.Sum256([]byte(checksumInput))
		expected := hex.EncodeToString(hash[:])
		if checksum != expected {
			violations = append(violations, fmt.Sprintf("seq %d: checksum mismatch (expected %s, got %s)", seq, expected, checksum))
		}

		prevChecksum = checksum
		verified++
	}

	return verified, violations, nil
}

// Count returns the total number of events
func (el *EventLog) Count() (int64, error) {
	var count int64
	err := el.db.QueryRow("SELECT COUNT(*) FROM event_log").Scan(&count)
	return count, err
}

// Tail returns the last N events
func (el *EventLog) Tail(n int) ([]EventRecord, error) {
	rows, err := el.db.Query(
		`SELECT seq, timestamp, type, entity_id, node_id, payload, checksum, prev_checksum
		 FROM event_log ORDER BY seq DESC LIMIT ?`, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]EventRecord, 0, n)
	for rows.Next() {
		var e EventRecord
		var entityID, nodeID sql.NullString
		if err := rows.Scan(&e.Seq, &e.Timestamp, &e.Type, &entityID, &nodeID,
			&e.Payload, &e.Checksum, &e.PrevChecksum); err != nil {
			continue
		}
		if entityID.Valid {
			e.EntityID = entityID.String
		}
		if nodeID.Valid {
			e.NodeID = nodeID.String
		}
		events = append(events, e)
	}
	return events, nil
}
