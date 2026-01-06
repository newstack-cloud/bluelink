package stageui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// StageOptionsSelectedMsg is sent when the user completes the stage options form.
type StageOptionsSelectedMsg struct {
	InstanceName   string
	Destroy        bool
	SkipDriftCheck bool
	// InstanceExists indicates whether the instance already exists.
	// This is useful for the caller to know if they're staging for a new or existing instance.
	InstanceExists bool
}

// instanceExistsMsg is sent after checking if an instance exists.
type instanceExistsMsg struct {
	exists bool
}

type stageOptionsPhase int

const (
	phaseInstanceName stageOptionsPhase = iota
	phaseCheckingInstance
	phaseExistingInstanceOptions
)

// StageOptionsFormModel provides a form for configuring staging options.
// It uses a two-phase approach:
// 1. First, prompt for instance name
// 2. If the instance exists, prompt for destroy and skip drift check options
// For new instances, destroy and skip drift check are not applicable.
type StageOptionsFormModel struct {
	phase  stageOptionsPhase
	styles *stylespkg.Styles
	engine engine.DeployEngine

	// Instance name form (phase 1)
	instanceNameForm *huh.Form
	instanceName     string

	// Existing instance options form (phase 2, only shown for existing instances)
	optionsForm    *huh.Form
	destroy        bool
	skipDriftCheck bool

	// State
	instanceExists bool
	submitted      bool

	// Initial values from config
	initialDestroy        bool
	initialSkipDriftCheck bool
}

// StageOptionsFormConfig holds the initial values for the stage options form.
type StageOptionsFormConfig struct {
	// InitialInstanceName is the pre-populated instance name (from flag/env/config).
	InitialInstanceName string
	// InitialDestroy is the pre-populated destroy value (from flag/env/config).
	InitialDestroy bool
	// InitialSkipDriftCheck is the pre-populated skip drift check value (from flag/env/config).
	InitialSkipDriftCheck bool
	// Engine is used to check if an instance exists.
	Engine engine.DeployEngine
}

// NewStageOptionsFormModel creates a new stage options form model.
func NewStageOptionsFormModel(styles *stylespkg.Styles, config StageOptionsFormConfig) *StageOptionsFormModel {
	model := &StageOptionsFormModel{
		phase:                 phaseInstanceName,
		styles:                styles,
		engine:                config.Engine,
		instanceName:          config.InitialInstanceName,
		destroy:               config.InitialDestroy,
		skipDriftCheck:        config.InitialSkipDriftCheck,
		initialDestroy:        config.InitialDestroy,
		initialSkipDriftCheck: config.InitialSkipDriftCheck,
	}

	model.instanceNameForm = shared.NewThemedForm(styles,
		huh.NewGroup(
			shared.NewInstanceNameInput(
				&model.instanceName,
				"Name of a new or existing blueprint instance.",
			),
		),
	)

	return model
}

// createOptionsForm creates the form for existing instance options.
func (m *StageOptionsFormModel) createOptionsForm() {
	m.optionsForm = shared.NewThemedForm(m.styles,
		huh.NewGroup(
			huh.NewConfirm().
				Key("destroy").
				Title("Destroy Mode").
				Description("Stage changes for destroying this instance.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&m.destroy),
			huh.NewConfirm().
				Key("skipDriftCheck").
				Title("Skip Drift Check").
				Description("Skip detection of external resource changes.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&m.skipDriftCheck),
		),
	)
}

func (m *StageOptionsFormModel) Init() tea.Cmd {
	return m.instanceNameForm.Init()
}

func (m *StageOptionsFormModel) Update(msg tea.Msg) (*StageOptionsFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case instanceExistsMsg:
		m.instanceExists = msg.exists
		if msg.exists {
			// Instance exists, show options form
			m.phase = phaseExistingInstanceOptions
			m.createOptionsForm()
			return m, m.optionsForm.Init()
		}
		// Instance doesn't exist - new deployment, no additional options needed
		m.submitted = true
		return m, func() tea.Msg {
			return StageOptionsSelectedMsg{
				InstanceName:   m.instanceName,
				Destroy:        false,
				SkipDriftCheck: false,
				InstanceExists: false,
			}
		}
	}

	switch m.phase {
	case phaseInstanceName:
		form, cmd := m.instanceNameForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.instanceNameForm = f
		}

		if m.instanceNameForm.State == huh.StateCompleted {
			m.instanceName = strings.TrimSpace(m.instanceNameForm.GetString("instanceName"))
			m.phase = phaseCheckingInstance
			return m, checkInstanceExistsCmd(m)
		}

		return m, cmd

	case phaseCheckingInstance:
		// Waiting for instance check to complete
		return m, nil

	case phaseExistingInstanceOptions:
		form, cmd := m.optionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.optionsForm = f
		}

		if m.optionsForm.State == huh.StateCompleted && !m.submitted {
			m.submitted = true
			m.destroy = m.optionsForm.GetBool("destroy")
			m.skipDriftCheck = m.optionsForm.GetBool("skipDriftCheck")
			return m, func() tea.Msg {
				return StageOptionsSelectedMsg{
					InstanceName:   m.instanceName,
					Destroy:        m.destroy,
					SkipDriftCheck: m.skipDriftCheck,
					InstanceExists: true,
				}
			}
		}

		return m, cmd
	}

	return m, nil
}

func (m *StageOptionsFormModel) View() string {
	switch m.phase {
	case phaseInstanceName:
		return m.instanceNameForm.View()
	case phaseCheckingInstance:
		return "  Checking if instance exists...\n"
	case phaseExistingInstanceOptions:
		header := "  Instance \"" + m.instanceName + "\" exists. Configure options:\n\n"
		return header + m.optionsForm.View()
	}
	return ""
}
