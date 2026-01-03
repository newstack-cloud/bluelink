package deployui

import (
	"context"
	"log"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
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

		instanceID, err := createOrUpdateInstance(model, payload)
		if err != nil {
			return handleDeployError(err, model.instanceID)
		}

		err = model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			instanceID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		return DeployStartedMsg{InstanceID: instanceID}
	}
}

// createOrUpdateInstance creates a new instance or updates an existing one.
func createOrUpdateInstance(model DeployModel, payload *types.BlueprintInstancePayload) (string, error) {
	if model.instanceID != "" {
		instance, err := model.engine.UpdateBlueprintInstance(
			context.TODO(),
			model.instanceID,
			payload,
		)
		if err != nil {
			return "", err
		}
		return instance.InstanceID, nil
	}

	instance, err := model.engine.CreateBlueprintInstance(
		context.TODO(),
		payload,
	)
	if err != nil {
		return "", err
	}
	return instance.InstanceID, nil
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
	docInfo, err := buildDocumentInfo(model.blueprintSource, model.blueprintFile)
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

// buildDocumentInfo creates BlueprintDocumentInfo based on the source type.
func buildDocumentInfo(source string, blueprintFile string) (types.BlueprintDocumentInfo, error) {
	switch source {
	case consts.BlueprintSourceHTTPS:
		return buildHTTPSDocumentInfo(blueprintFile)
	case consts.BlueprintSourceS3:
		return buildObjectStorageDocumentInfo(blueprintFile, "s3"), nil
	case consts.BlueprintSourceGCS:
		return buildObjectStorageDocumentInfo(blueprintFile, "gcs"), nil
	case consts.BlueprintSourceAzureBlob:
		return buildObjectStorageDocumentInfo(blueprintFile, "azureblob"), nil
	default:
		return buildLocalFileDocumentInfo(blueprintFile)
	}
}

func buildLocalFileDocumentInfo(blueprintFile string) (types.BlueprintDocumentInfo, error) {
	absPath, err := filepath.Abs(blueprintFile)
	if err != nil {
		return types.BlueprintDocumentInfo{}, err
	}
	return types.BlueprintDocumentInfo{
		FileSourceScheme: "file",
		Directory:        filepath.Dir(absPath),
		BlueprintFile:    filepath.Base(absPath),
	}, nil
}

func buildObjectStorageDocumentInfo(blueprintFile, scheme string) types.BlueprintDocumentInfo {
	return types.BlueprintDocumentInfo{
		FileSourceScheme: scheme,
		Directory:        path.Dir(blueprintFile),
		BlueprintFile:    path.Base(blueprintFile),
	}
}

func buildHTTPSDocumentInfo(blueprintFile string) (types.BlueprintDocumentInfo, error) {
	parsedURL, err := url.Parse(blueprintFile)
	if err != nil {
		return types.BlueprintDocumentInfo{}, err
	}

	basePath := path.Dir(parsedURL.Path)
	if basePath == "/" {
		basePath = ""
	}

	return types.BlueprintDocumentInfo{
		FileSourceScheme: "https",
		Directory:        basePath,
		BlueprintFile:    path.Base(parsedURL.Path),
		BlueprintLocationMetadata: map[string]any{
			"host": parsedURL.Host,
		},
	}, nil
}

func waitForNextDeployEventCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		log.Printf("DEBUG: waiting for next deploy event...\n")
		event, ok := <-model.eventStream
		if !ok {
			log.Printf("DEBUG: eventStream channel was CLOSED\n")
			return DeployStreamClosedMsg{}
		}
		log.Printf("received deploy event: %s\n\n", event.String())
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
			log.Printf("received deploy error: %+v\n\n", newErr)
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
		instanceID, instanceName := resolveInstanceIdentifiers(model)
		return InstanceResolvedMsg{
			InstanceID:   instanceID,
			InstanceName: instanceName,
		}
	}
}

// resolveInstanceIdentifiers looks up instance identifiers, returning the resolved ID and name.
func resolveInstanceIdentifiers(model MainModel) (instanceID, instanceName string) {
	// If we already have an instance ID, use it as-is
	if model.instanceID != "" {
		return model.instanceID, model.instanceName
	}

	// No instance name provided - new deployment
	if model.instanceName == "" {
		return "", ""
	}

	// Try to look up the instance by name
	// GetBlueprintInstance accepts an ID that can be either an ID or a name
	instance, err := model.engine.GetBlueprintInstance(context.TODO(), model.instanceName)
	if err != nil || instance == nil {
		// Instance doesn't exist - new deployment with no identifiers
		return "", ""
	}

	// Instance exists - use its ID
	return instance.InstanceID, model.instanceName
}

// applyReconciliationCmd applies reconciliation to accept external changes.
func applyReconciliationCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		if model.driftResult == nil {
			return driftui.ReconciliationErrorMsg{Err: nil}
		}

		payload := buildAcceptExternalPayload(model.driftResult, model)
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

// getEffectiveInstanceID returns the instance ID, falling back to instance name if ID is empty.
func getEffectiveInstanceID(instanceID, instanceName string) string {
	if instanceID != "" {
		return instanceID
	}
	return instanceName
}

// buildAcceptExternalPayload builds the reconciliation payload from the drift result.
func buildAcceptExternalPayload(
	result *container.ReconciliationCheckResult,
	model DeployModel,
) *types.ApplyReconciliationPayload {
	return &types.ApplyReconciliationPayload{
		BlueprintDocumentInfo: buildBlueprintDocumentInfo(model),
		ResourceActions:       buildResourceActions(result.Resources),
		LinkActions:           buildLinkActions(result.Links),
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

// buildBlueprintDocumentInfo creates BlueprintDocumentInfo from the deploy model.
// It reuses the payload creation logic and extracts just the document info.
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

		instanceID, err := createOrUpdateInstance(model, payload)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		err = model.engine.StreamBlueprintInstanceEvents(
			context.TODO(),
			instanceID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return DeployErrorMsg{Err: err}
		}

		return DeployStartedMsg{InstanceID: instanceID}
	}
}

// PostDeployInstanceStateFetchedMsg is sent when instance state has been fetched after deployment.
type PostDeployInstanceStateFetchedMsg struct {
	InstanceState *state.InstanceState
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
