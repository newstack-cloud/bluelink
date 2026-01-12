package cleanupui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// CleanupOptionsSelectedMsg is sent when user completes the options form.
type CleanupOptionsSelectedMsg struct {
	Validations           bool
	Changesets            bool
	ReconciliationResults bool
	Events                bool
}

// CleanupOptionsFormModel provides a form for selecting which resource types to clean up.
type CleanupOptionsFormModel struct {
	form   *huh.Form
	styles *stylespkg.Styles

	validations           bool
	changesets            bool
	reconciliationResults bool
	events                bool

	submitted bool
}

// NewCleanupOptionsFormModel creates a new cleanup options form model.
func NewCleanupOptionsFormModel(styles *stylespkg.Styles) *CleanupOptionsFormModel {
	m := &CleanupOptionsFormModel{
		styles:                styles,
		validations:           true,
		changesets:            true,
		reconciliationResults: true,
		events:                true,
	}

	m.form = shared.NewThemedForm(styles,
		huh.NewGroup(
			huh.NewConfirm().
				Key("validations").
				Title("Cleanup Validations").
				Description("Remove blueprint validation results that have exceeded their retention period.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&m.validations),

			huh.NewConfirm().
				Key("changesets").
				Title("Cleanup Changesets").
				Description("Remove change sets that have exceeded their retention period.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&m.changesets),

			huh.NewConfirm().
				Key("reconciliationResults").
				Title("Cleanup Reconciliation Results").
				Description("Remove reconciliation check results that have exceeded their retention period.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&m.reconciliationResults),

			huh.NewConfirm().
				Key("events").
				Title("Cleanup Events").
				Description("Remove streaming events that have exceeded their retention period.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&m.events),
		),
	)

	return m
}

func (m *CleanupOptionsFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *CleanupOptionsFormModel) Update(msg tea.Msg) (*CleanupOptionsFormModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true
		m.validations = m.form.GetBool("validations")
		m.changesets = m.form.GetBool("changesets")
		m.reconciliationResults = m.form.GetBool("reconciliationResults")
		m.events = m.form.GetBool("events")

		return m, func() tea.Msg {
			return CleanupOptionsSelectedMsg{
				Validations:           m.validations,
				Changesets:            m.changesets,
				ReconciliationResults: m.reconciliationResults,
				Events:                m.events,
			}
		}
	}

	return m, cmd
}

func (m *CleanupOptionsFormModel) View() string {
	header := "\n  Select resource types to clean up:\n\n"
	return header + m.form.View()
}
