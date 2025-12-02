package initui

import tea "github.com/charmbracelet/bubbletea"

// InputFormCompleteMsg signals completion of the input form stage.
// This message carries all the form values to the parent InitModel.
type InputFormCompleteMsg struct {
	ProjectName     string
	BlueprintFormat string
	NoGit           bool
}

func inputFormCompleteCmd(projectName, blueprintFormat string, noGit bool) tea.Cmd {
	return func() tea.Msg {
		return InputFormCompleteMsg{
			ProjectName:     projectName,
			BlueprintFormat: blueprintFormat,
			NoGit:           noGit,
		}
	}
}
