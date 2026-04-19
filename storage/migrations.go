package storage

import "fmt"

// migrate runs all database migrations
func (d *DB) migrate() error {
	// Create migration tracking table
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("cannot create schema_version table: %w", err)
	}

	// Get current version
	var currentVersion int
	row := d.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	row.Scan(&currentVersion)

	migrations := []string{
		// v1: audit trail
		`CREATE TABLE IF NOT EXISTS audit_trail (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			detail TEXT,
			layer TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_trail(timestamp);
		CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_trail(tenant_id);`,

		// v2: metrics
		`CREATE TABLE IF NOT EXISTS metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			value REAL NOT NULL,
			timestamp DATETIME NOT NULL,
			entity_id TEXT NOT NULL,
			labels TEXT,
			metadata TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_metrics_entity_time ON metrics(entity_id, timestamp);
		CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name);`,

		// v3: logs
		`CREATE TABLE IF NOT EXISTS logs (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			source TEXT NOT NULL,
			entity_id TEXT,
			fields TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'default'
		);
		CREATE INDEX IF NOT EXISTS idx_logs_entity_time ON logs(entity_id, timestamp);
		CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);`,

		// v4: traces
		`CREATE TABLE IF NOT EXISTS traces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration_ms REAL,
			status TEXT,
			entity_id TEXT,
			metadata TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'default'
		);
		CREATE TABLE IF NOT EXISTS spans (
			id TEXT PRIMARY KEY,
			trace_id TEXT NOT NULL,
			parent_id TEXT,
			name TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration_ms REAL,
			status TEXT,
			attributes TEXT,
			FOREIGN KEY (trace_id) REFERENCES traces(id)
		);
		CREATE INDEX IF NOT EXISTS idx_spans_trace ON spans(trace_id);`,

		// v5: events
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			type TEXT NOT NULL,
			source TEXT NOT NULL,
			entity_id TEXT,
			severity TEXT,
			data TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'default'
		);
		CREATE INDEX IF NOT EXISTS idx_events_entity_time ON events(entity_id, timestamp);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);`,

		// v6: incidents + learning data
		`CREATE TABLE IF NOT EXISTS incidents (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			type TEXT NOT NULL,
			entity_id TEXT,
			description TEXT,
			action_taken TEXT,
			action_succeeded INTEGER,
			resolution TEXT,
			duration_ms INTEGER,
			context TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'default'
		);
		CREATE INDEX IF NOT EXISTS idx_incidents_type ON incidents(type);
		CREATE TABLE IF NOT EXISTS learning_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pattern TEXT NOT NULL UNIQUE,
			successful_fixes TEXT,
			failed_fixes TEXT,
			frequency INTEGER DEFAULT 0,
			last_occurrence DATETIME,
			context TEXT
		);`,

		// v7: chat sessions + messages
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			metadata TEXT
		);
		CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (session_id) REFERENCES chat_sessions(id)
		);
		CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id);`,

		// v8: proposals
		`CREATE TABLE IF NOT EXISTS proposals (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			source TEXT NOT NULL,
			action TEXT NOT NULL,
			entity_id TEXT,
			priority INTEGER,
			risk TEXT,
			confidence REAL,
			reasoning TEXT,
			parameters TEXT,
			requires_approval INTEGER,
			status TEXT NOT NULL,
			approved_by TEXT,
			approved_at DATETIME,
			executed_at DATETIME,
			result TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'default'
		);
		CREATE INDEX IF NOT EXISTS idx_proposals_status ON proposals(status);`,

		// v9: policies
		`CREATE TABLE IF NOT EXISTS policies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			rules TEXT NOT NULL,
			effect TEXT NOT NULL DEFAULT 'deny',
			priority INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// v10: tenants
		`CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			config TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT OR IGNORE INTO tenants (id, name) VALUES ('default', 'Default Tenant');`,

		// v11: event log (append-only, tamper-evident)
		`CREATE TABLE IF NOT EXISTS event_log (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT NOT NULL UNIQUE,
			timestamp DATETIME NOT NULL,
			type TEXT NOT NULL,
			entity_id TEXT,
			node_id TEXT,
			payload TEXT NOT NULL,
			checksum TEXT NOT NULL,
			prev_checksum TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_eventlog_time ON event_log(timestamp);
		CREATE INDEX IF NOT EXISTS idx_eventlog_entity ON event_log(entity_id);
		CREATE INDEX IF NOT EXISTS idx_eventlog_type ON event_log(type);`,

		// v12: persistent entity state
		`CREATE TABLE IF NOT EXISTS entity_state_persistent (
			entity_id TEXT PRIMARY KEY,
			type TEXT,
			status TEXT,
			desired_state TEXT,
			actual_state TEXT,
			metadata TEXT,
			version INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for i, migration := range migrations {
		version := i + 1
		if version <= currentVersion {
			continue
		}
		if _, err := d.db.Exec(migration); err != nil {
			return fmt.Errorf("migration v%d failed: %w", version, err)
		}
		if _, err := d.db.Exec("INSERT INTO schema_version (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("cannot record migration v%d: %w", version, err)
		}
	}

	return nil
}
