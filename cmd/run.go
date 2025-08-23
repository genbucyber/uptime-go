package cmd

import (
	"fmt"
	"log"
	"os"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(configuration.Config.Monitor) <= 0 {
			return fmt.Errorf("no valid website configurations found in config file")
		}

		if noTimeInLog {
			log.SetFlags(0)
		}

		configs := configuration.Config.Monitor

		var urls []string

		for _, r := range configs {
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
		db.UpsertRecord(configs, "url", &[]string{
			"url",
			"enabled",
			"response_time_threshold",
			"interval",
			"certificate_monitoring",
			"certificate_expired_before",
		})
		db.DB.Where("url IN ?", urls).Find(&configs)

		// Initialize and start monitor
		uptimeMonitor, err := monitor.NewUptimeMonitor(db, configs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing monitor: %v\n", err)
			os.Exit(ExitErrorConfig)
		}

		uptimeMonitor.Start()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&noTimeInLog, "no-time", false, "hide time in log")
}
