package sharedui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type SelectBlueprintMsg struct {
	BlueprintFile string
	Source        string
}

func selectBlueprintCmd(blueprintFile string, source string) tea.Cmd {
	return func() tea.Msg {
		return SelectBlueprintMsg{
			BlueprintFile: blueprintFile,
			Source:        source,
		}
	}
}

type SelectBlueprintSourceMsg struct {
	Source string
}

func selectBlueprintSourceCmd(source string) tea.Cmd {
	return func() tea.Msg {
		return SelectBlueprintSourceMsg{
			Source: source,
		}
	}
}

type ClearSelectedBlueprintMsg struct{}

func clearSelectedBlueprintCmd() tea.Cmd {
	return func() tea.Msg {
		return ClearSelectedBlueprintMsg{}
	}
}

type SelectBlueprintFileErrorMsg struct {
	Err error
}

func selectBlueprintFileErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return SelectBlueprintFileErrorMsg{
			Err: err,
		}
	}
}

type SelectBlueprintStartMsg struct {
	Selection string
}

func selectBlueprintStartCmd(selection string) tea.Cmd {
	return func() tea.Msg {
		return SelectBlueprintStartMsg{
			Selection: selection,
		}
	}
}
