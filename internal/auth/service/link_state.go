package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LinkState represents the state data for OAuth linking
type LinkState struct {
	UserID      string    `json:"user_id"`
	Provider    string    `json:"provider"`
	RedirectURI string    `json:"redirect_uri"`
	Nonce       string    `json:"nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// LinkStateManager handles stateless OAuth linking state via signed tokens
type LinkStateManager struct {
	secret []byte
}

// NewLinkStateManager creates a new state manager
func NewLinkStateManager(secret []byte) *LinkStateManager {
	return &LinkStateManager{
		secret: secret,
	}
}

// GenerateState creates a signed state token for linking
func (m *LinkStateManager) GenerateState(userID, provider, redirectURI string) (string, error) {
	// Generate random nonce
	nonce, err := generateNonce()
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	state := LinkState{
		UserID:      userID,
		Provider:    provider,
		RedirectURI: redirectURI,
		Nonce:       nonce,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	// Marshal to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}

	// Base64 encode
	encoded := base64.URLEncoding.EncodeToString(data)

	// Generate HMAC signature
	signature := m.sign(encoded)

	// Combine: encoded.signature
	return fmt.Sprintf("link.%s.%s", encoded, signature), nil
}

// ValidateState verifies and decodes a state token
func (m *LinkStateManager) ValidateState(token string) (*LinkState, error) {
	// Split token
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "link" {
		return nil, fmt.Errorf("invalid state token format")
	}

	encoded := parts[1]
	signature := parts[2]

	// Verify signature
	expectedSignature := m.sign(encoded)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, fmt.Errorf("invalid state signature")
	}

	// Decode base64
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	// Unmarshal JSON
	var state LinkState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Check expiration
	if time.Now().After(state.ExpiresAt) {
		return nil, fmt.Errorf("state token expired")
	}

	return &state, nil
}

// sign generates HMAC signature for data
func (m *LinkStateManager) sign(data string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// generateNonce creates a random nonce
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
