package helper

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func GenerateRandomID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Error().Err(err).Msg("failed to generate random ID")
		return ""
	}

	return hex.EncodeToString(b)
}

func ParseDuration(input string, defaultValue string) time.Duration {
	re := regexp.MustCompile(`(\d+)([smhd])`)
	matches := re.FindAllStringSubmatch(input, -1)

	if len(matches) == 0 && defaultValue != "" {
		log.Warn().Msgf("invalid duration string: '%s'", input)
		log.Warn().Msgf("using default value: %s", defaultValue)
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

// NormalizeURL cleans and standardizes a URL string.
// It adds a default HTTPS scheme if missing, removes trailing slashes,
// and converts the host to lowercase.
func NormalizeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	// Add default scheme if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to parse URL: %s", rawURL)
		return rawURL // Return original if parsing fails
	}

	// Convert host to lowercase
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	// Remove trailing slash from path if it's just "/"
	if parsedURL.Path == "/" {
		parsedURL.Path = ""
	}

	// Reconstruct the URL
	return parsedURL.String()
}
