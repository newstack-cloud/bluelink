package commands

import (
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/config"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/github"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/plugins"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/service"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/shell"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"github.com/spf13/cobra"
)

type installOptions struct {
	cliVersion    string
	engineVersion string
	lsVersion     string
	noModifyPath  bool
	noService     bool
	noPlugins     bool
	corePlugins   string
	force         bool
}

func setupInstallCommand(rootCmd *cobra.Command) {
	opts := &installOptions{}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Bluelink components",
		Long: `Install the Bluelink CLI, Deploy Engine, and Blueprint Language Server.

This command will:
  1. Download and install all Bluelink components
  2. Configure authentication between CLI and Deploy Engine
  3. Add the bin directory to your PATH
  4. Install and start the Deploy Engine as a background service
  5. Install core plugins (e.g., AWS provider)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(opts)
		},
	}

	installCmd.Flags().StringVar(&opts.cliVersion, "cli-version", "", "CLI version to install (default: latest)")
	installCmd.Flags().StringVar(&opts.engineVersion, "engine-version", "", "Deploy Engine version to install (default: latest)")
	installCmd.Flags().StringVar(&opts.lsVersion, "ls-version", "", "Blueprint LS version to install (default: latest)")
	installCmd.Flags().BoolVar(&opts.noModifyPath, "no-modify-path", false, "Skip PATH modification")
	installCmd.Flags().BoolVar(&opts.noService, "no-service", false, "Skip service installation")
	installCmd.Flags().BoolVar(&opts.noPlugins, "no-plugins", false, "Skip core plugin installation")
	installCmd.Flags().StringVar(
		&opts.corePlugins,
		"core-plugins",
		"newstack-cloud/aws",
		"Comma-separated list of core plugins to install (default: newstack-cloud/aws)",
	)
	installCmd.Flags().BoolVar(&opts.force, "force", false, "Force reinstall/regenerate config")

	rootCmd.AddCommand(installCmd)
}

func runInstall(opts *installOptions) error {
	ui.PrintHeader("Bluelink Installer")

	// Detect platform
	platform, err := paths.DetectPlatform()
	if err != nil {
		return err
	}
	ui.Info("Detected platform: %s_%s", platform.OS, platform.Arch)

	// Create directories
	ui.Info("Creating directories...")
	if err := paths.EnsureDirectories(); err != nil {
		return err
	}

	// Resolve versions
	client := github.NewClient()

	cliVersion := opts.cliVersion
	if cliVersion == "" {
		ui.Info("Fetching latest CLI version...")
		cliVersion, err = client.GetLatestVersion("apps/cli")
		if err != nil {
			return err
		}
	}

	engineVersion := opts.engineVersion
	if engineVersion == "" {
		ui.Info("Fetching latest Deploy Engine version...")
		engineVersion, err = client.GetLatestVersion("apps/deploy-engine")
		if err != nil {
			return err
		}
	}

	lsVersion := opts.lsVersion
	if lsVersion == "" {
		ui.Info("Fetching latest Blueprint LS version...")
		lsVersion, err = client.GetLatestVersion("tools/blueprint-ls")
		if err != nil {
			return err
		}
	}

	ui.Info("Installing versions:")
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

	// Configure authentication
	if err := config.ConfigureAuth(opts.force); err != nil {
		return err
	}

	// Setup PATH
	if !opts.noModifyPath {
		if err := shell.SetupPath(); err != nil {
			return err
		}
	}

	// Install service
	if !opts.noService {
		if err := service.Install(); err != nil {
			return err
		}
	}

	// Install core plugins
	if !opts.noPlugins && opts.corePlugins != "" {
		if err := plugins.InstallCore(opts.corePlugins); err != nil {
			// Non-fatal, just warn
			ui.Warn("Some plugins failed to install: %v", err)
		}
	}

	// Print success
	ui.Println()
	ui.Success("Bluelink installed successfully!")
	ui.Println()
	ui.Info("Installation directory: %s", paths.InstallDir())
	ui.Println()
	ui.PrintNextSteps()

	return nil
}
