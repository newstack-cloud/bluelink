package deployui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// DeployEventMsg is a message containing a deployment event.
type DeployEventMsg types.BlueprintInstanceEvent

// DeployStreamClosedMsg is sent when the deploy event stream is closed.
// This typically happens due to a stream timeout or the connection being dropped.
type DeployStreamClosedMsg struct{}

// DeployErrorMsg is a message containing an error from the deployment process.
type DeployErrorMsg struct {
	Err error
}

// DestroyChangesetErrorMsg is sent when deployment fails because the changeset
// was created for a destroy operation.
type DestroyChangesetErrorMsg struct{}

// DeployStartedMsg is a message indicating that deployment has started.
type DeployStartedMsg struct {
	InstanceID string
}

// StartDeployMsg is a message to initiate deployment.
type StartDeployMsg struct{}

// ConfirmDeployMsg is a message to confirm deployment after staging review.
type ConfirmDeployMsg struct {
	Confirmed bool
}

// InstanceResolvedMsg is a message indicating instance identifiers have been resolved.
// This is used to handle the case where a user provides an instance name for a new deployment
// and we need to resolve it to an empty instance ID (since the instance doesn't exist yet).
type InstanceResolvedMsg struct {
	InstanceID   string
	InstanceName string
}

func startDeploymentCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		payload, err := createDeployPayload(model)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		response, err := createOrUpdateInstance(model, payload)
		if err != nil {
			return handleDeployError(err, model.instanceID)
		}

		err = model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			response.Data.InstanceID,
			response.LastEventID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		return DeployStartedMsg{InstanceID: response.Data.InstanceID}
	}
}

// createOrUpdateInstance creates a new instance or updates an existing one.
func createOrUpdateInstance(
	model DeployModel,
	payload *types.BlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	if model.instanceID != "" {
		return model.engine.UpdateBlueprintInstance(
			context.TODO(),
			model.instanceID,
			payload,
		)
	}
	return model.engine.CreateBlueprintInstance(context.TODO(), payload)
}

// handleDeployError converts deployment errors to appropriate messages,
// including drift detection for 409 responses and destroy changeset errors.
func handleDeployError(err error, fallbackInstanceID string) tea.Msg {
	// Check for destroy changeset error
	if _, isDestroyChangeset := engineerrors.IsDestroyChangesetError(err); isDestroyChangeset {
		return DestroyChangesetErrorMsg{}
	}

	// Check for drift blocked error
	clientErr, isDriftBlocked := engineerrors.IsDriftBlockedError(err)
	if !isDriftBlocked {
		return DeployErrorMsg{Err: err}
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

func createDeployPayload(model DeployModel) (*types.BlueprintInstancePayload, error) {
	docInfo, err := shared.BuildDocumentInfo(model.blueprintSource, model.blueprintFile)
	if err != nil {
		return nil, err
	}

	return &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: docInfo,
		InstanceName:          model.instanceName,
		ChangeSetID:           model.changesetID,
		AutoRollback:          model.autoRollback,
		Force:                 model.force,
	}, nil
}


func waitForNextDeployEventCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-model.eventStream
		if !ok {
			return DeployStreamClosedMsg{}
		}
		return DeployEventMsg(event)
	}
}

func checkForErrCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		var err error
		select {
		case <-time.After(1 * time.Second):
			break
		case newErr := <-model.errStream:
			err = newErr
		}
		return DeployErrorMsg{Err: err}
	}
}

// resolveInstanceIdentifiersCmd resolves instance identifiers for staging in the deploy context.
// When deploying with --stage, if the user provides an instance name but no instance ID,
// we need to check if the instance exists. If it doesn't exist (new deployment), we stage
// with no instance ID/name so staging treats it as a new deployment.
// If it exists, we use the instance ID for staging against the existing instance.
func resolveInstanceIdentifiersCmd(model MainModel) tea.Cmd {
	return func() tea.Msg {
		instanceID, instanceName := shared.ResolveInstanceIdentifiers(model)
		return InstanceResolvedMsg{
			InstanceID:   instanceID,
			InstanceName: instanceName,
		}
	}
}

