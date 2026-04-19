package layer1

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

const (
	discoveryAddr = "224.0.0.199:7778"
	beaconInterval = 5 * time.Second
)

// DiscoveryBeacon is broadcast by each node on the LAN
type DiscoveryBeacon struct {
	NodeID     string `json:"node_id"`
	ListenAddr string `json:"listen_addr"`
	OS         string `json:"os"`
	Version    string `json:"version"`
	Timestamp  int64  `json:"timestamp"`
}

// MeshDiscovery handles automatic peer discovery via multicast UDP
type MeshDiscovery struct {
	nodeID     string
	listenAddr string
	version    string
	onDiscover func(beacon DiscoveryBeacon)
	conn       *net.UDPConn
	known      map[string]DiscoveryBeacon
	stopChan   chan struct{}
	mu         sync.RWMutex
}

// NewMeshDiscovery creates a new discovery instance
func NewMeshDiscovery(nodeID, listenAddr, version string, onDiscover func(DiscoveryBeacon)) *MeshDiscovery {
	return &MeshDiscovery{
		nodeID:     nodeID,
		listenAddr: listenAddr,
		version:    version,
		onDiscover: onDiscover,
		known:      make(map[string]DiscoveryBeacon),
		stopChan:   make(chan struct{}),
	}
}

// Start begins broadcasting and listening for beacons
func (d *MeshDiscovery) Start() error {
	addr, err := net.ResolveUDPAddr("udp4", discoveryAddr)
	if err != nil {
		return fmt.Errorf("resolve multicast addr: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		// Multicast not available — fall back silently
		return nil
	}
	d.conn = conn
	conn.SetReadBuffer(8192)

	go d.listenLoop()
	go d.broadcastLoop()

	return nil
}

// Stop shuts down discovery
func (d *MeshDiscovery) Stop() {
	close(d.stopChan)
	if d.conn != nil {
		d.conn.Close()
	}
}

// KnownPeers returns all discovered peers
func (d *MeshDiscovery) KnownPeers() []DiscoveryBeacon {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peers := make([]DiscoveryBeacon, 0, len(d.known))
	for _, b := range d.known {
		peers = append(peers, b)
	}
	return peers
}

func (d *MeshDiscovery) listenLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-d.stopChan:
			return
		default:
		}

		d.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var beacon DiscoveryBeacon
		if err := json.Unmarshal(buf[:n], &beacon); err != nil {
			continue
		}

		// Ignore self
		if beacon.NodeID == d.nodeID {
			continue
		}

		d.mu.Lock()
		_, existed := d.known[beacon.NodeID]
		d.known[beacon.NodeID] = beacon
		d.mu.Unlock()

		// Notify on new peer
		if !existed && d.onDiscover != nil {
			d.onDiscover(beacon)
		}
	}
}

func (d *MeshDiscovery) broadcastLoop() {
	addr, err := net.ResolveUDPAddr("udp4", discoveryAddr)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		select {
		case <-d.stopChan:
			return
		case <-time.After(beaconInterval):
		}

		beacon := DiscoveryBeacon{
			NodeID:     d.nodeID,
			ListenAddr: d.listenAddr,
			OS:         detectOS(),
			Version:    d.version,
			Timestamp:  time.Now().UnixMilli(),
		}

		data, _ := json.Marshal(beacon)
		conn.Write(data)
	}
}

func detectOS() string {
	return runtime.GOOS
}
