package testprovider

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	lambdaFunctionResourceType = "aws/lambda/function"
	testResource1ID            = "test-resource-1"
	testInstance1ID            = "test-instance-1"
	testResource1Name          = "saveOrderFunction_0"
	testResource2Name          = "ordersTable"
)

func createDeployIntermediaryResourceChanges() *provider.Changes {
	return &provider.Changes{
		AppliedResourceInfo: createDeployResourceInfo(),
		MustRecreate:        false,
		ModifiedFields: []provider.FieldChange{
			{
				FieldPath: "spec.functionName",
				PrevValue: core.MappingNodeFromString("Save-Order-Function-0"),
				NewValue:  core.MappingNodeFromString("Save-Order-Function-0--Updated"),
			},
		},
		NewFields:                 []provider.FieldChange{},
		RemovedFields:             []string{},
		UnchangedFields:           []string{},
		ComputedFields:            []string{"spec.arn"},
		FieldChangesKnownOnDeploy: []string{},
		ConditionKnownOnDeploy:    false,
		NewOutboundLinks:          map[string]provider.LinkChanges{},
		OutboundLinkChanges:       map[string]provider.LinkChanges{},
		RemovedOutboundLinks:      []string{},
	}
}

func createDeployResourceInfo() provider.ResourceInfo {
	lastDriftDetected := 1742389743
	configCompleteDuration := 8.5
	totalDuration := 69.5
	return provider.ResourceInfo{
		ResourceID:   testResource1ID,
		ResourceName: testResource1Name,
		InstanceID:   testInstance1ID,
		CurrentResourceState: &state.ResourceState{
			ResourceID:                 testResource1ID,
			Name:                       testResource1Name,
			Type:                       lambdaFunctionResourceType,
			TemplateName:               "saveOrderFunction",
			InstanceID:                 testInstance1ID,
			Status:                     core.ResourceStatusCreated,
			PreciseStatus:              core.PreciseResourceStatusCreated,
			LastStatusUpdateTimestamp:  1742389743,
			LastDeployedTimestamp:      1742389743,
			LastDeployAttemptTimestamp: 1742389743,
			Description:                "Lambda function for saving orders",
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"arn":          core.MappingNodeFromString("arn:aws:lambda:us-west-2:123456789012:function:saveOrderFunction_0"),
					"functionName": core.MappingNodeFromString("Save-Order-Function-0"),
				},
			},
			Metadata: &state.ResourceMetadataState{
				DisplayName: "Save Order Function",
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
			Description: core.MappingNodeFromString("Lambda function for saving orders"),
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"functionName": core.MappingNodeFromString("Save-Order-Function-0--Updated"),
				},
			},
			Metadata: &provider.ResolvedResourceMetadata{
				DisplayName: core.MappingNodeFromString("Save Order Function"),
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
