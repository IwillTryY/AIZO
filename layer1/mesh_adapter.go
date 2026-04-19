package layer1

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MeshAdapter connects multiple machines into a single virtual cluster.
// It implements the Adapter interface and manages persistent TCP connections
// to all peer nodes. Commands and state requests are routed across the mesh.
type MeshAdapter struct {
	*BaseAdapter
	node     *MeshNode          // local server (listens for incoming peers)
	peers    map[string]net.Conn // peerAddr -> connection
	peerStates map[string]NodeState // peerAddr -> last known state
	mu       sync.RWMutex
}

// NewMeshAdapter creates a new mesh adapter
func NewMeshAdapter(config *AdapterConfig) *MeshAdapter {
	capabilities := []AdapterCapability{
		CapabilityReadState,
		CapabilitySendCommand,
		CapabilityStream,
		CapabilityBidirectional,
	}
	config.Type = AdapterTypeMesh

	return &MeshAdapter{
		BaseAdapter: NewBaseAdapter(config, capabilities),
		peers:       make(map[string]net.Conn),
		peerStates:  make(map[string]NodeState),
	}
}

// Connect starts the local mesh node and connects to all configured peers
func (a *MeshAdapter) Connect(ctx context.Context) error {
	// Start local node listener
	listenAddr := a.config.Target
	if listenAddr == "" {
		listenAddr = "0.0.0.0:7777"
	}

	a.node = NewMeshNode(a.config.ID, listenAddr)
	if err := a.node.Start(); err != nil {
		return fmt.Errorf("failed to start mesh node: %w", err)
	}

	// Connect to all configured peers
	peersRaw, _ := a.config.Metadata["peers"].(string)
	if peersRaw != "" {
		for _, peer := range strings.Split(peersRaw, ",") {
			peer = strings.TrimSpace(peer)
			if peer == "" {
				continue
			}
			if err := a.connectPeer(ctx, peer); err != nil {
				// Non-fatal: log and continue
				a.RecordError(fmt.Errorf("peer %s: %w", peer, err))
			}
		}
	}

	a.SetConnected(true)
	a.RecordSuccess()
	a.UpdateHealth(HealthStatusHealthy,
		fmt.Sprintf("mesh node running on %s, %d peers connected", listenAddr, len(a.peers)),
		0)
	return nil
}

