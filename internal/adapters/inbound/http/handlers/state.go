package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func newState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
