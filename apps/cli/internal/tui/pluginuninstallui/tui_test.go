package pluginuninstallui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type UninstallTUISuite struct {
	suite.Suite
	tempDir string
	styles  *stylespkg.Styles
}

func TestUninstallTUISuite(t *testing.T) {
	suite.Run(t, new(UninstallTUISuite))
}

func (s *UninstallTUISuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "uninstall-tui-test-*")
	s.Require().NoError(err)
	s.tempDir, err = filepath.EvalSymlinks(tempDir)
	s.Require().NoError(err)

	s.styles = stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
}

func (s *UninstallTUISuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *UninstallTUISuite) extractFinalModel(model tea.Model) *MainModel {
	switch m := model.(type) {
	case *MainModel:
		return m
	case MainModel:
		return &m
	default:
		s.FailNow("unexpected model type")
		return nil
	}
}

func (s *UninstallTUISuite) createTestManager() *plugins.Manager {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	return plugins.NewManagerWithPluginsDir(nil, nil, pluginsDir)
}

func (s *UninstallTUISuite) installTestPlugin(manager *plugins.Manager, namespace, name, version string) {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	// Create plugin directory
	pluginDir := filepath.Join(pluginsDir, "bin", namespace, name, version)
	err := os.MkdirAll(pluginDir, 0755)
	s.Require().NoError(err)
	err = os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("binary"), 0755)
	s.Require().NoError(err)

	// Add to manifest
	manifest := &plugins.PluginManifest{
		Plugins: map[string]*plugins.InstalledPlugin{
			plugins.DefaultRegistryHost + "/" + namespace + "/" + name: {
				ID:           namespace + "/" + name + "@" + version,
				Version:      version,
				RegistryHost: plugins.DefaultRegistryHost,
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err = manager.SaveManifest(manifest)
	s.Require().NoError(err)
}

func (s *UninstallTUISuite) Test_successful_uninstall_single_plugin() {
	manager := s.createTestManager()
	s.installTestPlugin(manager, "bluelink", "aws", "1.0.0")

	pluginID := &plugins.PluginID{
		RegistryHost: plugins.DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}

	headlessOutput := &bytes.Buffer{}
	model, err := NewUninstallApp(UninstallAppOptions{
		PluginIDs:      []*plugins.PluginID{pluginID},
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
		Manager:        manager,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)
	s.Equal(completeStage, finalModel.stage)
	s.Equal(1, finalModel.removedCount)

	output := headlessOutput.String()
	s.Contains(output, "bluelink/aws")
	s.Contains(output, "removed")
	s.Contains(output, "Removed: 1")
}

func (s *UninstallTUISuite) Test_uninstall_not_found_plugin() {
	manager := s.createTestManager()

	pluginID := &plugins.PluginID{
		RegistryHost: plugins.DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "nonexistent",
	}

	headlessOutput := &bytes.Buffer{}
	model, err := NewUninstallApp(UninstallAppOptions{
		PluginIDs:      []*plugins.PluginID{pluginID},
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
		Manager:        manager,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)
	s.Equal(completeStage, finalModel.stage)
	s.Equal(0, finalModel.removedCount)
	s.Equal(1, finalModel.notFoundCount)

	output := headlessOutput.String()
	s.Contains(output, "not found")
	s.Contains(output, "Not found: 1")
}

func (s *UninstallTUISuite) Test_uninstall_multiple_plugins() {
	manager := s.createTestManager()
	s.installTestPlugin(manager, "bluelink", "aws", "1.0.0")

	// Add second plugin to manifest
	manifest, err := manager.LoadManifest()
	s.Require().NoError(err)
	manifest.Plugins[plugins.DefaultRegistryHost+"/bluelink/gcp"] = &plugins.InstalledPlugin{
		ID:           "bluelink/gcp@2.0.0",
		Version:      "2.0.0",
		RegistryHost: plugins.DefaultRegistryHost,
		Shasum:       "def456",
		InstalledAt:  time.Now(),
	}
	err = manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Create gcp plugin directory
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	gcpDir := filepath.Join(pluginsDir, "bin", "bluelink", "gcp", "2.0.0")
	err = os.MkdirAll(gcpDir, 0755)
	s.Require().NoError(err)
	err = os.WriteFile(filepath.Join(gcpDir, "plugin"), []byte("binary"), 0755)
	s.Require().NoError(err)

	pluginIDs := []*plugins.PluginID{
		{RegistryHost: plugins.DefaultRegistryHost, Namespace: "bluelink", Name: "aws"},
		{RegistryHost: plugins.DefaultRegistryHost, Namespace: "bluelink", Name: "gcp"},
	}

	headlessOutput := &bytes.Buffer{}
	model, err := NewUninstallApp(UninstallAppOptions{
		PluginIDs:      pluginIDs,
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
		Manager:        manager,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)
	s.Equal(completeStage, finalModel.stage)
	s.Equal(2, finalModel.removedCount)

	output := headlessOutput.String()
	s.Contains(output, "Removed: 2")
}

func (s *UninstallTUISuite) Test_NewUninstallApp_requires_plugins() {
	_, err := NewUninstallApp(UninstallAppOptions{
		PluginIDs:      []*plugins.PluginID{},
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: &bytes.Buffer{},
	})
	s.Error(err)
	s.Contains(err.Error(), "no plugins to uninstall")
}
