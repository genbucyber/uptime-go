package configuration

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"uptime-go/internal/helper"
	"uptime-go/internal/models"

	"github.com/spf13/viper"
)

const (
	VERSION            = "0.1.2"
	OJTGUARDIAN_PATH   = "/etc/ojtguardian"
	OJTGUARDIAN_CONFIG = OJTGUARDIAN_PATH + "/main.yml"
	PLUGIN_PATH        = OJTGUARDIAN_PATH + "/plugins/uptime"
	CONFIG_PATH        = PLUGIN_PATH + "/config.yml"
	DB_PATH            = PLUGIN_PATH + "/uptime.db"
)

type MonitorConfig struct {
	URL                      string `mapstructure:"url" yaml:"url" json:"url"`
	Enabled                  bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	Interval                 string `mapstructure:"interval" yaml:"interval" json:"interval"`
	ResponseTimeThreshold    string `mapstructure:"response_time_threshold" yaml:"response_time_threshold" json:"response_time_threshold"`
	CertificateMonitoring    bool   `mapstructure:"certificate_monitoring" yaml:"certificate_monitoring" json:"certificate_monitoring"`
	CertificateExpiredBefore string `mapstructure:"certificate_expired_before" yaml:"certificate_expired_before" json:"certificate_expired_before"`
}

type AppConfig struct {
	ConfigFile string
	DBFile     string
	Main       struct {
		MasterHost string `yaml:"master_host" mapstructure:"master_host"`
		Auth       struct {
			Token string
		}
	}

	// Parsed config
	Monitor []*models.Monitor

	// Raw config
	MonitorConfig []MonitorConfig `mapstructure:"monitor" yaml:"monitor" json:"monitor"`
}

var Config AppConfig

func GetIncidentCreateURL() string {
	return Config.Main.MasterHost + "/api/v1/incidents/add"
}

func GetIncidentStatusURL(id uint64) string {
	return fmt.Sprintf("%s/api/v1/incidents/%d/update-status", Config.Main.MasterHost, id)
}

func Load() error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(PLUGIN_PATH, 0755); err != nil {
		fmt.Printf("failed to create directory: %v", err)
		return err
	}

	// Load main config
	vMain := viper.New()
	vMain.SetConfigFile(OJTGUARDIAN_CONFIG)
	vMain.SetConfigType("yaml")
	if err := vMain.ReadInConfig(); err != nil {
		return err
	}

	if err := vMain.Unmarshal(&Config.Main); err != nil {
		return err
	}

	// Ensure monitor config file is absolute
	if !filepath.IsAbs(Config.ConfigFile) {
		absPath, err := filepath.Abs(Config.ConfigFile)
		if err == nil {
			Config.ConfigFile = absPath
		}
	}

	// Load monitor config
	vMonitor := viper.New()
	vMonitor.SetConfigFile(Config.ConfigFile)
	vMonitor.SetConfigType("yaml")
	if err := vMonitor.ReadInConfig(); err != nil {
		return err
	}

	if err := vMonitor.UnmarshalKey("monitor", &Config.MonitorConfig); err != nil {
		return err
	}

	// Parse
	for _, monitor := range Config.MonitorConfig {
		if monitor.URL == "" {
			log.Print("[config] found record with empty url")
			continue
		}

		interval := helper.ParseDuration(monitor.Interval, "5m")
		timeout := helper.ParseDuration(monitor.ResponseTimeThreshold, "10s")
		certificateExpiredBefore := helper.ParseDuration(monitor.CertificateExpiredBefore, "31d")

		Config.Monitor = append(Config.Monitor, &models.Monitor{
			URL:                      monitor.URL,
			Enabled:                  monitor.Enabled,
			Interval:                 interval,
			ResponseTimeThreshold:    timeout,
			CertificateMonitoring:    monitor.CertificateMonitoring,
			CertificateExpiredBefore: &certificateExpiredBefore,
		})
	}

	return nil
}
