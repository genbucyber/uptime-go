/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"uptime-go/internal/configuration"
	"uptime-go/internal/monitor"

	"github.com/spf13/cobra"
)

type AppConfig struct {
	ConfigFile string
}

var Config AppConfig

// Constants for exit codes
const (
	ExitSuccess          = 0
	ExitErrorInvalidArgs = 1
	ExitErrorConnection  = 2
	ExitErrorConfig      = 3

	ConfigPath = "/var/uptime-go/etc/uptime.yml" // Default config file path
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uptime-go",
	Short: "An application to check website uptime",
	Long: `A command-line tool to monitor the uptime of websites.
It provides continuous monitoring of websites defined in the configuration file.

Usage: uptime-go [--config=path/to/uptime.yaml]`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runMonitorMode()
	},
}

// runMonitorMode reads the configuration file and starts continuous monitoring
func runMonitorMode() {
	if Config.ConfigFile == "" {
		Config.ConfigFile = ConfigPath
	}

	// Ensure config file is absolute
	if !filepath.IsAbs(Config.ConfigFile) {
		absPath, err := filepath.Abs(Config.ConfigFile)
		if err == nil {
			Config.ConfigFile = absPath
		}
	}

	// Read configuration
	fmt.Printf("Loading configuration from %s\n", Config.ConfigFile)
	configReader := configuration.NewConfigReader()
	err := configReader.ReadConfig(Config.ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %v\n", err)
		os.Exit(ExitErrorConfig)
	}

	// Get uptime configuration
	uptimeConfigs, err := configReader.GetUptimeConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing configuration: %v\n", err)
		os.Exit(ExitErrorConfig)
	}

	if len(uptimeConfigs) == 0 {
		fmt.Fprintln(os.Stderr, "No valid website configurations found in config file")
		os.Exit(ExitErrorConfig)
	}

	// Initialize and start monitor
	uptimeMonitor, err := monitor.NewUptimeMonitor(uptimeConfigs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing monitor: %v\n", err)
		os.Exit(ExitErrorConfig)
	}

	uptimeMonitor.Start()
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
	rootCmd.Flags().StringVarP(&Config.ConfigFile, "config", "c", "", "Path to configuration file")
}
