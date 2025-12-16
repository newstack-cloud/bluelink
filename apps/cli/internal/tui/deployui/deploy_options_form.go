package deployui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// DeployConfigMsg is sent when the user completes the deploy configuration form.
type DeployConfigMsg struct {
	InstanceName string
	InstanceID   string
	ChangesetID  string
	AsRollback   bool
	StageFirst   bool
	AutoApprove  bool
}

// DeployConfigFormInitialValues holds the initial values for the deploy config form.
type DeployConfigFormInitialValues struct {
	InstanceName string
	InstanceID   string
	ChangesetID  string
	AsRollback   bool
	StageFirst   bool
	AutoApprove  bool
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
	asRollback   bool
	stageFirst   bool
	autoApprove  bool

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
		asRollback:    initialValues.AsRollback,
		stageFirst:    initialValues.StageFirst,
		autoApprove:   initialValues.AutoApprove,
		hasInstanceID: initialValues.InstanceID != "",
	}

	// In interactive mode, always show the form so users can review settings.
	// The form will only be skipped in headless mode, which is handled by
	// the TUI state machine in tui.go.
	model.autoComplete = false

	// Build the form fields
	fields := []huh.Field{}

	// If instance ID is provided, show it as read-only note (not editable)
	// Otherwise, show instance name input
	if model.hasInstanceID {
		// Instance ID provided - show as info, no input needed
		fields = append(fields,
			huh.NewNote().
				Title("Instance ID").
				Description(model.instanceID),
		)
	} else {
		// New deployment - need instance name
		fields = append(fields,
			huh.NewInput().
				Key("instanceName").
				Title("Instance Name").
				Description("Name of an existing instance to update, or a new name to create.").
				Placeholder("my-app-production").
				Value(&model.instanceName).
				Validate(func(value string) error {
					trimmed := strings.TrimSpace(value)
					if trimmed == "" {
						return errors.New("instance name cannot be empty")
					}
					if len(trimmed) < 3 {
						return errors.New("instance name must be at least 3 characters")
					}
					if len(trimmed) > 128 {
						return errors.New("instance name must be at most 128 characters")
					}
					return nil
				}),
		)
	}

	// As-Rollback toggle
	fields = append(fields,
		huh.NewConfirm().
			Key("asRollback").
			Title("As Rollback?").
			Description("Mark deployment as rollback operation.").
			Affirmative("Yes, as rollback").
			Negative("No, normal deploy").
			Value(&model.asRollback),
	)

	// Stage first toggle
	fields = append(fields,
		huh.NewConfirm().
			Key("stageFirst").
			Title("Stage changes first?").
			Description("Stage now, or use an existing changeset ID.").
			Affirmative("Yes, stage now").
			Negative("No, use existing changeset").
			Value(&model.stageFirst),
	)

	// Changeset ID input in separate group (only shown when stageFirst is false)
	changesetIDGroup := huh.NewGroup(
		huh.NewInput().
			Key("changesetID").
			Title("Changeset ID").
			Description("The ID of a previously staged changeset to deploy.").
			Placeholder("changeset-abc123").
			Value(&model.changesetID).
			Validate(func(value string) error {
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					return errors.New("changeset ID is required when not staging first")
				}
				return nil
			}),
	).WithHideFunc(func() bool {
		return model.stageFirst
	})

	// Auto-approve toggle in separate group (only shown when stageFirst is true)
	autoApproveGroup := huh.NewGroup(
		huh.NewConfirm().
			Key("autoApprove").
			Title("Auto-approve staged changes?").
			Description("Skip confirmation after staging.").
			Affirmative("Yes, skip confirmation").
			Negative("No, ask before deploy").
			Value(&model.autoApprove),
	).WithHideFunc(func() bool {
		return !model.stageFirst
	})

	model.form = huh.NewForm(
		huh.NewGroup(fields...),
		changesetIDGroup,
		autoApproveGroup,
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	return model
}

// Init initializes the model.
func (m DeployConfigFormModel) Init() tea.Cmd {
	if m.autoComplete {
		return deployConfigCompleteCmd(
			m.instanceName,
			m.instanceID,
			m.changesetID,
			m.asRollback,
			m.stageFirst,
			m.autoApprove,
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
			m.form.GetBool("asRollback"),
			m.form.GetBool("stageFirst"),
			m.form.GetBool("autoApprove"),
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
	asRollback bool,
	stageFirst bool,
	autoApprove bool,
) tea.Cmd {
	return func() tea.Msg {
		return DeployConfigMsg{
			InstanceName: instanceName,
			InstanceID:   instanceID,
			ChangesetID:  changesetID,
			AsRollback:   asRollback,
			StageFirst:   stageFirst,
			AutoApprove:  autoApprove,
		}
	}
}
