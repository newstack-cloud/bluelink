package listui

import (
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/inspectui"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"go.uber.org/zap"
)

const pageSize = 20

var quitTextStyle = stylespkg.NewDefaultStyles(
	stylespkg.NewBluelinkPalette(),
).Muted.Margin(1, 0, 2, 4)

type listSessionState int

const (
	listLoading listSessionState = iota
	listViewing
	listSearching  // User is typing search query
	listInspecting // Viewing a selected instance
)

// MainModel is the top-level model for the list command TUI.
type MainModel struct {
	sessionState listSessionState
	quitting     bool

	// Pagination state
	currentPage int
	totalPages  int
	totalCount  int

	// Search state
	searchTerm  string
	searchInput textinput.Model

	// Current page data
	instances []state.InstanceSummary

	// Selection state
	cursor int

	// Selected instance (set when user presses enter)
	SelectedInstanceID   string
	SelectedInstanceName string

	// Embedded inspect model for viewing selected instance
	inspect *inspectui.InspectModel

	// Runtime state
	headless bool
	jsonMode bool

	// Dependencies
	engine engine.DeployEngine
	logger *zap.Logger
	styles *stylespkg.Styles

	// Output
	headlessWriter io.Writer
	printer        *headless.Printer

	// Window size
	width  int
	height int

	Error error
}

// Init initializes the main model.
func (m MainModel) Init() tea.Cmd {
	// In headless mode, load all instances at once (no pagination)
	if m.headless {
		return loadAllCmd(m.engine, m.searchTerm)
	}
	return loadPageCmd(m.engine, m.searchTerm, 0)
}

// Update handles messages for the main model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route messages to inspect model when in inspect mode
	if m.sessionState == listInspecting && m.inspect != nil {
		return m.handleInspectModeUpdate(msg)
	}

	// Handle search input mode
	if m.sessionState == listSearching {
		return m.handleSearchInput(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case PageLoadedMsg:
		return m.handlePageLoaded(msg)

	case PageLoadErrorMsg:
		return m.handlePageLoadError(msg)

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m MainModel) handlePageLoaded(msg PageLoadedMsg) (tea.Model, tea.Cmd) {
	m.instances = msg.Instances
	m.totalCount = msg.TotalCount
	m.currentPage = msg.Page
	m.totalPages = (msg.TotalCount + pageSize - 1) / pageSize
	if m.totalPages == 0 {
		m.totalPages = 1
	}
	m.sessionState = listViewing
	m.cursor = 0

	if m.headless {
		m.dispatchHeadlessOutput(msg.Instances, msg.TotalCount, nil)
		return m, tea.Quit
	}

	return m, nil
}

func (m MainModel) handlePageLoadError(msg PageLoadErrorMsg) (tea.Model, tea.Cmd) {
	m.Error = msg.Err
	if m.headless {
		m.dispatchHeadlessOutput(nil, 0, msg.Err)
		return m, tea.Quit
	}
	return m, nil
}

func (m MainModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sessionState == listLoading {
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.instances)-1 {
			m.cursor++
		}

	case "left", "h":
		if m.currentPage > 0 {
			m.sessionState = listLoading
			return m, loadPageCmd(m.engine, m.searchTerm, m.currentPage-1)
		}

	case "right", "l":
		if m.currentPage < m.totalPages-1 {
			m.sessionState = listLoading
			return m, loadPageCmd(m.engine, m.searchTerm, m.currentPage+1)
		}

	case "enter":
		if len(m.instances) > 0 && m.cursor < len(m.instances) {
			return m.navigateToInspect(m.instances[m.cursor])
		}

	case "/":
		// Enter search mode
		m.searchInput.SetValue(m.searchTerm)
		m.searchInput.Focus()
		m.sessionState = listSearching
		return m, textinput.Blink

	case "esc":
		// Clear search if there is one
		if m.searchTerm != "" {
			m.searchTerm = ""
			m.sessionState = listLoading
			return m, loadPageCmd(m.engine, "", 0)
		}
	}

	return m, nil
}

