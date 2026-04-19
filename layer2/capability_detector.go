package layer2

import (
	"context"
	"fmt"
)

// CapabilityDetector identifies what an entity can do
type CapabilityDetector struct {
	detectors map[EntityType][]CapabilityProbe
}

// CapabilityProbe tests for a specific capability
type CapabilityProbe interface {
	Detect(ctx context.Context, entity *Entity) (bool, error)
	Capability() Capability
}

// NewCapabilityDetector creates a new capability detector
func NewCapabilityDetector() *CapabilityDetector {
	detector := &CapabilityDetector{
		detectors: make(map[EntityType][]CapabilityProbe),
	}

	// Register default probes
	detector.RegisterProbe(EntityTypeServer, &MetricsProbe{})
	detector.RegisterProbe(EntityTypeServer, &CommandProbe{})
	detector.RegisterProbe(EntityTypeServer, &HealthCheckProbe{})

	detector.RegisterProbe(EntityTypeAPI, &MetricsProbe{})
	detector.RegisterProbe(EntityTypeAPI, &HealthCheckProbe{})
	detector.RegisterProbe(EntityTypeAPI, &LogsProbe{})

	detector.RegisterProbe(EntityTypeDatabase, &MetricsProbe{})
	detector.RegisterProbe(EntityTypeDatabase, &CommandProbe{})
	detector.RegisterProbe(EntityTypeDatabase, &HealthCheckProbe{})

	return detector
}

// RegisterProbe adds a capability probe for an entity type
func (d *CapabilityDetector) RegisterProbe(entityType EntityType, probe CapabilityProbe) {
	d.detectors[entityType] = append(d.detectors[entityType], probe)
}

// DetectCapabilities identifies all capabilities of an entity
func (d *CapabilityDetector) DetectCapabilities(ctx context.Context, entity *Entity) error {
	probes, exists := d.detectors[entity.Type]
	if !exists {
		return fmt.Errorf("no probes registered for entity type: %s", entity.Type)
	}

	entity.Capabilities = make([]Capability, 0)

	for _, probe := range probes {
		detected, err := probe.Detect(ctx, entity)
		if err != nil {
			// Log error but continue with other probes
			continue
		}

		if detected {
			entity.AddCapability(probe.Capability())
		}
	}

	return nil
}

// MetricsProbe detects if entity can provide metrics
type MetricsProbe struct{}

func (p *MetricsProbe) Capability() Capability {
	return CapabilityMetrics
}

func (p *MetricsProbe) Detect(ctx context.Context, entity *Entity) (bool, error) {
	// Check if entity has metrics endpoint or agent
	if endpoint, exists := entity.Metadata["metrics_endpoint"]; exists && endpoint != "" {
		return true, nil
	}

	// Check for common metrics ports
	if entity.Endpoint != "" {
		// In real implementation, would probe for Prometheus, StatsD, etc.
		return true, nil
	}

	return false, nil
}

// LogsProbe detects if entity can provide logs
type LogsProbe struct{}

func (p *LogsProbe) Capability() Capability {
	return CapabilityLogs
}

func (p *LogsProbe) Detect(ctx context.Context, entity *Entity) (bool, error) {
	// Check if entity has log endpoint or file path
	if logPath, exists := entity.Metadata["log_path"]; exists && logPath != "" {
		return true, nil
	}

	// Most servers and APIs can provide logs
	if entity.Type == EntityTypeServer || entity.Type == EntityTypeAPI {
		return true, nil
	}

	return false, nil
}

// CommandProbe detects if entity can execute commands
type CommandProbe struct{}

func (p *CommandProbe) Capability() Capability {
	return CapabilityCommands
}

func (p *CommandProbe) Detect(ctx context.Context, entity *Entity) (bool, error) {
	// Check if entity has SSH or API access
	if entity.Endpoint != "" {
		// In real implementation, would test SSH or API connectivity
		return true, nil
	}

	return false, nil
}

// HealthCheckProbe detects if entity supports health checks
type HealthCheckProbe struct{}

func (p *HealthCheckProbe) Capability() Capability {
	return CapabilityHealthChecks
}

func (p *HealthCheckProbe) Detect(ctx context.Context, entity *Entity) (bool, error) {
	// Check for health check endpoint
	if healthEndpoint, exists := entity.Metadata["health_endpoint"]; exists && healthEndpoint != "" {
		return true, nil
	}

	// Most modern services support health checks
	return true, nil
}

// TracesProbe detects if entity can provide distributed traces
type TracesProbe struct{}

func (p *TracesProbe) Capability() Capability {
	return CapabilityTraces
}

func (p *TracesProbe) Detect(ctx context.Context, entity *Entity) (bool, error) {
	// Check for tracing configuration
	if tracingEnabled, exists := entity.Metadata["tracing_enabled"]; exists && tracingEnabled == true {
		return true, nil
	}

	// Check for common tracing endpoints (Jaeger, Zipkin, etc.)
	if tracingEndpoint, exists := entity.Metadata["tracing_endpoint"]; exists && tracingEndpoint != "" {
		return true, nil
	}

	return false, nil
}
