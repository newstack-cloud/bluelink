package stageui

import (
	"context"
	"net/url"
	"path"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
)

// StageEventMsg is a message containing a change staging event.
type StageEventMsg types.ChangeStagingEvent

// StageErrorMsg is a message containing an error from the staging process.
type StageErrorMsg struct {
	Err error
}

// StageStartedMsg is a message indicating that staging has started.
type StageStartedMsg struct {
	ChangesetID string
}

func startStagingCmd(model StageModel) tea.Cmd {
	return func() tea.Msg {
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

		return StageStartedMsg{ChangesetID: changeset.ID}
	}
}

func createChangesetPayload(model StageModel) (*types.CreateChangesetPayload, error) {
	switch model.blueprintSource {
	case consts.BlueprintSourceHTTPS:
		return createChangesetPayloadForHTTPS(model)
	case consts.BlueprintSourceS3:
		return createChangesetPayloadForS3(model)
	case consts.BlueprintSourceGCS:
		return createChangesetPayloadForGCS(model)
	case consts.BlueprintSourceAzureBlob:
		return createChangesetPayloadForAzureBlob(model)
	default:
		return createChangesetPayloadForLocalFile(model)
	}
}

func createChangesetPayloadForLocalFile(
	model StageModel,
) (*types.CreateChangesetPayload, error) {
	// Convert to absolute path to ensure child blueprints can be resolved
	// relative to the parent blueprint's directory.
	absPath, err := filepath.Abs(model.blueprintFile)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(absPath)
	file := filepath.Base(absPath)
	return &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        directory,
			BlueprintFile:    file,
		},
		InstanceID:   model.instanceID,
		InstanceName: model.instanceName,
		Destroy:      model.destroy,
	}, nil
}

func createChangesetPayloadForS3(
	model StageModel,
) (*types.CreateChangesetPayload, error) {
	return createChangesetPayloadForObjectStorage(model, "s3")
}

func createChangesetPayloadForGCS(
	model StageModel,
) (*types.CreateChangesetPayload, error) {
	return createChangesetPayloadForObjectStorage(model, "gcs")
}

func createChangesetPayloadForAzureBlob(
	model StageModel,
) (*types.CreateChangesetPayload, error) {
	return createChangesetPayloadForObjectStorage(model, "azureblob")
}

func createChangesetPayloadForObjectStorage(
	model StageModel,
	scheme string,
) (*types.CreateChangesetPayload, error) {
	directory := path.Dir(model.blueprintFile)
	file := path.Base(model.blueprintFile)
	return &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: scheme,
			Directory:        directory,
			BlueprintFile:    file,
		},
		InstanceID:   model.instanceID,
		InstanceName: model.instanceName,
		Destroy:      model.destroy,
	}, nil
}

func createChangesetPayloadForHTTPS(
	model StageModel,
) (*types.CreateChangesetPayload, error) {
	url, err := url.Parse(model.blueprintFile)
	if err != nil {
		return nil, err
	}

	basePath := path.Dir(url.Path)
	if basePath == "/" {
		basePath = ""
	}
	file := path.Base(url.Path)
	return &types.CreateChangesetPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "https",
			Directory:        basePath,
			BlueprintFile:    file,
			BlueprintLocationMetadata: map[string]any{
				"host": url.Host,
			},
		},
		InstanceID:   model.instanceID,
		InstanceName: model.instanceName,
		Destroy:      model.destroy,
	}, nil
}

func waitForNextEventCmd(model StageModel) tea.Cmd {
	return func() tea.Msg {
		event := <-model.eventStream
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
