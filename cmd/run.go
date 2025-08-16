package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"uptime-go/internal/configuration"
	"uptime-go/internal/helper"
	"uptime-go/internal/monitor"
	"uptime-go/internal/net/database"

	"github.com/spf13/cobra"
)

var noTimeInLog bool

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts the continuous monitoring process for the configured websites",
	Long: `The 'run' command starts the monitoring service.
It loads websites from the configuration and continuously checks their uptime.

Use this command to start the monitoring service.
Example:
  uptime-go run --config /path/to/your/config.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		if noTimeInLog {
			log.SetFlags(0)
		}

		runMonitorMode()
	},
}

// runMonitorMode reads the configuration file and starts continuous monitoring
func runMonitorMode() {
	// Ensure config file is absolute
	if !filepath.IsAbs(configuration.Config.ConfigFile) {
		absPath, err := filepath.Abs(configuration.Config.ConfigFile)
		if err == nil {
			configuration.Config.ConfigFile = absPath
		}
	}

	// Read configuration
	fmt.Printf("Loading configuration from %s\n", configuration.Config.ConfigFile)
	configReader := configuration.NewConfigReader()
	if err := configReader.ReadConfig(configuration.Config.ConfigFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %v\n", err)
		os.Exit(ExitErrorConfig)
	}

	// Get uptime configuration
	uptimeConfigs, err := configReader.ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing configuration: %v\n", err)
		os.Exit(ExitErrorConfig)
	}

	var urls []string

	for _, r := range uptimeConfigs {
		r.ID = helper.GenerateRandomID()
		urls = append(urls, r.URL)
	}

	// Initialize database
	db, err := database.InitializeDatabase()
	if err != nil {
		fmt.Printf("failed to initialize database: %v", err)
		os.Exit(ExitErrorConnection)
	}

	// Merge config
	db.UpsertRecord(uptimeConfigs, "url")
	db.DB.Where("url IN ?", urls).Find(&uptimeConfigs)

	// Initialize and start monitor
	uptimeMonitor, err := monitor.NewUptimeMonitor(db, uptimeConfigs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing monitor: %v\n", err)
		os.Exit(ExitErrorConfig)
	}

	uptimeMonitor.Start()
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&noTimeInLog, "no-time", false, "hide time in log")
}
