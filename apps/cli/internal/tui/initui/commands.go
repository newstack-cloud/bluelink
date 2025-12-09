package initui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/git"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/project"
)

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

// InputDirectoryCompleteMsg signals completion of the directory input stage.
// This message carries the directory value to the parent InitModel.
type InputDirectoryCompleteMsg struct {
	Directory string
}

func inputDirectoryCompleteCmd(directory string) tea.Cmd {
	return func() tea.Msg {
		return InputDirectoryCompleteMsg{
			Directory: directory,
		}
	}
}

// DownloadCompleteMsg signals completion of the download repo stage.
// This message carries the repository path to the parent InitModel.
type DownloadCompleteMsg struct {
	RepoPath string
}

func downloadRepoCompleteCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		return DownloadCompleteMsg{
			RepoPath: repoPath,
		}
	}
}

var templateRepos = map[string]string{
	"scaffold":       "https://github.com/newstack-cloud/bluelink-template-scaffold.git",
	"aws-simple-api": "https://github.com/newstack-cloud/bluelink-template-aws-simple-api.git",
}

func startDownloadCmd(m *DownloadRepoModel) tea.Cmd {
	return func() tea.Msg {
		repoURL, ok := templateRepos[m.template]
		if !ok {
			return DownloadErrorMsg{Err: fmt.Errorf("template %q not found", m.template)}
		}

		err := m.git.Clone(repoURL, m.directory)
		if err != nil {
			return DownloadErrorMsg{Err: err}
		}

		return DownloadCompleteMsg{
			RepoPath: m.directory,
		}
	}
}

// DownloadErrorMsg signals an error during the download repo stage.
// This message carries the error to the parent InitModel.
type DownloadErrorMsg struct {
	Err error
}


// PrepareCompleteMsg signals completion of the prepare project stage.
type PrepareCompleteMsg struct{}

func prepareProjectCompleteCmd() tea.Cmd {
	return func() tea.Msg {
		return PrepareCompleteMsg{}
	}
}

// PrepareErrorMsg signals an error during the prepare project stage.
type PrepareErrorMsg struct {
	Err error
}

// RemoveGitHistoryCompleteMsg signals completion of the remove git history step.
type RemoveGitHistoryCompleteMsg struct{}

func removeGitHistoryCmd(directory string, preparer project.Preparer) tea.Cmd {
	return func() tea.Msg {
		if err := preparer.RemoveGitHistory(directory); err != nil {
			return PrepareErrorMsg{Err: err}
		}
		return RemoveGitHistoryCompleteMsg{}
	}
}

// RemoveMaintainerFilesCompleteMsg signals completion of the remove maintainer files step.
type RemoveMaintainerFilesCompleteMsg struct{}

func removeMaintainerFilesCmd(directory string, preparer project.Preparer) tea.Cmd {
	return func() tea.Msg {
		if err := preparer.RemoveMaintainerFiles(directory); err != nil {
			return PrepareErrorMsg{Err: err}
		}
		return RemoveMaintainerFilesCompleteMsg{}
	}
}

// InitGitRepoCompleteMsg signals completion of the git init step.
type InitGitRepoCompleteMsg struct{}

func initGitRepoCmd(directory string, g git.Git) tea.Cmd {
	return func() tea.Msg {
		if err := g.Init(directory); err != nil {
			return PrepareErrorMsg{Err: err}
		}
		return InitGitRepoCompleteMsg{}
	}
}

// SelectBlueprintFormatCompleteMsg signals completion of the blueprint format selection step.
type SelectBlueprintFormatCompleteMsg struct{}

func selectBlueprintFormatCmd(directory string, format string, preparer project.Preparer) tea.Cmd {
	return func() tea.Msg {
		if err := preparer.SelectBlueprintFormat(directory, format); err != nil {
			return PrepareErrorMsg{Err: err}
		}
		return SelectBlueprintFormatCompleteMsg{}
	}
}

// SubstitutePlaceholdersCompleteMsg signals completion of the placeholder substitution step.
type SubstitutePlaceholdersCompleteMsg struct{}

func substitutePlaceholdersCmd(
	directory string,
	projectName string,
	blueprintFormat string,
	preparer project.Preparer,
) tea.Cmd {
	return func() tea.Msg {
		values := project.NewTemplateValues(projectName, blueprintFormat)
		if err := preparer.SubstitutePlaceholders(directory, values); err != nil {
			return PrepareErrorMsg{Err: err}
		}
		return SubstitutePlaceholdersCompleteMsg{}
	}
}