// connectPeer establishes a persistent TCP connection to a peer
func (a *MeshAdapter) connectPeer(ctx context.Context, addr string) error {
	dialer := net.Dialer{Timeout: a.config.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	a.mu.Lock()
	a.peers[addr] = conn
	a.mu.Unlock()

	// Background goroutine to keep connection alive and handle reconnects
	go a.maintainPeer(addr)

	return nil
}

// maintainPeer monitors a peer connection and reconnects on failure
func (a *MeshAdapter) maintainPeer(addr string) {
	for {
		a.mu.RLock()
		conn, exists := a.peers[addr]
		a.mu.RUnlock()

		if !exists {
			return
		}

		// Ping every 5 seconds
		time.Sleep(5 * time.Second)

		err := writeMessage(conn, MsgPing, PingPayload{
			SenderID:  a.config.ID,
			Timestamp: time.Now(),
		})
		if err != nil {
			// Connection lost — reconnect
			conn.Close()
			a.mu.Lock()
			delete(a.peers, addr)
			// Mark peer as offline
			if state, ok := a.peerStates[addr]; ok {
				state.Online = false
				a.peerStates[addr] = state
			}
			a.mu.Unlock()

			// Retry with backoff
			for i := 0; i < 5; i++ {
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := a.connectPeer(ctx, addr)
				cancel()
				if err == nil {
					break
				}
			}
		}
	}
}

// Disconnect shuts down the mesh adapter
func (a *MeshAdapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for addr, conn := range a.peers {
		conn.Close()
		delete(a.peers, addr)
	}

	if a.node != nil {
		a.node.Stop()
	}

	a.SetConnected(false)
	a.UpdateHealth(HealthStatusUnknown, "disconnected", 0)
	return nil
}

// ReadState collects state from all connected peers and aggregates it
func (a *MeshAdapter) ReadState(ctx context.Context) (*StateData, error) {
	a.mu.RLock()
	peers := make(map[string]net.Conn, len(a.peers))
	for k, v := range a.peers {
		peers[k] = v
	}
	a.mu.RUnlock()

	// Also collect local state
	localState := a.node.collectLocalState()

	nodes := []NodeState{localState}
	var totalCPU, totalMem, totalDisk float64
	totalCPU += localState.CPU
	totalMem += localState.Memory
	totalDisk += localState.Disk

	// Collect from all peers concurrently
	type result struct {
		addr  string
		state NodeState
		err   error
	}
	results := make(chan result, len(peers))

	for addr, conn := range peers {
		go func(addr string, conn net.Conn) {
			reqID := uuid.New().String()
			err := writeMessage(conn, MsgStateRequest, StateRequestPayload{RequestID: reqID})
			if err != nil {
				results <- result{addr: addr, err: err}
				return
			}

			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			msgType, payload, err := readMessage(conn)
			conn.SetReadDeadline(time.Time{})

			if err != nil || msgType != MsgStateResponse {
				results <- result{addr: addr, err: fmt.Errorf("bad response")}
				return
			}

			var resp StateResponsePayload
			if err := decodePayload(payload, &resp); err != nil {
				results <- result{addr: addr, err: err}
				return
			}

			results <- result{addr: addr, state: resp.Node}
		}(addr, conn)
	}

	// Collect results with timeout
	deadline := time.After(6 * time.Second)
	for i := 0; i < len(peers); i++ {
		select {
		case r := <-results:
			if r.err == nil {
				nodes = append(nodes, r.state)
				totalCPU += r.state.CPU
				totalMem += r.state.Memory
				totalDisk += r.state.Disk
				a.mu.Lock()
				a.peerStates[r.addr] = r.state
				a.mu.Unlock()
			} else {
				// Use last known state marked offline
				a.mu.RLock()
				if last, ok := a.peerStates[r.addr]; ok {
					last.Online = false
					nodes = append(nodes, last)
				}
				a.mu.RUnlock()
			}
		case <-deadline:
			break
		}
	}

	count := float64(len(nodes))
	if count == 0 {
		count = 1
	}

	a.RecordSuccess()

	return &StateData{
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"nodes":        nodes,
			"total_nodes":  len(nodes),
			"online_nodes": countOnline(nodes),
			"avg_cpu":      totalCPU / count,
			"avg_memory":   totalMem / count,
			"avg_disk":     totalDisk / count,
		},
		Metadata: map[string]string{
			"adapter": a.config.ID,
			"type":    "mesh",
		},
	}, nil
}

