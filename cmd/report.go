package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"uptime-go/internal/models"
	"uptime-go/internal/net/database"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var domainURL string

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate monitoring report",
	Long: `Generate a JSON report of the monitoring status.

Without a URL flag, it reports all monitored sites.
With a URL flag, it provides a detailed report for the specified site, including the last 100 history records.`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.InitializeDatabase()
		if err != nil {
			models.Response{
				Message: "failed to initialize sqlite database",
			}.Print()
			os.Exit(ExitErrorConnection)
		}

		if domainURL == "" {
			var monitor []models.Monitor
			db.DB.Find(&monitor)

			output, err := json.Marshal(monitor)
			if err != nil {
				models.Response{
					Message: "Error while serializing output",
				}.Print()
				os.Exit(1)
			}

			fmt.Print(string(output))
			return
		}

		var monitor models.Monitor
		db.DB.
			Preload("Histories", func(db *gorm.DB) *gorm.DB {
				return db.Order("monitor_histories.created_at DESC").Limit(100)
			}).
			Where("url = ?", domainURL).
			Find(&monitor)

		if monitor.CreatedAt.IsZero() {
			models.Response{
				Message: "Record not found",
			}.Print()
			os.Exit(1)
		}

		// Reverse record
		for i, j := 0, len(monitor.Histories)-1; i < j; i, j = i+1, j-1 {
			monitor.Histories[i], monitor.Histories[j] = monitor.Histories[j], monitor.Histories[i]
		}

		output, err := json.Marshal(monitor)
		if err != nil {
			models.Response{
				Message: "Error while encoding result",
			}.Print()
			os.Exit(1)
		}

		fmt.Print(string(output))
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&domainURL, "url", "u", "", "URL")
}
