package helper

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

func GenerateRandomID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Printf("failed to generate random ID: %v", err)
		return ""
	}

	return hex.EncodeToString(b)
}