// handleSearchInput handles keyboard input when in search mode.
func (m MainModel) handleSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Execute search
			newSearch := strings.TrimSpace(m.searchInput.Value())
			m.searchTerm = newSearch
			m.searchInput.Blur()
			m.sessionState = listLoading
			return m, loadPageCmd(m.engine, newSearch, 0)

		case "esc":
			// Cancel search, return to viewing
			m.searchInput.Blur()
			m.sessionState = listViewing
			return m, nil

		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Forward to text input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// navigateToInspect transitions to inspect mode for the selected instance.
func (m MainModel) navigateToInspect(inst state.InstanceSummary) (tea.Model, tea.Cmd) {
	m.SelectedInstanceID = inst.InstanceID
	m.SelectedInstanceName = inst.InstanceName

	// Create the inspect model
	m.inspect = inspectui.NewInspectModel(
		m.engine,
		m.logger,
		inst.InstanceID,
		inst.InstanceName,
		m.styles,
		false, // headless
		m.headlessWriter,
		false, // jsonMode
	)

	// Mark as embedded in list so footer shows "esc back to list"
	m.inspect.SetEmbeddedInList(true)

	m.sessionState = listInspecting

	// Initialize and start fetching instance state
	cmds := []tea.Cmd{
		m.inspect.Init(),
		inspectui.FetchInstanceStateCmd(*m.inspect),
	}

	// Pass window size to inspect model
	if m.width > 0 && m.height > 0 {
		cmd := m.updateInspectModel(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleInspectModeUpdate routes messages to the inspect model and handles back navigation.
func (m MainModel) handleInspectModeUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle key events for quit/back navigation
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.inspect != nil && !m.inspect.IsInSubView() {
				return m.navigateBackToList()
			}
		}
	}

	// Track window size
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsMsg.Width
		m.height = wsMsg.Height
	}

	// Handle back-to-list message
	if _, ok := msg.(inspectui.BackToListMsg); ok {
		return m.navigateBackToList()
	}

	// Forward to inspect model
	cmd := m.updateInspectModel(msg)

	// Sync error state
	if m.inspect != nil && m.inspect.GetError() != nil {
		m.Error = m.inspect.GetError()
	}

	return m, cmd
}

// navigateBackToList transitions back to list view from inspect mode.
func (m MainModel) navigateBackToList() (tea.Model, tea.Cmd) {
	m.sessionState = listViewing
	m.inspect = nil
	m.SelectedInstanceID = ""
	m.SelectedInstanceName = ""
	return m, nil
}

// updateInspectModel forwards a message to the inspect model and updates the pointer.
func (m *MainModel) updateInspectModel(msg tea.Msg) tea.Cmd {
	if m.inspect == nil {
		return nil
	}
	var cmd tea.Cmd
	var model tea.Model
	model, cmd = m.inspect.Update(msg)
	if im, ok := model.(inspectui.InspectModel); ok {
		m.inspect = &im
	}
	return cmd
}

// View renders the main model.
func (m MainModel) View() string {
	if m.headless {
		return ""
	}

	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}

	if m.Error != nil {
		return m.renderError(m.Error)
	}

	switch m.sessionState {
	case listLoading:
		return m.styles.Muted.Margin(2, 4).Render("Loading instances...")
	case listSearching:
		return m.renderSearchView()
	case listInspecting:
		if m.inspect != nil {
			return m.inspect.View()
		}
		return ""
	default:
		return m.renderList()
	}
}

func (m MainModel) renderSearchView() string {
	var sb strings.Builder

	// Title
	sb.WriteString("\n")
	sb.WriteString(m.styles.Title.MarginLeft(2).Render("Search Instances"))
	sb.WriteString("\n\n")

	// Search input
	sb.WriteString("  ")
	sb.WriteString(m.styles.Key.Render("/"))
	sb.WriteString(" ")
	sb.WriteString(m.searchInput.View())
	sb.WriteString("\n\n")

	// Help text
	sb.WriteString(m.styles.Muted.MarginLeft(2).Render("enter to search • esc to cancel"))
	sb.WriteString("\n")

	return sb.String()
}

