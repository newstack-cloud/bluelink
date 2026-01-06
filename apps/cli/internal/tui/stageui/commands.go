package stageui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// StageEventMsg is a message containing a change staging event.
type StageEventMsg types.ChangeStagingEvent

// StageStreamClosedMsg is sent when the staging event stream is closed.
// This typically happens due to a stream timeout or the connection being dropped.
type StageStreamClosedMsg struct{}

// StageErrorMsg is a message containing an error from the staging process.
type StageErrorMsg struct {
	Err error
}

// StageStartedMsg is a message indicating that staging has started.
type StageStartedMsg struct {
	ChangesetID string
}

// StageCompleteMsg is a message indicating that staging has completed.
// This is emitted to allow parent models to react to staging completion.
type StageCompleteMsg struct {
	ChangesetID   string
	Changes       *changes.BlueprintChanges
	Items         []StageItem
	InstanceState *state.InstanceState // Pre-deployment instance state for unchanged items
}

// InstanceStateFetchedMsg is sent when instance state has been successfully fetched.
type InstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
}

func startStagingCmd(model StageModel) tea.Cmd {
	return func() tea.Msg {
		// Fetch instance state if we have an instance ID or name
		// This is used to show all resources (including those with no changes) in the UI
		instanceState := stateutil.FetchInstanceState(model.engine, model.instanceID, model.instanceName)

		payload, err := createChangesetPayload(model)
		if err != nil {
			return StageErrorMsg{Err: err}
		}

		response, err := model.engine.CreateChangeset(
			context.TODO(),
			payload,
		)
		if err != nil {
			// Return the original error to preserve type information
			// for detailed error rendering (ClientError, StreamError, etc.)
			return StageErrorMsg{Err: err}
		}

		// Start streaming events
		err = model.engine.StreamChangeStagingEvents(
			context.TODO(),
			response.Data.ID,
			response.LastEventID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return StageErrorMsg{Err: err}
		}

		// Return both the changeset ID and instance state
		return StageStartedWithStateMsg{
			ChangesetID:   response.Data.ID,
			InstanceState: instanceState,
		}
	}
}

// StageStartedWithStateMsg is a message indicating that staging has started
// and includes the fetched instance state (if available).
type StageStartedWithStateMsg struct {
	ChangesetID   string
	InstanceState *state.InstanceState
}

func createChangesetPayload(model StageModel) (*types.CreateChangesetPayload, error) {
	docInfo, err := shared.BuildDocumentInfo(model.blueprintSource, model.blueprintFile)
	if err != nil {
		return nil, err
	}

	return &types.CreateChangesetPayload{
		BlueprintDocumentInfo: docInfo,
		InstanceID:            model.instanceID,
		InstanceName:          model.instanceName,
		Destroy:               model.destroy,
		SkipDriftCheck:        model.skipDriftCheck,
	}, nil
}

func waitForNextEventCmd(model StageModel) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-model.eventStream
		if !ok {
			return StageStreamClosedMsg{}
		}
		return StageEventMsg(event)
	}
}

func checkForErrCmd(model StageModel) tea.Cmd {
	return func() tea.Msg {
		var err error
		select {
		case <-time.After(1 * time.Second):
			break
		case newErr := <-model.errStream:
			err = newErr
		}
		return StageErrorMsg{Err: err}
	}
}

// applyReconciliationCmd applies reconciliation actions to accept external changes.
func applyReconciliationCmd(model StageModel) tea.Cmd {
	return func() tea.Msg {
		if model.driftResult == nil {
			return driftui.ReconciliationErrorMsg{
				Err: nil, // No drift result to reconcile
			}
		}

		payload := buildAcceptExternalPayload(model.driftResult, model)
		result, err := model.engine.ApplyReconciliation(
			context.TODO(),
			model.instanceID,
			payload,
		)
		if err != nil {
			return driftui.ReconciliationErrorMsg{Err: err}
		}

		return driftui.ReconciliationCompleteMsg{
			InstanceID:       result.InstanceID,
			ResourcesUpdated: result.ResourcesUpdated,
			LinksUpdated:     result.LinksUpdated,
		}
	}
}

// buildAcceptExternalPayload builds the reconciliation payload to accept all external changes.
func buildAcceptExternalPayload(
	result *container.ReconciliationCheckResult,
	model StageModel,
) *types.ApplyReconciliationPayload {
	return &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: buildBlueprintDocumentInfoFromModel(model),
		ResourceActions:       shared.BuildResourceActions(result.Resources),
		LinkActions:           shared.BuildLinkActions(result.Links),
	}
}

// buildBlueprintDocumentInfoFromModel creates BlueprintDocumentInfo from the model.
func buildBlueprintDocumentInfoFromModel(model StageModel) types.BlueprintDocumentInfo {
	docInfo, err := shared.BuildDocumentInfo(model.blueprintSource, model.blueprintFile)
	if err != nil {
		return types.BlueprintDocumentInfo{
			BlueprintFile: model.blueprintFile,
		}
	}
	return docInfo
}

// checkInstanceExistsCmd returns a command that checks if an instance exists.
func checkInstanceExistsCmd(model *StageOptionsFormModel) tea.Cmd {
	return func() tea.Msg {
		if model.engine == nil {
			return instanceExistsMsg{exists: false}
		}

		instance, err := model.engine.GetBlueprintInstance(context.TODO(), model.instanceName)
		if err != nil || instance == nil {
			return instanceExistsMsg{exists: false}
		}
		return instanceExistsMsg{exists: true}
	}
}
