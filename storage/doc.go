// Package storage provides SQLite-backed persistence for AIZO.
// It manages the database lifecycle, schema migrations, and a
// centralized audit trail. All layers use this package for
// durable storage of metrics, logs, incidents, proposals, and policies.
package storage
