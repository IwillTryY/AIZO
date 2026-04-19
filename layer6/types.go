package layer6

import (
	"time"
)

// SystemSummary represents aggregated system data
type SystemSummary struct {
	Timestamp         time.Time              `json:"timestamp"`
	TotalEntities     int                    `json:"total_entities"`
	HealthyEntities   int                    `json:"healthy_entities"`
	DegradedEntities  int                    `json:"degraded_entities"`
	UnhealthyEntities int                    `json:"unhealthy_entities"`
	TotalContainers   int                    `json:"total_containers"`
	RunningContainers int                    `json:"running_containers"`
	FailedContainers  int                    `json:"failed_containers"`
	AverageLoad       float64                `json:"average_load"`
	MemoryUsage       float64                `json:"memory_usage"`
	CPUUsage          float64                `json:"cpu_usage"`
	NetworkErrors     int                    `json:"network_errors"`
	DiskUsage         float64                `json:"disk_usage"`
	RecentIncidents   int                    `json:"recent_incidents"`
	Trends            map[string]interface{} `json:"trends"`
	Metrics           map[string]float64     `json:"metrics"`
}

// SystemEvent represents a specific incident or event
type SystemEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Type          EventType              `json:"type"`
	Severity      string                 `json:"severity"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Description   string                 `json:"description"`
	Context       map[string]interface{} `json:"context"`
	Metrics       map[string]float64     `json:"metrics"`
	RelatedEvents []string               `json:"related_events"`
}

// EventType represents the type of system event
type EventType string

const (
	EventContainerCrash  EventType = "container_crash"
	EventNodeFailure     EventType = "node_failure"
	EventNetworkIssue    EventType = "network_issue"
	EventHighMemory      EventType = "high_memory"
	EventHighCPU         EventType = "high_cpu"
	EventDiskFull        EventType = "disk_full"
	EventHealthCheckFail EventType = "health_check_fail"
	EventServiceDown     EventType = "service_down"
	EventAnomalyDetected EventType = "anomaly_detected"
)

// HealthStatus represents system health
type HealthStatus string

const (
	HealthHealthy  HealthStatus = "Healthy"
	HealthDegraded HealthStatus = "Degraded"
	HealthCritical HealthStatus = "Critical"
)

// IncidentHistory represents past incidents
type IncidentHistory struct {
	Incidents []HistoricalIncident `json:"incidents"`
}

// HistoricalIncident represents a past incident
type HistoricalIncident struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Type            EventType              `json:"type"`
	EntityID        string                 `json:"entity_id"`
	Description     string                 `json:"description"`
	ActionTaken     string                 `json:"action_taken"`
	ActionSucceeded bool                   `json:"action_succeeded"`
	Resolution      string                 `json:"resolution"`
	Duration        time.Duration          `json:"duration"`
	Context         map[string]interface{} `json:"context"`
}

// LearningData stores pattern recognition data
type LearningData struct {
	IncidentPattern string                 `json:"incident_pattern"`
	SuccessfulFixes []string               `json:"successful_fixes"`
	FailedFixes     []string               `json:"failed_fixes"`
	Frequency       int                    `json:"frequency"`
	LastOccurrence  time.Time              `json:"last_occurrence"`
	Context         map[string]interface{} `json:"context"`
}

// ProposalStatus represents the state of an action proposal
type ProposalStatus string

const (
	ProposalPending   ProposalStatus = "pending"
	ProposalApproved  ProposalStatus = "approved"
	ProposalRejected  ProposalStatus = "rejected"
	ProposalExecuting ProposalStatus = "executing"
	ProposalCompleted ProposalStatus = "completed"
	ProposalFailed    ProposalStatus = "failed"
)

// ActionProposal represents a proposed corrective action
type ActionProposal struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	Source           string                 `json:"source"` // "rule:<rule_id>"
	RuleID           string                 `json:"rule_id"`
	Action           string                 `json:"action"`
	EntityID         string                 `json:"entity_id"`
	Priority         int                    `json:"priority"`
	Risk             string                 `json:"risk"`
	Confidence       float64                `json:"confidence"`
	Reasoning        string                 `json:"reasoning"`
	Parameters       map[string]interface{} `json:"parameters"`
	RequiresApproval bool                   `json:"requires_approval"`
	Status           ProposalStatus         `json:"status"`
	ApprovedBy       string                 `json:"approved_by,omitempty"`
	ApprovedAt       time.Time              `json:"approved_at,omitempty"`
	ExecutedAt       time.Time              `json:"executed_at,omitempty"`
	Result           string                 `json:"result,omitempty"`
}

// ProposedAction is the action to be executed
type ProposedAction struct {
	Description string                 `json:"description"`
	ActionType  string                 `json:"action_type"`
	EntityID    string                 `json:"entity_id"`
	Parameters  map[string]interface{} `json:"parameters"`
	Risk        string                 `json:"risk"`
	Reversible  bool                   `json:"reversible"`
	Reasoning   string                 `json:"reasoning"`
}

// ExecutionResult represents the result of executing an action
type ExecutionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// RecommendedAction is a recommended action from rule evaluation
type RecommendedAction struct {
	Priority   int                    `json:"priority"`
	Action     string                 `json:"action"`
	Reason     string                 `json:"reason"`
	Risk       string                 `json:"risk"`
	Reversible bool                   `json:"reversible"`
	EntityID   string                 `json:"entity_id,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}
