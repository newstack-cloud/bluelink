package commands

import (
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/github"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/service"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	cliVersion    string
	engineVersion string
	lsVersion     string
}

func setupUpdateCommand(rootCmd *cobra.Command) {
	opts := &updateOptions{}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update all Bluelink components to latest versions",
		Long: `Update the Bluelink CLI, Deploy Engine, and Blueprint Language Server
to their latest versions.

The Deploy Engine service will be stopped during the update and
restarted afterwards.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(opts)
		},
	}

	updateCmd.Flags().StringVar(&opts.cliVersion, "cli-version", "", "CLI version to install (default: latest)")
	updateCmd.Flags().StringVar(&opts.engineVersion, "engine-version", "", "Deploy Engine version to install (default: latest)")
	updateCmd.Flags().StringVar(&opts.lsVersion, "ls-version", "", "Blueprint LS version to install (default: latest)")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(opts *updateOptions) error {
	ui.PrintHeader("Updating Bluelink")

	// Detect platform
	platform, err := paths.DetectPlatform()
	if err != nil {
		return err
	}

	// Stop service before updating
	ui.Info("Stopping Deploy Engine...")
	_ = service.Stop() // Ignore error if not running

	// Resolve versions
	client := github.NewClient()

	cliVersion := opts.cliVersion
	if cliVersion == "" {
		cliVersion, err = client.GetLatestVersion("apps/cli")
		if err != nil {
			return err
		}
	}

	engineVersion := opts.engineVersion
	if engineVersion == "" {
		engineVersion, err = client.GetLatestVersion("apps/deploy-engine")
		if err != nil {
			return err
		}
	}

	lsVersion := opts.lsVersion
	if lsVersion == "" {
		lsVersion, err = client.GetLatestVersion("tools/blueprint-ls")
		if err != nil {
			return err
		}
	}

	ui.Info("Updating to versions:")
	ui.Info("  CLI:           v%s", cliVersion)
	ui.Info("  Deploy Engine: v%s", engineVersion)
	ui.Info("  Blueprint LS:  v%s", lsVersion)
	ui.Println()

	// Download components
	if err := client.DownloadComponent(
		"Bluelink CLI",
		"apps/cli",
		cliVersion,
		"bluelink",
		"bluelink",
		platform,
	); err != nil {
		return err
	}
	if err := client.DownloadComponent(
		"Deploy Engine",
		"apps/deploy-engine",
		engineVersion,
		"deploy-engine",
		"deploy-engine",
		platform,
	); err != nil {
		return err
	}
	if err := client.DownloadComponent(
		"Blueprint LS",
		"tools/blueprint-ls",
		lsVersion,
		"blueprint-ls",
		"blueprint-ls",
		platform,
	); err != nil {
		return err
	}

	// Start service again
	ui.Info("Starting Deploy Engine...")
	if err := service.Start(); err != nil {
		return err
	}

	ui.Println()
	ui.Success("Bluelink updated successfully!")

	return nil
}
