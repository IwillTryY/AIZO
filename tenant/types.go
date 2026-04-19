package tenant

import "time"

// Tenant represents an isolated tenant/team/environment
type Tenant struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Config    map[string]string `json:"config"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
