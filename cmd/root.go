/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"uptime-go/internal/configuration"

	"github.com/spf13/cobra"
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
