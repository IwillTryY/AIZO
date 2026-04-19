package layer1

import (
	"context"
	"time"
)

// AdapterType represents the type of adapter
type AdapterType string

const (
	AdapterTypeHTTP      AdapterType = "http"
	AdapterTypeSSH       AdapterType = "ssh"
	AdapterTypeGRPC      AdapterType = "grpc"
	AdapterTypeWebSocket AdapterType = "websocket"
	AdapterTypeMQTT      AdapterType = "mqtt"
	AdapterTypeSNMP      AdapterType = "snmp"
	AdapterTypeMesh      AdapterType = "mesh"
)

// HealthStatus represents the health state of an adapter
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// AdapterCapability represents what an adapter can do
type AdapterCapability string

const (
	CapabilityReadState   AdapterCapability = "read_state"
	CapabilitySendCommand AdapterCapability = "send_command"
	CapabilityStream      AdapterCapability = "stream"
	CapabilityBidirectional AdapterCapability = "bidirectional"
)

// AdapterConfig holds configuration for an adapter
type AdapterConfig struct {
	ID              string
	Type            AdapterType
	Target          string // Connection target (URL, IP, etc.)
	Credentials     map[string]string
	Timeout         time.Duration
	RetryAttempts   int
	RetryBackoff    time.Duration
	HealthCheckInterval time.Duration
	Metadata        map[string]interface{}
}

// AdapterHealth represents the health status of an adapter
type AdapterHealth struct {
	Status      HealthStatus
	LastCheck   time.Time
	Message     string
	Latency     time.Duration
	ErrorCount  int
	SuccessRate float64
}

// CommandRequest represents a command to be sent to a system
type CommandRequest struct {
	ID        string
	Command   string
	Args      map[string]interface{}
	Timeout   time.Duration
	Metadata  map[string]string
}

// CommandResponse represents the response from a command
type CommandResponse struct {
	RequestID  string
	Success    bool
	Output     interface{}
	Error      error
	Duration   time.Duration
	Timestamp  time.Time
}

// StateData represents the state read from a system
type StateData struct {
	Timestamp  time.Time
	Data       map[string]interface{}
	Metadata   map[string]string
}

// Adapter is the core interface that all adapters must implement
type Adapter interface {
	// Connect establishes connection to the target system
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// ReadState reads the current state from the target system
	ReadState(ctx context.Context) (*StateData, error)

	// SendCommand sends a command to the target system
	SendCommand(ctx context.Context, req *CommandRequest) (*CommandResponse, error)

	// HealthCheck performs a health check on the adapter
	HealthCheck(ctx context.Context) (*AdapterHealth, error)

	// GetCapabilities returns what this adapter can do
	GetCapabilities() []AdapterCapability

	// GetConfig returns the adapter configuration
	GetConfig() *AdapterConfig

	// GetType returns the adapter type
	GetType() AdapterType
	GetHealth() *AdapterHealth  // add this line
}

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	config        *AdapterConfig
	health        *AdapterHealth
	capabilities  []AdapterCapability
	connected     bool
	lastError     error
	errorCount    int
	successCount  int
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(config *AdapterConfig, capabilities []AdapterCapability) *BaseAdapter {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 1 * time.Second
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	return &BaseAdapter{
		config:       config,
		capabilities: capabilities,
		health: &AdapterHealth{
			Status:    HealthStatusUnknown,
			LastCheck: time.Now(),
		},
	}
}

// GetCapabilities returns the adapter capabilities
func (b *BaseAdapter) GetCapabilities() []AdapterCapability {
	return b.capabilities
}

// GetConfig returns the adapter configuration
func (b *BaseAdapter) GetConfig() *AdapterConfig {
	return b.config
}

// GetType returns the adapter type
func (b *BaseAdapter) GetType() AdapterType {
	return b.config.Type
}

// UpdateHealth updates the health status
func (b *BaseAdapter) UpdateHealth(status HealthStatus, message string, latency time.Duration) {
	b.health = &AdapterHealth{
		Status:      status,
		LastCheck:   time.Now(),
		Message:     message,
		Latency:     latency,
		ErrorCount:  b.errorCount,
		SuccessRate: b.calculateSuccessRate(),
	}
}

// RecordSuccess records a successful operation
func (b *BaseAdapter) RecordSuccess() {
	b.successCount++
	b.lastError = nil
}

// RecordError records a failed operation
func (b *BaseAdapter) RecordError(err error) {
	b.errorCount++
	b.lastError = err
}

// calculateSuccessRate calculates the success rate
func (b *BaseAdapter) calculateSuccessRate() float64 {
	total := b.successCount + b.errorCount
	if total == 0 {
		return 0
	}
	return float64(b.successCount) / float64(total)
}

// IsConnected returns whether the adapter is connected
func (b *BaseAdapter) IsConnected() bool {
	return b.connected
}

// SetConnected sets the connection status
func (b *BaseAdapter) SetConnected(connected bool) {
	b.connected = connected
}

// GetHealth returns the current health status
func (b *BaseAdapter) GetHealth() *AdapterHealth {
	return b.health
}

// RetryWithBackoff executes a function with retry logic
func (b *BaseAdapter) RetryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= b.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			backoff := b.config.RetryBackoff * time.Duration(attempt)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}
