package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/plugininstallui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/pluginloginui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errLoginFailed = errors.New("login failed")
var errInstallFailed = errors.New("install failed")

func setupPluginsCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	pluginsCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage plugins and login to plugin registries",
		Long:  `Commands for managing plugins and signing into plugin registries.`,
	}

	setupPluginsLoginCommand(pluginsCmd)
	setupPluginsInstallCommand(pluginsCmd, confProvider)

	rootCmd.AddCommand(pluginsCmd)
}

func setupPluginsLoginCommand(pluginsCmd *cobra.Command) {
	loginCmd := &cobra.Command{
		Use:   "login <registry-host>",
		Short: "Authenticate with a plugin registry",
		Long: `Authenticate with a plugin registry to enable downloading plugins.

The command discovers available authentication methods from the registry's
service discovery endpoint and prompts for credentials accordingly.

Supported authentication methods:
  - API Key: Enter your API key when prompted
  - OAuth2 Client Credentials: Enter client ID and secret when prompted
  - OAuth2 Authorization Code: Complete authentication in your browser

Credentials are stored in:
  - Linux/macOS: $HOME/.bluelink/clients/plugins.auth.json
  - Windows: %LOCALAPPDATA%\NewStack\Bluelink\clients\plugins.auth.json

Examples:
  # Login to a registry (interactive)
  bluelink plugins login registry.example.com

  # Login to the official Bluelink registry
  bluelink plugins login registry.bluelink.dev`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runPluginsLogin(args[0])
		},
	}

	pluginsCmd.AddCommand(loginCmd)
}

func runPluginsLogin(registryHost string) error {
	// Detect if running in a terminal
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal

	// Create styles
	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)

	// Create the login app
	loginApp, err := pluginloginui.NewLoginApp(
		context.Background(),
		pluginloginui.LoginAppOptions{
			RegistryHost:   registryHost,
			Styles:         styles,
			Headless:       headlessMode,
			HeadlessWriter: os.Stdout,
		},
	)
	if err != nil {
		return err
	}

	// Run the TUI
	var teaOpts []tea.ProgramOption
	if headlessMode {
		teaOpts = append(teaOpts, tea.WithoutRenderer(), tea.WithInput(nil))
	} else {
		teaOpts = append(teaOpts, tea.WithAltScreen())
	}

	p := tea.NewProgram(loginApp, teaOpts...)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if there was an error in the final model
	// The model may be returned as either MainModel (value) or *MainModel (pointer)
	// depending on which Update path was taken
	switch m := finalModel.(type) {
	case pluginloginui.MainModel:
		if m.Error != nil {
			return errLoginFailed
		}
	case *pluginloginui.MainModel:
		if m.Error != nil {
			return errLoginFailed
		}
	}

	return nil
}

func setupPluginsInstallCommand(pluginsCmd *cobra.Command, confProvider *config.Provider) {
	installCmd := &cobra.Command{
		Use:   "install [plugin-id@version] ...",
		Short: "Install one or more plugins",
		Long: `Install plugins from a registry to the local machine.

Plugin IDs can be specified in the following formats:
  - Default registry: bluelink/aws (resolves to registry.bluelink.dev/bluelink/aws)
  - With version: bluelink/aws@1.0.0
  - Custom registry: registry.example.com/my-org/plugin@1.0.0 (full host required)

If no plugins are specified, dependencies are read from the deploy config file
(--deploy-config-file flag or BLUELINK_CLI_DEPLOY_CONFIG_FILE env var).

Plugins are installed to the path specified by BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH
environment variable, or the default platform-specific path if not set:
  - Linux/macOS: ~/.bluelink/engine/plugins
  - Windows: %LOCALAPPDATA%\NewStack\Bluelink\engine\plugins

Examples:
  # Install a single plugin from the default registry
  bluelink plugins install bluelink/aws

  # Install a specific version
  bluelink plugins install bluelink/aws@1.0.0

  # Install multiple plugins
  bluelink plugins install bluelink/aws bluelink/gcp

  # Install from a custom registry (full host required)
  bluelink plugins install registry.example.com/my-org/custom@1.0.0

  # Install dependencies from deploy config
  bluelink plugins install

  # Install dependencies from a specific config file
  bluelink plugins install --deploy-config-file=my-config.json`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			deployConfigFile, _ := confProvider.GetString("deployConfigFile")
			return runPluginsInstall(args, deployConfigFile)
		},
	}

	pluginsCmd.AddCommand(installCmd)
}

func runPluginsInstall(args []string, deployConfigFile string) error {
	var pluginIDs []*plugins.PluginID

	if len(args) == 0 {
		// Read from deploy config file (from flag/env/config or default)
		config, err := plugins.LoadDeployConfig(deployConfigFile)
		if err != nil {
			return fmt.Errorf("no plugins specified and failed to load deploy config %q: %w",
				deployConfigFile, err)
		}

		pluginIDs, err = config.GetPluginIDs()
		if err != nil {
			return err
		}

		if len(pluginIDs) == 0 {
			return fmt.Errorf("no dependencies found in %s", deployConfigFile)
		}
	} else {
		// Parse plugin IDs from arguments
		for _, arg := range args {
			id, err := plugins.ParsePluginID(arg)
			if err != nil {
				return fmt.Errorf("invalid plugin ID %q: %w", arg, err)
			}
			pluginIDs = append(pluginIDs, id)
		}
	}

	// Detect if running in a terminal
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal

	// Create styles
	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)

	// Create the install app
	installApp, err := plugininstallui.NewInstallApp(
		context.Background(),
		plugininstallui.InstallAppOptions{
			PluginIDs:      pluginIDs,
			Styles:         styles,
			Headless:       headlessMode,
			HeadlessWriter: os.Stdout,
		},
	)
	if err != nil {
		return err
	}

	// Run the TUI
	var teaOpts []tea.ProgramOption
	if headlessMode {
		teaOpts = append(teaOpts, tea.WithoutRenderer(), tea.WithInput(nil))
	} else {
		teaOpts = append(teaOpts, tea.WithAltScreen())
	}

	p := tea.NewProgram(installApp, teaOpts...)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if there was an error in the final model
	switch m := finalModel.(type) {
	case plugininstallui.MainModel:
		if m.Error != nil {
			return errInstallFailed
		}
	case *plugininstallui.MainModel:
		if m.Error != nil {
			return errInstallFailed
		}
	}

	return nil
}
