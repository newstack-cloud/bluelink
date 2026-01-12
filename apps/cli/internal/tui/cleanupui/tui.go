package cleanupui

import (
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"go.uber.org/zap"
)

var errCleanupFailed = errors.New("cleanup failed")

var quitTextStyle = stylespkg.NewDefaultStyles(
	stylespkg.NewBluelinkPalette(),
).Muted.Margin(1, 0, 2, 4)

type cleanupSessionState uint32

const (
	cleanupOptionsForm cleanupSessionState = iota
	cleanupExecuting
	cleanupComplete
)

// MainModel is the top-level model for the cleanup TUI.
type MainModel struct {
	sessionState cleanupSessionState
	quitting     bool

	optionsForm *CleanupOptionsFormModel
	cleanup     *CleanupModel

	cleanupValidations           bool
	cleanupChangesets            bool
	cleanupReconciliationResults bool
	cleanupEvents                bool
	showOptionsForm              bool

	styles *stylespkg.Styles
	Error  error

	engine         engine.DeployEngine
	logger         *zap.Logger
	headless       bool
	headlessWriter io.Writer
}

func (m MainModel) Init() tea.Cmd {
	if m.showOptionsForm {
		return m.optionsForm.Init()
	}
	return m.cleanup.Init()
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case CleanupOptionsSelectedMsg:
		m.cleanupValidations = msg.Validations
		m.cleanupChangesets = msg.Changesets
		m.cleanupReconciliationResults = msg.ReconciliationResults
		m.cleanupEvents = msg.Events

		m.cleanup = NewCleanupModel(
			m.engine,
			m.logger,
			m.cleanupValidations,
			m.cleanupChangesets,
			m.cleanupReconciliationResults,
			m.cleanupEvents,
			m.headless,
			m.headlessWriter,
			m.styles,
		)
		m.sessionState = cleanupExecuting
		return m, m.cleanup.Init()

	case AllCleanupsDoneMsg:
		m.sessionState = cleanupComplete
		if m.cleanup.err != nil {
			m.Error = m.cleanup.err
		} else if m.cleanup.hasFailures {
			// Set Error to indicate some operations failed (for non-zero exit code)
			m.Error = errCleanupFailed
		}
		// In headless mode, quit immediately after completion
		if m.headless {
			return m, tea.Quit
		}
		return m, nil

	case CleanupErrorMsg:
		m.sessionState = cleanupComplete
		m.Error = msg.Err
		if m.cleanup != nil {
			m.cleanup.err = msg.Err
			m.cleanup.done = true
		}
		// In headless mode, quit immediately after error
		if m.headless {
			return m, tea.Quit
		}
		return m, nil
	}

	switch m.sessionState {
	case cleanupOptionsForm:
		newForm, cmd := m.optionsForm.Update(msg)
		m.optionsForm = newForm
		cmds = append(cmds, cmd)

	case cleanupExecuting:
		newCleanup, cmd := m.cleanup.Update(msg)
		if cm, ok := newCleanup.(*CleanupModel); ok {
			m.cleanup = cm
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("Cleanup cancelled.")
	}

	switch m.sessionState {
	case cleanupOptionsForm:
		return m.optionsForm.View()
	case cleanupExecuting, cleanupComplete:
		return m.cleanup.View()
	}

	return ""
}

// NewCleanupApp creates a new cleanup TUI application.
func NewCleanupApp(
	engine engine.DeployEngine,
	logger *zap.Logger,
	cleanupValidations bool,
	cleanupChangesets bool,
	cleanupReconciliationResults bool,
	cleanupEvents bool,
	showOptionsForm bool,
	styles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
) (*MainModel, error) {
	model := &MainModel{
		cleanupValidations:           cleanupValidations,
		cleanupChangesets:            cleanupChangesets,
		cleanupReconciliationResults: cleanupReconciliationResults,
		cleanupEvents:                cleanupEvents,
		showOptionsForm:              showOptionsForm,
		styles:                       styles,
		engine:                       engine,
		logger:                       logger,
		headless:                     headless,
		headlessWriter:               headlessWriter,
	}

	if showOptionsForm {
		model.sessionState = cleanupOptionsForm
		model.optionsForm = NewCleanupOptionsFormModel(styles)
	} else {
		model.sessionState = cleanupExecuting
		model.cleanup = NewCleanupModel(
			engine,
			logger,
			cleanupValidations,
			cleanupChangesets,
			cleanupReconciliationResults,
			cleanupEvents,
			headless,
			headlessWriter,
			styles,
		)
	}

	return model, nil
}
