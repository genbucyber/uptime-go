package configuration

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"
	"uptime-go/internal/models"

	"github.com/spf13/viper"
)

type ConfigReader struct {
	viper *viper.Viper
}

func NewConfigReader() *ConfigReader {
	return &ConfigReader{
		viper: viper.New(),
	}
}

func (cr *ConfigReader) ReadConfig(filePath string) error {
	// Set the file name and path
	cr.viper.SetConfigFile(filePath)

	// Set the file type
	cr.viper.SetConfigType("yaml")

	if err := cr.viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

func (c *ConfigReader) ParseConfig() ([]*models.Monitor, error) {
	var configList []*models.Monitor

	// Get the monitor configurations
	monitors := c.viper.Get("monitor")
	monitorsList, ok := monitors.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid monitor configuration format")
	}

	for _, m := range monitorsList {
		monitor, ok := m.(map[string]any)
		if !ok {
			continue
		}

		config := &models.Monitor{}

		// Get URL
		if url, ok := monitor["url"].(string); ok {
			config.URL = url
		} else {
			log.Printf("invalid url: %s", url)
			continue
		}

		// Get enabled
		if enabled, ok := monitor["enabled"].(bool); ok {
			config.Enabled = enabled
		} else {
			config.Enabled = true
		}

		// Get refresh interval
		if refreshInterval, ok := monitor["interval"].(string); ok {
			config.Interval = ParseDuration(refreshInterval, "1m")
		} else {
			config.Interval = 60 * time.Second // Default refresh interval
		}

		// Get timeout
		if timeout, ok := monitor["response_time_threshold"].(string); ok {
			config.ResponseTimeThreshold = ParseDuration(timeout, "5s")
		} else {
			config.ResponseTimeThreshold = 5 * time.Second // Default timeout
		}

		// Get skip SSL verification
		if skipSSL, ok := monitor["certificate_monitoring"].(bool); ok {
			config.CertificateMonitoring = skipSSL
		} else {
			config.CertificateMonitoring = false // Default skip SSL
		}

		// Get SSL expired before
		if sslExpired, ok := monitor["certificate_expired_before"].(string); ok {
			expired := ParseDuration(sslExpired, "1M")
			config.CertificateExpiredBefore = &expired
		} else {
			expired := ParseDuration(sslExpired, "1M")
			config.CertificateExpiredBefore = &expired
		}

		configList = append(configList, config)
	}

	return configList, nil
}

func ParseDuration(input string, defaultValue string) time.Duration {
	re := regexp.MustCompile(`(\d+)([smhdM])`)
	matches := re.FindAllStringSubmatch(input, -1)

	if len(matches) == 0 {
		log.Printf("invalid duration string: %s", input)
		return ParseDuration(defaultValue, "1s")
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
		case "M":
			total += time.Duration(value) * 24 * time.Hour * 30
		}
	}

	return total
}
