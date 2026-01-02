package stageui

import (
	"context"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
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

		changeset, err := model.engine.CreateChangeset(
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
			changeset.ID,
			model.eventStream,
			model.errStream,
		)
		if err != nil {
			return StageErrorMsg{Err: err}
		}

		// Return both the changeset ID and instance state
		return StageStartedWithStateMsg{
			ChangesetID:   changeset.ID,
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
	docInfo, err := buildDocumentInfo(model.blueprintSource, model.blueprintFile)
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

// buildBlueprintDocumentInfoFromModel creates BlueprintDocumentInfo from the model.
// It reuses buildDocumentInfo and returns a fallback on error.
func buildBlueprintDocumentInfoFromModel(model StageModel) types.BlueprintDocumentInfo {
	docInfo, err := buildDocumentInfo(model.blueprintSource, model.blueprintFile)
	if err != nil {
		return types.BlueprintDocumentInfo{
			BlueprintFile: model.blueprintFile,
		}
	}
	return docInfo
}
