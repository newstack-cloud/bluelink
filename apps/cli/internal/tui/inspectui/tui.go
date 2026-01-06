package inspectui

import (
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"go.uber.org/zap"
)

var quitTextStyle = stylespkg.NewDefaultStyles(
	stylespkg.NewBluelinkPalette(),
).Muted.Margin(1, 0, 2, 4)

type inspectSessionState uint32

const (
	inspectInstanceInput inspectSessionState = iota
	inspectLoading
	inspectViewing
)

// MainModel is the top-level model for the inspect command TUI.
type MainModel struct {
	sessionState inspectSessionState
	quitting     bool

	// Sub-models
	instanceForm *InstanceInputFormModel
	inspect      *InspectModel

	// Config from flags
	instanceID   string
	instanceName string

	// Runtime state
	headless bool
	jsonMode bool
	engine   engine.DeployEngine
	logger   *zap.Logger

	styles *stylespkg.Styles
	Error  error
}

// Init initializes the main model.
func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	if m.instanceForm != nil {
		cmds = append(cmds, m.instanceForm.Init())
	}

	if m.inspect != nil {
		cmds = append(cmds, m.inspect.Init())
	}

	// If we're starting in loading state (identifier already provided), fetch the instance state
	if m.sessionState == inspectLoading {
		cmds = append(cmds, fetchInstanceStateCmd(*m.inspect))
	}

	return tea.Batch(cmds...)
}

