package deployui

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

// DeployEventMsg is a message containing a deployment event.
type DeployEventMsg types.BlueprintInstanceEvent

// DeployErrorMsg is a message containing an error from the deployment process.
type DeployErrorMsg struct {
	Err error
}

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

		var instanceID string

		if model.instanceID != "" {
			// Update existing instance
			instance, err := model.engine.UpdateBlueprintInstance(
				context.TODO(),
				model.instanceID,
				payload,
			)
			if err != nil {
				return DeployErrorMsg{Err: err}
			}
			instanceID = instance.InstanceID
		} else {
			// Create new instance
			instance, err := model.engine.CreateBlueprintInstance(
				context.TODO(),
				payload,
			)
			if err != nil {
				return DeployErrorMsg{Err: err}
			}
			instanceID = instance.InstanceID
		}

		// Start streaming events
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

func createDeployPayload(model DeployModel) (*types.BlueprintInstancePayload, error) {
	switch model.blueprintSource {
	case consts.BlueprintSourceHTTPS:
		return createDeployPayloadForHTTPS(model)
	case consts.BlueprintSourceS3:
		return createDeployPayloadForS3(model)
	case consts.BlueprintSourceGCS:
		return createDeployPayloadForGCS(model)
	case consts.BlueprintSourceAzureBlob:
		return createDeployPayloadForAzureBlob(model)
	default:
		return createDeployPayloadForLocalFile(model)
	}
}

func createDeployPayloadForLocalFile(
	model DeployModel,
) (*types.BlueprintInstancePayload, error) {
	// Convert to absolute path to ensure child blueprints can be resolved
	// relative to the parent blueprint's directory.
	absPath, err := filepath.Abs(model.blueprintFile)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(absPath)
	file := filepath.Base(absPath)
	return &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        directory,
			BlueprintFile:    file,
		},
		InstanceName: model.instanceName,
		ChangeSetID:  model.changesetID,
		AsRollback:   model.asRollback,
		AutoRollback: model.autoRollback,
		Force:        model.force,
	}, nil
}

func createDeployPayloadForS3(
	model DeployModel,
) (*types.BlueprintInstancePayload, error) {
	return createDeployPayloadForObjectStorage(model, "s3")
}

func createDeployPayloadForGCS(
	model DeployModel,
) (*types.BlueprintInstancePayload, error) {
	return createDeployPayloadForObjectStorage(model, "gcs")
}

func createDeployPayloadForAzureBlob(
	model DeployModel,
) (*types.BlueprintInstancePayload, error) {
	return createDeployPayloadForObjectStorage(model, "azureblob")
}

func createDeployPayloadForObjectStorage(
	model DeployModel,
	scheme string,
) (*types.BlueprintInstancePayload, error) {
	directory := path.Dir(model.blueprintFile)
	file := path.Base(model.blueprintFile)
	return &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: scheme,
			Directory:        directory,
			BlueprintFile:    file,
		},
		InstanceName: model.instanceName,
		ChangeSetID:  model.changesetID,
		AsRollback:   model.asRollback,
		AutoRollback: model.autoRollback,
		Force:        model.force,
	}, nil
}

func createDeployPayloadForHTTPS(
	model DeployModel,
) (*types.BlueprintInstancePayload, error) {
	parsedURL, err := url.Parse(model.blueprintFile)
	if err != nil {
		return nil, err
	}

	basePath := path.Dir(parsedURL.Path)
	if basePath == "/" {
		basePath = ""
	}
	file := path.Base(parsedURL.Path)
	return &types.BlueprintInstancePayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "https",
			Directory:        basePath,
			BlueprintFile:    file,
			BlueprintLocationMetadata: map[string]any{
				"host": parsedURL.Host,
			},
		},
		InstanceName: model.instanceName,
		ChangeSetID:  model.changesetID,
		AsRollback:   model.asRollback,
		AutoRollback: model.autoRollback,
		Force:        model.force,
	}, nil
}

func waitForNextDeployEventCmd(model DeployModel) tea.Cmd {
	return func() tea.Msg {
		event := <-model.eventStream
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
		// If we already have an instance ID, use it as-is
		if model.instanceID != "" {
			return InstanceResolvedMsg{
				InstanceID:   model.instanceID,
				InstanceName: model.instanceName,
			}
		}

		// If we have an instance name but no ID, try to look it up
		// GetBlueprintInstance accepts an ID that can be either an ID or a name
		if model.instanceName != "" {
			instance, err := model.engine.GetBlueprintInstance(
				context.TODO(),
				model.instanceName,
			)
			if err == nil && instance != nil {
				// Instance exists - use its ID
				return InstanceResolvedMsg{
					InstanceID:   instance.InstanceID,
					InstanceName: model.instanceName,
				}
			}
			// Instance doesn't exist - this is a new deployment
			// Stage with no instance identifiers
			return InstanceResolvedMsg{
				InstanceID:   "",
				InstanceName: "",
			}
		}

		// No instance identifiers provided
		return InstanceResolvedMsg{
			InstanceID:   "",
			InstanceName: "",
		}
	}
}
