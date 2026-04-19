package tenant

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Manager manages tenants with SQLite persistence
type Manager struct {
	db      *sql.DB
	current string // current active tenant ID
}

// NewManager creates a new tenant manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db, current: "default"}
}

// Create creates a new tenant
func (m *Manager) Create(id, name string, config map[string]string) (*Tenant, error) {
	configJSON, _ := json.Marshal(config)
	now := time.Now()
	_, err := m.db.Exec(
		`INSERT INTO tenants (id, name, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, name, string(configJSON), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	return &Tenant{ID: id, Name: name, Config: config, CreatedAt: now, UpdatedAt: now}, nil
}

// Get returns a tenant by ID
func (m *Manager) Get(id string) (*Tenant, error) {
	var t Tenant
	var configJSON sql.NullString
	err := m.db.QueryRow(
		`SELECT id, name, config, created_at, updated_at FROM tenants WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &configJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	if configJSON.Valid {
		json.Unmarshal([]byte(configJSON.String), &t.Config)
	}
	return &t, nil
}

// List returns all tenants
func (m *Manager) List() ([]*Tenant, error) {
	rows, err := m.db.Query(`SELECT id, name, config, created_at, updated_at FROM tenants ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenants := make([]*Tenant, 0)
	for rows.Next() {
		var t Tenant
		var configJSON sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &configJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		if configJSON.Valid {
			json.Unmarshal([]byte(configJSON.String), &t.Config)
		}
		tenants = append(tenants, &t)
	}
	return tenants, nil
}

// Delete deletes a tenant (cannot delete "default")
func (m *Manager) Delete(id string) error {
	if id == "default" {
		return fmt.Errorf("cannot delete default tenant")
	}
	_, err := m.db.Exec(`DELETE FROM tenants WHERE id = ?`, id)
	return err
}

// Switch sets the active tenant
func (m *Manager) Switch(id string) error {
	// Verify tenant exists
	var count int
	m.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE id = ?`, id).Scan(&count)
	if count == 0 {
		return fmt.Errorf("tenant %s not found", id)
	}
	m.current = id
	return nil
}

// Current returns the current active tenant ID
func (m *Manager) Current() string {
	return m.current
}

// PrefixKey namespaces a key with the current tenant
func (m *Manager) PrefixKey(key string) string {
	return m.current + ":" + key
}
