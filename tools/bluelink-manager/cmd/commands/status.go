package commands

import (
	"os"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/service"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"github.com/spf13/cobra"
)

func setupStatusCommand(rootCmd *cobra.Command) {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show Bluelink installation and service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}

	rootCmd.AddCommand(statusCmd)
}

func runStatus() error {
	ui.Bold("Bluelink Status")
	ui.Println("================")
	ui.Println()

	ui.Println("Installation: %s", paths.InstallDir())
	ui.Println()

	// Check installed components
	ui.Bold("Components:")
	binDir := paths.BinDir()

	components := []struct {
		name   string
		binary string
	}{
		{"bluelink", "bluelink"},
		{"deploy-engine", "deploy-engine"},
		{"blueprint-ls", "blueprint-ls"},
		{"bluelink-manager", "bluelink-manager"},
	}

	for _, c := range components {
		path := filepath.Join(binDir, c.binary)
		if _, err := os.Stat(path); err == nil {
			ui.Println("  %s: installed", c.name)
		} else {
			ui.PrintRed("  %s: not installed", c.name)
		}
	}

	ui.Println()
	ui.Bold("Service Status:")

	running, err := service.IsRunning()
	if err != nil {
		ui.Println("  Deploy Engine: unknown (%v)", err)
	} else if running {
		ui.PrintGreen("  Deploy Engine: running")
	} else {
		ui.PrintYellow("  Deploy Engine: stopped")
	}

	return nil
}
