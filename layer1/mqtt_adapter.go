package layer1

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTAdapter implements the Adapter interface for MQTT brokers
type MQTTAdapter struct {
	*BaseAdapter
	client     mqtt.Client
	lastState  map[string]interface{}
	stateMu    sync.RWMutex
	stateTopic string
	cmdTopic   string
}

// NewMQTTAdapter creates a new MQTT adapter
func NewMQTTAdapter(config *AdapterConfig) *MQTTAdapter {
	capabilities := []AdapterCapability{
		CapabilityReadState,
		CapabilitySendCommand,
		CapabilityStream,
		CapabilityBidirectional,
	}
	config.Type = AdapterTypeMQTT

	stateTopic := "aizo/state/#"
	cmdTopic := "aizo/cmd"
	if t, ok := config.Metadata["state_topic"]; ok {
		stateTopic, _ = t.(string)
	}
	if t, ok := config.Metadata["cmd_topic"]; ok {
		cmdTopic, _ = t.(string)
	}

	return &MQTTAdapter{
		BaseAdapter: NewBaseAdapter(config, capabilities),
		lastState:   make(map[string]interface{}),
		stateTopic:  stateTopic,
		cmdTopic:    cmdTopic,
	}
}

// Connect establishes connection to the MQTT broker
func (m *MQTTAdapter) Connect(ctx context.Context) error {
	return m.RetryWithBackoff(ctx, func() error {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(m.config.Target)
		opts.SetClientID("aizo-" + m.config.ID)
		opts.SetConnectTimeout(m.config.Timeout)
		opts.SetAutoReconnect(true)
		opts.SetCleanSession(true)

		// Auth
		if user, ok := m.config.Credentials["username"]; ok {
			opts.SetUsername(user)
		}
		if pass, ok := m.config.Credentials["password"]; ok {
			opts.SetPassword(pass)
		}

		opts.SetOnConnectHandler(func(c mqtt.Client) {
			// Subscribe to state topic on connect/reconnect
			c.Subscribe(m.stateTopic, 1, m.onStateMessage)
		})

		opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
			m.RecordError(err)
			m.UpdateHealth(HealthStatusUnhealthy, "connection lost: "+err.Error(), 0)
		})

		client := mqtt.NewClient(opts)
		token := client.Connect()

		// Wait with context
		done := make(chan struct{})
		go func() {
			token.Wait()
			close(done)
		}()

		select {
		case <-done:
			if token.Error() != nil {
				m.RecordError(token.Error())
				return fmt.Errorf("mqtt connect failed: %w", token.Error())
			}
		case <-ctx.Done():
			return ctx.Err()
		}

		m.client = client
		m.SetConnected(true)
		m.RecordSuccess()
		m.UpdateHealth(HealthStatusHealthy, "connected", 0)
		return nil
	})
}

// onStateMessage handles incoming state messages
func (m *MQTTAdapter) onStateMessage(client mqtt.Client, msg mqtt.Message) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	var data interface{}
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		m.lastState[msg.Topic()] = string(msg.Payload())
	} else {
		m.lastState[msg.Topic()] = data
	}
}

// Disconnect closes the MQTT connection
func (m *MQTTAdapter) Disconnect(ctx context.Context) error {
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(250)
	}
	m.SetConnected(false)
	m.UpdateHealth(HealthStatusUnknown, "disconnected", 0)
	return nil
}

// ReadState returns the last received state from subscribed topics
func (m *MQTTAdapter) ReadState(ctx context.Context) (*StateData, error) {
	if m.client == nil || !m.client.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	m.stateMu.RLock()
	defer m.stateMu.RUnlock()

	// Copy last state
	data := make(map[string]interface{})
	for k, v := range m.lastState {
		data[k] = v
	}

	m.RecordSuccess()
	return &StateData{
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  map[string]string{"adapter": m.config.ID, "type": "mqtt", "state_topic": m.stateTopic},
	}, nil
}

// SendCommand publishes a command to the MQTT command topic
func (m *MQTTAdapter) SendCommand(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	if m.client == nil || !m.client.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()

	// Build payload
	payload := map[string]interface{}{
		"command": req.Command,
		"args":    req.Args,
		"id":      req.ID,
	}
	data, _ := json.Marshal(payload)

	// Publish to command topic (or custom topic from command string)
	topic := m.cmdTopic
	if req.Command != "" {
		topic = req.Command
	}

	token := m.client.Publish(topic, 1, false, data)

	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-done:
		if token.Error() != nil {
			m.RecordError(token.Error())
			return &CommandResponse{
				RequestID: req.ID,
				Success:   false,
				Error:     token.Error(),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}, nil
		}
	case <-ctx.Done():
		return &CommandResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     ctx.Err(),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	m.RecordSuccess()
	return &CommandResponse{
		RequestID: req.ID,
		Success:   true,
		Output:    fmt.Sprintf("published to %s", topic),
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}

// HealthCheck checks MQTT broker connectivity
func (m *MQTTAdapter) HealthCheck(ctx context.Context) (*AdapterHealth, error) {
	if m.client == nil || !m.client.IsConnected() {
		health := &AdapterHealth{
			Status:    HealthStatusUnhealthy,
			LastCheck: time.Now(),
			Message:   "not connected",
		}
		return health, nil
	}

	m.RecordSuccess()
	m.UpdateHealth(HealthStatusHealthy, "connected", 0)
	return m.GetHealth(), nil
}
