// Package layer6 provides the intelligence layer for AIZO.
// It implements a self-learning rule engine that detects infrastructure
// issues, proposes corrective actions, and learns from outcomes.
// Rules are evaluated against system events and summaries. Thresholds
// auto-tune based on historical success rates. New rules are suggested
// by mining incident patterns. No external AI APIs required.
package layer6
