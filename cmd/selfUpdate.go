package cmd

import (
	"errors"
	"runtime"
	"uptime-go/internal/selfupdate"

	"github.com/spf13/cobra"
)

var (
	dryRun bool
	force  bool
)

// selfUpdateCmd represents the selfUpdate command
var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update the uptime-go binary to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS != "linux" {
			return errors.New("self-update is only supported on Linux")
		}

		return selfupdate.Run(VERSION, dryRun, force)
	},
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
	selfUpdateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without replacing the binary")
	selfUpdateCmd.Flags().BoolVar(&force, "force", false, "Skip delete confirmation")
}
