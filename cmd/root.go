package cmd

import (
	"os"

	"uptime-go/internal/configuration"
	"uptime-go/pkg/log"

	"github.com/spf13/cobra"
)

// Constants for exit codes
const (
	ExitSuccess          = 0
	ExitErrorInvalidArgs = 1
	ExitErrorConnection  = 2
	ExitErrorConfig      = 3
)

var (
	configPath   string
	databasePath string
	logLevel     string
	logPath      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "uptime-go",
	Version: VERSION,
	Short:   "An application to check website uptime",
	Long: `A command-line tool to monitor the uptime of websites.
It provides continuous monitoring of websites defined in the configuration file.

Usage: uptime-go [--config=path/to/uptime.yaml] run`,
	Args: cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		log.InitLogger(logPath)
		log.SetLogLevel(logLevel)

		if err := configuration.Load(configPath); err != nil {
			return err
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
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/uptime-go/config.yml", "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&databasePath, "database", "d", "/var/lib/uptime-go/uptime.db", "Path to database file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logPath, "log-path", "", "Path to log file")
}
