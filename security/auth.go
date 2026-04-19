package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// AuthToken represents a signed authentication token for mesh messages
type AuthToken struct {
	NodeID    string `json:"node_id"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

// SignMessage creates an HMAC-SHA256 signature over a payload
func SignMessage(secret []byte, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyMessage verifies an HMAC-SHA256 signature
func VerifyMessage(secret []byte, payload []byte, signature string) bool {
	expected := SignMessage(secret, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// GenerateToken creates a signed auth token for a node
func GenerateToken(nodeID string, secret []byte) (*AuthToken, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	token := &AuthToken{
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
		Nonce:     hex.EncodeToString(nonce),
	}

	// Sign: nodeID + timestamp + nonce
	payload := fmt.Sprintf("%s:%d:%s", token.NodeID, token.Timestamp, token.Nonce)
	token.Signature = SignMessage(secret, []byte(payload))

	return token, nil
}

// VerifyToken verifies a signed auth token
// Returns error if invalid, expired (>30s), or signature mismatch
func VerifyToken(token *AuthToken, secret []byte) error {
	if token == nil {
		return fmt.Errorf("nil token")
	}

	// Check expiry (30 second window)
	age := time.Since(time.UnixMilli(token.Timestamp))
	if age > 30*time.Second || age < -30*time.Second {
		return fmt.Errorf("token expired (age: %v)", age)
	}

	// Verify signature
	payload := fmt.Sprintf("%s:%d:%s", token.NodeID, token.Timestamp, token.Nonce)
	if !VerifyMessage(secret, []byte(payload), token.Signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// HashPayload returns SHA256 hash of arbitrary data (for integrity checks)
func HashPayload(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