// applyReconciliationCmd applies reconciliation to accept external changes.
func applyReconciliationCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		if model.driftResult == nil {
			return driftui.ReconciliationErrorMsg{Err: nil}
		}

		payload := buildAcceptExternalPayload(model.driftResult, model)
		instanceID := shared.GetEffectiveInstanceID(model.instanceID, model.instanceName)

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

// buildAcceptExternalPayload builds the reconciliation payload from the drift result.
func buildAcceptExternalPayload(
	result *container.ReconciliationCheckResult,
	model DeployModel,
) *types.ApplyReconciliationPayload {
	return &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: buildBlueprintDocumentInfo(model),
		ResourceActions:       shared.BuildResourceActions(result.Resources),
		LinkActions:           shared.BuildLinkActions(result.Links),
	}
}

// buildBlueprintDocumentInfo creates BlueprintDocumentInfo from the deploy model.
func buildBlueprintDocumentInfo(model DeployModel) types.BlueprintDocumentInfo {
	payload, err := createDeployPayload(model)
	if err != nil {
		return types.BlueprintDocumentInfo{}
	}
	return payload.BlueprintDocumentInfo
}

// continueDeploymentCmd continues deployment after reconciliation.
// This uses the changeset ID from the 409 response to resume deployment.
func continueDeploymentCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		payload, err := createDeployPayload(model)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		response, err := createOrUpdateInstance(model, payload)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		err = model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			response.Data.InstanceID,
			response.LastEventID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		return DeployStartedMsg{InstanceID: response.Data.InstanceID}
	}
}

// PostDeployInstanceStateFetchedMsg is sent when instance state has been fetched after deployment.
type PostDeployInstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
}

// DeployStateRefreshedMsg is sent when the instance state has been refreshed during deployment.
type DeployStateRefreshedMsg struct {
	InstanceState *state.InstanceState
}

// DeployStateRefreshTickMsg triggers a periodic state refresh during deployment.
type DeployStateRefreshTickMsg struct{}

// deployStateRefreshInterval is the interval between state refreshes during deployment.
const deployStateRefreshInterval = 5 * time.Second

// startDeployStateRefreshTickerCmd starts the periodic state refresh ticker for deployment.
func startDeployStateRefreshTickerCmd() tea.Cmd {
	return tea.Tick(deployStateRefreshInterval, func(t time.Time) tea.Msg {
		return DeployStateRefreshTickMsg{}
	})
}

// refreshDeployInstanceStateCmd refreshes the instance state during deployment.
func refreshDeployInstanceStateCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(model.engine, model.instanceID, model.instanceName)
		if instanceState == nil {
			return nil
		}
		return DeployStateRefreshedMsg{
			InstanceState: instanceState,
		}
	}
}

// fetchPostDeployInstanceStateCmd fetches the instance state after deployment completes.
// This is used to get updated computed fields (outputs) for display in the UI.
func fetchPostDeployInstanceStateCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(model.engine, model.instanceID, model.instanceName)
		return PostDeployInstanceStateFetchedMsg{
			InstanceState: instanceState,
		}
	}
}

// PreDeployInstanceStateFetchedMsg is sent when instance state has been fetched before deployment.
// This is used for direct deployments (without staging) to populate unchanged items.
type PreDeployInstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
}

// fetchPreDeployInstanceStateCmd fetches the instance state before deployment starts.
// This is used when deploying directly without going through the staging flow.
func fetchPreDeployInstanceStateCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		instanceState := stateutil.FetchInstanceState(model.engine, model.instanceID, model.instanceName)
		return PreDeployInstanceStateFetchedMsg{
			InstanceState: instanceState,
		}
	}
}
