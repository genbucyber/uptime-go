package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"uptime-go/internal/api"
	"uptime-go/internal/configuration"
	"uptime-go/internal/helper"
	"uptime-go/internal/monitor"
	"uptime-go/internal/net"
	"uptime-go/internal/net/database"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	enableAPI bool
	apiBind   string
	apiPort   string
)

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
		// Monitoring Section
		configs := configuration.Config.Monitor

		var urls []string

		for _, r := range configs {
			r.ID = helper.GenerateRandomID()
			urls = append(urls, r.URL)
		}

		// Initialize database
		db, err := database.New(databasePath)
		if err != nil {
			log.Error().Err(err).Str("database_path", databasePath).Msg("Error initializing database")
			return err
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

		go func() {
			uptimeMonitor.Start()
		}()

		go func() {
			log.Info().Msg("fetching ip address...")
			ip, err := net.GetIPAddress()
			if err != nil {
				log.Error().Err(err).Msg("failed to fetch ip address")
				return
			}
			log.Info().Str("ip", ip).Msg("ip fetched successfully")
		}()

		// API Section
		var apiServer *api.Server
		if enableAPI {
			log.Info().Msg("API server enabled, starting...")

			apiServer = api.NewServer(api.ServerConfig{
				Bind:       apiBind,
				Port:       apiPort,
				ConfigPath: configPath,
			}, db)

			go func() {
				if err := apiServer.Start(); err != nil {
					log.Error().Err(err).Msg("API server failed")
				}
			}()
		}

		// Set up signal handling for graceful shutdown
		log.Info().Msg("Press Ctrl+C to stop")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

		// Wait for shutdown signal
		<-sigChan
		log.Info().Msg("Shutdown signal received, shutting down...")

		uptimeMonitor.Shutdown()

		if apiServer != nil {
			apiServer.Shutdown()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// API flags
	runCmd.Flags().BoolVar(&enableAPI, "api", false, "Enable API server for remote management")
	runCmd.Flags().StringVar(&apiPort, "api-port", "5004", "API server port")
	runCmd.Flags().StringVar(&apiBind, "api-bind", "127.0.0.1", "API server bind address")
}
