package layer1

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MeshNode is the server-side component that runs on each machine.
// It listens for incoming peer connections and responds to requests.
type MeshNode struct {
	id       string
	listenAddr string
	listener net.Listener
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewMeshNode creates a new mesh node server
func NewMeshNode(id, listenAddr string) *MeshNode {
	return &MeshNode{
		id:         id,
		listenAddr: listenAddr,
		stopChan:   make(chan struct{}),
	}
}

// Start begins listening for peer connections
func (n *MeshNode) Start() error {
	ln, err := net.Listen("tcp", n.listenAddr)
	if err != nil {
		return fmt.Errorf("mesh node listen failed: %w", err)
	}
	n.listener = ln

	go n.acceptLoop()
	return nil
}

// Stop shuts down the mesh node
func (n *MeshNode) Stop() {
	close(n.stopChan)
	if n.listener != nil {
		n.listener.Close()
	}
}

func (n *MeshNode) acceptLoop() {
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			select {
			case <-n.stopChan:
				return
			default:
				continue
			}
		}
		go n.handleConn(conn)
	}
}

func (n *MeshNode) handleConn(conn net.Conn) {
	defer conn.Close()

	for {
		msgType, payload, err := readMessage(conn)
		if err != nil {
			return
		}

		switch msgType {
		case MsgPing:
			var ping PingPayload
			if err := decodePayload(payload, &ping); err != nil {
				return
			}
			writeMessage(conn, MsgPong, PongPayload{
				SenderID:  n.id,
				Timestamp: time.Now(),
			})

		case MsgStateRequest:
			var req StateRequestPayload
			if err := decodePayload(payload, &req); err != nil {
				return
			}
			state := n.collectLocalState()
			writeMessage(conn, MsgStateResponse, StateResponsePayload{
				RequestID: req.RequestID,
				Node:      state,
			})

		case MsgCommand:
			var cmd CommandPayload
			if err := decodePayload(payload, &cmd); err != nil {
				return
			}
			output, err := n.executeCommand(cmd)
			resp := CommandRespPayload{
				RequestID: cmd.RequestID,
				NodeID:    n.id,
				Success:   err == nil,
				Output:    output,
			}
			if err != nil {
				resp.Error = err.Error()
			}
			writeMessage(conn, MsgCommandResp, resp)

		case MsgFileChunk:
			var chunk FileChunkPayload
			if err := decodePayload(payload, &chunk); err != nil {
				return
			}
			err := n.writeFileChunk(chunk)
			ack := FileAckPayload{
				TransferID: chunk.TransferID,
				Offset:     chunk.Offset,
				Success:    err == nil,
			}
			if err != nil {
				ack.Error = err.Error()
			}
			writeMessage(conn, MsgFileAck, ack)
		}
	}
}

// collectLocalState gathers OS metrics from the current machine
func (n *MeshNode) collectLocalState() NodeState {
	hostname, _ := os.Hostname()
	state := NodeState{
		ID:       n.id,
		OS:       runtime.GOOS,
		Hostname: hostname,
		Online:   true,
	}

	state.CPU = collectCPU()
	state.Memory = collectMemory()
	state.Disk = collectDisk()
	state.Uptime = collectUptime()

	return state
}

// executeCommand runs a shell command and returns output
func (n *MeshNode) executeCommand(cmd CommandPayload) (string, error) {
	timeout := cmd.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("cmd", "/C", cmd.Command)
	default:
		c = exec.Command("sh", "-c", cmd.Command)
	}

	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out

	if err := c.Run(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// writeFileChunk writes a received file chunk to disk
func (n *MeshNode) writeFileChunk(chunk FileChunkPayload) error {
	flags := os.O_WRONLY | os.O_CREATE
	if chunk.Offset == 0 {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(chunk.Path, flags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteAt(chunk.Data, chunk.Offset); err != nil {
		return err
	}
	return nil
}

// --- Cross-OS metric collection ---

func collectCPU() float64 {
	switch runtime.GOOS {
	case "linux":
		return parseProcStat()
	case "darwin":
		out, err := exec.Command("sh", "-c", "top -l 1 | grep 'CPU usage' | awk '{print $3}' | tr -d '%'").Output()
		if err != nil {
			return 0
		}
		v, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
		return v
	case "windows":
		out, err := exec.Command("wmic", "cpu", "get", "loadpercentage", "/value").Output()
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "LoadPercentage=") {
				v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, "LoadPercentage=")), 64)
				return v
			}
		}
	}
	return 0
}

