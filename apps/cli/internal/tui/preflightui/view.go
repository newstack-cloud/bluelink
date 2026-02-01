package preflightui

import (
	"fmt"
	"strings"

	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
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
	sb.WriteString(renderInstallSummary(
		m.styles, pluginNames, m.installedCount, m.restartInstructions, m.commandName,
	))
	sb.WriteString("  ")
	sb.WriteString(m.styles.Key.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" quit"))
	sb.WriteString("\n")
	return sb.String()
}

// RenderInstallSummary renders the plugin installation summary for use
// in both the preflight complete view and parent TUI quitting views.
func RenderInstallSummary(
	styles *stylespkg.Styles,
	plugins []string,
	installedCount int,
	restartInstructions string,
	commandName string,
) string {
	return renderInstallSummary(styles, plugins, installedCount, restartInstructions, commandName)
}

func renderInstallSummary(
	styles *stylespkg.Styles,
	plugins []string,
	installedCount int,
	restartInstructions string,
	commandName string,
) string {
	var sb strings.Builder

	sb.WriteString("\n  ")
	sb.WriteString(styles.Muted.Render(
		"The deploy configuration requires plugin(s) that were not installed.",
	))
	sb.WriteString("\n  ")
	sb.WriteString(styles.Selected.Render(
		fmt.Sprintf("%d missing plugin(s) installed:", installedCount),
	))

	sb.WriteString("\n\n")
	for _, p := range plugins {
		sb.WriteString("  ")
		sb.WriteString(styles.Muted.Render("• "))
		sb.WriteString(p)
		sb.WriteString("\n")
	}

	sb.WriteString("\n  ")
	sb.WriteString(restartInstructions)
	if commandName != "" {
		sb.WriteString("\n  ")
		sb.WriteString(styles.Muted.Render(
			fmt.Sprintf("Re-run `bluelink %s` after restarting the engine.", commandName),
		))
	}
	sb.WriteString("\n\n")
	return sb.String()
}
