package helper

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
