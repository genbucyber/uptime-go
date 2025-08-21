package configuration

import (
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
