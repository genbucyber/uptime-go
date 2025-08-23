package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"uptime-go/internal/configuration"
	"uptime-go/internal/models"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// setConfigCmd represents the set-config command
var setConfigCmd = &cobra.Command{
	Use:   "set-config",
	Short: "Reads a JSON string, converts it to YAML, and saves it to configuration file",
	Long:  `This command takes a JSON string as an argument, parses it, and writes the configuration to configuration file in YAML format.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonConfig := args[0]

		var config struct {
			Monitor []configuration.MonitorConfig `json:"monitor"`
		}

		if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
			models.Response{
				Message: fmt.Sprintf("Error while decode config: %v", err),
				Data:    jsonConfig,
			}.Print()
			os.Exit(1)
		}

		yamlConfig, err := yaml.Marshal(config)

		if err != nil {
			models.Response{
				Message: fmt.Sprintf("Error marshalling to YAML: %v", err),
				Data:    jsonConfig,
			}.Print()
			os.Exit(1)
		}

		err = os.WriteFile(configuration.Config.ConfigFile, yamlConfig, 0644)
		if err != nil {
			models.Response{
				Message: fmt.Sprintf("Error writing YAML file: %v\n", err),
				Data:    jsonConfig,
			}.Print()
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(setConfigCmd)
}
