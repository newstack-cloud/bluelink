package pluginlistui

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

type ListTUISuite struct {
	suite.Suite
	tempDir string
	styles  *stylespkg.Styles
}

func TestListTUISuite(t *testing.T) {
	suite.Run(t, new(ListTUISuite))
}

func (s *ListTUISuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "pluginlist-tui-test-*")
	s.Require().NoError(err)
	s.tempDir, err = filepath.EvalSymlinks(tempDir)
	s.Require().NoError(err)

	s.styles = stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
}

func (s *ListTUISuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *ListTUISuite) extractFinalModel(model tea.Model) *MainModel {
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

func (s *ListTUISuite) setupManifest(installedPlugins map[string]*plugins.InstalledPlugin) {
	pluginsDir := plugins.GetPluginsDir()
	err := os.MkdirAll(pluginsDir, 0755)
	s.Require().NoError(err)

	manager := plugins.NewManager(nil, nil)
	manifest := &plugins.PluginManifest{Plugins: installedPlugins}
	err = manager.SaveManifest(manifest)
	s.Require().NoError(err)
}

func (s *ListTUISuite) Test_list_empty_plugins() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		TypeFilter:     "all",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
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

	output := headlessOutput.String()
	s.Contains(output, "No plugins found")
	s.Contains(output, "Total: 0 plugin(s)")
}

func (s *ListTUISuite) Test_list_all_plugins() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	s.setupManifest(map[string]*plugins.InstalledPlugin{
		plugins.DefaultRegistryHost + "/bluelink/aws": {
			ID:           "bluelink/aws@1.0.0",
			Version:      "1.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "abc123",
			InstalledAt:  time.Now(),
			Type:         "provider",
		},
		plugins.DefaultRegistryHost + "/bluelink/celerity": {
			ID:           "bluelink/celerity@2.0.0",
			Version:      "2.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "def456",
			InstalledAt:  time.Now(),
			Type:         "transformer",
			Dependencies: map[string]string{"bluelink/aws": "^1.0.0"},
		},
	})

	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		TypeFilter:     "all",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
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

	output := headlessOutput.String()
	s.Contains(output, "bluelink/aws@1.0.0")
	s.Contains(output, "bluelink/celerity@2.0.0")
	s.Contains(output, "[provider]")
	s.Contains(output, "[transformer]")
	s.Contains(output, "Total: 2 plugin(s)")
}

func (s *ListTUISuite) Test_list_filter_provider() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	s.setupManifest(map[string]*plugins.InstalledPlugin{
		plugins.DefaultRegistryHost + "/bluelink/aws": {
			ID:           "bluelink/aws@1.0.0",
			Version:      "1.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "abc123",
			InstalledAt:  time.Now(),
			Type:         "provider",
		},
		plugins.DefaultRegistryHost + "/bluelink/celerity": {
			ID:           "bluelink/celerity@2.0.0",
			Version:      "2.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "def456",
			InstalledAt:  time.Now(),
			Type:         "transformer",
		},
	})

	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		TypeFilter:     "provider",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
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

	output := headlessOutput.String()
	s.Contains(output, "bluelink/aws@1.0.0")
	s.Contains(output, "(type: provider)")
	s.NotContains(output, "bluelink/celerity")
	s.Contains(output, "Total: 1 plugin(s)")
}

func (s *ListTUISuite) Test_list_filter_transformer() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	s.setupManifest(map[string]*plugins.InstalledPlugin{
		plugins.DefaultRegistryHost + "/bluelink/aws": {
			ID:           "bluelink/aws@1.0.0",
			Version:      "1.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "abc123",
			InstalledAt:  time.Now(),
			Type:         "provider",
		},
		plugins.DefaultRegistryHost + "/bluelink/celerity": {
			ID:           "bluelink/celerity@2.0.0",
			Version:      "2.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "def456",
			InstalledAt:  time.Now(),
			Type:         "transformer",
		},
	})

	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		TypeFilter:     "transformer",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
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

	output := headlessOutput.String()
	s.Contains(output, "bluelink/celerity@2.0.0")
	s.Contains(output, "(type: transformer)")
	s.NotContains(output, "bluelink/aws")
	s.Contains(output, "Total: 1 plugin(s)")
}

func (s *ListTUISuite) Test_list_search_by_name() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	s.setupManifest(map[string]*plugins.InstalledPlugin{
		plugins.DefaultRegistryHost + "/bluelink/aws": {
			ID:           "bluelink/aws@1.0.0",
			Version:      "1.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "abc123",
			InstalledAt:  time.Now(),
			Type:         "provider",
		},
		plugins.DefaultRegistryHost + "/bluelink/gcp": {
			ID:           "bluelink/gcp@1.0.0",
			Version:      "1.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "def456",
			InstalledAt:  time.Now(),
			Type:         "provider",
		},
	})

	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		TypeFilter:     "all",
		Search:         "aws",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
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

	output := headlessOutput.String()
	s.Contains(output, "bluelink/aws@1.0.0")
	s.NotContains(output, "bluelink/gcp")
	s.Contains(output, "Total: 1 plugin(s)")
}

func (s *ListTUISuite) Test_list_shows_dependencies() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	s.setupManifest(map[string]*plugins.InstalledPlugin{
		plugins.DefaultRegistryHost + "/bluelink/celerity": {
			ID:           "bluelink/celerity@2.0.0",
			Version:      "2.0.0",
			RegistryHost: plugins.DefaultRegistryHost,
			Shasum:       "def456",
			InstalledAt:  time.Now(),
			Type:         "transformer",
			Dependencies: map[string]string{
				"bluelink/aws": "^1.0.0",
				"bluelink/gcp": "^2.0.0",
			},
		},
	})

	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		TypeFilter:     "all",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
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

	output := headlessOutput.String()
	s.Contains(output, "Dependencies:")
	s.Contains(output, "bluelink/aws@^1.0.0")
	s.Contains(output, "bluelink/gcp@^2.0.0")
}

func (s *ListTUISuite) Test_list_backward_compatible_manifest() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", pluginsDir)

	// Simulate old manifest without Type or Dependencies fields
	err := os.MkdirAll(pluginsDir, 0755)
	s.Require().NoError(err)

	oldManifest := `{
		"plugins": {
			"registry.bluelink.dev/bluelink/aws": {
				"id": "bluelink/aws@1.0.0",
				"version": "1.0.0",
				"registryHost": "registry.bluelink.dev",
				"shasum": "abc123",
				"installedAt": "2025-01-01T00:00:00Z"
			}
		}
	}`
	err = os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), []byte(oldManifest), 0644)
	s.Require().NoError(err)

	headlessOutput := &bytes.Buffer{}
	model, appErr := NewListApp(ListAppOptions{
		TypeFilter:     "all",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(appErr)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)

	output := headlessOutput.String()
	s.Contains(output, "bluelink/aws@1.0.0")
	s.Contains(output, "[unknown]")
	s.Contains(output, "Total: 1 plugin(s)")
}
