package validateui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/diagutils"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"
)

var (
	bluelinkStyles = stylespkg.NewDefaultStyles(stylespkg.NewBluelinkPalette())

	validateCategoryStyle     = bluelinkStyles.Category
	diagnosticLevelErrorStyle = bluelinkStyles.Error.MarginLeft(2)
	diagnosticLevelWarnStyle  = bluelinkStyles.Warning.MarginLeft(2)
	diagnosticLevelInfoStyle  = bluelinkStyles.Info.MarginLeft(2)
	diagnosticMessageStyle    = bluelinkStyles.DiagnosticMessage
	locationStyle             = bluelinkStyles.Location
	diagnosticActionStyle     = bluelinkStyles.DiagnosticAction

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return bluelinkStyles.Title.BorderStyle(b)
	}()
)

type ValidateResultMsg *types.BlueprintValidationEvent

type ValidateErrMsg struct {
	err error
}

type ValidateStreamMsg struct{}

type ValidateModel struct {
	spinner         spinner.Model
	viewport        viewport.Model
	hasDimensions   bool
	engine          engine.DeployEngine
	blueprintFile   string
	blueprintSource string
	resultStream    chan types.BlueprintValidationEvent
	collected       []*types.BlueprintValidationEvent
	errStream       chan error
	streaming       bool
	err             error
	width           int
	finished        bool
	// This is separate from an error in the sense that it indicates
	// that validation failed to the parent context to ensure that the program
	// exits with a non-zero exit code.
	// This is separate from the err field so that it allows us to render the
	// diagnostics in the TUI before exiting.
	validationFailed bool
	logger           *zap.Logger
	renderer         *glamour.TermRenderer
	headless         bool
	headlessWriter   io.Writer
}

