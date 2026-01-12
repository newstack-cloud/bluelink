package main

import (
	"errors"
	"os"

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
	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		// If it's a sentinel error, exit silently with error code 1
		// (detailed error was already displayed by the TUI)
		if isSilentError(err) {
			os.Exit(1)
		}
		// For other errors, let cobra/log handle the output
		os.Exit(1)
	}
}
