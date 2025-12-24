package plugintestsuites

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testprovider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

func linkStageChangesInput() *provider.LinkStageChangesInput {
	return &provider.LinkStageChangesInput{
		ResourceAChanges: createDeployResourceChanges( /* mustRecreate */ false),
		ResourceBChanges: createDeployResourceChanges( /* mustRecreate */ false),
		CurrentLinkState: createCurrentLinkState(),
		LinkContext:      testutils.CreateTestLinkContext(),
	}
}

func linkUpdateResourceAInput() *provider.LinkUpdateResourceInput {
	return &provider.LinkUpdateResourceInput{
		Changes:           testprovider.LinkLambdaDynamoDBChangesOutput().Changes,
		ResourceInfo:      createLinkResourceAInfo(),
		OtherResourceInfo: createLinkResourceBInfo(),
		LinkUpdateType:    provider.LinkUpdateTypeCreate,
		CurrentLinkState:  createCurrentLinkState(),
		LinkContext:       testutils.CreateTestLinkContext(),
	}
}

func linkUpdateResourceBInput() *provider.LinkUpdateResourceInput {
	return &provider.LinkUpdateResourceInput{
		Changes:           testprovider.LinkLambdaDynamoDBChangesOutput().Changes,
		ResourceInfo:      createLinkResourceBInfo(),
		OtherResourceInfo: createLinkResourceAInfo(),
		LinkUpdateType:    provider.LinkUpdateTypeCreate,
		CurrentLinkState:  createCurrentLinkState(),
		LinkContext:       testutils.CreateTestLinkContext(),
	}
}

func createLinkResourceAInfo() *provider.ResourceInfo {
	resourceInfo := createDeployResourceInfo()
	return &resourceInfo
}

func createLinkResourceBInfo() *provider.ResourceInfo {
	lastDriftDetected := 1742389743
	configCompleteDuration := 8.5
	totalDuration := 69.5
	return &provider.ResourceInfo{
		ResourceID:   testResource2ID,
		ResourceName: testResource2Name,
		InstanceID:   testInstance1ID,
		CurrentResourceState: &state.ResourceState{
			ResourceID:                 testResource2ID,
			Name:                       testResource2Name,
			Type:                       dynamoDBTableResourceType,
			InstanceID:                 testInstance1ID,
			Status:                     core.ResourceStatusCreated,
			PreciseStatus:              core.PreciseResourceStatusCreated,
			LastStatusUpdateTimestamp:  1742389743,
			LastDeployedTimestamp:      1742389743,
			LastDeployAttemptTimestamp: 1742389743,
			Description:                "Table for storing orders",
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"arn":       core.MappingNodeFromString("arn:aws:dynamodb:us-west-2:123456789012:table:ordersTable"),
					"tableName": core.MappingNodeFromString("orders"),
				},
			},
			Metadata: &state.ResourceMetadataState{
				DisplayName: "Orders Table",
				Annotations: map[string]*core.MappingNode{
					"example.annotation.v1": core.MappingNodeFromString("example-value"),
				},
				Labels: map[string]string{
					"app": "orders",
				},
				Custom: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"customField": core.MappingNodeFromString("customValue"),
					},
				},
			},
			DependsOnResources:         []string{"orderQueue"},
			DependsOnChildren:          []string{"coreInfra"},
			Drifted:                    true,
			LastDriftDetectedTimestamp: &lastDriftDetected,
			Durations: &state.ResourceCompletionDurations{
				ConfigCompleteDuration: &configCompleteDuration,
				TotalDuration:          &totalDuration,
				AttemptDurations:       []float64{10.5, 20.5, 30.5},
			},
		},
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{
				Value: lambdaFunctionResourceType,
			},
			Description: core.MappingNodeFromString("Table for storing orders"),
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"tableName": core.MappingNodeFromString("orders--updated"),
				},
			},
			Metadata: &provider.ResolvedResourceMetadata{
				DisplayName: core.MappingNodeFromString("Orders Table"),
				Annotations: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"example.annotation.v1": core.MappingNodeFromString("example-value-updated"),
					},
				},
				Labels: &schema.StringMap{
					Values: map[string]string{
						"app": "orders",
					},
				},
				Custom: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"customField": core.MappingNodeFromString("customValueUpdated"),
					},
				},
			},
			Condition: &provider.ResolvedResourceCondition{
				StringValue: core.MappingNodeFromBool(true),
			},
			LinkSelector: &schema.LinkSelector{
				ByLabel: &schema.StringMap{
					Values: map[string]string{
						"app": "orders",
					},
				},
			},
		},
	}
}

