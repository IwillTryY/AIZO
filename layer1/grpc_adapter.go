package layer1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCAdapter implements the Adapter interface for gRPC endpoints
type GRPCAdapter struct {
	*BaseAdapter
	conn   *grpc.ClientConn
}

// NewGRPCAdapter creates a new gRPC adapter
func NewGRPCAdapter(config *AdapterConfig) *GRPCAdapter {
	capabilities := []AdapterCapability{
		CapabilityReadState,
		CapabilitySendCommand,
		CapabilityBidirectional,
	}
	config.Type = AdapterTypeGRPC

	return &GRPCAdapter{
		BaseAdapter: NewBaseAdapter(config, capabilities),
	}
}

// Connect establishes a gRPC connection
func (g *GRPCAdapter) Connect(ctx context.Context) error {
	return g.RetryWithBackoff(ctx, func() error {
		opts := []grpc.DialOption{
			grpc.WithBlock(),
		}

		// Use TLS if configured, otherwise insecure
		if _, ok := g.config.Credentials["tls_cert"]; ok {
			// TODO: load TLS credentials from config
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		} else {
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		dialCtx, cancel := context.WithTimeout(ctx, g.config.Timeout)
		defer cancel()

		conn, err := grpc.DialContext(dialCtx, g.config.Target, opts...)
		if err != nil {
			g.RecordError(err)
			return fmt.Errorf("grpc dial failed: %w", err)
		}

		g.conn = conn
		g.SetConnected(true)
		g.RecordSuccess()
		g.UpdateHealth(HealthStatusHealthy, "connected", 0)
		return nil
	})
}

// Disconnect closes the gRPC connection
func (g *GRPCAdapter) Disconnect(ctx context.Context) error {
	if g.conn != nil {
		err := g.conn.Close()
		g.conn = nil
		g.SetConnected(false)
		g.UpdateHealth(HealthStatusUnknown, "disconnected", 0)
		return err
	}
	return nil
}

// ReadState reads state by invoking a gRPC method configured in metadata
func (g *GRPCAdapter) ReadState(ctx context.Context) (*StateData, error) {
	if g.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()

	// Use the gRPC health check as default state reader
	healthClient := grpc_health_v1.NewHealthClient(g.conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		g.RecordError(err)
		return nil, fmt.Errorf("grpc health check failed: %w", err)
	}

	g.RecordSuccess()
	latency := time.Since(start)
	g.UpdateHealth(HealthStatusHealthy, "state read ok", latency)

	data := map[string]interface{}{
		"status":  resp.Status.String(),
		"latency": latency.Milliseconds(),
	}

	// If a custom method is configured, invoke it
	if method, ok := g.config.Metadata["grpc_method"]; ok {
		methodStr, _ := method.(string)
		if methodStr != "" {
			var reply json.RawMessage
			err := g.conn.Invoke(ctx, methodStr, nil, &reply)
			if err == nil {
				data["response"] = string(reply)
			} else {
				data["method_error"] = err.Error()
			}
		}
	}

	return &StateData{
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  map[string]string{"adapter": g.config.ID, "type": "grpc"},
	}, nil
}

// SendCommand sends a command via gRPC
func (g *GRPCAdapter) SendCommand(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	if g.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()

	// The command string is the gRPC method path (e.g. "/package.Service/Method")
	method := req.Command
	if method == "" {
		return &CommandResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     fmt.Errorf("command must be a gRPC method path"),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Marshal args as the request payload
	payload, _ := json.Marshal(req.Args)

	var reply json.RawMessage
	err := g.conn.Invoke(ctx, method, payload, &reply)

	duration := time.Since(start)

	if err != nil {
		g.RecordError(err)
		return &CommandResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     err,
			Duration:  duration,
			Timestamp: time.Now(),
		}, nil
	}

	g.RecordSuccess()
	return &CommandResponse{
		RequestID: req.ID,
		Success:   true,
		Output:    string(reply),
		Duration:  duration,
		Timestamp: time.Now(),
	}, nil
}

// HealthCheck performs a gRPC health check
func (g *GRPCAdapter) HealthCheck(ctx context.Context) (*AdapterHealth, error) {
	if g.conn == nil {
		return &AdapterHealth{
			Status:    HealthStatusUnhealthy,
			LastCheck: time.Now(),
			Message:   "not connected",
		}, nil
	}

	start := time.Now()
	healthClient := grpc_health_v1.NewHealthClient(g.conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	latency := time.Since(start)

	if err != nil {
		g.RecordError(err)
		health := &AdapterHealth{
			Status:    HealthStatusUnhealthy,
			LastCheck: time.Now(),
			Message:   err.Error(),
			Latency:   latency,
		}
		g.UpdateHealth(HealthStatusUnhealthy, err.Error(), latency)
		return health, nil
	}

	g.RecordSuccess()
	status := HealthStatusHealthy
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		status = HealthStatusDegraded
	}

	g.UpdateHealth(status, resp.Status.String(), latency)
	return g.GetHealth(), nil
}
