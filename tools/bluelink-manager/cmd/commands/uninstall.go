package commands

import (
	"os"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/service"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"github.com/spf13/cobra"
)

type uninstallOptions struct {
	all bool
}

func setupUninstallCommand(rootCmd *cobra.Command) {
	opts := &uninstallOptions{}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Bluelink binaries",
		Long: `Uninstall Bluelink binaries and stop the Deploy Engine service.

By default, configuration and data are preserved. Use --all to remove
everything including configuration, plugins, and state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(opts)
		},
	}

	uninstallCmd.Flags().BoolVar(&opts.all, "all", false, "Remove everything including config and data")

	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(opts *uninstallOptions) error {
	ui.PrintHeader("Uninstalling Bluelink")

	// Stop and remove service
	ui.Info("Stopping Deploy Engine service...")
	_ = service.Stop()

	ui.Info("Removing Deploy Engine service...")
	_ = service.Uninstall()

	// Remove binaries
	binDir := paths.BinDir()
	binaries := []string{"bluelink", "deploy-engine", "blueprint-ls", "bluelink-manager"}

	for _, binary := range binaries {
		path := filepath.Join(binDir, binary)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			ui.Warn("Failed to remove %s: %v", path, err)
		}
	}
	ui.Success("Removed Bluelink binaries")

	if opts.all {
		// Remove entire install directory
		installDir := paths.InstallDir()
		ui.Info("Removing all data from %s...", installDir)
		if err := os.RemoveAll(installDir); err != nil {
			return err
		}
		ui.Success("Removed all Bluelink data")
	} else {
		ui.Info("Configuration and data preserved in: %s", paths.InstallDir())
		ui.Info("To completely remove: rm -rf %s", paths.InstallDir())
	}

	return nil
}
