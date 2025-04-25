package configuration

import (
	"fmt"
	"time"
	"uptime-go/internal/net/config"

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

	// Set the environment variable prefix
	cr.setDefaults()

	if err := cr.viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

func (c *ConfigReader) setDefaults() {
	c.viper.SetDefault("timeout", "5s")
	c.viper.SetDefault("refresh_interval", "10")
	c.viper.SetDefault("follow_redirects", true)
	c.viper.SetDefault("skip_ssl", false)
}

func (c *ConfigReader) GetUptimeConfig() ([]*config.NetworkConfig, error) {
	var configList []*config.NetworkConfig

	// Get the monitor configurations
	monitors := c.viper.Get("monitor")
	monitorsList, ok := monitors.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid monitor configuration format")
	}

	for _, m := range monitorsList {
		monitor, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		config := &config.NetworkConfig{}

		// Get URL
		if url, ok := monitor["url"].(string); ok {
			config.URL = url
		}

		// Get refresh interval
		if refreshInterval, ok := monitor["refresh_interval"].(int); ok {
			config.RefreshInterval = time.Duration(refreshInterval) * time.Second
		} else {
			config.RefreshInterval = 60 * time.Second // Default refresh interval
		}

		// Get timeout
		if timeout, ok := monitor["timeout"].(int); ok {
			config.Timeout = time.Duration(timeout) * time.Second
		} else {
			config.Timeout = 5 * time.Second // Default timeout
		}

		// Get follow redirects
		if followRedirects, ok := monitor["follow_redirects"].(bool); ok {
			config.FollowRedirects = followRedirects
		} else {
			config.FollowRedirects = true // Default follow redirects
		}

		// Get skip SSL verification
		if skipSSL, ok := monitor["skip_ssl_verification"].(bool); ok {
			config.SkipSSL = skipSSL
		} else {
			config.SkipSSL = false // Default skip SSL
		}

		configList = append(configList, config)
	}

	return configList, nil
}
