package initui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

// InputFormModel handles the input form stage where users provide
// project name, blueprint format, and git initialization preference.
type InputFormModel struct {
	form         *huh.Form
	styles       *styles.BluelinkStyles
	autoComplete bool

	// Bound form values
	projectName     string
	blueprintFormat string
	noGit           bool
}

// InputFormInitialValues holds the initial values for the input form
// passed from InitialState.
type InputFormInitialValues struct {
	ProjectName              string
	BlueprintFormat          string
	IsDefaultBlueprintFormat bool
	NoGit                    *bool
	IsDefaultNoGit           bool
}

func (m InputFormModel) Init() tea.Cmd {
	if m.autoComplete {
		return inputFormCompleteCmd(
			m.projectName,
			m.blueprintFormat,
			m.noGit,
		)
	}

	return m.form.Init()
}

func (m InputFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		cmds = append(cmds, inputFormCompleteCmd(
			m.projectName,
			m.blueprintFormat,
			m.noGit,
		))
	}

	return m, tea.Batch(cmds...)
}

func (m InputFormModel) View() string {
	if m.autoComplete {
		return ""
	}
	return m.form.View()
}

// NewInputFormModel creates a new InputFormModel with the given initial values.
func NewInputFormModel(
	initialValues InputFormInitialValues,
	bluelinkStyles *styles.BluelinkStyles,
) *InputFormModel {
	model := &InputFormModel{
		styles:          bluelinkStyles,
		projectName:     initialValues.ProjectName,
		blueprintFormat: initialValues.BlueprintFormat,
		noGit:           false,
	}

	// Set default blueprint format if empty
	if model.blueprintFormat == "" {
		model.blueprintFormat = "yaml"
	}

	// Handle noGit pointer
	if initialValues.NoGit != nil {
		model.noGit = *initialValues.NoGit
	}

	// Determine if we should auto-complete (skip the form)
	// Skip only if ALL values are explicitly set (non-default)
	projectNameSet := strings.TrimSpace(initialValues.ProjectName) != ""
	blueprintFormatSet := initialValues.BlueprintFormat != "" && !initialValues.IsDefaultBlueprintFormat
	noGitSet := initialValues.NoGit != nil && !initialValues.IsDefaultNoGit

	model.autoComplete = projectNameSet && blueprintFormatSet && noGitSet

	// Build the form with Bluelink theme
	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("projectName").
				Title("Project Name").
				Description("The name for your Bluelink project.").
				Placeholder("my-project").
				Value(&model.projectName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("project name cannot be empty")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Key("blueprintFormat").
				Title("Blueprint Format").
				Description("The format for blueprint configuration files.").
				Options(
					huh.NewOption("YAML", "yaml"),
					huh.NewOption("JSON with Comments", "jsonc"),
				).
				Value(&model.blueprintFormat),

			huh.NewConfirm().
				Key("noGit").
				Title("Skip Git Initialization?").
				Description("Choose whether to initialize a git repository.").
				Affirmative("Yes, skip git").
				Negative("No, use git").
				Value(&model.noGit),
		),
	).WithTheme(styles.NewBluelinkHuhTheme())

	return model
}
