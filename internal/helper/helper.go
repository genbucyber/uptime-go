package helper

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"regexp"
	"strconv"
	"time"
)

func GenerateRandomID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Printf("failed to generate random ID: %v", err)
		return ""
	}

	return hex.EncodeToString(b)
}

func ParseDuration(input string, defaultValue string) time.Duration {
	re := regexp.MustCompile(`(\d+)([smhd])`)
	matches := re.FindAllStringSubmatch(input, -1)

	if len(matches) == 0 && defaultValue != "" {
		log.Printf("[helper] invalid duration string: '%s'", input)
		log.Printf("[helper] using default value: %s", defaultValue)
		return ParseDuration(defaultValue, "")
	}

	var total time.Duration
	for _, match := range matches {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "s":
			total += time.Duration(value) * time.Second
		case "m":
			total += time.Duration(value) * time.Minute
		case "h":
			total += time.Duration(value) * time.Hour
		case "d":
			total += time.Duration(value) * 24 * time.Hour
		}
	}

	return total
}
