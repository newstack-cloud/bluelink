package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/newstack-cloud/bluelink/apps/cli/cmd/commands"
)

// Sentinel errors that should exit silently (detailed errors already displayed by TUI)
var silentErrors = []error{
	errors.New("deployment failed"),
	errors.New("destroy failed"),
	errors.New("staging failed"),
	errors.New("state import failed"),
}

func isSilentError(err error) bool {
	for _, sentinelErr := range silentErrors {
		if errors.Is(err, sentinelErr) || err.Error() == sentinelErr.Error() {
			return true
		}
	}
	return false
}

func main() {
	// Establish a root context that is cancelled on interrupt/terminate so
	// long-running engine calls (validation, change staging, deployment) can
	// stop promptly when the user presses Ctrl+C. Commands read this via
	// cmd.Context().
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := commands.NewRootCmd()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// If it's a sentinel error, exit silently with error code 1
		// (detailed error was already displayed by the TUI)
		if isSilentError(err) {
			os.Exit(1)
		}
		// For other errors, let cobra/log handle the output
		os.Exit(1)
	}
}
