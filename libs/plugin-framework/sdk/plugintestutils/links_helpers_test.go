package plugintestutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func newMockLink(
	linkDeps pluginutils.LinkServiceDeps[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	],
) provider.Link {
	actions := newMockLinkActions(
		linkDeps.ResourceAService.ServiceFactory,
		linkDeps.ResourceAService.ConfigStore,
	)
	return &providerv1.LinkDefinition{
		ResourceTypeA:                   "aws/lambda/function",
		ResourceTypeB:                   "aws/dynamodb/table",
		Kind:                            provider.LinkKindSoft,
		UpdateResourceAFunc:             actions.updateResourceA,
		UpdateResourceBFunc:             actions.updateResourceB,
		UpdateIntermediaryResourcesFunc: actions.updateIntermediaryResources,
		StageChangesFunc:                actions.stageChanges,
	}
}

type mockLinkActions struct {
	serviceFactory pluginutils.ServiceFactory[*mockConfig, *mockService]
	configStore    pluginutils.ServiceConfigStore[*mockConfig]
}

func newMockLinkActions(
	serviceFactory pluginutils.ServiceFactory[*mockConfig, *mockService],
	configStore pluginutils.ServiceConfigStore[*mockConfig],
) *mockLinkActions {
	return &mockLinkActions{
		serviceFactory: serviceFactory,
		configStore:    configStore,
	}
}

func (a *mockLinkActions) updateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	providerContext := provider.NewProviderContextFromLinkContext(
		input.LinkContext,
		"aws",
	)
	serviceConfig, err := a.configStore.FromProviderContext(
		ctx,
		providerContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := a.serviceFactory(serviceConfig, providerContext)

	saveOutput, err := service.SaveResource(ctx, &saveMockResourceInput{})
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateResourceOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"resourceAId": core.MappingNodeFromString(saveOutput.ID),
			},
		},
	}, nil
}

func (a *mockLinkActions) updateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	providerContext := provider.NewProviderContextFromLinkContext(
		input.LinkContext,
		"aws",
	)
	serviceConfig, err := a.configStore.FromProviderContext(
		ctx,
		providerContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := a.serviceFactory(serviceConfig, providerContext)

	saveOutput, err := service.SaveResource(ctx, &saveMockResourceInput{})
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateResourceOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"resourceBId": core.MappingNodeFromString(saveOutput.ID),
			},
		},
	}, nil
}

func (a *mockLinkActions) updateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	providerContext := provider.NewProviderContextFromLinkContext(
		input.LinkContext,
		"aws",
	)
	serviceConfig, err := a.configStore.FromProviderContext(
		ctx,
		providerContext,
		map[string]*core.MappingNode{},
	)
	if err != nil {
		return nil, err
	}

	service := a.serviceFactory(serviceConfig, providerContext)

	saveOutput, err := service.SaveResource(ctx, &saveMockResourceInput{})
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateIntermediaryResourcesOutput{
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
			{
				ResourceID: saveOutput.ID,
			},
		},
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"intermediaryResourceId": core.MappingNodeFromString(saveOutput.ID),
			},
		},
	}, nil
}

func (a *mockLinkActions) stageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			ModifiedFields: []*provider.FieldChange{
				{
					FieldPath: "testField",
					NewValue:  core.MappingNodeFromString("newValue"),
				},
			},
		},
	}, nil
}
