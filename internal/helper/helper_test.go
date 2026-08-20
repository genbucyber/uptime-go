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

func TestCalculateMedian(t *testing.T) {
	resSize := []int64{120, 109, 106, 90, 102, 104, 124, 119, 108}
	result := CalculateMedian(resSize)

	assert.Equal(t, result, int64(108))
}

func TestExtractVisibleText(t *testing.T){
	htmlContent := `
		<html>
			<body>
				<!-- login page --!>
				<h1>Welcome to Dashboard</h1>
				<a href="#">Login</a>
				<script>console.log("400 Bad Request")</script>
			</body>
		</html>
	`
	visible := ExtractVisibleText(htmlContent)
	assert.Contains(t, visible, "welcome to dashboard")
	assert.Contains(t, visible, "login")
	assert.NotContains(t, visible, "400 Bad Request")
	assert.NotContains(t, visible, "login page")
}