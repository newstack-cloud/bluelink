package destroyui

import (
	"context"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// DestroyEventMsg is a message containing a destroy event.
type DestroyEventMsg types.BlueprintInstanceEvent

// DestroyStreamClosedMsg is sent when the destroy event stream is closed.
type DestroyStreamClosedMsg struct{}

// DestroyErrorMsg is a message containing an error from the destroy process.
type DestroyErrorMsg struct {
	Err error
}

// DestroyStartedMsg is a message indicating that destroy has started.
type DestroyStartedMsg struct {
	InstanceID string
}

// StartDestroyMsg is a message to initiate destroy.
type StartDestroyMsg struct{}

// ConfirmDestroyMsg is a message to confirm destroy after staging review.
type ConfirmDestroyMsg struct {
	Confirmed bool
}

// InstanceResolvedMsg is a message indicating instance identifiers have been resolved.
type InstanceResolvedMsg struct {
	InstanceID   string
	InstanceName string
}

// PostDestroyInstanceStateFetchedMsg is sent when instance state has been fetched after destroy.
type PostDestroyInstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
}

// PreDestroyInstanceStateFetchedMsg is sent when instance state has been fetched before destroy.
type PreDestroyInstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
}

// DeployChangesetErrorMsg is sent when destroy fails because the changeset
// was created for a deploy operation, not a destroy operation.
type DeployChangesetErrorMsg struct{}

func startDestroyCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		payload := createDestroyPayload(model)

		response, err := executeDestroy(model, payload)
		if err != nil {
			return handleDestroyError(err, model.instanceID)
		}

		err = model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			response.Data.InstanceID,
			response.LastEventID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return DestroyErrorMsg{Err: err}
		}

		return DestroyStartedMsg{InstanceID: response.Data.InstanceID}
	}
}

func executeDestroy(
	model DestroyModel,
	payload *types.DestroyBlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	instanceID := getEffectiveInstanceID(model.instanceID, model.instanceName)
	return model.engine.DestroyBlueprintInstance(context.TODO(), instanceID, payload)
}

func handleDestroyError(err error, fallbackInstanceID string) tea.Msg {
	// Check for deploy changeset error (trying to destroy with a non-destroy changeset)
	if _, isDeployChangeset := engineerrors.IsDeployChangesetError(err); isDeployChangeset {
		return DeployChangesetErrorMsg{}
	}

	// Check for drift blocked error
	clientErr, isDriftBlocked := engineerrors.IsDriftBlockedError(err)
	if !isDriftBlocked {
		return DestroyErrorMsg{Err: err}
	}

	instanceID := clientErr.DriftBlockedResponse.InstanceID
	if instanceID == "" {
		instanceID = fallbackInstanceID
	}

	return driftui.DriftDetectedMsg{
		ReconciliationResult: clientErr.DriftBlockedResponse.ReconciliationResult,
		Message:              clientErr.Message,
		InstanceID:           instanceID,
		ChangesetID:          clientErr.DriftBlockedResponse.ChangesetID,
	}
}

func createDestroyPayload(model DestroyModel) *types.DestroyBlueprintInstancePayload {
	return &types.DestroyBlueprintInstancePayload{
		ChangeSetID: model.changesetID,
		Force:       model.force,
	}
}

func waitForNextDestroyEventCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-model.eventStream
		if !ok {
			return DestroyStreamClosedMsg{}
		}
		return DestroyEventMsg(event)
	}
}

func checkForDestroyErrCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		var err error
		select {
		case <-time.After(1 * time.Second):
			break
		case newErr := <-model.errStream:
			err = newErr
		}
		return DestroyErrorMsg{Err: err}
	}
}

// resolveInstanceIdentifiersCmd resolves instance identifiers for staging in the destroy context.
func resolveInstanceIdentifiersCmd(model MainModel) tea.Cmd {
	return func() tea.Msg {
		instanceID, instanceName := resolveInstanceIdentifiers(model)
		return InstanceResolvedMsg{
			InstanceID:   instanceID,
			InstanceName: instanceName,
		}
	}
}

func resolveInstanceIdentifiers(model MainModel) (instanceID, instanceName string) {
	if model.instanceID != "" {
		return model.instanceID, model.instanceName
	}

	if model.instanceName == "" {
		return "", ""
	}

	instance, err := model.engine.GetBlueprintInstance(context.TODO(), model.instanceName)
	if err != nil || instance == nil {
		return "", ""
	}

	return instance.InstanceID, model.instanceName
}

func applyReconciliationCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		if model.driftResult == nil {
			return driftui.ReconciliationErrorMsg{Err: nil}
		}

		payload := buildAcceptExternalPayload(model.driftResult)
		instanceID := getEffectiveInstanceID(model.instanceID, model.instanceName)

		_, err := model.engine.ApplyReconciliation(context.TODO(), instanceID, payload)
		if err != nil {
			return driftui.ReconciliationErrorMsg{Err: err}
		}

		return driftui.ReconciliationCompleteMsg{
			ResourcesUpdated: len(model.driftResult.Resources),
			LinksUpdated:     len(model.driftResult.Links),
		}
	}
}

func getEffectiveInstanceID(instanceID, instanceName string) string {
	if instanceID != "" {
		return instanceID
	}
	return instanceName
}

func buildAcceptExternalPayload(
	result *container.ReconciliationCheckResult,
) *types.ApplyReconciliationPayload {
	return &types.ApplyReconciliationPayload{
		ResourceActions: buildResourceActions(result.Resources),
		LinkActions:     buildLinkActions(result.Links),
	}
}

func buildResourceActions(resources []container.ResourceReconcileResult) []types.ResourceReconcileActionPayload {
	actions := make([]types.ResourceReconcileActionPayload, 0, len(resources))
	for _, r := range resources {
		actions = append(actions, types.ResourceReconcileActionPayload{
			ResourceID:    r.ResourceID,
			ChildPath:     r.ChildPath,
			Action:        string(r.RecommendedAction),
			ExternalState: r.ExternalState,
			NewStatus:     strconv.Itoa(int(r.NewStatus)),
		})
	}
	return actions
}

func buildLinkActions(links []container.LinkReconcileResult) []types.LinkReconcileActionPayload {
	actions := make([]types.LinkReconcileActionPayload, 0, len(links))
	for _, l := range links {
		actions = append(actions, types.LinkReconcileActionPayload{
			LinkID:              l.LinkID,
			ChildPath:           l.ChildPath,
			Action:              string(l.RecommendedAction),
			NewStatus:           strconv.Itoa(int(l.NewStatus)),
			LinkDataUpdates:     l.LinkDataUpdates,
			IntermediaryActions: buildIntermediaryActions(l.IntermediaryChanges),
		})
	}
	return actions
}

func buildIntermediaryActions(
	changes map[string]*container.IntermediaryReconcileResult,
) map[string]*types.IntermediaryReconcileActionPayload {
	if len(changes) == 0 {
		return nil
	}

	actions := make(map[string]*types.IntermediaryReconcileActionPayload, len(changes))
	for name, intResult := range changes {
		actions[name] = &types.IntermediaryReconcileActionPayload{
			Action:        string(container.ReconciliationActionAcceptExternal),
			ExternalState: intResult.ExternalState,
			NewStatus:     "created",
		}
	}
	return actions
}

func continueDestroyCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		payload := createDestroyPayload(model)

		response, err := executeDestroy(model, payload)
		if err != nil {
			return handleDestroyError(err, model.instanceID)
		}

		err = model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			response.Data.InstanceID,
			response.LastEventID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return DestroyErrorMsg{Err: err}
		}

		return DestroyStartedMsg{InstanceID: response.Data.InstanceID}
	}
}

func fetchPostDestroyInstanceStateCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(model.engine, model.instanceID, model.instanceName)
		return PostDestroyInstanceStateFetchedMsg{
			InstanceState: instanceState,
		}
	}
}

func fetchPreDestroyInstanceStateCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(model.engine, model.instanceID, model.instanceName)
		return PreDestroyInstanceStateFetchedMsg{
			InstanceState: instanceState,
		}
	}
}

// ChangesetFetchedMsg is sent when changeset changes have been fetched.
type ChangesetFetchedMsg struct {
	Changes *changes.BlueprintChanges
}

func fetchChangesetChangesCmd(model DestroyModel) tea.Cmd {
	return func() tea.Msg {
		if model.changesetID == "" {
			return ChangesetFetchedMsg{Changes: nil}
		}

		changeset, err := model.engine.GetChangeset(context.TODO(), model.changesetID)
		if err != nil || changeset == nil {
			return ChangesetFetchedMsg{Changes: nil}
		}

		return ChangesetFetchedMsg{Changes: changeset.Changes}
	}
}
