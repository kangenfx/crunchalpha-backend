package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	KeyPrefix = "crunch_"
	KeyLength = 32 // bytes (64 hex chars)
)

// GenerateAPIKey creates a new cryptographically secure API key
func GenerateAPIKey() (string, error) {
	b := make([]byte, KeyLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	
	key := KeyPrefix + hex.EncodeToString(b)
	return key, nil
}

// HashAPIKey creates SHA-256 hash of API key for storage
func HashAPIKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

// GetKeyPrefix extracts first 12 chars for display
func GetKeyPrefix(key string) string {
	if len(key) < 12 {
		return key
	}
	return key[:12] + "..."
}

// ValidateKeyFormat checks if key has correct format
func ValidateKeyFormat(key string) bool {
	if len(key) != len(KeyPrefix)+KeyLength*2 {
		return false
	}
	if key[:len(KeyPrefix)] != KeyPrefix {
		return false
	}
	return true
}