// SendCommand routes a command to a specific peer or broadcasts to all
func (a *MeshAdapter) SendCommand(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	start := time.Now()

	// Determine target node
	targetNode, _ := req.Metadata["node"]

	a.mu.RLock()
	peers := make(map[string]net.Conn, len(a.peers))
	for k, v := range a.peers {
		peers[k] = v
	}
	a.mu.RUnlock()

	if len(peers) == 0 && targetNode == "" {
		// Run locally
		output, err := a.node.executeCommand(CommandPayload{
			RequestID: req.ID,
			Command:   req.Command,
			Timeout:   req.Timeout,
		})
		if err != nil {
			return &CommandResponse{RequestID: req.ID, Success: false, Error: err, Duration: time.Since(start), Timestamp: time.Now()}, nil
		}
		return &CommandResponse{RequestID: req.ID, Success: true, Output: output, Duration: time.Since(start), Timestamp: time.Now()}, nil
	}

	// Broadcast to all peers (or specific node)
	type cmdResult struct {
		nodeID string
		output string
		err    error
	}
	results := make(chan cmdResult, len(peers)+1)

	for addr, conn := range peers {
		go func(addr string, conn net.Conn) {
			payload := CommandPayload{
				RequestID: req.ID,
				NodeID:    targetNode,
				Command:   req.Command,
				Timeout:   req.Timeout,
			}
			if err := writeMessage(conn, MsgCommand, payload); err != nil {
				results <- cmdResult{nodeID: addr, err: err}
				return
			}

			conn.SetReadDeadline(time.Now().Add(req.Timeout + 5*time.Second))
			msgType, respPayload, err := readMessage(conn)
			conn.SetReadDeadline(time.Time{})

			if err != nil || msgType != MsgCommandResp {
				results <- cmdResult{nodeID: addr, err: fmt.Errorf("bad response")}
				return
			}

			var resp CommandRespPayload
			if err := decodePayload(respPayload, &resp); err != nil {
				results <- cmdResult{nodeID: addr, err: err}
				return
			}

			if !resp.Success {
				results <- cmdResult{nodeID: resp.NodeID, err: fmt.Errorf(resp.Error)}
				return
			}
			results <- cmdResult{nodeID: resp.NodeID, output: resp.Output}
		}(addr, conn)
	}

	// Aggregate outputs
	var combined strings.Builder
	deadline := time.After(req.Timeout + 6*time.Second)
	for i := 0; i < len(peers); i++ {
		select {
		case r := <-results:
			combined.WriteString(fmt.Sprintf("=== %s ===\n%s\n", r.nodeID, r.output))
		case <-deadline:
			break
		}
	}

	a.RecordSuccess()
	return &CommandResponse{
		RequestID: req.ID,
		Success:   true,
		Output:    combined.String(),
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}

// HealthCheck pings all peers and reports latency
func (a *MeshAdapter) HealthCheck(ctx context.Context) (*AdapterHealth, error) {
	a.mu.RLock()
	peers := make(map[string]net.Conn, len(a.peers))
	for k, v := range a.peers {
		peers[k] = v
	}
	a.mu.RUnlock()

	online := 0
	var totalLatency time.Duration

	for addr, conn := range peers {
		start := time.Now()
		err := writeMessage(conn, MsgPing, PingPayload{
			SenderID:  a.config.ID,
			Timestamp: start,
		})
		if err != nil {
			continue
		}

		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		msgType, _, err := readMessage(conn)
		conn.SetReadDeadline(time.Time{})

		if err == nil && msgType == MsgPong {
			online++
			latency := time.Since(start)
			totalLatency += latency
			a.mu.Lock()
			if state, ok := a.peerStates[addr]; ok {
				state.LatencyMs = latency.Milliseconds()
				state.Online = true
				a.peerStates[addr] = state
			}
			a.mu.Unlock()
		}
	}

	status := HealthStatusHealthy
	msg := fmt.Sprintf("%d/%d peers online", online, len(peers))
	var avgLatency time.Duration
	if online > 0 {
		avgLatency = totalLatency / time.Duration(online)
	}
	if online < len(peers) {
		status = HealthStatusDegraded
	}
	if online == 0 && len(peers) > 0 {
		status = HealthStatusUnhealthy
	}

	a.UpdateHealth(status, msg, avgLatency)
	return a.GetHealth(), nil
}

// SendFile transfers a file to all peers or a specific peer
func (a *MeshAdapter) SendFile(ctx context.Context, localPath, remotePath, targetNode string) error {
	a.mu.RLock()
	peers := make(map[string]net.Conn, len(a.peers))
	for k, v := range a.peers {
		peers[k] = v
	}
	a.mu.RUnlock()

	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	transferID := uuid.New().String()
	chunkSize := 64 * 1024 // 64KB chunks

	for addr, conn := range peers {
		if targetNode != "" && !strings.Contains(addr, targetNode) {
			continue
		}

		for offset := 0; offset < len(data); offset += chunkSize {
			end := offset + chunkSize
			if end > len(data) {
				end = len(data)
			}

			chunk := FileChunkPayload{
				TransferID: transferID,
				Path:       remotePath,
				Offset:     int64(offset),
				Data:       data[offset:end],
				Final:      end == len(data),
			}

			if err := writeMessage(conn, MsgFileChunk, chunk); err != nil {
				return fmt.Errorf("send chunk to %s: %w", addr, err)
			}

			// Wait for ack
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			msgType, payload, err := readMessage(conn)
			conn.SetReadDeadline(time.Time{})

			if err != nil || msgType != MsgFileAck {
				return fmt.Errorf("ack from %s: %w", addr, err)
			}

			var ack FileAckPayload
			if err := decodePayload(payload, &ack); err != nil || !ack.Success {
				return fmt.Errorf("chunk rejected by %s: %s", addr, ack.Error)
			}
		}
	}

	return nil
}

func countOnline(nodes []NodeState) int {
	n := 0
	for _, node := range nodes {
		if node.Online {
			n++
		}
	}
	return n
}
