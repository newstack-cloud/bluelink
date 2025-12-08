package validateui

import (
	"context"
	"net/url"
	"path"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"go.uber.org/zap"
)

func startValidateStreamCmd(model ValidateModel, logger *zap.Logger) tea.Cmd {
	return func() tea.Msg {
		payload, err := createValidationPayload(model)
		if err != nil {
			return ValidateErrMsg{err}
		}

		blueprintValidation, err := model.engine.CreateBlueprintValidation(
			context.TODO(),
			payload,
			&types.CreateBlueprintValidationQuery{},
		)
		if err != nil {
			time.Sleep(10 * time.Second)
			return ValidateErrMsg{engine.SimplifyError(err, logger)}
		}

		err = model.engine.StreamBlueprintValidationEvents(
			context.TODO(),
			blueprintValidation.ID,
			model.resultStream,
			model.errStream,
		)
		if err != nil {
			return ValidateErrMsg{err}
		}
		return nil
	}
}

func createValidationPayload(model ValidateModel) (*types.CreateBlueprintValidationPayload, error) {
	switch model.blueprintSource {
	case consts.BlueprintSourceHTTPS:
		return createValidationPayloadForHTTPS(model)
	case consts.BlueprintSourceS3:
		return createValidationPayloadForS3(model)
	case consts.BlueprintSourceGCS:
		return createValidationPayloadForGCS(model)
	case consts.BlueprintSourceAzureBlob:
		return createValidationPayloadForAzureBlob(model)
	default:
		return createValidationPayloadForLocalFile(model)
	}
}

func createValidationPayloadForLocalFile(
	model ValidateModel,
) (*types.CreateBlueprintValidationPayload, error) {
	directory := path.Dir(model.blueprintFile)
	file := path.Base(model.blueprintFile)
	return &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "file",
			Directory:        directory,
			BlueprintFile:    file,
		},
	}, nil
}

func createValidationPayloadForS3(
	model ValidateModel,
) (*types.CreateBlueprintValidationPayload, error) {
	return createValidationPayloadForObjectStorage(model, "s3")
}

func createValidationPayloadForGCS(
	model ValidateModel,
) (*types.CreateBlueprintValidationPayload, error) {
	return createValidationPayloadForObjectStorage(model, "gcs")
}

func createValidationPayloadForAzureBlob(
	model ValidateModel,
) (*types.CreateBlueprintValidationPayload, error) {
	return createValidationPayloadForObjectStorage(model, "azureblob")
}

func createValidationPayloadForObjectStorage(
	model ValidateModel,
	scheme string,
) (*types.CreateBlueprintValidationPayload, error) {
	directory := path.Dir(model.blueprintFile)
	file := path.Base(model.blueprintFile)
	return &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: scheme,
			Directory:        directory,
			BlueprintFile:    file,
		},
	}, nil
}

func createValidationPayloadForHTTPS(
	model ValidateModel,
) (*types.CreateBlueprintValidationPayload, error) {
	url, err := url.Parse(model.blueprintFile)
	if err != nil {
		return nil, err
	}

	basePath := path.Dir(url.Path)
	if basePath == "/" {
		basePath = ""
	}
	file := path.Base(url.Path)
	return &types.CreateBlueprintValidationPayload{
		BlueprintDocumentInfo: types.BlueprintDocumentInfo{
			FileSourceScheme: "https",
			Directory:        basePath,
			BlueprintFile:    file,
			BlueprintLocationMetadata: map[string]any{
				"host": url.Host,
			},
		},
	}, nil
}

func waitForNextResultCmd(model ValidateModel) tea.Cmd {
	return func() tea.Msg {
		event := <-model.resultStream
		return ValidateResultMsg(&event)
	}
}

func checkForErrCmd(model ValidateModel) tea.Cmd {
	return func() tea.Msg {
		var err error
		select {
		case <-time.After(1 * time.Second):
			break
		case newErr := <-model.errStream:
			err = newErr
		}
		return ValidateErrMsg{err}
	}
}
