package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomID(t *testing.T) {
	result := GenerateRandomID()

	assert.Equal(t, len(result), 8)
}
