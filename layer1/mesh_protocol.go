package layer1

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"time"
)

// Magic bytes identifying AIZO mesh protocol packets
var meshMagic = [4]byte{0xA1, 0xAA, 0xBB, 0xCC}

// ProtocolVersion is the current wire protocol version.
// Increment on breaking changes. Peers reject mismatched versions.
const ProtocolVersion uint8 = 1

// MsgType identifies the type of mesh message
type MsgType byte

const (
	MsgPing          MsgType = 0x01
	MsgPong          MsgType = 0x02
	MsgStateRequest  MsgType = 0x03
	MsgStateResponse MsgType = 0x04
	MsgCommand       MsgType = 0x05
	MsgCommandResp   MsgType = 0x06
	MsgFileChunk     MsgType = 0x07
	MsgFileAck       MsgType = 0x08
)

// meshHeader is the fixed header for every message (10 bytes)
//
//	[0-3]  Magic: 0xA1 0xAA 0xBB 0xCC
//	[4]    Version: protocol version (currently 1)
//	[5]    Type: message type
//	[6-9]  PayloadLen: uint32 big-endian
type meshHeader struct {
	Magic      [4]byte
	Version    uint8
	Type       MsgType
	PayloadLen uint32
}

// NodeState represents the state of a single mesh node
type NodeState struct {
	ID       string  `json:"id"`
	OS       string  `json:"os"`
	Hostname string  `json:"hostname"`
	CPU      float64 `json:"cpu"`
	Memory   float64 `json:"memory"`
	Disk     float64 `json:"disk"`
	Uptime   int64   `json:"uptime"`
	Online   bool    `json:"online"`
	LatencyMs int64  `json:"latency_ms"`
}

// MeshMessage is the top-level message envelope
type MeshMessage struct {
	Type    MsgType
	Payload interface{}
}

// PingPayload is sent to check liveness
type PingPayload struct {
	SenderID  string
	Timestamp time.Time
}

// PongPayload is the response to a ping
type PongPayload struct {
	SenderID  string
	Timestamp time.Time
}

// StateRequestPayload requests node state
type StateRequestPayload struct {
	RequestID string
}

// StateResponsePayload carries node state
type StateResponsePayload struct {
	RequestID string
	Node      NodeState
}

// CommandPayload carries a command to execute
type CommandPayload struct {
	RequestID string
	NodeID    string // empty = this node
	Command   string
	Args      []string
	Timeout   time.Duration
}

// CommandRespPayload carries command output
type CommandRespPayload struct {
	RequestID string
	NodeID    string
	Success   bool
	Output    string
	Error     string
	Duration  time.Duration
}

// FileChunkPayload carries a chunk of file data
type FileChunkPayload struct {
	TransferID string
	Path       string
	Offset     int64
	Data       []byte
	Final      bool
}

// FileAckPayload acknowledges a file chunk
type FileAckPayload struct {
	TransferID string
	Offset     int64
	Success    bool
	Error      string
}

// writeMessage encodes and writes a message to a connection
func writeMessage(conn net.Conn, msgType MsgType, payload interface{}) error {
	// Encode payload with gob
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	// Write header
	hdr := meshHeader{
		Magic:      meshMagic,
		Version:    ProtocolVersion,
		Type:       msgType,
		PayloadLen: uint32(buf.Len()),
	}

	if err := binary.Write(conn, binary.BigEndian, hdr); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write payload
	if _, err := conn.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// readMessage reads and decodes a message from a connection
func readMessage(conn net.Conn) (MsgType, []byte, error) {
	// Read header
	var hdr meshHeader
	if err := binary.Read(conn, binary.BigEndian, &hdr); err != nil {
		return 0, nil, fmt.Errorf("read header: %w", err)
	}

	// Validate magic
	if hdr.Magic != meshMagic {
		return 0, nil, fmt.Errorf("invalid magic bytes")
	}

	// Validate version
	if hdr.Version != ProtocolVersion {
		return 0, nil, fmt.Errorf("protocol version mismatch: got %d, want %d", hdr.Version, ProtocolVersion)
	}

	// Read payload
	if hdr.PayloadLen == 0 {
		return hdr.Type, nil, nil
	}

	payload := make([]byte, hdr.PayloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, nil, fmt.Errorf("read payload: %w", err)
	}

	return hdr.Type, payload, nil
}

// decodePayload decodes a gob-encoded payload into target
func decodePayload(data []byte, target interface{}) error {
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(target)
}

func init() {
	// Register all payload types with gob
	gob.Register(PingPayload{})
	gob.Register(PongPayload{})
	gob.Register(StateRequestPayload{})
	gob.Register(StateResponsePayload{})
	gob.Register(CommandPayload{})
	gob.Register(CommandRespPayload{})
	gob.Register(FileChunkPayload{})
	gob.Register(FileAckPayload{})
}
