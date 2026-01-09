package database

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomString generates a secure random hex string of length 2*n
func GenerateRandomString(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
