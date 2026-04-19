# Layer 6: Intelligence Layer - AI-Powered Self-Healing - Version 1

## Overview
Layer 6 is the Intelligence Layer for RealityOS - an AI-powered system that analyzes infrastructure state, detects issues, and proposes safe, minimal-risk corrective actions. It operates in two modes: scheduled audits and event response.

## Architecture

### Core Components

1. **Types** (`types.go`)
   - SystemSummary: Aggregated system data for audits
   - SystemEvent: Specific incidents or events
   - AuditReport: Scheduled audit output
   - EventResponse: Event analysis output
   - ActionProposal: Proposed corrective actions
   - IncidentHistory: Learning from past incidents
   - **SystemPrompt**: Exact AI instructions (verbatim as specified)

2. **AI Client** (`ai_client.go`)
   - API provider integration (OpenAI, Anthropic, etc.)
   - Ollama fallback for local models
   - Prompt building and response parsing
   - Automatic failover between providers

3. **Intelligence Engine** (`intelligence_engine.go`)
   - Scheduled audit execution
   - Event response handling
   - Action proposal management
   - Incident recording and learning
   - Pattern recognition

4. **Manager** (`manager.go`)
   - High-level orchestration
   - Periodic audit scheduling
   - Approval workflow
   - Statistics and reporting

## Key Features

### ✅ Two Operating Modes

**1. Scheduled Audit Mode (every 2 hours)**
- Analyzes aggregated system summary
- Assesses overall health
- Identifies trends and emerging risks
- Produces structured reports
- Recommends prioritized actions

**2. Event Response Mode (triggered by incidents)**
- Analyzes specific system events
- Determines root cause
- Proposes minimal safe corrective action
- Provides confidence scores
- Suggests alternatives

### ✅ Safety Guardrails

- **No Direct Execution**: AI only proposes, never executes
- **Approval Workflow**: High-risk actions require human approval
- **Reversible Actions**: Prefers actions that can be undone
- **Avoids Destruction**: Never suggests destructive operations
- **Learning-Based**: Uses incident history to improve
- **Observation First**: Recommends monitoring when uncertain

### ✅ Learning Behavior

- Records all incidents and resolutions
- Tracks successful vs failed fixes
- Recognizes recurring patterns
- Prefers proven solutions
- Avoids repeating failures
- Reduces analysis time for known issues

### ✅ Action Management

- Proposal queue with priorities
- Risk assessment (low, medium, high)
- Confidence scoring (0-1)
- Approval workflow
- Execution tracking
- Result recording

## Usage Example

```go
// Create Layer 6 manager
config := &layer6.ManagerConfig{
    AIConfig: &layer6.AIConfig{
        Provider:           "api",
        APIEndpoint:        "https://api.openai.com/v1/chat/completions",
        APIKey:             "your-api-key",
        Model:              "gpt-4",
        OllamaEndpoint:     "http://localhost:11434",
        OllamaModel:        "llama2",
        FallbackToOllama:   true,
        MaxTokens:          2000,
        Temperature:        0.7,
        AuditInterval:      2 * time.Hour,
        EnableAutoApproval: false,
        AutoApprovalRisk:   "low",
    },
    AuditInterval: 2 * time.Hour,
}

manager := layer6.NewManager(config)
ctx := context.Background()

// Start manager (begins periodic audits)
manager.Start(ctx)

// Scheduled Audit
summary := &layer6.SystemSummary{
    TotalEntities:     50,
    HealthyEntities:   45,
    DegradedEntities:  3,
    UnhealthyEntities: 2,
    CPUUsage:          60.2,
    MemoryUsage:       75.5,
    DiskUsage:         82.0,
    RecentIncidents:   3,
}

auditReport, err := manager.ScheduledAudit(ctx, summary)
fmt.Printf("Health: %s\n", auditReport.HealthStatus)
for _, action := range auditReport.RecommendedActions {
    fmt.Printf("Priority %d: %s (Risk: %s)\n",
        action.Priority, action.Action, action.Risk)
}

// Event Response
event := &layer6.SystemEvent{
    Type:        layer6.EventContainerCrash,
    Severity:    "high",
    EntityID:    "web-server-1",
    Description: "Container crashed with exit code 137 (OOM)",
    Context: map[string]interface{}{
        "exit_code": 137,
        "memory_usage": "510Mi",
        "memory_limit": "512Mi",
    },
}

response, err := manager.RespondToEvent(ctx, event)
fmt.Printf("Cause: %s (Confidence: %.0f%%)\n",
    response.LikelyCause, response.Confidence*100)
fmt.Printf("Action: %s\n", response.SuggestedAction)
fmt.Printf("Risk: %s\n", response.RiskOfAction)

// Get pending proposals
proposals := manager.GetPendingProposals()
for _, proposal := range proposals {
    if proposal.RequiresApproval {
        // Manual approval
        err := manager.ApproveProposal(proposal.ID, "operator@example.com")
    }
}

// Execute approved proposal
err = manager.ExecuteProposal(proposal.ID, true, "Action completed successfully")

// Record incident for learning
incident := layer6.HistoricalIncident{
    Type:            event.Type,
    EntityID:        event.EntityID,
    ActionTaken:     response.SuggestedAction,
    ActionSucceeded: true,
    Duration:        5 * time.Minute,
}
manager.RecordIncident(incident)
```

