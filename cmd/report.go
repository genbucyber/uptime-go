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
			fmt.Printf("failed to initialize database: %v\n", err)
			os.Exit(ExitErrorConnection)
		}

		if domainURL == "" {
			var monitor []models.Monitor
			db.DB.Find(&monitor)

			output, err := json.Marshal(monitor)
			if err != nil {
				fmt.Println("error while serializing output")
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
			fmt.Printf("%s: record not found\n", domainURL)
			os.Exit(1)
		}

		output, err := json.Marshal(monitor)
		if err != nil {
			fmt.Printf("%s: error while encoding result\n", domainURL)
			os.Exit(1)
		}

		fmt.Print(string(output))
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&domainURL, "url", "u", "", "URL")
}
