package layer1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPAdapter implements the Adapter interface for HTTP/REST endpoints
type HTTPAdapter struct {
	*BaseAdapter
	client *http.Client
}

// NewHTTPAdapter creates a new HTTP adapter
func NewHTTPAdapter(config *AdapterConfig) *HTTPAdapter {
	capabilities := []AdapterCapability{
		CapabilityReadState,
		CapabilitySendCommand,
		CapabilityBidirectional,
	}

	config.Type = AdapterTypeHTTP

	return &HTTPAdapter{
		BaseAdapter: NewBaseAdapter(config, capabilities),
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Connect establishes connection (validates endpoint is reachable)
func (h *HTTPAdapter) Connect(ctx context.Context) error {
	return h.RetryWithBackoff(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, "HEAD", h.config.Target, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		h.addAuth(req)

		resp, err := h.client.Do(req)
		if err != nil {
			h.RecordError(err)
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			err := fmt.Errorf("connection failed with status: %d", resp.StatusCode)
			h.RecordError(err)
			return err
		}

		h.SetConnected(true)
		h.RecordSuccess()
		h.UpdateHealth(HealthStatusHealthy, "Connected successfully", 0)
		return nil
	})
}

// Disconnect closes the connection
func (h *HTTPAdapter) Disconnect(ctx context.Context) error {
	h.SetConnected(false)
	h.client.CloseIdleConnections()
	return nil
}

// ReadState reads the current state from the HTTP endpoint
func (h *HTTPAdapter) ReadState(ctx context.Context) (*StateData, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", h.config.Target, nil)
	if err != nil {
		h.RecordError(err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	h.addAuth(req)

	resp, err := h.client.Do(req)
	if err != nil {
		h.RecordError(err)
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		err := fmt.Errorf("read state failed with status: %d", resp.StatusCode)
		h.RecordError(err)
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// If not JSON, store as raw string
		data = map[string]interface{}{
			"raw": string(body),
		}
	}

	h.RecordSuccess()
	h.UpdateHealth(HealthStatusHealthy, "State read successfully", time.Since(start))

	return &StateData{
		Timestamp: time.Now(),
		Data:      data,
		Metadata: map[string]string{
			"status_code": fmt.Sprintf("%d", resp.StatusCode),
			"content_type": resp.Header.Get("Content-Type"),
		},
	}, nil
}

// SendCommand sends a command to the HTTP endpoint
func (h *HTTPAdapter) SendCommand(ctx context.Context, cmdReq *CommandRequest) (*CommandResponse, error) {
	start := time.Now()

	method := "POST"
	if m, ok := cmdReq.Args["method"].(string); ok {
		method = m
	}

	var body io.Reader
	if cmdReq.Args != nil {
		jsonData, err := json.Marshal(cmdReq.Args)
		if err != nil {
			h.RecordError(err)
			return nil, fmt.Errorf("failed to marshal args: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	endpoint := h.config.Target
	if cmdReq.Command != "" {
		endpoint = fmt.Sprintf("%s/%s", h.config.Target, cmdReq.Command)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		h.RecordError(err)
		return &CommandResponse{
			RequestID: cmdReq.ID,
			Success:   false,
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	h.addAuth(req)

	resp, err := h.client.Do(req)
	if err != nil {
		h.RecordError(err)
		return &CommandResponse{
			RequestID: cmdReq.ID,
			Success:   false,
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.RecordError(err)
		return &CommandResponse{
			RequestID: cmdReq.ID,
			Success:   false,
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, err
	}

	var output interface{}
	if err := json.Unmarshal(respBody, &output); err != nil {
		output = string(respBody)
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if success {
		h.RecordSuccess()
	} else {
		h.RecordError(fmt.Errorf("command failed with status: %d", resp.StatusCode))
	}

	return &CommandResponse{
		RequestID: cmdReq.ID,
		Success:   success,
		Output:    output,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}

// HealthCheck performs a health check
func (h *HTTPAdapter) HealthCheck(ctx context.Context) (*AdapterHealth, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", h.config.Target, nil)
	if err != nil {
		h.UpdateHealth(HealthStatusUnhealthy, err.Error(), 0)
		return h.GetHealth(), err
	}

	h.addAuth(req)

	resp, err := h.client.Do(req)
	if err != nil {
		h.UpdateHealth(HealthStatusUnhealthy, err.Error(), time.Since(start))
		return h.GetHealth(), err
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode >= 500 {
		h.UpdateHealth(HealthStatusUnhealthy, fmt.Sprintf("Server error: %d", resp.StatusCode), latency)
	} else if resp.StatusCode >= 400 {
		h.UpdateHealth(HealthStatusDegraded, fmt.Sprintf("Client error: %d", resp.StatusCode), latency)
	} else {
		h.UpdateHealth(HealthStatusHealthy, "Endpoint is healthy", latency)
	}

	return h.GetHealth(), nil
}

// addAuth adds authentication to the request
func (h *HTTPAdapter) addAuth(req *http.Request) {
	if token, ok := h.config.Credentials["bearer_token"]; ok {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if apiKey, ok := h.config.Credentials["api_key"]; ok {
		req.Header.Set("X-API-Key", apiKey)
	} else if username, ok := h.config.Credentials["username"]; ok {
		if password, ok := h.config.Credentials["password"]; ok {
			req.SetBasicAuth(username, password)
		}
	}
}
