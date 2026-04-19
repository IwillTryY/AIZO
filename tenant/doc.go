// Package tenant provides multi-tenancy support for AIZO.
// Each tenant is an isolated namespace with its own entities, policies,
// and audit trail. Tenant isolation is enforced at the storage layer
// via tenant_id prefixing on all queries.
package tenant
