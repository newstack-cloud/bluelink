package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/plugininstallui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/pluginlistui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/pluginloginui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/pluginuninstallui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errLoginFailed = errors.New("login failed")
var errInstallFailed = errors.New("install failed")
var errUninstallFailed = errors.New("uninstall failed")
var errListPluginsFailed = errors.New("list plugins failed")

func setupPluginsCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	pluginsCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage plugins and login to plugin registries",
		Long:  `Commands for managing plugins and signing into plugin registries.`,
	}

	setupPluginsLoginCommand(pluginsCmd)
	setupPluginsInstallCommand(pluginsCmd, confProvider)
	setupPluginsUninstallCommand(pluginsCmd)
	setupPluginsListCommand(pluginsCmd, confProvider)

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

	manager := createPluginManager()

	resolvedPlugins, err := manager.ResolveDependencies(context.TODO(), pluginIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve plugin dependencies: %v\n", err)
		return errInstallFailed
	}

	if len(resolvedPlugins) == 0 {
		fmt.Fprintln(os.Stdout, "All plugins are already installed.")
		return nil
	}

	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)

	installApp, err := plugininstallui.NewInstallApp(
		context.Background(),
		plugininstallui.InstallAppOptions{
			PluginIDs:        resolvedPlugins,
			UserRequestedIDs: pluginIDs,
			Manager:          manager,
			Styles:           styles,
			Headless:         headlessMode,
			HeadlessWriter:   os.Stdout,
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

func setupPluginsUninstallCommand(pluginsCmd *cobra.Command) {
	uninstallCmd := &cobra.Command{
		Use:   "uninstall <plugin-id> [plugin-id] ...",
		Short: "Uninstall one or more plugins",
		Long: `Uninstall plugins from the local machine.

Plugin IDs can be specified in the following formats:
  - Default registry: bluelink/aws (resolves to registry.bluelink.dev/bluelink/aws)
  - Custom registry: registry.example.com/my-org/plugin (full host required)

Plugins are removed from the path specified by BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH
environment variable, or the default platform-specific path if not set:
  - Linux/macOS: ~/.bluelink/engine/plugins/bin
  - Windows: %LOCALAPPDATA%\NewStack\Bluelink\engine\plugins

Examples:
  # Uninstall a single plugin
  bluelink plugins uninstall bluelink/aws

  # Uninstall multiple plugins
  bluelink plugins uninstall bluelink/aws bluelink/gcp

  # Uninstall from a custom registry (full host required)
  bluelink plugins uninstall registry.example.com/my-org/custom`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runPluginsUninstall(args)
		},
	}

	pluginsCmd.AddCommand(uninstallCmd)
}

func runPluginsUninstall(args []string) error {
	// Parse plugin IDs from arguments
	var pluginIDs []*plugins.PluginID
	for _, arg := range args {
		id, err := plugins.ParsePluginID(arg)
		if err != nil {
			return fmt.Errorf("invalid plugin ID %q: %w", arg, err)
		}
		pluginIDs = append(pluginIDs, id)
	}

	// Detect if running in a terminal
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal

	// Create styles
	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)

	// Create the uninstall app
	uninstallApp, err := pluginuninstallui.NewUninstallApp(
		pluginuninstallui.UninstallAppOptions{
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

	p := tea.NewProgram(uninstallApp, teaOpts...)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if there was an error in the final model
	switch m := finalModel.(type) {
	case pluginuninstallui.MainModel:
		if m.Error != nil {
			return errUninstallFailed
		}
	case *pluginuninstallui.MainModel:
		if m.Error != nil {
			return errUninstallFailed
		}
	}

	return nil
}

func setupPluginsListCommand(pluginsCmd *cobra.Command, confProvider *config.Provider) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Long: `Lists all plugins installed on the local machine.
Each plugin is displayed with its dependency tree. Transformer plugins
typically depend on provider plugins to deploy concrete resources.

Use --type to filter by plugin type and --search to filter by name.

Examples:
  # List all installed plugins
  bluelink plugins list

  # List only provider plugins
  bluelink plugins list --type provider

  # Search plugins by name
  bluelink plugins list --search "aws"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runPluginsList(confProvider)
		},
	}

	listCmd.PersistentFlags().String(
		"type",
		"all",
		"Filter plugins by type. Allowed values: provider, transformer, all.",
	)
	confProvider.BindPFlag("pluginsListType", listCmd.PersistentFlags().Lookup("type"))
	confProvider.BindEnvVar("pluginsListType", "BLUELINK_CLI_PLUGINS_LIST_TYPE")

	listCmd.PersistentFlags().String(
		"search",
		"",
		"Filter plugins by name (case-insensitive substring match).",
	)
	confProvider.BindPFlag("pluginsListSearch", listCmd.PersistentFlags().Lookup("search"))
	confProvider.BindEnvVar("pluginsListSearch", "BLUELINK_CLI_PLUGINS_LIST_SEARCH")

	pluginsCmd.AddCommand(listCmd)
}

func runPluginsList(confProvider *config.Provider) error {
	typeFilter, _ := confProvider.GetString("pluginsListType")
	search, _ := confProvider.GetString("pluginsListSearch")

	switch typeFilter {
	case "all", "provider", "transformer":
	default:
		return fmt.Errorf(
			"invalid type filter %q: allowed values are provider, transformer, all",
			typeFilter,
		)
	}

	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)

	app, err := pluginlistui.NewListApp(pluginlistui.ListAppOptions{
		TypeFilter:     typeFilter,
		Search:         search,
		Styles:         styles,
		Headless:       headlessMode,
		HeadlessWriter: os.Stdout,
	})
	if err != nil {
		return err
	}

	var teaOpts []tea.ProgramOption
	if headlessMode {
		teaOpts = append(teaOpts, tea.WithoutRenderer(), tea.WithInput(nil))
	} else {
		teaOpts = append(teaOpts, tea.WithAltScreen())
	}

	p := tea.NewProgram(app, teaOpts...)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	switch m := finalModel.(type) {
	case pluginlistui.MainModel:
		if m.Error != nil {
			return errListPluginsFailed
		}
	case *pluginlistui.MainModel:
		if m.Error != nil {
			return errListPluginsFailed
		}
	}

	return nil
}

func createPluginManager() *plugins.Manager {
	authStore := registries.NewAuthConfigStore()
	tokenStore := registries.NewTokenStore()
	discoveryClient := registries.NewServiceDiscoveryClient()
	registryClient := registries.NewRegistryClient(authStore, tokenStore, discoveryClient)
	return plugins.NewManager(registryClient, discoveryClient)
}
