package commands

import (
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/github"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"github.com/spf13/cobra"
)

func setupSelfUpdateCommand(rootCmd *cobra.Command) {
	selfUpdateCmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update bluelink-manager itself to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpdate()
		},
	}

	rootCmd.AddCommand(selfUpdateCmd)
}

func runSelfUpdate() error {
	ui.Info("Checking for updates...")

	platform, err := paths.DetectPlatform()
	if err != nil {
		return err
	}

	client := github.NewClient()

	latestVersion, err := client.GetLatestVersion("tools/bluelink-manager")
	if err != nil {
		return err
	}

	if latestVersion == Version {
		ui.Info("Already at latest version (%s)", Version)
		return nil
	}

	ui.Info("Updating from v%s to v%s...", Version, latestVersion)

	if err := client.DownloadComponent(
		"bluelink-manager",
		"tools/bluelink-manager",
		latestVersion,
		"bluelink-manager",
		"bluelink-manager",
		platform,
	); err != nil {
		return err
	}

	ui.Success("Updated to bluelink-manager v%s", latestVersion)
	ui.Info("Run 'bluelink-manager version' to verify")

	return nil
}
