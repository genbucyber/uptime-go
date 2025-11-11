package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the logger.
// It sets up a dual-output logger if a log file path is provided and accessible.
// Otherwise, it falls back to a console-only logger.
func InitLogger(logFilePath string) {
	// Console writer for human-readable output is always used.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.ANSIC,
		FormatLevel: func(i any) string {
			return colorizeLevel(i.(string))
		},
		FormatMessage: func(i any) string {
			if i == nil {
				return ""
			}
			return fmt.Sprintf("> %s", i)
		},
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter) // Always write to console

	if logFilePath != "" {
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			// Log a warning to the console (which is already set up)
			log.Warn().Msgf("Could not create log directory '%s', file logging will be disabled: %v", logDir, err)
		} else {
			logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				// Log a warning to the console
				log.Warn().Msgf("Could not open log file '%s', file logging will be disabled: %v", logFilePath, err)
			} else {
				writers = append(writers, logFile) // Add file writer if successful
			}
		}
	}

	// Create a multi-level writer from all successful writers
	multi := zerolog.MultiLevelWriter(writers...)

	// Global logger configuration
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()

	// Set default global log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// SetLogLevel sets the global logging level.
func SetLogLevel(level string) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		// If the level is invalid, default to Info
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Invalid log level '%s'. Using 'info' level.", level)
		return
	}

	zerolog.SetGlobalLevel(logLevel)
}

// Helper function to colorize console output
func colorizeLevel(level string) string {
	switch strings.ToLower(level) {
	case "debug":
		return "\033[36mDBG\033[0m" // Cyan
	case "info":
		return "\033[32mINF\033[0m" // Green
	case "warn":
		return "\033[33mWRN\033[0m" // Yellow
	case "error":
		return "\033[31mERR\033[0m" // Red
	case "fatal":
		return "\033[35mFTL\033[0m" // Magenta
	case "panic":
		return "\033[41mPNC\033[0m" // Red background
	default:
		return level
	}
}
