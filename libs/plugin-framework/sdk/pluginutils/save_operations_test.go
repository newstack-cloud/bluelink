package pluginutils

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type SaveOperationsSuite struct {
	suite.Suite
}

func (s *SaveOperationsSuite) Test_save_operations() {
	hasValues, saveOpCtx, err := RunSaveOperations(
		context.Background(),
		SaveOperationContext{
			Data: map[string]any{},
		},
		[]SaveOperation[*mockService]{
			&createResourceOp{},
			&putResourceExtraConfigOp{},
		},
		&provider.ResourceDeployInput{},
		&mockService{},
	)
	s.Assert().NoError(err)
	s.Assert().True(hasValues)
	s.Assert().Equal(
		SaveOperationContext{
			ProviderUpstreamID: "test-id-103922",
			Data: map[string]any{
				"metadata": map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
				"extraConfig": map[string]any{
					"extraKey1": "extraValue1",
					"extraKey2": "extraValue2",
				},
			},
		},
		saveOpCtx,
	)
}

func (s *SaveOperationsSuite) Test_save_operations_with_preparation_error() {
	hasValues, saveOpCtx, err := RunSaveOperations(
		context.Background(),
		SaveOperationContext{
			Data: map[string]any{},
		},
		[]SaveOperation[*mockService]{
			&createResourceOp{errorOnPrepare: true},
			&putResourceExtraConfigOp{},
		},
		&provider.ResourceDeployInput{},
		&mockService{},
	)
	s.Assert().Error(err)
	s.Assert().False(hasValues)
	s.Assert().Equal(
		SaveOperationContext{},
		saveOpCtx,
	)
}

func (s *SaveOperationsSuite) Test_save_operations_with_execution_error() {
	hasValues, saveOpCtx, err := RunSaveOperations(
		context.Background(),
		SaveOperationContext{
			Data: map[string]any{},
		},
		[]SaveOperation[*mockService]{
			&createResourceOp{errorOnExecute: true},
			&putResourceExtraConfigOp{},
		},
		&provider.ResourceDeployInput{},
		&mockService{},
	)
	s.Assert().Error(err)
	s.Assert().False(hasValues)
	s.Assert().Equal(
		SaveOperationContext{},
		saveOpCtx,
	)
}

type createResourceOp struct {
	errorOnPrepare bool
	errorOnExecute bool
}

func (op *createResourceOp) Name() string {
	return "create resource"
}

func (op *createResourceOp) Prepare(
	saveOpCtx SaveOperationContext,
	specData *core.MappingNode,
	changes *provider.Changes,
) (bool, SaveOperationContext, error) {
	if op.errorOnPrepare {
		return false, saveOpCtx, errors.New(
			"error preparing create resource operation",
		)
	}

	return true, SaveOperationContext{
		ProviderUpstreamID: "test-id-103922",
		Data: map[string]any{
			"metadata": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}, nil
}

func (op *createResourceOp) Execute(
	ctx context.Context,
	saveOpCtx SaveOperationContext,
	service *mockService,
) (SaveOperationContext, error) {
	if op.errorOnExecute {
		return SaveOperationContext{}, errors.New(
			"error executing create resource operation",
		)
	}
	return saveOpCtx, nil
}

type putResourceExtraConfigOp struct{}

func (op *putResourceExtraConfigOp) Name() string {
	return "put resource extra config"
}

func (op *putResourceExtraConfigOp) Prepare(
	saveOpCtx SaveOperationContext,
	specData *core.MappingNode,
	changes *provider.Changes,
) (bool, SaveOperationContext, error) {
	newSaveOpCtx := SaveOperationContext{
		ProviderUpstreamID: saveOpCtx.ProviderUpstreamID,
		Data:               map[string]any{},
	}
	maps.Copy(newSaveOpCtx.Data, saveOpCtx.Data)
	newSaveOpCtx.Data["extraConfig"] = map[string]any{
		"extraKey1": "extraValue1",
		"extraKey2": "extraValue2",
	}

	return true, newSaveOpCtx, nil
}

func (op *putResourceExtraConfigOp) Execute(
	ctx context.Context,
	saveOpCtx SaveOperationContext,
	service *mockService,
) (SaveOperationContext, error) {
	return saveOpCtx, nil
}

type mockService struct{}

func TestSaveOperationsSuite(t *testing.T) {
	suite.Run(t, new(SaveOperationsSuite))
}
