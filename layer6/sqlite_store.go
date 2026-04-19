package layer6

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SQLiteStore persists Layer 6 data to SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed Layer 6 store
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// --- Incidents ---

// StoreIncident persists an incident
func (s *SQLiteStore) StoreIncident(incident HistoricalIncident) error {
	if incident.ID == "" {
		incident.ID = uuid.New().String()
	}
	ctx, _ := json.Marshal(incident.Context)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO incidents (id, timestamp, type, entity_id, description, action_taken, action_succeeded, resolution, duration_ms, context)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		incident.ID, incident.Timestamp, string(incident.Type), incident.EntityID,
		incident.Description, incident.ActionTaken, incident.ActionSucceeded,
		incident.Resolution, incident.Duration.Milliseconds(), string(ctx),
	)
	return err
}

// LoadIncidents loads incidents from SQLite
func (s *SQLiteStore) LoadIncidents(limit int) ([]HistoricalIncident, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id, timestamp, type, entity_id, description, action_taken, action_succeeded, resolution, duration_ms, context
		 FROM incidents ORDER BY timestamp DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]HistoricalIncident, 0)
	for rows.Next() {
		var inc HistoricalIncident
		var etype string
		var durationMs int64
		var ctxJSON sql.NullString
		if err := rows.Scan(&inc.ID, &inc.Timestamp, &etype, &inc.EntityID, &inc.Description,
			&inc.ActionTaken, &inc.ActionSucceeded, &inc.Resolution, &durationMs, &ctxJSON); err != nil {
			continue
		}
		inc.Type = EventType(etype)
		inc.Duration = time.Duration(durationMs) * time.Millisecond
		if ctxJSON.Valid {
			json.Unmarshal([]byte(ctxJSON.String), &inc.Context)
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

// LoadSimilarIncidents loads incidents matching a type
func (s *SQLiteStore) LoadSimilarIncidents(eventType EventType, limit int) ([]HistoricalIncident, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.Query(
		`SELECT id, timestamp, type, entity_id, description, action_taken, action_succeeded, resolution, duration_ms, context
		 FROM incidents WHERE type = ? ORDER BY timestamp DESC LIMIT ?`, string(eventType), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]HistoricalIncident, 0)
	for rows.Next() {
		var inc HistoricalIncident
		var etype string
		var durationMs int64
		var ctxJSON sql.NullString
		if err := rows.Scan(&inc.ID, &inc.Timestamp, &etype, &inc.EntityID, &inc.Description,
			&inc.ActionTaken, &inc.ActionSucceeded, &inc.Resolution, &durationMs, &ctxJSON); err != nil {
			continue
		}
		inc.Type = EventType(etype)
		inc.Duration = time.Duration(durationMs) * time.Millisecond
		if ctxJSON.Valid {
			json.Unmarshal([]byte(ctxJSON.String), &inc.Context)
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

// --- Learning Data ---

// StoreLearningData persists learning data
func (s *SQLiteStore) StoreLearningData(data *LearningData) error {
	successJSON, _ := json.Marshal(data.SuccessfulFixes)
	failedJSON, _ := json.Marshal(data.FailedFixes)
	ctxJSON, _ := json.Marshal(data.Context)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO learning_data (pattern, successful_fixes, failed_fixes, frequency, last_occurrence, context)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		data.IncidentPattern, string(successJSON), string(failedJSON),
		data.Frequency, data.LastOccurrence, string(ctxJSON),
	)
	return err
}

// LoadLearningData loads learning data for a pattern
func (s *SQLiteStore) LoadLearningData(pattern string) (*LearningData, error) {
	var data LearningData
	var successJSON, failedJSON, ctxJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT pattern, successful_fixes, failed_fixes, frequency, last_occurrence, context
		 FROM learning_data WHERE pattern = ?`, pattern,
	).Scan(&data.IncidentPattern, &successJSON, &failedJSON, &data.Frequency, &data.LastOccurrence, &ctxJSON)
	if err != nil {
		return nil, err
	}
	if successJSON.Valid {
		json.Unmarshal([]byte(successJSON.String), &data.SuccessfulFixes)
	}
	if failedJSON.Valid {
		json.Unmarshal([]byte(failedJSON.String), &data.FailedFixes)
	}
	if ctxJSON.Valid {
		json.Unmarshal([]byte(ctxJSON.String), &data.Context)
	}
	return &data, nil
}

// --- Proposals ---

// StoreProposal persists an action proposal
func (s *SQLiteStore) StoreProposal(p *ActionProposal) error {
	params, _ := json.Marshal(p.Parameters)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO proposals (id, timestamp, source, action, entity_id, priority, risk, confidence, reasoning, parameters, requires_approval, status, approved_by, approved_at, executed_at, result)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Timestamp, p.Source, p.Action, p.EntityID, p.Priority, p.Risk, p.Confidence,
		p.Reasoning, string(params), p.RequiresApproval, string(p.Status),
		p.ApprovedBy, p.ApprovedAt, p.ExecutedAt, p.Result,
	)
	return err
}

// LoadProposals loads proposals with optional status filter
func (s *SQLiteStore) LoadProposals(status string, limit int) ([]*ActionProposal, error) {
	if limit <= 0 {
		limit = 50
	}
	query := "SELECT id, timestamp, source, action, entity_id, priority, risk, confidence, reasoning, parameters, requires_approval, status, approved_by, approved_at, executed_at, result FROM proposals"
	args := make([]interface{}, 0)
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	proposals := make([]*ActionProposal, 0)
	for rows.Next() {
		var p ActionProposal
		var statusStr string
		var paramsJSON sql.NullString
		var approvedBy sql.NullString
		var approvedAt, executedAt sql.NullTime
		var result sql.NullString
		if err := rows.Scan(&p.ID, &p.Timestamp, &p.Source, &p.Action, &p.EntityID, &p.Priority,
			&p.Risk, &p.Confidence, &p.Reasoning, &paramsJSON, &p.RequiresApproval, &statusStr,
			&approvedBy, &approvedAt, &executedAt, &result); err != nil {
			continue
		}
		p.Status = ProposalStatus(statusStr)
		if paramsJSON.Valid {
			json.Unmarshal([]byte(paramsJSON.String), &p.Parameters)
		}
		if approvedBy.Valid {
			p.ApprovedBy = approvedBy.String
		}
		if approvedAt.Valid {
			p.ApprovedAt = approvedAt.Time
		}
		if executedAt.Valid {
			p.ExecutedAt = executedAt.Time
		}
		if result.Valid {
			p.Result = result.String
		}
		proposals = append(proposals, &p)
	}
	return proposals, nil
}

// --- Rule Stats ---

// RuleStats holds persisted success/failure counts for a rule
type RuleStats struct {
	SuccessCount int
	FailureCount int
}

// LoadRuleStats loads success/failure counts per rule from incident history
func (s *SQLiteStore) LoadRuleStats() (map[string]RuleStats, error) {
	rows, err := s.db.Query(
		`SELECT description, action_succeeded FROM incidents WHERE description LIKE 'rule:%'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]RuleStats)
	for rows.Next() {
		var desc string
		var succeeded bool
		if err := rows.Scan(&desc, &succeeded); err != nil {
			continue
		}
		s := stats[desc]
		if succeeded {
			s.SuccessCount++
		} else {
			s.FailureCount++
		}
		stats[desc] = s
	}
	return stats, nil
}
