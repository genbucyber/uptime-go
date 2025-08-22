package configuration

import "fmt"

const (
	OJTGUARDIAN_PATH    = "/etc/ojtguardian"
	OJTGUARDIAN_CONFIG  = OJTGUARDIAN_PATH + "/main.yml"
	PLUGIN_PATH         = "/etc/ojtguardian/plugins/uptime"
	CONFIG_PATH         = PLUGIN_PATH + "/config.yml"
	DB_PATH             = PLUGIN_PATH + "/uptime.db"
	MASTER_SERVER_URL   = "http://localhost:8000"
	INCIDENT_CREATE_URL = MASTER_SERVER_URL + "/api/v1/incidents/add"
)

type AppConfig struct {
	ConfigFile string
	DBFile     string
}

var Config AppConfig

func GetIncidentStatusURL(id uint64) string {
	return fmt.Sprintf("%s/api/v1/incidents/%d/update-status", MASTER_SERVER_URL, id)
}
