// Package layer4 provides state management for AIZO.
// It tracks desired vs actual state for all entities, detects drift,
// and drives reconciliation to restore intended state. Supports
// snapshot-based history and time-travel queries for debugging.
package layer4
