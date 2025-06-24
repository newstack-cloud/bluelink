package validateui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/engine"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"go.uber.org/zap"
)

var (
	validateCategroyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#4f46e5"))
	diagnosticLevelErrorStyle = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("#dc2626"))
	diagnosticLevelWarnStyle  = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("#f97316"))
	diagnosticLevelInfoStyle  = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("#2563eb"))
	diagnosticMessageStyle    = lipgloss.NewStyle().MarginLeft(2)
	locationStyle             = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("#4f46e5"))
)

type ValidateResultMsg *types.BlueprintValidationEvent

type ValidateErrMsg struct {
	err error
}

type ValidateStreamMsg struct{}

type item struct {
	result     *types.BlueprintValidationEvent
	filterText string
}

func (i item) FilterValue() string {
	return i.filterText
}

type ValidateModel struct {
	spinner       spinner.Model
	list          list.Model
	engine        engine.DeployEngine
	blueprintFile string
	resultStream  chan types.BlueprintValidationEvent
	collected     []*types.BlueprintValidationEvent
	errStream     chan error
	streaming     bool
	err           error
	width         int
	finished      bool
	logger        *zap.Logger
}

func (m ValidateModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ValidateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case SelectBlueprintMsg:
		m.blueprintFile = msg.blueprintFile
		// SelectBlueprintMsg can be sent multiple times, we need to make sure we aren't collecting
		// duplicate results from the stream by not dispatching commands that will create multiple
		// consumers.
		if !m.streaming {
			cmds = append(cmds, startValidateStreamCmd(m, m.logger), waitForNextResultCmd(m), checkForErrCmd(m))
		}
		m.streaming = true
	case ValidateResultMsg:
		if msg == nil {
			m.finished = true
			return m, tea.Quit
		}
		m.collected = append(m.collected, msg)
		setListItemsCmd := m.list.SetItems(listItemsFromResults(m.collected))
		cmds = append(cmds, setListItemsCmd, waitForNextResultCmd(m), checkForErrCmd(m))
	case spinner.TickMsg:
		log.Println("ValidateModel: spinner tick")
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ValidateErrMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	cmds = append(cmds, listCmd)

	return m, tea.Batch(cmds...)
}

func (m ValidateModel) View() string {
	log.Printf("ValidateModel: Rendering view m.collected length: %d", len(m.collected))
	if m.err != nil {
		return renderError(m.err)
	}

	sb := strings.Builder{}

	for _, result := range m.collected {
		containerStyle := lipgloss.NewStyle().Padding(1, 1).Width(m.width)

		itemSB := strings.Builder{}
		itemSB.WriteString(validateCategroyStyle.Render("diagnostic"))
		if result.Diagnostic.Level == bpcore.DiagnosticLevelError {
			itemSB.WriteString(
				diagnosticLevelErrorStyle.Render(
					diagnosticLevelName(result.Diagnostic.Level),
				),
			)
		} else if result.Diagnostic.Level == bpcore.DiagnosticLevelWarning {
			itemSB.WriteString(
				diagnosticLevelWarnStyle.Render(
					diagnosticLevelName(result.Diagnostic.Level),
				),
			)
		} else if result.Diagnostic.Level == bpcore.DiagnosticLevelInfo {
			itemSB.WriteString(
				diagnosticLevelInfoStyle.Render(
					diagnosticLevelName(result.Diagnostic.Level),
				),
			)
		}
		itemSB.WriteString(diagnosticMessageStyle.Render(result.Diagnostic.Message))
		if hasPreciseRange(result.Diagnostic.Range) {
			itemSB.WriteString(
				locationStyle.Render(
					fmt.Sprintf("(line %d, column %d)", result.Diagnostic.Range.Start.Line, result.Diagnostic.Range.Start.Column),
				),
			)
		}
		containerRendered := containerStyle.Render(itemSB.String())
		sb.WriteString(containerRendered)
		sb.WriteString("\n")
	}
	if !m.finished {
		sb.WriteString(fmt.Sprintf("\n\n %s Validating project...\n\n", m.spinner.View()))
	}
	return sb.String()
}

func NewValidateModel(engine engine.DeployEngine, logger *zap.Logger) ValidateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return ValidateModel{
		spinner:      s,
		engine:       engine,
		logger:       logger,
		list:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		resultStream: make(chan types.BlueprintValidationEvent),
		errStream:    make(chan error),
	}
}

func diagnosticLevelName(level bpcore.DiagnosticLevel) string {
	switch level {
	case bpcore.DiagnosticLevelError:
		return "error"
	case bpcore.DiagnosticLevelWarning:
		return "warning"
	case bpcore.DiagnosticLevelInfo:
		return "info"
	default:
		return "unknown"
	}
}

func hasPreciseRange(r *bpcore.DiagnosticRange) bool {
	return r != nil && r.Start.Line > 0 && r.Start.Column > 0
}

func listItemsFromResults(results []*types.BlueprintValidationEvent) []list.Item {
	items := []list.Item{}
	for _, result := range results {
		items = append(items, item{
			result:     result,
			filterText: resultToPlainText(result),
		})
	}
	return items
}

func resultToPlainText(result *types.BlueprintValidationEvent) string {
	sb := strings.Builder{}
	sb.WriteString("diagnostic")
	sb.WriteString(" ")
	sb.WriteString(diagnosticLevelName(result.Diagnostic.Level))
	sb.WriteString(" ")
	sb.WriteString(result.Diagnostic.Message)
	if hasPreciseRange(result.Diagnostic.Range) {
		sb.WriteString(fmt.Sprintf(" (line %d, column %d)", result.Diagnostic.Range.Start.Line, result.Diagnostic.Range.Start.Column))
	}
	return sb.String()
}

func renderError(err error) string {
	sb := strings.Builder{}
	sb.WriteString(diagnosticLevelErrorStyle.Render(err.Error()))
	sb.WriteString("\n")
	return sb.String()
}
