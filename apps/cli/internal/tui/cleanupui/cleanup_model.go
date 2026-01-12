package cleanupui

import (
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"go.uber.org/zap"
)

// cleanupStep represents a single cleanup operation to perform.
type cleanupStep struct {
	name        string
	cleanupType manage.CleanupType
}

// CleanupModel handles the execution of cleanup operations.
type CleanupModel struct {
	engine  engine.DeployEngine
	logger  *zap.Logger
	spinner spinner.Model
	styles  *stylespkg.Styles

	steps            []cleanupStep
	currentStepIndex int
	completedOps     []*manage.CleanupOperation
	currentOp        *manage.CleanupOperation
	err              error
	done             bool
	hasFailures      bool // Track if any operations failed

	headless               bool
	headlessWriter         io.Writer
	headlessLastPrinted    int  // Last step index printed in headless mode
	headlessSummaryPrinted bool // Whether the summary has been printed
}

// NewCleanupModel creates a new cleanup execution model.
func NewCleanupModel(
	engine engine.DeployEngine,
	logger *zap.Logger,
	cleanupValidations bool,
	cleanupChangesets bool,
	cleanupReconciliationResults bool,
	cleanupEvents bool,
	headless bool,
	headlessWriter io.Writer,
	styles *stylespkg.Styles,
) *CleanupModel {
	var steps []cleanupStep

	if cleanupValidations {
		steps = append(steps, cleanupStep{
			name:        "validations",
			cleanupType: manage.CleanupTypeValidations,
		})
	}
	if cleanupChangesets {
		steps = append(steps, cleanupStep{
			name:        "changesets",
			cleanupType: manage.CleanupTypeChangesets,
		})
	}
	if cleanupReconciliationResults {
		steps = append(steps, cleanupStep{
			name:        "reconciliation results",
			cleanupType: manage.CleanupTypeReconciliationResults,
		})
	}
	if cleanupEvents {
		steps = append(steps, cleanupStep{
			name:        "events",
			cleanupType: manage.CleanupTypeEvents,
		})
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = stylespkg.NewDefaultStyles(stylespkg.NewBluelinkPalette()).Spinner

	return &CleanupModel{
		engine:              engine,
		logger:              logger,
		spinner:             s,
		styles:              styles,
		steps:               steps,
		currentStepIndex:    0,
		headless:            headless,
		headlessWriter:      headlessWriter,
		headlessLastPrinted: -1, // Start at -1 so step 0 gets printed
	}
}

func (m *CleanupModel) Init() tea.Cmd {
	if len(m.steps) == 0 {
		m.done = true
		return func() tea.Msg { return AllCleanupsDoneMsg{} }
	}
	return tea.Batch(m.spinner.Tick, m.startNextCleanup())
}

func (m *CleanupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case CleanupStartedMsg:
		m.currentOp = msg.Operation
		cmds = append(cmds, m.waitForCompletion(msg.Operation))

	case CleanupCompletedMsg:
		m.completedOps = append(m.completedOps, msg.Operation)
		// Track if the operation failed
		if msg.Operation.Status == manage.CleanupOperationStatusFailed {
			m.hasFailures = true
		}
		m.currentStepIndex += 1

		if m.currentStepIndex >= len(m.steps) {
			m.done = true
			return m, func() tea.Msg { return AllCleanupsDoneMsg{} }
		}

		cmds = append(cmds, m.startNextCleanup())

	case CleanupErrorMsg:
		m.err = msg.Err
		m.done = true
		return m, func() tea.Msg { return AllCleanupsDoneMsg{} }
	}

	return m, tea.Batch(cmds...)
}

func (m *CleanupModel) View() string {
	if m.headless {
		m.renderHeadless()
		return ""
	}
	return m.renderInteractive()
}

func (m *CleanupModel) startNextCleanup() tea.Cmd {
	if m.currentStepIndex >= len(m.steps) {
		return func() tea.Msg { return AllCleanupsDoneMsg{} }
	}

	step := m.steps[m.currentStepIndex]
	return startCleanupCmd(m.engine, step.cleanupType)
}

func (m *CleanupModel) waitForCompletion(op *manage.CleanupOperation) tea.Cmd {
	return waitForCleanupCompletionCmd(m.engine, op)
}
