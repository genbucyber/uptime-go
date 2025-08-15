package configuration

type AppConfig struct {
	ConfigFile string
	DBFile     string
}

var Config AppConfig
