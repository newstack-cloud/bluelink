package preflightui

import (
	"context"
	"net/url"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
)

// preflightCheckResultMsg is the internal result of checking plugin satisfaction.
type preflightCheckResultMsg struct {
	unsatisfied  []*plugins.PluginID
	allToInstall []*plugins.PluginID
	manager      *plugins.Manager
}

func checkPluginsCmd(confProvider *config.Provider) tea.Cmd {
	return func() tea.Msg {
		if !isLocalEngine(confProvider) {
			return PreflightSatisfiedMsg{}
		}

		deployConfigFile, _ := confProvider.GetString("deployConfigFile")
		pluginIDs, err := loadDeployConfigPlugins(deployConfigFile)
		if err != nil || len(pluginIDs) == 0 {
			return PreflightSatisfiedMsg{}
		}

		manager := createPluginManager()

		unsatisfied, err := manager.GetUnsatisfiedPlugins(pluginIDs)
		if err != nil {
			return PreflightErrorMsg{Err: err}
		}

		if len(unsatisfied) == 0 {
			return PreflightSatisfiedMsg{}
		}

		allToInstall, err := manager.ResolveDependencies(context.TODO(), unsatisfied)
		if err != nil {
			return PreflightErrorMsg{Err: err}
		}

		return preflightCheckResultMsg{
			unsatisfied:  unsatisfied,
			allToInstall: allToInstall,
			manager:      manager,
		}
	}
}

func loadDeployConfigPlugins(configPath string) ([]*plugins.PluginID, error) {
	configPath = ResolveDeployConfigPath(configPath)
	if configPath == "" {
		return nil, nil
	}

	deployConfig, err := plugins.LoadDeployConfig(configPath)
	if err != nil {
		return nil, err
	}

	if len(deployConfig.Dependencies) == 0 {
		return nil, nil
	}

	return deployConfig.GetPluginIDs()
}

// ResolveDeployConfigPath returns the path to the deploy config file,
// trying the alternate extension (.json <-> .jsonc) if the original
// path does not exist.
func ResolveDeployConfigPath(configPath string) string {
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	if strings.HasSuffix(configPath, ".jsonc") {
		alt := strings.TrimSuffix(configPath, "c")
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	} else if strings.HasSuffix(configPath, ".json") {
		alt := configPath + "c"
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}

	return ""
}

func isLocalEngine(confProvider *config.Provider) bool {
	protocol, _ := confProvider.GetString("connectProtocol")
	if protocol == "unix" {
		return true
	}

	endpoint, _ := confProvider.GetString("engineEndpoint")
	return IsLocalhostEndpoint(endpoint)
}

// IsLocalhostEndpoint returns true if the given endpoint URL refers to
// the local machine (localhost, 127.0.0.1, or ::1).
func IsLocalhostEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func createPluginManager() *plugins.Manager {
	authStore := registries.NewAuthConfigStore()
	tokenStore := registries.NewTokenStore()
	discoveryClient := registries.NewServiceDiscoveryClient()
	registryClient := registries.NewRegistryClient(authStore, tokenStore, discoveryClient)
	return plugins.NewManager(registryClient, discoveryClient)
}
