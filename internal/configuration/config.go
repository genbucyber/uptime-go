package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"uptime-go/internal/helper"
	"uptime-go/internal/models"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	OJTGUARDIAN_PATH   = "/etc/ojtguardian"
	OJTGUARDIAN_CONFIG = OJTGUARDIAN_PATH + "/main.yml"
)

type MonitorConfig struct {
	URL                      string `mapstructure:"url" yaml:"url" json:"url"`
	Enabled                  bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	Interval                 string `mapstructure:"interval" yaml:"interval" json:"interval"`
	ResponseTimeThreshold    string `mapstructure:"response_time_threshold" yaml:"response_time_threshold" json:"response_time_threshold"`
	CertificateMonitoring    bool   `mapstructure:"certificate_monitoring" yaml:"certificate_monitoring" json:"certificate_monitoring"`
	CertificateExpiredBefore string `mapstructure:"certificate_expired_before" yaml:"certificate_expired_before" json:"certificate_expired_before"`

	// Retry configuration
	MaxRetries    int    `mapstructure:"max_retries" yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	RetryInterval string `mapstructure:"retry_interval" yaml:"retry_interval,omitempty" json:"retry_interval,omitempty"`

	// Granular timeout configuration
	DNSTimeout            string `mapstructure:"dns_timeout" yaml:"dns_timeout,omitempty" json:"dns_timeout,omitempty"`
	DialTimeout           string `mapstructure:"dial_timeout" yaml:"dial_timeout,omitempty" json:"dial_timeout,omitempty"`
	TLSHandshakeTimeout   string `mapstructure:"tls_handshake_timeout" yaml:"tls_handshake_timeout,omitempty" json:"tls_handshake_timeout,omitempty"`
	ResponseHeaderTimeout string `mapstructure:"response_header_timeout" yaml:"response_header_timeout,omitempty" json:"response_header_timeout,omitempty"`
	FollowRedirects       *bool  `mapstructure:"follow_redirects" yaml:"follow_redirects" json:"follow_redirects"`
}

type AppConfig struct {
	Agent struct {
		MasterHost string `yaml:"master_host" mapstructure:"master_host"`
		Auth       struct {
			Token string
		}
	}

	Monitor []*models.Monitor
}

var Config AppConfig

func GetIncidentCreateURL() string {
	return Config.Agent.MasterHost + "/api/v1/incidents/add"
}

func GetIncidentStatusURL(id uint64) string {
	return fmt.Sprintf("%s/api/v1/incidents/%d/update-status", Config.Agent.MasterHost, id)
}

func Load(configPath string) error {
	// Load agent config
	agentConfig := viper.New()
	agentConfig.SetConfigFile(OJTGUARDIAN_CONFIG)
	agentConfig.SetConfigType("yaml")
	if err := agentConfig.ReadInConfig(); err != nil {
		return err
	}

	if err := agentConfig.Unmarshal(&Config.Agent); err != nil {
		return err
	}

	// Load monitor config
	if !filepath.IsAbs(configPath) {
		absPath, err := filepath.Abs(configPath)
		if err == nil {
			configPath = absPath
		}
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		log.Error().Err(err).Msg("failed to create configuration directory")
		return err
	}

	monitorConfig := viper.New()
	monitorConfig.SetConfigFile(configPath)
	monitorConfig.SetConfigType("yml")

	if err := monitorConfig.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		log.Info().Msg("config file created with default site")
		setDefaultMonitor(monitorConfig)
	}

	var rawMonitor []MonitorConfig

	if err := monitorConfig.UnmarshalKey("monitor", &rawMonitor); err != nil {
		return err
	}

	if len(rawMonitor) <= 0 {
		log.Info().Msg("no sites to monitor, adding default site...")
		setDefaultMonitor(monitorConfig)
		if err := monitorConfig.UnmarshalKey("monitor", &rawMonitor); err != nil {
			return err
		}
	}

	// Parse
	for _, monitor := range rawMonitor {
		if monitor.URL == "" {
			log.Warn().Msg("found record with empty url")
			continue
		}

		URL := helper.NormalizeURL(monitor.URL)
		interval := helper.ParseDuration(monitor.Interval, "5m")
		timeout := helper.ParseDuration(monitor.ResponseTimeThreshold, "30s")
		certificateExpiredBefore := helper.ParseDuration(monitor.CertificateExpiredBefore, "31d")
		followRedirects := true
		if monitor.FollowRedirects != nil {
			followRedirects = *monitor.FollowRedirects
		}

		// Parse retry configuration
		maxRetries := monitor.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3 // Default to 3 retries
		}
		retryInterval := helper.ParseDuration(monitor.RetryInterval, "60s")

		// Parse granular timeouts
		dnsTimeout := helper.ParseDuration(monitor.DNSTimeout, "5s")
		dialTimeout := helper.ParseDuration(monitor.DialTimeout, "10s")
		tlsTimeout := helper.ParseDuration(monitor.TLSHandshakeTimeout, "10s")
		headerTimeout := helper.ParseDuration(monitor.ResponseHeaderTimeout, "20s")

		Config.Monitor = append(Config.Monitor, &models.Monitor{
			URL:                      URL,
			Enabled:                  monitor.Enabled,
			Interval:                 interval,
			ResponseTimeThreshold:    timeout,
			CertificateMonitoring:    monitor.CertificateMonitoring,
			CertificateExpiredBefore: &certificateExpiredBefore,
			FollowRedirects:          followRedirects,
			MaxRetries:               maxRetries,
			RetryInterval:            retryInterval,
			DNSTimeout:               dnsTimeout,
			DialTimeout:              dialTimeout,
			TLSHandshakeTimeout:      tlsTimeout,
			ResponseHeaderTimeout:    headerTimeout,
		})
	}

	return nil
}

func UpdateConfig(configPath string, jsonConfig []byte) error {
	var config struct {
		Monitor []MonitorConfig `json:"monitor"`
	}

	if err := json.Unmarshal(jsonConfig, &config); err != nil {
		return fmt.Errorf("error while decoding config: %w", err)
	}

	yamlConfig, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling to YAML: %w", err)
	}

	err = os.WriteFile(configPath, yamlConfig, 0644)
	if err != nil {
		return fmt.Errorf("error writing YAML file: %w", err)
	}

	return nil
}

func setDefaultMonitor(v *viper.Viper) error {
	followRedirects := true
	v.Set("monitor", []MonitorConfig{
		{
			URL:                   "https://genbucyber.com",
			Enabled:               true,
			Interval:              "5m",
			ResponseTimeThreshold: "10s",
			FollowRedirects:       &followRedirects,
		},
	})

	if err := v.WriteConfig(); err != nil {
		return err
	}

	return nil
}
