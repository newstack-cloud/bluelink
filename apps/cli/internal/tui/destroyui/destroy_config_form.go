package destroyui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// DestroyConfigMsg is sent when the user completes the destroy configuration form.
type DestroyConfigMsg struct {
	InstanceName string
	InstanceID   string
	ChangesetID  string
	StageFirst   bool
	AutoApprove  bool
}

// DestroyConfigFormInitialValues holds the initial values for the destroy config form.
type DestroyConfigFormInitialValues struct {
	InstanceName string
	InstanceID   string
	ChangesetID  string
	StageFirst   bool
	AutoApprove  bool
}

// DestroyConfigFormModel provides a combined form for destroy configuration.
type DestroyConfigFormModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	autoComplete bool

	// Bound form values
	instanceName string
	instanceID   string
	changesetID  string
	stageFirst   bool
	autoApprove  bool

	// Read-only instance ID (shown but not editable)
	hasInstanceID bool
}

// NewDestroyConfigFormModel creates a new destroy config form model.
func NewDestroyConfigFormModel(
	initialValues DestroyConfigFormInitialValues,
	styles *stylespkg.Styles,
) *DestroyConfigFormModel {
	model := &DestroyConfigFormModel{
		styles:        styles,
		instanceName:  initialValues.InstanceName,
		instanceID:    initialValues.InstanceID,
		changesetID:   initialValues.ChangesetID,
		stageFirst:    initialValues.StageFirst,
		autoApprove:   initialValues.AutoApprove,
		hasInstanceID: initialValues.InstanceID != "",
	}

	model.autoComplete = false

	fields := []huh.Field{}

	if model.hasInstanceID {
		fields = append(fields,
			huh.NewNote().
				Title("Instance ID").
				Description(model.instanceID),
		)
	} else {
		fields = append(fields,
			huh.NewInput().
				Key("instanceName").
				Title("Instance Name").
				Description("Name of the instance to destroy.").
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

	fields = append(fields,
		huh.NewConfirm().
			Key("stageFirst").
			Title("Stage destroy changes first?").
			Description("Stage now to preview destruction, or use an existing changeset ID.").
			Affirmative("Yes, stage now").
			Negative("No, use existing changeset").
			WithButtonAlignment(lipgloss.Left).
			Value(&model.stageFirst),
	)

	changesetIDGroup := huh.NewGroup(
		huh.NewInput().
			Key("changesetID").
			Title("Changeset ID").
			Description("The ID of a previously staged destroy changeset.").
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

	autoApproveGroup := huh.NewGroup(
		huh.NewConfirm().
			Key("autoApprove").
			Title("Auto-approve staged changes?").
			Description("Skip confirmation after staging.").
			Affirmative("Yes, skip confirmation").
			Negative("No, ask before destroy").
			WithButtonAlignment(lipgloss.Left).
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
func (m DestroyConfigFormModel) Init() tea.Cmd {
	if m.autoComplete {
		return destroyConfigCompleteCmd(
			m.instanceName,
			m.instanceID,
			m.changesetID,
			m.stageFirst,
			m.autoApprove,
		)
	}
	return m.form.Init()
}

// Update handles messages.
func (m DestroyConfigFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		instanceName := m.instanceName
		if !m.hasInstanceID {
			instanceName = strings.TrimSpace(m.form.GetString("instanceName"))
		}

		changesetID := strings.TrimSpace(m.form.GetString("changesetID"))

		cmds = append(cmds, destroyConfigCompleteCmd(
			instanceName,
			m.instanceID,
			changesetID,
			m.form.GetBool("stageFirst"),
			m.form.GetBool("autoApprove"),
		))
	}

	return m, tea.Batch(cmds...)
}

// View renders the model.
func (m DestroyConfigFormModel) View() string {
	if m.autoComplete {
		return ""
	}

	sb := strings.Builder{}
	sb.WriteString("\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Palette.Primary()).
		MarginLeft(2)
	sb.WriteString(headerStyle.Render("Destroy Options"))
	sb.WriteString("\n\n")

	sb.WriteString(m.form.View())
	sb.WriteString("\n")

	return sb.String()
}

func destroyConfigCompleteCmd(
	instanceName string,
	instanceID string,
	changesetID string,
	stageFirst bool,
	autoApprove bool,
) tea.Cmd {
	return func() tea.Msg {
		return DestroyConfigMsg{
			InstanceName: instanceName,
			InstanceID:   instanceID,
			ChangesetID:  changesetID,
			StageFirst:   stageFirst,
			AutoApprove:  autoApprove,
		}
	}
}