func (m ValidateModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ValidateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		footerHeight := lipgloss.Height(m.footerView())

		if !m.hasDimensions {
			m.viewport = viewport.New(msg.Width, msg.Height-footerHeight)
			m.hasDimensions = true
		}
	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = msg.BlueprintFile
		m.blueprintSource = msg.Source
		// SelectBlueprintMsg can be sent multiple times, we need to make sure we aren't collecting
		// duplicate results from the stream by not dispatching commands that will create multiple
		// consumers.
		if !m.streaming {
			cmds = append(cmds, startValidateStreamCmd(m, m.logger), waitForNextResultCmd(m), checkForErrCmd(m))
		}
		m.streaming = true
	case ValidateResultMsg:
		m.collected = append(m.collected, msg)
		cmds = append(cmds, checkForErrCmd(m))
		if !msg.End {
			cmds = append(cmds, waitForNextResultCmd(m))
		} else {
			m.finished = true
			m.validationFailed = checkForValidationFailure(m.collected)
			m.viewport.SetContent(m.resultContents())
			if m.headless {
				// Make sure we exit after validation completes in headless mode.
				cmds = append(cmds, tea.Quit)
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ValidateErrMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
	}

	var viewportCmd tea.Cmd
	m.viewport, viewportCmd = m.viewport.Update(msg)
	cmds = append(cmds, viewportCmd)

	return m, tea.Batch(cmds...)
}

func (m ValidateModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m ValidateModel) View() string {
	if m.headless {
		// In headless mode, print directly to configured writer and return empty string
		m.renderHeadless()
		return ""
	}

	if m.err != nil {
		return renderError(m.err)
	}

	if !m.finished {
		return fmt.Sprintf("\n\n %s Validating project...\n\n", m.spinner.View())
	}

	return fmt.Sprintf("%s\n%s", m.viewport.View(), m.footerView())
}

func (m ValidateModel) resultContents() string {
	sb := strings.Builder{}

	for _, result := range m.collected {
		containerStyle := lipgloss.NewStyle().Padding(1, 1).Width(m.width)

		itemSB := strings.Builder{}
		itemSB.WriteString(validateCategoryStyle.Render("diagnostic"))

		levelOutput := renderDiagnosticLevel(result.Diagnostic.Level)
		itemSB.WriteString(levelOutput)

		messageOutput, err := renderDiagnosticMessage(&result.Diagnostic, m.renderer)
		if err != nil {
			return renderError(err)
		}

		itemSB.WriteString(messageOutput)

		containerRendered := containerStyle.Render(itemSB.String())
		sb.WriteString(containerRendered)
	}

	return sb.String()
}

func (m ValidateModel) renderHeadless() {
	if m.err != nil {
		fmt.Fprintln(m.headlessWriter, renderError(m.err))
		return
	}

	fmt.Fprintf(m.headlessWriter, "Validating blueprint: %s\n\n", m.blueprintFile)

	for _, result := range m.collected {
		containerStyle := lipgloss.NewStyle().Padding(1, 1).Width(m.width)

		itemSB := strings.Builder{}
		itemSB.WriteString(validateCategoryStyle.Render("diagnostic"))

		levelOutput := renderDiagnosticLevel(result.Diagnostic.Level)
		itemSB.WriteString(levelOutput)

		messageOutput, err := renderDiagnosticMessagePlain(&result.Diagnostic)
		if err != nil {
			fmt.Fprintln(m.headlessWriter, renderError(err))
			continue
		}

		itemSB.WriteString(messageOutput)

		containerRendered := containerStyle.Render(itemSB.String())
		fmt.Fprintln(m.headlessWriter, containerRendered)
	}

	if !m.finished {
		fmt.Fprintln(m.headlessWriter, "\nValidating project...")
	} else {
		fmt.Fprintln(m.headlessWriter, "\nValidation complete.")
	}
}

func renderDiagnosticMessagePlain(diagnostic *bpcore.Diagnostic) (string, error) {
	sb := strings.Builder{}

	if diagnostic.Context == nil {
		sb.WriteString(diagnosticMessageStyle.Render(diagnostic.Message))
		if hasPreciseRange(diagnostic.Range) {
			sb.WriteString(locationStyle.Render(
				fmt.Sprintf(
					"(line %d, column %d)",
					diagnostic.Range.Start.Line,
					diagnostic.Range.Start.Column,
				),
			))
		}
		return sb.String(), nil
	}

	sb.WriteString(diagnosticMessageStyle.Render(string(diagnostic.Message)))
	suggestedActions, err := renderSuggestedActionsPlain(
		diagnostic.Context.SuggestedActions,
		diagnostic.Context.Metadata,
	)
	if err != nil {
		return sb.String(), err
	}

	sb.WriteString(suggestedActions)
	return sb.String(), nil
}

func renderSuggestedActionsPlain(
	suggestedActions []errors.SuggestedAction,
	diagMetadata map[string]any,
) (string, error) {
	sb := strings.Builder{}

	if len(suggestedActions) > 0 {
		sb.WriteString("\nSuggested Actions:\n")
	}

	for i, suggestedAction := range suggestedActions {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestedAction.Title))
		sb.WriteString(fmt.Sprintf("     %s\n", suggestedAction.Description))

		concreteAction := diagutils.GetConcreteAction(suggestedAction, diagMetadata)
		if concreteAction != nil {
			concreteActionOutput := concreteActionPlain(concreteAction)
			sb.WriteString(fmt.Sprintf("     %s\n", concreteActionOutput))
		}

		if i < len(suggestedActions)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

func concreteActionPlain(concreteAction *diagutils.ConcreteAction) string {
	if concreteAction == nil {
		return ""
	}

	sb := strings.Builder{}
	if len(concreteAction.Commands) > 0 {
		sb.WriteString("Commands:\n")
		for _, command := range concreteAction.Commands {
			sb.WriteString(fmt.Sprintf("  - %s\n", command))
		}
		sb.WriteString("\n")
	}

	if len(concreteAction.Links) > 0 {
		sb.WriteString("Links:\n")
		for _, link := range concreteAction.Links {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", link.Title, link.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderDiagnosticMessage(diagnostic *bpcore.Diagnostic, renderer *glamour.TermRenderer) (string, error) {
	sb := strings.Builder{}

	if diagnostic.Context == nil {
		sb.WriteString(diagnosticMessageStyle.Render(diagnostic.Message))
		if hasPreciseRange(diagnostic.Range) {
			sb.WriteString(locationStyle.Render(
				fmt.Sprintf(
					"(line %d, column %d)",
					diagnostic.Range.Start.Line,
					diagnostic.Range.Start.Column,
				),
			))
		}
		return sb.String(), nil
	}

	sb.WriteString(diagnosticMessageStyle.Render(string(diagnostic.Message)))
	suggestedActions, err := renderSuggestedActions(
		diagnostic.Context.SuggestedActions,
		renderer,
		diagnostic.Context.Metadata,
	)
	if err != nil {
		return sb.String(), err
	}

	sb.WriteString(diagnosticActionStyle.Render(suggestedActions))
	return sb.String(), nil
}

func renderSuggestedActions(
	suggestedActions []errors.SuggestedAction,
	renderer *glamour.TermRenderer,
	diagMetadata map[string]any,
) (string, error) {
	sb := strings.Builder{}

	if len(suggestedActions) > 0 {
		sb.WriteString("\n# Suggested Actions\n")
	}

	for _, suggestedAction := range suggestedActions {
		markdown := suggestedActionMarkdown(suggestedAction, diagMetadata)
		sb.WriteString(markdown)

	}

	return renderer.Render(sb.String())
}

func suggestedActionMarkdown(suggestedAction errors.SuggestedAction, diagMetadata map[string]any) string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("## %s\n\n%s\n", suggestedAction.Title, suggestedAction.Description))
	concreteAction := diagutils.GetConcreteAction(suggestedAction, diagMetadata)
	if concreteAction != nil {
		concreteActionOutput := concreteActionMarkdown(concreteAction)
		sb.WriteString(fmt.Sprintf("\n%s\n", concreteActionOutput))
	}

	return sb.String()
}

func concreteActionMarkdown(concreteAction *diagutils.ConcreteAction) string {
	if concreteAction == nil {
		return ""
	}

	sb := strings.Builder{}
	if len(concreteAction.Commands) > 0 {
		sb.WriteString("Try running one of the following commands:\n")
		for _, command := range concreteAction.Commands {
			sb.WriteString(fmt.Sprintf("  ```bash\n%s\n```\n", command))
		}
		sb.WriteString("\n")
	}

	if len(concreteAction.Links) > 0 {
		sb.WriteString("Try visiting one of the following links:\n\n")
		for _, link := range concreteAction.Links {
			sb.WriteString(fmt.Sprintf("[%s](%s)\n", link.Title, link.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderDiagnosticLevel(level bpcore.DiagnosticLevel) string {
	switch level {
	case bpcore.DiagnosticLevelError:
		return diagnosticLevelErrorStyle.Render(
			diagnosticLevelName(level),
		)
	case bpcore.DiagnosticLevelWarning:
		return diagnosticLevelWarnStyle.Render(
			diagnosticLevelName(level),
		)
	case bpcore.DiagnosticLevelInfo:
		return diagnosticLevelInfoStyle.Render(
			diagnosticLevelName(level),
		)
	default:
		return ""
	}
}

func NewValidateModel(
	engine engine.DeployEngine,
	logger *zap.Logger,
	headless bool,
	headlessWriter io.Writer,
) ValidateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = bluelinkStyles.Spinner
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	return ValidateModel{
		spinner:        s,
		engine:         engine,
		logger:         logger,
		resultStream:   make(chan types.BlueprintValidationEvent),
		errStream:      make(chan error),
		renderer:       renderer,
		headless:       headless,
		headlessWriter: headlessWriter,
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

func renderError(err error) string {
	sb := strings.Builder{}
	sb.WriteString(diagnosticLevelErrorStyle.Render(err.Error()))
	sb.WriteString("\n")
	return sb.String()
}

func checkForValidationFailure(diagnostics []*types.BlueprintValidationEvent) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Diagnostic.Level == bpcore.DiagnosticLevelError {
			return true
		}
	}
	return false
}
