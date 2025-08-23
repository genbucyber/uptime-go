package helper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomID(t *testing.T) {
	result := GenerateRandomID()

	assert.Equal(t, len(result), 8)
}

func TestParseDurationDays(t *testing.T) {
	result := ParseDuration("19d", "1d")

	assert.Equal(t, result, time.Duration(19)*24*time.Hour)
}

func TestParseDurationMinutes(t *testing.T) {
	result := ParseDuration("19m", "1m")

	assert.Equal(t, result, time.Duration(19)*time.Minute)
}

func TestParseDurationDefault(t *testing.T) {
	result := ParseDuration("19M", "19s")

	assert.Equal(t, result, time.Duration(19)*time.Second)
}
