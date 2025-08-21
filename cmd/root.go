package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"uptime-go/internal/configuration"
	"uptime-go/internal/helper"
	"uptime-go/internal/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Constants for exit codes
const (
	ExitSuccess          = 0
	ExitErrorInvalidArgs = 1
	ExitErrorConnection  = 2
	ExitErrorConfig      = 3
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uptime-go",
	Short: "An application to check website uptime",
	Long: `A command-line tool to monitor the uptime of websites.
It provides continuous monitoring of websites defined in the configuration file.

Usage: uptime-go [--config=path/to/uptime.yaml] run`,
	Args: cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		// Create the directory if it doesn't exist
		if err := os.MkdirAll(configuration.PLUGIN_PATH, 0755); err != nil {
			fmt.Printf("failed to create directory: %v", err)
			return err
		}

		// Load main config
		vMain := viper.New()
		vMain.SetConfigFile(configuration.OJTGUARDIAN_CONFIG)
		vMain.SetConfigType("yaml")
		if err := vMain.ReadInConfig(); err != nil {
			return err
		}

		if err := vMain.Unmarshal(&configuration.Config.Main); err != nil {
			return err
		}

		// Ensure monitor config file is absolute
		if !filepath.IsAbs(configuration.Config.ConfigFile) {
			absPath, err := filepath.Abs(configuration.Config.ConfigFile)
			if err == nil {
				configuration.Config.ConfigFile = absPath
			}
		}

		// Load monitor config
		vMonitor := viper.New()
		vMonitor.SetConfigFile(configuration.Config.ConfigFile)
		vMonitor.SetConfigType("yaml")
		if err := vMonitor.ReadInConfig(); err != nil {
			return err
		}

		if err := vMonitor.UnmarshalKey("monitor", &configuration.Config.MonitorConfig); err != nil {
			return err
		}

		// Parse
		for _, monitor := range configuration.Config.MonitorConfig {
			interval := helper.ParseDuration(monitor.Interval, "5m")
			timeout := helper.ParseDuration(monitor.ResponseTimeThreshold, "10s")
			certificateExpiredBefore := helper.ParseDuration(monitor.CertificateExpiredBefore, "31d")

			configuration.Config.Monitor = append(configuration.Config.Monitor, &models.Monitor{
				URL:                      monitor.URL,
				Enabled:                  monitor.Enabled,
				Interval:                 interval,
				ResponseTimeThreshold:    timeout,
				CertificateMonitoring:    monitor.CertificateMonitoring,
				CertificateExpiredBefore: &certificateExpiredBefore,
			})
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configuration.Config.ConfigFile, "config", "c", configuration.CONFIG_PATH, "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&configuration.Config.DBFile, "database", "", configuration.DB_PATH, "Path to database file")
}
