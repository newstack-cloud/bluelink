package templatelistui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/templates"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

type listSessionState int

const (
	listLoading listSessionState = iota
	listViewing
)

// MainModel is the top-level model for the templates list command TUI.
type MainModel struct {
	sessionState listSessionState
	quitting     bool

	allTemplates     []templates.Template
	selectWithPreview tea.Model

	searchTerm     string
	headless       bool
	headlessWriter io.Writer
	printer        *headless.Printer
	styles         *stylespkg.Styles

	width  int
	height int

	Error error
}

// ListAppOptions contains options for creating a new template list app.
type ListAppOptions struct {
	Search         string
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
}

// NewListApp creates a new template list TUI application.
func NewListApp(opts ListAppOptions) (*MainModel, error) {
	printer := createHeadlessPrinter(opts.Headless, opts.HeadlessWriter)

	return &MainModel{
		sessionState:   listLoading,
		searchTerm:     opts.Search,
		headless:       opts.Headless,
		headlessWriter: opts.HeadlessWriter,
		printer:        printer,
		styles:         opts.Styles,
		width:          80,
	}, nil
}

func (m MainModel) Init() tea.Cmd {
	return loadTemplatesCmd(m.searchTerm)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.selectWithPreview != nil {
			var cmd tea.Cmd
			m.selectWithPreview, cmd = m.selectWithPreview.Update(msg)
			return m, cmd
		}
		return m, nil

	case TemplatesLoadedMsg:
		return m.handleTemplatesLoaded(msg)

	case TemplatesLoadErrorMsg:
		return m.handleTemplatesLoadError(msg)

	case TemplateSelectedMsg:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}

	if m.selectWithPreview != nil {
		var cmd tea.Cmd
		m.selectWithPreview, cmd = m.selectWithPreview.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m MainModel) handleTemplatesLoaded(
	msg TemplatesLoadedMsg,
) (tea.Model, tea.Cmd) {
	m.allTemplates = msg.Templates
	m.sessionState = listViewing

	if m.headless {
		m.dispatchHeadlessOutput(msg.Templates, nil)
		return m, tea.Quit
	}

	items := templatesToListItems(msg.Templates)
	m.selectWithPreview = sharedui.NewSelectWithPreview(
		"Available Templates",
		items,
		m.styles,
		selectTemplateCmd,
		true,
	)

	cmds := []tea.Cmd{m.selectWithPreview.Init()}
	if m.width > 0 && m.height > 0 {
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleTemplatesLoadError(
	msg TemplatesLoadErrorMsg,
) (tea.Model, tea.Cmd) {
	m.Error = msg.Err
	if m.headless {
		m.dispatchHeadlessOutput(nil, msg.Err)
		return m, tea.Quit
	}
	return m, nil
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

	if m.sessionState == listLoading {
		return m.styles.Muted.Margin(2, 4).Render("Loading templates...")
	}

	if m.selectWithPreview != nil {
		return "\n" + m.selectWithPreview.View()
	}

	return ""
}

func createHeadlessPrinter(
	isHeadless bool,
	headlessWriter io.Writer,
) *headless.Printer {
	if !isHeadless || headlessWriter == nil {
		return nil
	}
	prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[templates] ")
	return headless.NewPrinter(prefixedWriter, 80)
}
