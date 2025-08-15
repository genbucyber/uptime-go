package cmd

import (
	"bytes"
	"fmt"
	"os"
	"uptime-go/internal/configuration"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		v := viper.New()
		v.SetConfigType("json")
		if err := v.ReadConfig(bytes.NewBuffer([]byte(jsonConfig))); err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading JSON: %v\n", err)
			os.Exit(1)
		}

		configs := v.Get("configs")
		if configs == nil {
			msg := "Error: 'configs' key not found in the input JSON"
			fmt.Fprintln(os.Stderr, msg)
			fmt.Printf(`"message":"%s"`, msg)
			os.Exit(1)
		}

		yamlData, err := yaml.Marshal(map[string]any{"monitor": configs})

		if err != nil {
			msg := fmt.Sprintf("Error marshalling to YAML: %v\n", err)
			fmt.Fprintln(os.Stderr, msg)
			fmt.Printf(`"message":"%s"`, msg)
			os.Exit(1)
		}

		err = os.WriteFile(configuration.Config.ConfigFile, yamlData, 0644)
		if err != nil {
			msg := fmt.Sprintf("Error writing YAML file: %v\n", err)
			fmt.Fprintln(os.Stderr, msg)
			fmt.Printf(`"message":"%s"`, msg)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(setConfigCmd)
}
