package preflightui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/plugininstallui"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type PreflightSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestPreflightSuite(t *testing.T) {
	suite.Run(t, new(PreflightSuite))
}

func (s *PreflightSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
}

// --- IsLocalhostEndpoint tests ---

func (s *PreflightSuite) TestIsLocalhostEndpoint_localhost() {
	s.True(IsLocalhostEndpoint("http://localhost:8325"))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_ipv4_loopback() {
	s.True(IsLocalhostEndpoint("http://127.0.0.1:8325"))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_ipv6_loopback() {
	s.True(IsLocalhostEndpoint("http://[::1]:8325"))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_remote_host() {
	s.False(IsLocalhostEndpoint("http://remote.server:8325"))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_https_remote() {
	s.False(IsLocalhostEndpoint("https://deploy.example.com:8325"))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_invalid_url() {
	s.False(IsLocalhostEndpoint("://not-a-url"))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_empty() {
	s.False(IsLocalhostEndpoint(""))
}

func (s *PreflightSuite) TestIsLocalhostEndpoint_localhost_no_port() {
	s.True(IsLocalhostEndpoint("http://localhost"))
}

// --- ResolveDeployConfigPath tests ---

func (s *PreflightSuite) TestResolveDeployConfigPath_exact_json_exists() {
	dir := s.T().TempDir()
	jsonPath := filepath.Join(dir, "bluelink.deploy.json")
	os.WriteFile(jsonPath, []byte("{}"), 0644)

	s.Equal(jsonPath, ResolveDeployConfigPath(jsonPath))
}

func (s *PreflightSuite) TestResolveDeployConfigPath_exact_jsonc_exists() {
	dir := s.T().TempDir()
	jsoncPath := filepath.Join(dir, "bluelink.deploy.jsonc")
	os.WriteFile(jsoncPath, []byte("{}"), 0644)

	s.Equal(jsoncPath, ResolveDeployConfigPath(jsoncPath))
}

func (s *PreflightSuite) TestResolveDeployConfigPath_json_fallback_to_jsonc() {
	dir := s.T().TempDir()
	jsoncPath := filepath.Join(dir, "bluelink.deploy.jsonc")
	os.WriteFile(jsoncPath, []byte("{}"), 0644)

	jsonPath := filepath.Join(dir, "bluelink.deploy.json")
	s.Equal(jsoncPath, ResolveDeployConfigPath(jsonPath))
}

func (s *PreflightSuite) TestResolveDeployConfigPath_jsonc_fallback_to_json() {
	dir := s.T().TempDir()
	jsonPath := filepath.Join(dir, "bluelink.deploy.json")
	os.WriteFile(jsonPath, []byte("{}"), 0644)

	jsoncPath := filepath.Join(dir, "bluelink.deploy.jsonc")
	s.Equal(jsonPath, ResolveDeployConfigPath(jsoncPath))
}

func (s *PreflightSuite) TestResolveDeployConfigPath_neither_exists() {
	dir := s.T().TempDir()
	jsonPath := filepath.Join(dir, "bluelink.deploy.json")

	s.Equal("", ResolveDeployConfigPath(jsonPath))
}

func (s *PreflightSuite) TestResolveDeployConfigPath_both_exist_prefers_exact() {
	dir := s.T().TempDir()
	jsonPath := filepath.Join(dir, "bluelink.deploy.json")
	jsoncPath := filepath.Join(dir, "bluelink.deploy.jsonc")
	os.WriteFile(jsonPath, []byte("{}"), 0644)
	os.WriteFile(jsoncPath, []byte("{}"), 0644)

	s.Equal(jsonPath, ResolveDeployConfigPath(jsonPath))
	s.Equal(jsoncPath, ResolveDeployConfigPath(jsoncPath))
}

func (s *PreflightSuite) TestResolveDeployConfigPath_non_json_extension() {
	dir := s.T().TempDir()
	tomlPath := filepath.Join(dir, "config.toml")

	s.Equal("", ResolveDeployConfigPath(tomlPath))
}

// --- PreflightModel tests ---

func (s *PreflightSuite) newModel(
	opts ...func(*PreflightOptions),
) *PreflightModel {
	o := PreflightOptions{
		Styles:      s.styles,
		CommandName: "stage",
	}
	for _, fn := range opts {
		fn(&o)
	}
	return NewPreflightModel(o)
}

func (s *PreflightSuite) Test_NewPreflightModel_view_shows_checking() {
	model := s.newModel()
	view := model.View()
	s.Contains(view, "Checking plugin dependencies")
}

func (s *PreflightSuite) Test_NewPreflightModel_headless_view_returns_empty() {
	model := s.newModel(func(o *PreflightOptions) {
		o.Headless = true
		o.HeadlessWriter = &bytes.Buffer{}
	})
	s.Empty(model.View())
}

func (s *PreflightSuite) Test_Init_returns_commands() {
	model := s.newModel()
	cmd := model.Init()
	s.NotNil(cmd)
}

func (s *PreflightSuite) Test_Init_headless_prints_checking_message() {
	buf := new(bytes.Buffer)
	model := s.newModel(func(o *PreflightOptions) {
		o.Headless = true
		o.HeadlessWriter = buf
	})
	model.Init()
	s.Contains(buf.String(), "Checking plugin dependencies")
}

func (s *PreflightSuite) Test_Update_satisfied_msg_returns_nil_cmd() {
	model := s.newModel()
	updated, cmd := model.Update(PreflightSatisfiedMsg{})
	s.Nil(cmd)
	s.Nil(updated.Error)
}

func (s *PreflightSuite) Test_Update_error_msg_sets_error() {
	model := s.newModel()
	testErr := errors.New("plugin check failed")
	updated, cmd := model.Update(PreflightErrorMsg{Err: testErr})
	s.Nil(cmd)
	s.Equal(testErr, updated.Error)
}

func (s *PreflightSuite) Test_Update_error_msg_view_returns_empty() {
	model := s.newModel()
	testErr := errors.New("plugin check failed")
	updated, _ := model.Update(PreflightErrorMsg{Err: testErr})
	s.Empty(updated.View())
}

func (s *PreflightSuite) Test_Update_installed_msg_is_passthrough() {
	model := s.newModel()
	updated, cmd := model.Update(PreflightInstalledMsg{
		CommandName:         "stage",
		RestartInstructions: "Restart the deploy engine.",
		InstalledPlugins:    []string{"plugin-a"},
		InstalledCount:      1,
	})
	s.Nil(cmd)
	s.Nil(updated.Error)
}

func (s *PreflightSuite) Test_Update_install_complete_error_produces_error_msg() {
	model := s.newModel()
	testErr := errors.New("install failed")
	updated, cmd := model.Update(plugininstallui.InstallCompleteMsg{
		Error: testErr,
	})
	s.Equal(testErr, updated.Error)
	s.NotNil(cmd)

	msg := cmd()
	errMsg, ok := msg.(PreflightErrorMsg)
	s.True(ok)
	s.Equal(testErr, errMsg.Err)
}

func (s *PreflightSuite) Test_Update_install_complete_interactive_stays_in_tui() {
	model := s.newModel()
	updated, cmd := model.Update(plugininstallui.InstallCompleteMsg{
		InstalledCount: 2,
	})
	s.Nil(updated.Error)
	// In interactive mode, no command is returned (model stays in TUI).
	s.Nil(cmd)

	view := updated.View()
	s.Contains(view, "missing plugin(s) installed")
	s.Contains(view, "quit")
}

func (s *PreflightSuite) Test_Update_install_complete_interactive_view_shows_restart() {
	model := s.newModel()
	updated, _ := model.Update(plugininstallui.InstallCompleteMsg{
		InstalledCount: 3,
	})
	view := updated.View()
	s.Contains(view, "requires plugin(s) that were not installed")
	s.Contains(view, "3 missing plugin(s) installed")
}

func (s *PreflightSuite) Test_Update_install_complete_headless_sends_installed_msg() {
	buf := new(bytes.Buffer)
	model := s.newModel(func(o *PreflightOptions) {
		o.Headless = true
		o.HeadlessWriter = buf
	})
	_, cmd := model.Update(plugininstallui.InstallCompleteMsg{
		InstalledCount: 2,
	})
	s.NotNil(cmd)

	msg := cmd()
	_, ok := msg.(PreflightInstalledMsg)
	s.True(ok)
}

func (s *PreflightSuite) Test_Update_install_complete_headless_writes_summary() {
	buf := new(bytes.Buffer)
	model := s.newModel(func(o *PreflightOptions) {
		o.Headless = true
		o.HeadlessWriter = buf
	})
	_, _ = model.Update(plugininstallui.InstallCompleteMsg{
		InstalledCount: 1,
	})
	output := buf.String()
	s.Contains(output, "requires plugin(s) that were not installed")
	s.Contains(output, "1 missing plugin(s) installed")
}

func (s *PreflightSuite) Test_Update_install_complete_headless_includes_command_name() {
	buf := new(bytes.Buffer)
	model := s.newModel(func(o *PreflightOptions) {
		o.Headless = true
		o.HeadlessWriter = buf
		o.CommandName = "deploy"
	})
	_, _ = model.Update(plugininstallui.InstallCompleteMsg{
		InstalledCount: 1,
	})
	s.Contains(buf.String(), "Re-run `bluelink deploy`")
}

func (s *PreflightSuite) Test_key_q_in_complete_stage_produces_installed_msg() {
	model := s.newModel()
	// Transition to complete stage via install complete.
	updated, _ := model.Update(plugininstallui.InstallCompleteMsg{
		InstalledCount: 1,
	})

	// Press q to quit.
	final, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	s.Nil(final.Error)
	s.NotNil(cmd)

	msg := cmd()
	installedMsg, ok := msg.(PreflightInstalledMsg)
	s.True(ok)
	s.Equal("stage", installedMsg.CommandName)
}

func (s *PreflightSuite) Test_key_q_in_checking_stage_does_not_quit() {
	model := s.newModel()
	// Model starts in checking stage. Pressing q should not produce an installed msg.
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		msg := cmd()
		_, isInstalled := msg.(PreflightInstalledMsg)
		s.False(isInstalled)
	}
}

// --- RenderInstallSummary tests ---

func (s *PreflightSuite) Test_RenderInstallSummary_shows_plugin_list() {
	output := RenderInstallSummary(
		s.styles,
		[]string{"bluelink/aws-provider", "bluelink/gcp-provider"},
		2,
		"Run `bluelink-manager restart` to restart the deploy engine.",
		"stage",
	)
	s.Contains(output, "bluelink/aws-provider")
	s.Contains(output, "bluelink/gcp-provider")
	s.Contains(output, "2 missing plugin(s) installed")
	s.Contains(output, "bluelink-manager restart")
	s.Contains(output, "Re-run `bluelink stage`")
}

func (s *PreflightSuite) Test_RenderInstallSummary_without_command_name() {
	output := RenderInstallSummary(
		s.styles,
		[]string{"bluelink/aws-provider"},
		1,
		"Restart the deploy engine to load the newly installed plugins.",
		"",
	)
	s.Contains(output, "bluelink/aws-provider")
	s.Contains(output, "1 missing plugin(s) installed")
	s.NotContains(output, "Re-run")
}

func (s *PreflightSuite) Test_RenderInstallSummary_shows_context_message() {
	output := RenderInstallSummary(
		s.styles,
		nil,
		0,
		"Restart the deploy engine.",
		"",
	)
	s.Contains(output, "requires plugin(s) that were not installed")
}
