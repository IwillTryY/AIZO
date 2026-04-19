package layer1

import (
	"sort"
	"sync"
)

// MeshTopology defines how nodes connect
type MeshTopology string

const (
	TopologyFullMesh     MeshTopology = "full"
	TopologyHierarchical MeshTopology = "hierarchical"
)

// MeshCoordinator handles coordinator election and command routing
// in hierarchical topology mode
type MeshCoordinator struct {
	localID       string
	peers         []string // all known peer IDs
	coordinatorID string
	isCoordinator bool
	mu            sync.RWMutex
}

// NewMeshCoordinator creates a new coordinator
func NewMeshCoordinator(localID string) *MeshCoordinator {
	return &MeshCoordinator{
		localID: localID,
		peers:   make([]string, 0),
	}
}

// UpdatePeers updates the known peer list and re-elects coordinator
func (c *MeshCoordinator) UpdatePeers(peerIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.peers = make([]string, len(peerIDs))
	copy(c.peers, peerIDs)
	c.elect()
}

// AddPeer adds a peer and re-elects
func (c *MeshCoordinator) AddPeer(peerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, p := range c.peers {
		if p == peerID {
			return
		}
	}
	c.peers = append(c.peers, peerID)
	c.elect()
}

// RemovePeer removes a peer and re-elects
func (c *MeshCoordinator) RemovePeer(peerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, p := range c.peers {
		if p == peerID {
			c.peers = append(c.peers[:i], c.peers[i+1:]...)
			break
		}
	}
	c.elect()
}

// elect picks the coordinator — deterministic: lowest ID wins
func (c *MeshCoordinator) elect() {
	all := make([]string, 0, len(c.peers)+1)
	all = append(all, c.localID)
	all = append(all, c.peers...)
	sort.Strings(all)

	if len(all) > 0 {
		c.coordinatorID = all[0]
	} else {
		c.coordinatorID = c.localID
	}
	c.isCoordinator = (c.coordinatorID == c.localID)
}

// IsCoordinator returns true if this node is the coordinator
func (c *MeshCoordinator) IsCoordinator() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isCoordinator
}

// CoordinatorID returns the current coordinator's ID
func (c *MeshCoordinator) CoordinatorID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.coordinatorID
}

// RecommendTopology returns the recommended topology based on cluster size
func RecommendTopology(nodeCount int) MeshTopology {
	if nodeCount <= 10 {
		return TopologyFullMesh
	}
	return TopologyHierarchical
}

// ShouldConnectTo returns which peers this node should connect to
// In full mesh: all peers. In hierarchical: only the coordinator.
func (c *MeshCoordinator) ShouldConnectTo(topology MeshTopology) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if topology == TopologyFullMesh {
		result := make([]string, len(c.peers))
		copy(result, c.peers)
		return result
	}

	// Hierarchical: connect only to coordinator (unless we ARE the coordinator)
	if c.isCoordinator {
		// Coordinator connects to all
		result := make([]string, len(c.peers))
		copy(result, c.peers)
		return result
	}

	// Non-coordinator connects only to coordinator
	if c.coordinatorID != "" && c.coordinatorID != c.localID {
		return []string{c.coordinatorID}
	}
	return nil
}
