package commands

import (
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/cmd/utils"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cobra.AddTemplateFunc("wrappedFlagUsages", utils.WrappedFlagUsages)

	rootCmd := &cobra.Command{
		Use:   "bluelink-manager",
		Short: "Install and manage Bluelink on Unix systems",
		Long: `Bluelink Manager installs, updates, and manages Bluelink components.

Components managed:
  - bluelink CLI
  - Deploy Engine (background service)
  - Blueprint Language Server

Use "bluelink-manager [command] --help" for more information about a command.`,
	}

	rootCmd.SetUsageTemplate(utils.UsageTemplate)
	rootCmd.SetHelpTemplate(utils.HelpTemplate)

	// Add subcommands
	setupInstallCommand(rootCmd)
	setupUpdateCommand(rootCmd)
	setupUninstallCommand(rootCmd)
	setupStatusCommand(rootCmd)
	setupServiceCommands(rootCmd)
	setupSelfUpdateCommand(rootCmd)
	setupVersionCommand(rootCmd)

	return rootCmd
}