// Update handles messages for the main model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case InstanceInputMsg:
		m.instanceID = msg.InstanceID
		m.instanceName = msg.InstanceName

		// Update inspect model with the identifiers
		m.inspect.instanceID = msg.InstanceID
		m.inspect.instanceName = msg.InstanceName
		m.inspect.footerRenderer.InstanceID = msg.InstanceID
		m.inspect.footerRenderer.InstanceName = msg.InstanceName

		// Transition to loading state and start fetching
		m.sessionState = inspectLoading
		cmds = append(cmds, fetchInstanceStateCmd(*m.inspect))

	case InstanceStateFetchedMsg:
		m.sessionState = inspectViewing

		// Update the inspect model with fetched state
		m.inspect.SetInstanceState(msg.InstanceState)

		if msg.IsInProgress {
			// Start streaming events and periodic state refresh
			m.inspect.streaming = true
			m.inspect.footerRenderer.Streaming = true
			cmds = append(cmds, startStreamingCmd(*m.inspect))
			cmds = append(cmds, startStateRefreshTickerCmd())
		} else {
			// Static view - just display the state
			m.inspect.finished = true
			m.inspect.footerRenderer.Finished = true
			m.inspect.detailsRenderer.Finished = true

			// In headless mode, output now and quit
			if m.headless {
				if m.jsonMode {
					m.inspect.outputJSON()
				} else {
					m.inspect.printHeadlessInstanceState()
				}
				return m, tea.Quit
			}
		}
		// Return early to avoid double-handling by InspectModel
		return m, tea.Batch(cmds...)

	case InstanceNotFoundMsg:
		m.Error = msg.Err
		if m.headless {
			if m.jsonMode {
				m.inspect.outputJSONError(msg.Err)
			} else {
				m.inspect.printHeadlessError(msg.Err)
			}
			return m, tea.Quit
		}
		return m, nil

	case InspectStreamStartedMsg:
		cmds = append(cmds, waitForNextEventCmd(*m.inspect))
		return m, tea.Batch(cmds...)

	case StateRefreshTickMsg:
		// Only refresh if still streaming
		if m.inspect.streaming && !m.inspect.finished {
			cmds = append(cmds, refreshInstanceStateCmd(*m.inspect))
			cmds = append(cmds, startStateRefreshTickerCmd())
		}
		return m, tea.Batch(cmds...)

	case InstanceStateRefreshedMsg:
		// Hydrate existing items with updated state
		// This runs both during streaming (periodic refresh) and after streaming ends (final refresh)
		if msg.InstanceState != nil {
			m.inspect.RefreshInstanceState(msg.InstanceState)
		}
		return m, tea.Batch(cmds...)

	case InspectEventMsg:
		m.inspect.processEvent(&msg)
		m.inspect.splitPane.UpdateItems(ToSplitPaneItems(m.inspect.items))

		finishData, isFinish := msg.AsFinish()
		if isFinish && finishData.EndOfStream {
			m.inspect.finished = true
			m.inspect.streaming = false
			m.inspect.footerRenderer.Streaming = false
			m.inspect.footerRenderer.Finished = true
			m.inspect.footerRenderer.CurrentStatus = finishData.Status
			m.inspect.detailsRenderer.Finished = true

			if m.headless {
				if m.jsonMode {
					m.inspect.outputJSON()
				} else {
					m.inspect.printHeadlessInstanceState()
				}
				return m, tea.Quit
			}

			// Trigger a final state refresh to hydrate all items with ResourceState
			// for resources that completed during streaming
			cmds = append(cmds, refreshInstanceStateCmd(*m.inspect))
			return m, tea.Batch(cmds...)
		}

		cmds = append(cmds, waitForNextEventCmd(*m.inspect))
		return m, tea.Batch(cmds...)

	case InspectStreamClosedMsg:
		m.inspect.streaming = false
		m.inspect.footerRenderer.Streaming = false
		if !m.inspect.finished {
			m.inspect.err = errors.New("event stream closed unexpectedly")
			m.Error = m.inspect.err
			if m.headless {
				if m.jsonMode {
					m.inspect.outputJSONError(m.inspect.err)
				} else {
					m.inspect.printHeadlessError(m.inspect.err)
				}
				return m, tea.Quit
			}
		}
		return m, tea.Batch(cmds...)

	case InspectErrorMsg:
		if msg.Err != nil {
			m.inspect.err = msg.Err
			m.Error = msg.Err
			if m.headless {
				if m.jsonMode {
					m.inspect.outputJSONError(msg.Err)
				} else {
					m.inspect.printHeadlessError(msg.Err)
				}
				return m, tea.Quit
			}
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		// Always forward spinner ticks to the inspect model regardless of session state
		if m.inspect != nil {
			var cmd tea.Cmd
			m.inspect.spinner, cmd = m.inspect.spinner.Update(msg)
			m.inspect.footerRenderer.SpinnerView = m.inspect.spinner.View()
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		if m.instanceForm != nil {
			var formModel tea.Model
			var formCmd tea.Cmd
			formModel, formCmd = m.instanceForm.Update(msg)
			if fm, ok := formModel.(*InstanceInputFormModel); ok {
				m.instanceForm = fm
			}
			cmds = append(cmds, formCmd)
		}

		if m.inspect != nil {
			var inspectCmd tea.Cmd
			var inspectModel tea.Model
			inspectModel, inspectCmd = m.inspect.handleWindowSize(msg)
			if im, ok := inspectModel.(InspectModel); ok {
				m.inspect = &im
			}
			cmds = append(cmds, inspectCmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.sessionState == inspectViewing && m.inspect.finished {
				m.quitting = true
				return m, tea.Quit
			}
			if m.Error != nil {
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	// Route messages to the appropriate sub-model based on session state
	switch m.sessionState {
	case inspectInstanceInput:
		if m.instanceForm != nil {
			var formModel tea.Model
			var formCmd tea.Cmd
			formModel, formCmd = m.instanceForm.Update(msg)
			if fm, ok := formModel.(*InstanceInputFormModel); ok {
				m.instanceForm = fm
			}
			cmds = append(cmds, formCmd)
		}

	case inspectLoading:
		// Just waiting for state fetch

	case inspectViewing:
		if m.inspect != nil {
			var inspectModel tea.Model
			var inspectCmd tea.Cmd
			inspectModel, inspectCmd = m.inspect.Update(msg)
			if im, ok := inspectModel.(InspectModel); ok {
				m.inspect = &im
			}
			cmds = append(cmds, inspectCmd)

			if m.inspect.err != nil {
				m.Error = m.inspect.err
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the main model.
func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}

	if m.Error != nil && !m.headless {
		return m.inspect.renderError(m.Error)
	}

	switch m.sessionState {
	case inspectInstanceInput:
		if m.instanceForm != nil {
			return m.instanceForm.View()
		}
		return ""
	case inspectLoading:
		return m.renderLoading()
	case inspectViewing:
		if m.inspect != nil {
			return m.inspect.View()
		}
		return ""
	default:
		return ""
	}
}

func (m MainModel) renderLoading() string {
	return m.styles.Muted.Margin(2, 4).Render("Loading instance state...")
}

// NewInspectApp creates a new inspect application with the given configuration.
func NewInspectApp(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	instanceID string,
	instanceName string,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) (*MainModel, error) {
	// Determine initial session state
	sessionState := inspectInstanceInput
	hasIdentifier := instanceID != "" || instanceName != ""

	if hasIdentifier || headless {
		// Skip input form, go straight to loading
		sessionState = inspectLoading
	}

	// Create instance input form (for interactive mode without pre-set identifier)
	var instanceForm *InstanceInputFormModel
	if !hasIdentifier && !headless {
		instanceForm = NewInstanceInputFormModel(bluelinkStyles)
	}

	// Create the inspect model
	inspect := NewInspectModel(
		deployEngine,
		logger,
		instanceID,
		instanceName,
		bluelinkStyles,
		headless,
		headlessWriter,
		jsonMode,
	)

	model := &MainModel{
		sessionState: sessionState,
		instanceForm: instanceForm,
		inspect:      inspect,
		instanceID:   instanceID,
		instanceName: instanceName,
		headless:     headless,
		jsonMode:     jsonMode,
		engine:       deployEngine,
		logger:       logger,
		styles:       bluelinkStyles,
	}

	return model, nil
}

// InstanceInputMsg is sent when the user enters an instance identifier.
type InstanceInputMsg struct {
	InstanceID   string
	InstanceName string
}

// InstanceInputFormModel provides a form for entering instance name/ID.
type InstanceInputFormModel struct {
	form   *huh.Form
	styles *stylespkg.Styles

	instanceName string
}

// NewInstanceInputFormModel creates a new instance input form.
func NewInstanceInputFormModel(styles *stylespkg.Styles) *InstanceInputFormModel {
	model := &InstanceInputFormModel{
		styles: styles,
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("instanceName").
				Title("Instance Name or ID").
				Description("Enter the name or ID of the instance to inspect.").
				Placeholder("my-app-production").
				Value(&model.instanceName).
				Validate(func(value string) error {
					trimmed := strings.TrimSpace(value)
					if trimmed == "" {
						return errors.New("instance name or ID is required")
					}
					return nil
				}),
		),
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	return model
}

// Init initializes the form.
func (m *InstanceInputFormModel) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages.
func (m *InstanceInputFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
	}

	if m.form.State == huh.StateCompleted {
		instanceName := strings.TrimSpace(m.form.GetString("instanceName"))
		return m, func() tea.Msg {
			return InstanceInputMsg{
				InstanceName: instanceName,
			}
		}
	}

	return m, cmd
}

// View renders the form.
func (m *InstanceInputFormModel) View() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Title.MarginLeft(2).Render("Inspect Instance"))
	sb.WriteString("\n\n")
	sb.WriteString(m.form.View())
	return sb.String()
}
