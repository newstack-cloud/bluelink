package inspectui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// InspectEventMsg wraps a deployment event for the inspect flow.
type InspectEventMsg types.BlueprintInstanceEvent

// AsFinish returns the finish data if this is a finish event.
func (m InspectEventMsg) AsFinish() (*container.DeploymentFinishedMessage, bool) {
	event := types.BlueprintInstanceEvent(m)
	return event.AsFinish()
}

// InspectStreamClosedMsg is sent when the event stream is closed.
type InspectStreamClosedMsg struct{}

// InspectStreamStartedMsg is sent when the event stream has been started.
type InspectStreamStartedMsg struct{}

// InspectErrorMsg is sent when an error occurs.
type InspectErrorMsg struct {
	Err error
}

// InstanceStateFetchedMsg is sent when the instance state has been fetched.
type InstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
	IsInProgress  bool
}

// InstanceNotFoundMsg is sent when the instance is not found.
type InstanceNotFoundMsg struct {
	Err error
}

// InstanceStateRefreshedMsg is sent when the instance state has been refreshed during streaming.
type InstanceStateRefreshedMsg struct {
	InstanceState *state.InstanceState
}

// StateRefreshTickMsg triggers a periodic state refresh during streaming.
type StateRefreshTickMsg struct{}

// BackToListMsg is sent when the user wants to navigate back to the list view.
type BackToListMsg struct{}

// stateRefreshInterval is the interval between state refreshes during streaming.
const stateRefreshInterval = 5 * time.Second

// FetchInstanceStateCmd fetches the instance state from the engine (exported for embedding).
func FetchInstanceStateCmd(model InspectModel) tea.Cmd {
	return fetchInstanceStateCmd(model)
}

// fetchInstanceStateCmd fetches the instance state from the engine.
func fetchInstanceStateCmd(model InspectModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(
			model.engine,
			model.instanceID,
			model.instanceName,
		)

		if instanceState == nil {
			return InstanceNotFoundMsg{
				Err: errInstanceNotFound(model.instanceID, model.instanceName),
			}
		}

		isInProgress := isInProgressStatus(instanceState.Status)

		return InstanceStateFetchedMsg{
			InstanceState: instanceState,
			IsInProgress:  isInProgress,
		}
	}
}

// startStreamingCmd starts the event stream for the instance.
func startStreamingCmd(model InspectModel) tea.Cmd {
	return func() tea.Msg {
		err := model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			model.instanceID,
			"", // Start from beginning
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return InspectErrorMsg{Err: err}
		}

		return InspectStreamStartedMsg{}
	}
}

// waitForNextEventMsg returns a command to wait for the next event.
func waitForNextEventMsg(model InspectModel) tea.Msg {
	event, ok := <-model.eventStream
	if !ok {
		return InspectStreamClosedMsg{}
	}
	return InspectEventMsg(event)
}

// waitForNextEventCmd returns a command to wait for the next event.
func waitForNextEventCmd(model InspectModel) tea.Cmd {
	return func() tea.Msg {
		return waitForNextEventMsg(model)
	}
}

func errInstanceNotFound(instanceID, instanceName string) error {
	identifier := instanceID
	if identifier == "" {
		identifier = instanceName
	}
	return &instanceNotFoundError{identifier: identifier}
}

type instanceNotFoundError struct {
	identifier string
}

func (e *instanceNotFoundError) Error() string {
	return "instance not found: " + e.identifier
}

// startStateRefreshTickerCmd starts the periodic state refresh ticker.
func startStateRefreshTickerCmd() tea.Cmd {
	return tea.Tick(stateRefreshInterval, func(t time.Time) tea.Msg {
		return StateRefreshTickMsg{}
	})
}

// refreshInstanceStateCmd refreshes the instance state from the engine.
func refreshInstanceStateCmd(model InspectModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(
			model.engine,
			model.instanceID,
			model.instanceName,
		)

		if instanceState == nil {
			// Instance disappeared - this shouldn't happen during streaming
			// but we handle it gracefully by not updating the state
			return nil
		}

		return InstanceStateRefreshedMsg{
			InstanceState: instanceState,
		}
	}
}