func (m MainModel) renderList() string {
	var sb strings.Builder

	// Title
	sb.WriteString("\n")
	title := "Blueprint Instances"
	if m.searchTerm != "" {
		title = "Blueprint Instances (search: \"" + m.searchTerm + "\")"
	}
	sb.WriteString(m.styles.Title.MarginLeft(2).Render(title))
	sb.WriteString("\n\n")

	if len(m.instances) == 0 {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render("No instances found."))
		sb.WriteString("\n")
	} else {
		for i, inst := range m.instances {
			line := m.renderInstanceLine(i, inst)
			sb.WriteString(line)
			sb.WriteString("\n\n") // Extra newline for vertical spacing between items
		}
	}

	// Footer with pagination info and keybindings
	sb.WriteString("\n")
	footer := m.renderFooter()
	sb.WriteString(m.styles.Muted.Render(footer))
	sb.WriteString("\n")

	return sb.String()
}

func (m MainModel) renderInstanceLine(index int, inst state.InstanceSummary) string {
	var sb strings.Builder

	cursor := "  "
	if index == m.cursor {
		cursor = m.styles.Selected.Render("> ")
	}

	name := inst.InstanceName
	if index == m.cursor {
		name = m.styles.Selected.Render(name)
	}

	status := renderStatus(inst.Status, m.styles)
	timestamp := formatTimestamp(inst.LastDeployedTimestamp)

	// Line 1: cursor + name + status
	sb.WriteString(cursor)
	sb.WriteString(name)
	sb.WriteString("  ")
	sb.WriteString(status)
	sb.WriteString("\n")

	// Line 2: indented ID
	sb.WriteString("    ")
	sb.WriteString(m.styles.Muted.Render(inst.InstanceID))
	sb.WriteString("\n")

	// Line 3: last deployed timestamp
	sb.WriteString("    ")
	sb.WriteString(m.styles.Muted.Render("Last deployed: " + timestamp))

	return sb.String()
}

func (m MainModel) renderFooter() string {
	var sb strings.Builder

	sb.WriteString("  Page ")
	sb.WriteString(itoa(m.currentPage + 1))
	sb.WriteString(" of ")
	sb.WriteString(itoa(m.totalPages))
	sb.WriteString(" (")
	sb.WriteString(itoa(m.totalCount))
	sb.WriteString(" total)")
	sb.WriteString("\n")

	sb.WriteString("  ")
	sb.WriteString(m.styles.Key.Render("↑/↓"))
	sb.WriteString(m.styles.Muted.Render(" navigate  "))
	sb.WriteString(m.styles.Key.Render("←/→"))
	sb.WriteString(m.styles.Muted.Render(" pages  "))
	sb.WriteString(m.styles.Key.Render("/"))
	sb.WriteString(m.styles.Muted.Render(" search  "))
	if m.searchTerm != "" {
		sb.WriteString(m.styles.Key.Render("esc"))
		sb.WriteString(m.styles.Muted.Render(" clear  "))
	}
	sb.WriteString(m.styles.Key.Render("enter"))
	sb.WriteString(m.styles.Muted.Render(" select  "))
	sb.WriteString(m.styles.Key.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" quit"))

	return sb.String()
}

func (m MainModel) renderError(err error) string {
	return m.styles.Error.Margin(2, 4).Render("Error: " + err.Error())
}

// NewListApp creates a new list application with the given configuration.
func NewListApp(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	search string,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) (*MainModel, error) {
	printer := createHeadlessPrinter(headless, headlessWriter)

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "search by instance name..."
	searchInput.CharLimit = 100
	searchInput.Width = 40

	model := &MainModel{
		sessionState:   listLoading,
		searchTerm:     search,
		searchInput:    searchInput,
		engine:         deployEngine,
		logger:         logger,
		styles:         bluelinkStyles,
		headless:       headless,
		headlessWriter: headlessWriter,
		printer:        printer,
		jsonMode:       jsonMode,
	}

	return model, nil
}

func createHeadlessPrinter(isHeadless bool, headlessWriter io.Writer) *headless.Printer {
	if !isHeadless || headlessWriter == nil {
		return nil
	}
	prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[list] ")
	return headless.NewPrinter(prefixedWriter, 80)
}