func linkUpdateIntermediaryResourcesInput() *provider.LinkUpdateIntermediaryResourcesInput {
	return &provider.LinkUpdateIntermediaryResourcesInput{
		ResourceAInfo:    createLinkResourceAInfo(),
		ResourceBInfo:    createLinkResourceBInfo(),
		LinkUpdateType:   provider.LinkUpdateTypeCreate,
		CurrentLinkState: createCurrentLinkState(),
		Changes:          testprovider.LinkLambdaDynamoDBChangesOutput().Changes,
		LinkContext:      testutils.CreateTestLinkContext(),
	}
}

func createCurrentLinkState() *state.LinkState {
	resourceAUpdateDuration := 8.5
	resourceBUpdateDuration := 12.5
	intermediaryResourcesUpdateDuraition := 18.2
	return &state.LinkState{
		LinkID:                     testLinkID,
		Name:                       testLinkName,
		InstanceID:                 testInstance1ID,
		Status:                     core.LinkStatusCreated,
		PreciseStatus:              core.PreciseLinkStatusIntermediaryResourcesUpdated,
		LastStatusUpdateTimestamp:  1742389743,
		LastDeployedTimestamp:      1742389743,
		LastDeployAttemptTimestamp: 1742389743,
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{},
		Data: map[string]*core.MappingNode{
			testResource1Name: {
				Fields: map[string]*core.MappingNode{
					"environmentVariables": {
						Fields: map[string]*core.MappingNode{
							"TABLE_NAME_ordersTable": core.MappingNodeFromString("orders"),
						},
					},
				},
			},
		},
		Durations: &state.LinkCompletionDurations{
			ResourceAUpdate: &state.LinkComponentCompletionDurations{
				TotalDuration:    &resourceAUpdateDuration,
				AttemptDurations: []float64{resourceAUpdateDuration},
			},
			ResourceBUpdate: &state.LinkComponentCompletionDurations{
				TotalDuration:    &resourceBUpdateDuration,
				AttemptDurations: []float64{resourceBUpdateDuration},
			},
			IntermediaryResources: &state.LinkComponentCompletionDurations{
				TotalDuration:    &intermediaryResourcesUpdateDuraition,
				AttemptDurations: []float64{intermediaryResourcesUpdateDuraition},
			},
		},
	}
}

func linkGetPriorityResourceInput() *provider.LinkGetPriorityResourceInput {
	return &provider.LinkGetPriorityResourceInput{
		LinkContext: testutils.CreateTestLinkContext(),
	}
}

func linkGetTypeInput() *provider.LinkGetTypeInput {
	return &provider.LinkGetTypeInput{
		LinkContext: testutils.CreateTestLinkContext(),
	}
}

func linkGetTypeDescriptionInput() *provider.LinkGetTypeDescriptionInput {
	return &provider.LinkGetTypeDescriptionInput{
		LinkContext: testutils.CreateTestLinkContext(),
	}
}

func linkGetAnnotationDefnitionsInput() *provider.LinkGetAnnotationDefinitionsInput {
	return &provider.LinkGetAnnotationDefinitionsInput{
		LinkContext: testutils.CreateTestLinkContext(),
	}
}

func linkGetKindInput() *provider.LinkGetKindInput {
	return &provider.LinkGetKindInput{
		LinkContext: testutils.CreateTestLinkContext(),
	}
}

func linkGetIntermediaryExternalStateInput() *provider.LinkGetIntermediaryExternalStateInput {
	return &provider.LinkGetIntermediaryExternalStateInput{
		ResourceAInfo:            createLinkResourceAInfo(),
		ResourceBInfo:            createLinkResourceBInfo(),
		IntermediaryResourceIDs:  []string{"iam-role-1", "iam-policy-1"},
		CurrentLinkState:         createCurrentLinkState(),
		LinkContext:              testutils.CreateTestLinkContext(),
	}
}
