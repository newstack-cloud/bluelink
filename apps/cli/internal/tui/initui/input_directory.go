package initui

import (
	"errors"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// InputDirectoryModel handles the directory input stage where users provide
// the directory for the project.
type InputDirectoryModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	autoComplete bool

	// Bound form values
	directory string
}

// InputDirectoryInitialValues holds the initial values for the directory input
// passed from InitialState.
type InputDirectoryInitialValues struct {
	Directory string
}

func (m InputDirectoryModel) Init() tea.Cmd {
	if m.autoComplete {
		return inputDirectoryCompleteCmd(
			m.directory,
		)
	}

	return m.form.Init()
}

func (m InputDirectoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		cmds = append(cmds, inputDirectoryCompleteCmd(
			m.form.GetString("directory"),
		))
	}

	return m, tea.Batch(cmds...)
}

func (m InputDirectoryModel) View() string {
	if m.autoComplete {
		return ""
	}
	return m.form.View()
}

// NewInputDirectoryModel creates a new InputDirectoryModel with the given initial values.
func NewInputDirectoryModel(
	initialValues InputDirectoryInitialValues,
	projectName string,
	bluelinkStyles *stylespkg.Styles,
) *InputDirectoryModel {
	model := &InputDirectoryModel{
		styles:    bluelinkStyles,
		directory: initialValues.Directory,
	}

	log.Printf("DEBUG: projectName=%q, directory=%q\n", projectName, model.directory)

	// Determine if we should auto-complete (skip the form)
	// Skip only if the directory value is explicitly set (non-default)
	directorySet := strings.TrimSpace(initialValues.Directory) != ""

	model.autoComplete = directorySet

	if model.directory == "" {
		model.directory = projectName
	}

	// Build the form with Bluelink theme
	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("directory").
				Title("Directory").
				Description(
					"The directory for your Bluelink project, " +
						"you can use \".\" for the current directory.",
				).
				Placeholder(".").
				Value(&model.directory).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("directory cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(stylespkg.NewBluelinkHuhTheme())

	return model
}