## System Prompt

The AI operates with this exact system prompt (as specified):

```
You are Layer 6: the Systems Intelligence Layer for a distributed infrastructure platform.

Your role is NOT to directly control systems.
Your role is to analyze system state, detect issues, and propose safe, minimal-risk actions.

[Full prompt included in types.go - see SystemPrompt constant]
```

## Data Models

### SystemSummary (Audit Mode)
```go
SystemSummary {
    total_entities: int
    healthy_entities: int
    degraded_entities: int
    unhealthy_entities: int
    total_containers: int
    running_containers: int
    failed_containers: int
    cpu_usage: float64 (percentage)
    memory_usage: float64 (percentage)
    disk_usage: float64 (percentage)
    network_errors: int
    recent_incidents: int
    trends: map[string]interface{}
    metrics: map[string]float64
}
```

### SystemEvent (Event Mode)
```go
SystemEvent {
    type: container_crash | node_failure | network_issue | high_memory | ...
    severity: low | medium | high | critical
    entity_id: string
    entity_type: string
    description: string
    context: map[string]interface{}
    metrics: map[string]float64
    related_events: []string
}
```

### AuditReport (Output)
```go
AuditReport {
    health_status: Healthy | Degraded | Critical
    key_issues: []string
    emerging_risks: []string
    recommended_actions: []{
        priority: int
        action: string
        reason: string
        risk: low | medium | high
        reversible: bool
    }
    confidence: 0-1
}
```

### EventResponse (Output)
```go
EventResponse {
    severity: Low | Medium | High | Critical
    likely_cause: string
    confidence: 0-1
    suggested_action: string
    risk_of_action: string
    alternative_options: []string
    reasoning: string
    requires_approval: bool
}
```

## Running the Demo

```bash
cd examples
go run layer6_demo.go
```

The demo demonstrates:
1. Scheduled audit mode with system summary
2. Event response mode with container crash
3. Action proposal creation
4. Approval workflow
5. Execution tracking
6. Incident learning
7. System statistics
8. Safety guardrails

**Note**: Demo simulates AI responses. Set API keys for real AI analysis.

## Integration with Previous Layers

### With Layer 1-5
```go
// Layer 3: Event triggers AI analysis
event := &layer3.Event{
    Type: layer3.EventTypeAlert,
    EntityID: "server-1",
}
layer3Manager.PublishEvent(event)

// Layer 6: AI analyzes and proposes action
systemEvent := &layer6.SystemEvent{
    Type: layer6.EventHighMemory,
    EntityID: "server-1",
}
response, _ := layer6Manager.RespondToEvent(ctx, systemEvent)

// Layer 5: Execute approved action
if approved {
    // Restart container, scale resources, etc.
    layer5Manager.RestartContainer(ctx, containerID, 10)
}

// Layer 4: Update state
layer4Manager.UpdateActualState(ctx, entityID, newState)

// Layer 6: Record outcome
layer6Manager.ExecuteProposal(proposalID, true, "Action successful")
```

