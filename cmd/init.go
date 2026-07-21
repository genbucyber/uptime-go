package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//go:embed sync.yml
var syncFile string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration and files",
	Long:  `Initialize the configuration file, database, and sync manifest for the uptime monitoring agent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := filepath.Dir(configPath)
		databaseDir := filepath.Dir(databasePath)
		agentDir := "/etc/ojtguardian"
		agentConfigPath := filepath.Join(agentDir, "main.yml")

		log.Info().Msg("Initializing uptime-go configuration...")

		createDir := func(path, name string) error {
			if info, err := os.Stat(path); err == nil {
				if !info.IsDir() {
					return fmt.Errorf("%s path exists but is not a directory: %s", name, path)
				}

				log.Info().
					Str("path", path).
					Msgf("%s directory already exists", name)

				return nil
			}

			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("failed to create %s directory %s: %w", name, path, err)
			}

			log.Info().
				Str("path", path).
				Msgf("%s directory created", name)

			return nil
		}

		writeFile := func(path, name, content string) error {
			if _, err := os.Stat(path); err == nil {
				log.Info().
					Str("path", path).
					Msgf("%s already exists", name)

				return nil
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create %s %s: %w", name, path, err)
			}

			log.Info().
				Str("path", path).
				Msgf("%s created", name)

			return nil
		}

		if err := createDir(configDir, "Config"); err != nil {
			return fmt.Errorf("failed to initialize config directory: %w. Try running the command with 'sudo'", err)
		}

		if err := createDir(databaseDir, "Database"); err != nil {
			return fmt.Errorf("failed to initialize database directory: %w. Try running the command with 'sudo'", err)
		}

		if err := createDir(agentDir, "Agent configuration"); err != nil {
			return fmt.Errorf("failed to initialize agent configuration directory: %w. Try running the command with 'sudo'", err)
		}

		// if err := writeFile(filepath.Join(configDir, "config.yml"), "config.yml", defaultUptimeConfig); err != nil {
		// 	return err
		// }

		if err := writeFile(filepath.Join(configDir, "sync.yml"), "sync.yml", syncFile); err != nil {
			return err
		}

		// if err := writeFile(agentConfigPath, "main.yml", defaultAgentConfig); err != nil {
		// 	return err
		// }

		log.Info().Msg("Initialization completed successfully")

		log.Info().
			Str("command", fmt.Sprintf(
				"sudo ./uptime-go run --config %s --database %s",
				filepath.Join(configDir, "config.yml"),
				databasePath,
			)).
			Msg("Run the following command to start the uptime monitor")

		return nil
	},
}

func runInit() {
	
}


func init() {
	rootCmd.AddCommand(initCmd)
}