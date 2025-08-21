package configuration

import (
	"fmt"
	"uptime-go/internal/models"
)

const (
	OJTGUARDIAN_PATH   = "/etc/ojtguardian"
	OJTGUARDIAN_CONFIG = OJTGUARDIAN_PATH + "/main.yml"
	PLUGIN_PATH        = "/etc/ojtguardian/plugins/uptime"
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
