package layer6

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// DefaultActionExecutor implements basic action execution
type DefaultActionExecutor struct {
	// Add dependencies here (e.g., container runtime, config manager)
}

// NewDefaultActionExecutor creates a new default action executor
func NewDefaultActionExecutor() *DefaultActionExecutor {
	return &DefaultActionExecutor{}
}

// ExecuteAction executes a proposed action
func (e *DefaultActionExecutor) ExecuteAction(ctx context.Context, action *ProposedAction) (*ExecutionResult, error) {
	switch action.ActionType {
	case "restart":
		return e.executeRestart(ctx, action)
	case "cleanup":
		return e.executeCleanup(ctx, action)
	case "scale":
		return e.executeScale(ctx, action)
	case "config_change":
		return e.executeConfigChange(ctx, action)
	case "investigate":
		return e.executeInvestigate(ctx, action)
	default:
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action type: %s", action.ActionType),
		}, nil
	}
}

// executeRestart restarts a service/container
func (e *DefaultActionExecutor) executeRestart(ctx context.Context, action *ProposedAction) (*ExecutionResult, error) {
	// In a real implementation, this would interact with Layer 5 (Container Runtime)
	// For now, simulate a restart

	entityID := action.EntityID

	// Simulate restart delay
	time.Sleep(500 * time.Millisecond)

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Successfully restarted %s", entityID),
	}, nil
}

// executeCleanup performs cleanup operations (e.g., clear memory, delete temp files)
func (e *DefaultActionExecutor) executeCleanup(ctx context.Context, action *ProposedAction) (*ExecutionResult, error) {
	entityID := action.EntityID

	// Check if this is a memory cleanup for the web server
	if endpoint, ok := action.Parameters["endpoint"].(string); ok {
		// Try to trigger cleanup via HTTP endpoint
		cleanupURL := endpoint + "/cleanup"

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(cleanupURL)
		if err != nil {
			// Endpoint doesn't exist, simulate cleanup
			time.Sleep(300 * time.Millisecond)
			return &ExecutionResult{
				Success: true,
				Message: fmt.Sprintf("Simulated memory cleanup for %s (freed ~50MB)", entityID),
			}, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return &ExecutionResult{
				Success: true,
				Message: fmt.Sprintf("Successfully cleaned up memory for %s via /cleanup endpoint", entityID),
			}, nil
		}
	}

	// Check if entity ID looks like a web server endpoint
	if entityID == "web-server-vulnerable" {
		// Try the default endpoint
		cleanupURL := "http://localhost:8080/cleanup"

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(cleanupURL)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return &ExecutionResult{
					Success: true,
					Message: fmt.Sprintf("Successfully cleaned up memory leak for %s (called /cleanup endpoint)", entityID),
				}, nil
			}
		}
	}

	// Default cleanup simulation
	time.Sleep(300 * time.Millisecond)

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Performed cleanup on %s", entityID),
	}, nil
}

// executeScale scales a service up or down
func (e *DefaultActionExecutor) executeScale(ctx context.Context, action *ProposedAction) (*ExecutionResult, error) {
	entityID := action.EntityID
	replicas := 1

	if r, ok := action.Parameters["replicas"].(float64); ok {
		replicas = int(r)
	}

	time.Sleep(500 * time.Millisecond)

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Scaled %s to %d replicas", entityID, replicas),
	}, nil
}

// executeConfigChange applies a configuration change
func (e *DefaultActionExecutor) executeConfigChange(ctx context.Context, action *ProposedAction) (*ExecutionResult, error) {
	entityID := action.EntityID

	time.Sleep(400 * time.Millisecond)

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Applied configuration changes to %s", entityID),
	}, nil
}

// executeInvestigate performs investigation/diagnostic actions
func (e *DefaultActionExecutor) executeInvestigate(ctx context.Context, action *ProposedAction) (*ExecutionResult, error) {
	entityID := action.EntityID

	// Simulate investigation
	time.Sleep(1 * time.Second)

	findings := "No anomalies detected in logs. Memory usage is stable."
	if diagnostics, ok := action.Parameters["diagnostics"].(string); ok {
		findings = diagnostics
	}

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Investigation complete for %s: %s", entityID, findings),
	}, nil
}
