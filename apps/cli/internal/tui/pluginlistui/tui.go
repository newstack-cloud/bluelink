package pluginlistui

import (
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type listSessionState int

const (
	listLoading listSessionState = iota
	listViewing
	listSearching
)

// MainModel is the top-level model for the plugin list command TUI.
type MainModel struct {
	sessionState listSessionState
	quitting     bool

	// All loaded plugins (post type-filter, pre search-filter).
	allPlugins []*plugins.InstalledPlugin
	// Plugins after search filter is applied.
	filteredPlugins []*plugins.InstalledPlugin

	cursor int

	searchTerm  string
	searchInput textinput.Model

	typeFilter     string
	headless       bool
	headlessWriter io.Writer
	printer        *headless.Printer
	styles         *stylespkg.Styles

	width  int
	height int

	Error error
}

// ListAppOptions contains options for creating a new plugin list app.
type ListAppOptions struct {
	TypeFilter     string
	Search         string
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
}

// NewListApp creates a new plugin list TUI application.
func NewListApp(opts ListAppOptions) (*MainModel, error) {
	printer := createHeadlessPrinter(opts.Headless, opts.HeadlessWriter)

	searchInput := textinput.New()
	searchInput.Placeholder = "search by plugin name..."
	searchInput.CharLimit = 100
	searchInput.Width = 40

	return &MainModel{
		sessionState:   listLoading,
		typeFilter:     opts.TypeFilter,
		searchTerm:     opts.Search,
		searchInput:    searchInput,
		headless:       opts.Headless,
		headlessWriter: opts.HeadlessWriter,
		printer:        printer,
		styles:         opts.Styles,
		width:          80,
	}, nil
}

func (m MainModel) Init() tea.Cmd {
	return loadPluginsCmd(m.typeFilter)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.sessionState == listSearching {
		return m.handleSearchInput(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case PluginsLoadedMsg:
		return m.handlePluginsLoaded(msg)

	case PluginsLoadErrorMsg:
		return m.handlePluginsLoadError(msg)

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m MainModel) handlePluginsLoaded(msg PluginsLoadedMsg) (tea.Model, tea.Cmd) {
	m.allPlugins = msg.Plugins
	m.filteredPlugins = filterBySearch(m.allPlugins, m.searchTerm)
	m.sessionState = listViewing
	m.cursor = 0

	if m.headless {
		m.dispatchHeadlessOutput(m.filteredPlugins, nil)
		return m, tea.Quit
	}

	return m, nil
}

func (m MainModel) handlePluginsLoadError(msg PluginsLoadErrorMsg) (tea.Model, tea.Cmd) {
	m.Error = msg.Err
	if m.headless {
		m.dispatchHeadlessOutput(nil, msg.Err)
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
		if m.cursor < len(m.filteredPlugins)-1 {
			m.cursor++
		}

	case "/":
		m.searchInput.SetValue(m.searchTerm)
		m.searchInput.Focus()
		m.sessionState = listSearching
		return m, textinput.Blink

	case "esc":
		if m.searchTerm != "" {
			m.searchTerm = ""
			m.filteredPlugins = filterBySearch(m.allPlugins, "")
			m.cursor = 0
		}
	}

	return m, nil
}

func (m MainModel) handleSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			newSearch := strings.TrimSpace(m.searchInput.Value())
			m.searchTerm = newSearch
			m.searchInput.Blur()
			m.filteredPlugins = filterBySearch(m.allPlugins, newSearch)
			m.cursor = 0
			m.sessionState = listViewing
			return m, nil

		case "esc":
			m.searchInput.Blur()
			m.sessionState = listViewing
			return m, nil

		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m MainModel) View() string {
	if m.headless {
		return ""
	}

	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("See you next time.")
	}

	if m.Error != nil {
		return m.styles.Error.Margin(2, 4).Render("Error: " + m.Error.Error())
	}

	switch m.sessionState {
	case listLoading:
		return m.styles.Muted.Margin(2, 4).Render("Loading plugins...")
	case listSearching:
		return m.renderSearchView()
	default:
		return m.renderList()
	}
}

func createHeadlessPrinter(isHeadless bool, headlessWriter io.Writer) *headless.Printer {
	if !isHeadless || headlessWriter == nil {
		return nil
	}
	prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[plugins] ")
	return headless.NewPrinter(prefixedWriter, 80)
}
