package hooks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func Execute(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	
	if err != nil {
		log.Error().Err(err).Str("hook", command).Str("output", outputStr).Msg("Bash hook execution failed")
		return outputStr, fmt.Errorf("Bash hook execution failed: %w", err)
	}

	log.Info().Str("hook", command).Str("output", outputStr).Msg("Bash hook executed successfully")
	return outputStr, nil
}