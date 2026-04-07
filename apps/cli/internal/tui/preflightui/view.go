package preflightui

import (
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/tui/preflight"
)

func (m PreflightModel) renderChecking() string {
	var sb strings.Builder
	sb.WriteString("\n  ")
	sb.WriteString(m.spinner.View())
	sb.WriteString(" Checking plugin dependencies...\n")
	return sb.String()
}

func (m PreflightModel) renderComplete() string {
	pluginNames := make([]string, len(m.allToInstall))
	for i, p := range m.allToInstall {
		pluginNames[i] = p.String()
	}
	var sb strings.Builder
	sb.WriteString(preflight.RenderInstallSummary(
		m.styles, pluginNames, m.installedCount, m.restartInstructions, m.commandName,
	))
	sb.WriteString("  ")
	sb.WriteString(m.styles.Key.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" quit"))
	sb.WriteString("\n")
	return sb.String()
}