func parseProcStat() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0
	}
	var vals []float64
	for _, f := range fields[1:] {
		v, _ := strconv.ParseFloat(f, 64)
		vals = append(vals, v)
	}
	if len(vals) < 4 {
		return 0
	}
	idle := vals[3]
	total := 0.0
	for _, v := range vals {
		total += v
	}
	if total == 0 {
		return 0
	}
	return (1 - idle/total) * 100
}

func collectMemory() float64 {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		var total, available float64
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			v, _ := strconv.ParseFloat(fields[1], 64)
			switch fields[0] {
			case "MemTotal:":
				total = v
			case "MemAvailable:":
				available = v
			}
		}
		if total == 0 {
			return 0
		}
		return (1 - available/total) * 100

	case "darwin":
		out, err := exec.Command("sh", "-c", "vm_stat | grep 'Pages free' | awk '{print $3}' | tr -d '.'").Output()
		if err != nil {
			return 0
		}
		free, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
		// page size is 4096 bytes on macOS
		freeBytes := free * 4096
		totalOut, err := exec.Command("sh", "-c", "sysctl hw.memsize | awk '{print $2}'").Output()
		if err != nil {
			return 0
		}
		total, _ := strconv.ParseFloat(strings.TrimSpace(string(totalOut)), 64)
		if total == 0 {
			return 0
		}
		return (1 - freeBytes/total) * 100

	case "windows":
		out, err := exec.Command("wmic", "OS", "get", "FreePhysicalMemory,TotalVisibleMemorySize", "/value").Output()
		if err != nil {
			return 0
		}
		var free, total float64
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "FreePhysicalMemory=") {
				free, _ = strconv.ParseFloat(strings.TrimPrefix(line, "FreePhysicalMemory="), 64)
			}
			if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
				total, _ = strconv.ParseFloat(strings.TrimPrefix(line, "TotalVisibleMemorySize="), 64)
			}
		}
		if total == 0 {
			return 0
		}
		return (1 - free/total) * 100
	}
	return 0
}

func collectDisk() float64 {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'", "get", "Size,FreeSpace", "/value").Output()
		if err != nil {
			return 0
		}
		var free, size float64
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "FreeSpace=") {
				free, _ = strconv.ParseFloat(strings.TrimPrefix(line, "FreeSpace="), 64)
			}
			if strings.HasPrefix(line, "Size=") {
				size, _ = strconv.ParseFloat(strings.TrimPrefix(line, "Size="), 64)
			}
		}
		if size == 0 {
			return 0
		}
		return (1 - free/size) * 100
	default:
		out, err := exec.Command("df", "-k", "/").Output()
		if err != nil {
			return 0
		}
		lines := strings.Split(string(out), "\n")
		if len(lines) < 2 {
			return 0
		}
		fields := strings.Fields(lines[1])
		if len(fields) < 5 {
			return 0
		}
		pct := strings.TrimSuffix(fields[4], "%")
		v, _ := strconv.ParseFloat(pct, 64)
		return v
	}
}

func collectUptime() int64 {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return 0
		}
		fields := strings.Fields(string(data))
		if len(fields) == 0 {
			return 0
		}
		v, _ := strconv.ParseFloat(fields[0], 64)
		return int64(v)
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
		if err != nil {
			return 0
		}
		// Output: { sec = 1234567890, usec = 0 } ...
		parts := strings.Split(string(out), "=")
		if len(parts) < 2 {
			return 0
		}
		sec, _ := strconv.ParseInt(strings.TrimSpace(strings.Split(parts[1], ",")[0]), 10, 64)
		return time.Now().Unix() - sec
	case "windows":
		out, err := exec.Command("wmic", "os", "get", "LastBootUpTime", "/value").Output()
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "LastBootUpTime=") {
				val := strings.TrimPrefix(strings.TrimSpace(line), "LastBootUpTime=")
				// Format: 20240101120000.000000+000
				if len(val) >= 14 {
					t, err := time.Parse("20060102150405", val[:14])
					if err == nil {
						return int64(time.Since(t).Seconds())
					}
				}
			}
		}
	}
	return 0
}