## Configuration

### AI Provider Setup

**OpenAI API:**
```go
AIConfig{
    Provider: "api",
    APIEndpoint: "https://api.openai.com/v1/chat/completions",
    APIKey: "sk-...",
    Model: "gpt-4",
}
```

**Anthropic Claude:**
```go
AIConfig{
    Provider: "api",
    APIEndpoint: "https://api.anthropic.com/v1/messages",
    APIKey: "sk-ant-...",
    Model: "claude-3-opus-20240229",
}
```

**Ollama (Local):**
```go
AIConfig{
    Provider: "ollama",
    OllamaEndpoint: "http://localhost:11434",
    OllamaModel: "llama2",
}
```

**With Fallback:**
```go
AIConfig{
    Provider: "api",
    APIEndpoint: "https://api.openai.com/v1/chat/completions",
    APIKey: "sk-...",
    Model: "gpt-4",
    FallbackToOllama: true,
    OllamaEndpoint: "http://localhost:11434",
    OllamaModel: "llama2",
}
```

## Safety Features

### Guardrails
1. **No Direct Execution**: AI cannot run commands
2. **Proposal-Only**: All actions go through approval
3. **Risk Assessment**: Every action has risk level
4. **Approval Gates**: High-risk requires human approval
5. **Reversibility Check**: Prefers undoable actions
6. **Destructive Prevention**: Blocks dangerous operations
7. **Pattern Learning**: Avoids known failures
8. **Confidence Scoring**: Uncertainty triggers observation

### Auto-Approval Rules
```go
AIConfig{
    EnableAutoApproval: true,
    AutoApprovalRisk: "low", // Only auto-approve low-risk
}
```

- Low risk + high confidence = auto-approve
- Medium/high risk = require approval
- Failed previously = require approval
- Uncertain = recommend observation

## API Reference

### Manager
- `Start(ctx)` - Start periodic audits
- `Stop()` - Stop manager
- `ScheduledAudit(ctx, summary)` - Run audit
- `RespondToEvent(ctx, event)` - Analyze event
- `GetPendingProposals()` - List pending actions
- `ApproveProposal(id, approver)` - Approve action
- `RejectProposal(id, reason)` - Reject action
- `ExecuteProposal(id, success, result)` - Record execution
- `RecordIncident(incident)` - Add to history
- `GetIncidentHistory()` - Get learning data
- `GetStats()` - Get statistics

### Intelligence Engine
- `ScheduledAudit(ctx, summary)` - Perform audit
- `RespondToEvent(ctx, event)` - Respond to event
- `GetPendingProposals()` - Get proposals
- `ApproveProposal(id, approver)` - Approve
- `RejectProposal(id, reason)` - Reject
- `MarkProposalExecuted(id, success, result)` - Mark done
- `RecordIncident(incident)` - Record
- `GetIncidentHistory()` - Get history

## Limitations (V1)

- Simulated response parsing (needs proper JSON parsing)
- Basic prompt building (can be enhanced)
- In-memory incident history (no persistence)
- Simple pattern matching for similar incidents
- No advanced ML for anomaly detection
- Limited context window management

## Future Enhancements

- Advanced prompt engineering
- Multi-turn conversations with AI
- Structured output parsing (JSON mode)
- Vector database for incident similarity
- Fine-tuned models on infrastructure data
- Predictive analytics
- Automated root cause analysis
- Integration with runbook automation
- Multi-agent collaboration
- Continuous learning pipeline

## Performance Considerations

- AI calls have latency (1-5 seconds typical)
- Use fallback for reliability
- Cache similar queries
- Batch audit requests
- Async event processing
- Rate limit AI calls
- Monitor token usage

## Security Notes

- Store API keys securely (environment variables, secrets manager)
- Validate all AI responses
- Sanitize inputs to AI
- Log all proposals and approvals
- Audit trail for compliance
- Role-based approval permissions
- Encrypt incident history

## Cost Management

- Use cheaper models for routine audits
- Reserve expensive models for critical events
- Implement caching for similar queries
- Set token limits
- Monitor API usage
- Consider local models (Ollama) for cost savings
- Batch requests when possible
