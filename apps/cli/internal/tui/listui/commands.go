package listui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
)

// PageLoadedMsg is sent when a page of instances has been loaded.
type PageLoadedMsg struct {
	Instances  []state.InstanceSummary
	TotalCount int
	Page       int
}

// PageLoadErrorMsg is sent when loading a page fails.
type PageLoadErrorMsg struct {
	Err error
}

// loadPageCmd creates a command that fetches a page of instances from the server.
func loadPageCmd(deployEngine engine.DeployEngine, search string, page int) tea.Cmd {
	return func() tea.Msg {
		offset := page * pageSize
		result, err := deployEngine.ListBlueprintInstances(
			context.Background(),
			state.ListInstancesParams{
				Search: search,
				Offset: offset,
				Limit:  pageSize,
			},
		)
		if err != nil {
			return PageLoadErrorMsg{Err: err}
		}
		return PageLoadedMsg{
			Instances:  result.Instances,
			TotalCount: result.TotalCount,
			Page:       page,
		}
	}
}

// loadAllCmd creates a command that fetches all instances (for headless mode).
func loadAllCmd(deployEngine engine.DeployEngine, search string) tea.Cmd {
	return func() tea.Msg {
		result, err := deployEngine.ListBlueprintInstances(
			context.Background(),
			state.ListInstancesParams{
				Search: search,
				Limit:  0, // No limit - fetch all
			},
		)
		if err != nil {
			return PageLoadErrorMsg{Err: err}
		}
		return PageLoadedMsg{
			Instances:  result.Instances,
			TotalCount: result.TotalCount,
			Page:       0,
		}
	}
}
