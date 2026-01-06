package deployui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// DeployConfigMsg is sent when the user completes the deploy configuration form.
type DeployConfigMsg struct {
	InstanceName string
	InstanceID   string
	ChangesetID  string
	StageFirst   bool
	AutoApprove  bool
	AutoRollback bool
}

// DeployConfigFormInitialValues holds the initial values for the deploy config form.
type DeployConfigFormInitialValues struct {
	InstanceName string
	InstanceID   string
	ChangesetID  string
	StageFirst   bool
	AutoApprove  bool
	AutoRollback bool
}

// DeployConfigFormModel provides a combined form for deploy configuration.
type DeployConfigFormModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	autoComplete bool

	// Bound form values
	instanceName string
	instanceID   string
	changesetID  string
	stageFirst   bool
	autoApprove  bool
	autoRollback bool

	// Read-only instance ID (shown but not editable)
	hasInstanceID bool
}

// NewDeployConfigFormModel creates a new deploy config form model.
func NewDeployConfigFormModel(
	initialValues DeployConfigFormInitialValues,
	styles *stylespkg.Styles,
) *DeployConfigFormModel {
	model := &DeployConfigFormModel{
		styles:        styles,
		instanceName:  initialValues.InstanceName,
		instanceID:    initialValues.InstanceID,
		changesetID:   initialValues.ChangesetID,
		stageFirst:    initialValues.StageFirst,
		autoApprove:   initialValues.AutoApprove,
		autoRollback:  initialValues.AutoRollback,
		hasInstanceID: initialValues.InstanceID != "",
	}

	// In interactive mode, always show the form so users can review settings.
	// The form will only be skipped in headless mode, which is handled by
	// the TUI state machine in tui.go.
	model.autoComplete = false

	// Build the form fields
	fields := []huh.Field{}

	if model.hasInstanceID {
		fields = append(fields, shared.NewInstanceIDNote(model.instanceID))
	} else {
		fields = append(fields, shared.NewInstanceNameInput(
			&model.instanceName,
			"Name of an existing instance to update, or a new name to create.",
		))
	}

	fields = append(fields, shared.NewStageFirstConfirm(
		&model.stageFirst,
		"Stage changes first?",
		"Stage now, or use an existing changeset ID.",
	))

	changesetIDGroup := shared.NewChangesetIDGroup(
		&model.changesetID,
		&model.stageFirst,
		"The ID of a previously staged changeset to deploy.",
	)

	autoApproveGroup := shared.NewAutoApproveGroup(
		&model.autoApprove,
		&model.stageFirst,
		"No, ask before deploy",
	)

	// Auto-rollback toggle (deploy-specific)
	autoRollbackGroup := huh.NewGroup(
		huh.NewConfirm().
			Key("autoRollback").
			Title("Enable auto-rollback?").
			Description("Automatically rollback on deployment failure.").
			Affirmative("Yes, auto-rollback").
			Negative("No, keep failed state").
			WithButtonAlignment(lipgloss.Left).
			Value(&model.autoRollback),
	)

	model.form = shared.NewThemedForm(styles,
		huh.NewGroup(fields...),
		changesetIDGroup,
		autoApproveGroup,
		autoRollbackGroup,
	)

	return model
}

// Init initializes the model.
func (m DeployConfigFormModel) Init() tea.Cmd {
	if m.autoComplete {
		return deployConfigCompleteCmd(
			m.instanceName,
			m.instanceID,
			m.changesetID,
			m.stageFirst,
			m.autoApprove,
			m.autoRollback,
		)
	}
	return m.form.Init()
}

// Update handles messages.
func (m DeployConfigFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.autoComplete {
		return m, nil
	}

	cmds := []tea.Cmd{}

	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		// Get values from form (or use pre-set values for instance ID case)
		instanceName := m.instanceName
		if !m.hasInstanceID {
			instanceName = strings.TrimSpace(m.form.GetString("instanceName"))
		}

		// Get changeset ID (only relevant when not staging first)
		changesetID := strings.TrimSpace(m.form.GetString("changesetID"))

		cmds = append(cmds, deployConfigCompleteCmd(
			instanceName,
			m.instanceID,
			changesetID,
			m.form.GetBool("stageFirst"),
			m.form.GetBool("autoApprove"),
			m.form.GetBool("autoRollback"),
		))
	}

	return m, tea.Batch(cmds...)
}

// View renders the model.
func (m DeployConfigFormModel) View() string {
	if m.autoComplete {
		return ""
	}

	sb := strings.Builder{}
	sb.WriteString("\n")

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Palette.Primary()).
		MarginLeft(2)
	sb.WriteString(headerStyle.Render("Deployment Options"))
	sb.WriteString("\n\n")

	sb.WriteString(m.form.View())
	sb.WriteString("\n")

	return sb.String()
}

func deployConfigCompleteCmd(
	instanceName string,
	instanceID string,
	changesetID string,
	stageFirst bool,
	autoApprove bool,
	autoRollback bool,
) tea.Cmd {
	return func() tea.Msg {
		return DeployConfigMsg{
			InstanceName: instanceName,
			InstanceID:   instanceID,
			ChangesetID:  changesetID,
			StageFirst:   stageFirst,
			AutoApprove:  autoApprove,
			AutoRollback: autoRollback,
		}
	}
}
