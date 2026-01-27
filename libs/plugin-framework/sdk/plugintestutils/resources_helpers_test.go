package plugintestutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

type mockConfig struct{}

type mockConfigStore struct {
	config *mockConfig
}

func (m *mockConfigStore) FromProviderContext(
	ctx context.Context,
	providerCtx provider.Context,
	meta map[string]*core.MappingNode,
) (*mockConfig, error) {
	return m.config, nil
}

type mockService struct {
	MockCalls

	updateMockResourceConfigOutput *updateMockResourceConfigOutput
	updateMockResourceConfigError  error
	updateMockResourceCodeOutput   *updateMockResourceCodeOutput
	updateMockResourceCodeError    error
	getMockResourceOutput          *getMockResourceOutput
	getMockResourceError           error
	saveMockResourceOutput         *saveMockResourceOutput
	saveMockResourceError          error
	deleteMockResourceConfigOutput *deleteMockResourceConfigOutput
	deleteMockResourceConfigError  error
	deleteMockResourceCodeOutput   *deleteMockResourceCodeOutput
	deleteMockResourceCodeError    error
}

type mockServiceOption func(*mockService)

func newMockServiceFactory(
	opts ...mockServiceOption,
) func(mockConf *mockConfig, providerContext provider.Context) *mockService {
	mock := newMockService(opts...)
	return func(mockConfig *mockConfig, providerContext provider.Context) *mockService {
		return mock
	}
}

func newMockService(
	opts ...mockServiceOption,
) *mockService {
	mock := &mockService{}

	for _, opt := range opts {
		opt(mock)
	}

	return mock
}

func withUpdateMockResourceConfigOutput(
	output *updateMockResourceConfigOutput,
) mockServiceOption {
	return func(m *mockService) {
		m.updateMockResourceConfigOutput = output
	}
}

func withUpdateMockResourceCodeOutput(
	output *updateMockResourceCodeOutput,
) mockServiceOption {
	return func(m *mockService) {
		m.updateMockResourceCodeOutput = output
	}
}

func withGetMockResourceOutput(
	output *getMockResourceOutput,
) mockServiceOption {
	return func(m *mockService) {
		m.getMockResourceOutput = output
	}
}

func withGetMockResourceError(
	error error,
) mockServiceOption {
	return func(m *mockService) {
		m.getMockResourceError = error
	}
}

func withSaveMockResourceOutput(
	output *saveMockResourceOutput,
) mockServiceOption {
	return func(m *mockService) {
		m.saveMockResourceOutput = output
	}
}

func withSaveMockResourceError(
	error error,
) mockServiceOption {
	return func(m *mockService) {
		m.saveMockResourceError = error
	}
}

func withDeleteMockResourceConfigOutput(
	output *deleteMockResourceConfigOutput,
) mockServiceOption {
	return func(m *mockService) {
		m.deleteMockResourceConfigOutput = output
	}
}

func withDeleteMockResourceConfigError(
	error error,
) mockServiceOption {
	return func(m *mockService) {
		m.deleteMockResourceConfigError = error
	}
}

func withDeleteMockResourceCodeOutput(
	output *deleteMockResourceCodeOutput,
) mockServiceOption {
	return func(m *mockService) {
		m.deleteMockResourceCodeOutput = output
	}
}

type updateMockResourceConfigInput struct {
	Name      string
	DebugMode bool
	Param1    int
	Param2    string
	Param3    float64
}

type updateMockResourceConfigOutput struct {
	ID string
	updateMockResourceConfigInput
}

func (m *mockService) UpdateConfig(
	ctx context.Context,
	input *updateMockResourceConfigInput,
) (*updateMockResourceConfigOutput, error) {
	m.RegisterCall(ctx, input)
	return m.updateMockResourceConfigOutput, m.updateMockResourceConfigError
}

type updateMockResourceCodeInput struct {
	InlineCode   string
	CodeLocation string
}

type updateMockResourceCodeOutput struct {
	ID string
	updateMockResourceCodeInput
}

func (m *mockService) UpdateCode(
	ctx context.Context,
	input *updateMockResourceCodeInput,
) (*updateMockResourceCodeOutput, error) {
	m.RegisterCall(ctx, input)
	return m.updateMockResourceCodeOutput, m.updateMockResourceCodeError
}

type getMockResourceInput struct {
	ID string
}

type getMockResourceOutput struct {
	ID           string
	Name         string
	DebugMode    bool
	Param1       int
	Param2       string
	Param3       float64
	InlineCode   string
	CodeLocation string
	Status       string
	Tags         []mockTag
}

type mockTag struct {
	CustomKey   string
	CustomValue string
}

func (m *mockService) GetResource(
	ctx context.Context,
	input *getMockResourceInput,
) (*getMockResourceOutput, error) {
	m.RegisterCall(ctx, input)
	return m.getMockResourceOutput, m.getMockResourceError
}

type saveMockResourceInput struct {
	Name         string
	DebugMode    bool
	Param1       int
	Param2       string
	Param3       float64
	InlineCode   string
	CodeLocation string
}

type saveMockResourceOutput struct {
	ID           string
	Name         string
	DebugMode    bool
	Param1       int
	Param2       string
	Param3       float64
	InlineCode   string
	CodeLocation string
}

func (m *mockService) SaveResource(
	ctx context.Context,
	input *saveMockResourceInput,
) (*saveMockResourceOutput, error) {
	m.RegisterCall(ctx, input)
	return m.saveMockResourceOutput, m.saveMockResourceError
}

type deleteMockResourceConfigInput struct {
	ID string
}

type deleteMockResourceConfigOutput struct {
	ID string
}

func (m *mockService) DeleteResourceConfig(
	ctx context.Context,
	input *deleteMockResourceConfigInput,
) (*deleteMockResourceConfigOutput, error) {
	m.RegisterCall(ctx, input)
	return m.deleteMockResourceConfigOutput, m.deleteMockResourceConfigError
}

type deleteMockResourceCodeInput struct {
	ID string
}

type deleteMockResourceCodeOutput struct {
	ID string
}

func (m *mockService) DeleteResourceCode(
	ctx context.Context,
	input *deleteMockResourceCodeInput,
) (*deleteMockResourceCodeOutput, error) {
	m.RegisterCall(ctx, input)
	return m.deleteMockResourceCodeOutput, m.deleteMockResourceCodeError
}

func newMockResource(
	serviceFactory pluginutils.ServiceFactory[*mockConfig, *mockService],
	configStore pluginutils.ServiceConfigStore[*mockConfig],
) provider.Resource {
	actions := newMockResourceActions(serviceFactory, configStore)
	return &providerv1.ResourceDefinition{
		Type:                   "aws/lambda/function",
		Label:                  "AWS Lambda Function",
		IDField:                "arn",
		StabilisedDependencies: []string{"aws/sqs/queue"},
		CreateFunc:             actions.createMockResource,
		UpdateFunc:             actions.updateMockResource,
		DestroyFunc:            actions.destroyMockResource,
		GetExternalStateFunc:   actions.getMockResourceExternalState,
		StabilisedFunc:         actions.hasStabilisedMockResource,
	}
}

type mockResourceActions struct {
	serviceFactory pluginutils.ServiceFactory[*mockConfig, *mockService]
	configStore    pluginutils.ServiceConfigStore[*mockConfig]
}

func newMockResourceActions(
	serviceFactory pluginutils.ServiceFactory[*mockConfig, *mockService],
	configStore pluginutils.ServiceConfigStore[*mockConfig],
) *mockResourceActions {
	return &mockResourceActions{
		serviceFactory: serviceFactory,
		configStore:    configStore,
	}
}

func (m *mockResourceActions) createMockResource(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	serviceConfig, err := m.configStore.FromProviderContext(
		ctx,
		input.ProviderContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := m.serviceFactory(serviceConfig, input.ProviderContext)

	saveOutput, err := service.SaveResource(ctx, &saveMockResourceInput{})
	if err != nil {
		return nil, err
	}

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString(saveOutput.ID),
		},
	}, nil
}

func (m *mockResourceActions) updateMockResource(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	serviceConfig, err := m.configStore.FromProviderContext(
		ctx,
		input.ProviderContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := m.serviceFactory(serviceConfig, input.ProviderContext)

	updateConfOutput, err := service.UpdateConfig(
		ctx,
		&updateMockResourceConfigInput{},
	)
	if err != nil {
		return nil, err
	}

	// Second call to test assertions on multiple calls to the same method
	// with different inputs.
	_, err = service.UpdateConfig(
		ctx,
		&updateMockResourceConfigInput{
			Name:      "Second Update Name",
			DebugMode: true,
		},
	)
	if err != nil {
		return nil, err
	}

	_, err = service.UpdateCode(
		ctx,
		&updateMockResourceCodeInput{},
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString(updateConfOutput.ID),
		},
	}, nil
}

func (m *mockResourceActions) destroyMockResource(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	serviceConfig, err := m.configStore.FromProviderContext(
		ctx,
		input.ProviderContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return err
	}

	service := m.serviceFactory(serviceConfig, input.ProviderContext)
	_, err = service.DeleteResourceCode(
		ctx,
		&deleteMockResourceCodeInput{},
	)
	if err != nil {
		return err
	}

	_, err = service.DeleteResourceConfig(
		ctx,
		&deleteMockResourceConfigInput{},
	)
	if err != nil {
		return err
	}

	return err
}

func (m *mockResourceActions) hasStabilisedMockResource(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	serviceConfig, err := m.configStore.FromProviderContext(
		ctx,
		input.ProviderContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := m.serviceFactory(serviceConfig, input.ProviderContext)

	hasStabilisedOutput, err := service.GetResource(
		ctx,
		&getMockResourceInput{
			ID: input.ResourceID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceHasStabilisedOutput{
		Stabilised: hasStabilisedOutput.Status == "active",
	}, nil
}

func (m *mockResourceActions) getMockResourceExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	serviceConfig, err := m.configStore.FromProviderContext(
		ctx,
		input.ProviderContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := m.serviceFactory(serviceConfig, input.ProviderContext)

	getOutput, err := service.GetResource(
		ctx,
		&getMockResourceInput{
			ID: input.ResourceID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"id":           core.MappingNodeFromString(getOutput.ID),
				"name":         core.MappingNodeFromString(getOutput.Name),
				"debugMode":    core.MappingNodeFromBool(getOutput.DebugMode),
				"param1":       core.MappingNodeFromInt(getOutput.Param1),
				"param2":       core.MappingNodeFromString(getOutput.Param2),
				"param3":       core.MappingNodeFromFloat(getOutput.Param3),
				"inlineCode":   core.MappingNodeFromString(getOutput.InlineCode),
				"codeLocation": core.MappingNodeFromString(getOutput.CodeLocation),
				"tags":         mockTagsToMappingNodeSlice(getOutput.Tags),
			},
		},
	}, nil
}

func mockTagsToMappingNodeSlice(tags []mockTag) *core.MappingNode {
	mapping := &core.MappingNode{
		Items: []*core.MappingNode{},
	}

	for _, tag := range tags {
		tagObject := &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"customKey":   core.MappingNodeFromString(tag.CustomKey),
				"customValue": core.MappingNodeFromString(tag.CustomValue),
			},
		}
		mapping.Items = append(mapping.Items, tagObject)
	}

	return mapping
}
