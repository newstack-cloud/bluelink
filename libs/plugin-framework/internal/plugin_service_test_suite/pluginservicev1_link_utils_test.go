package pluginservicetestsuite

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testprovider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

const (
	lambdaFunctionResourceType = "aws/lambda/function"
	dynamoDBTableResourceType  = "aws/dynamodb/table"
	testResource1ID            = "test-resource-1"
	testInstance1ID            = "test-instance-1"
	testResource1Name          = "processOrderFunction_0"
	testResource2Name          = "ordersTable"
	testLinkID                 = "link-id-1"
	testLinkName               = "processOrderFunction_0::ordersTable"
	testResource2ID            = "test-resource-2"
)

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

func linkUpdateIntermediaryResourcesInput(
	linkUpdateType provider.LinkUpdateType,
) *provider.LinkUpdateIntermediaryResourcesInput {
	return &provider.LinkUpdateIntermediaryResourcesInput{
		ResourceAInfo:  createLinkResourceAInfo(),
		ResourceBInfo:  createLinkResourceBInfo(),
		LinkUpdateType: linkUpdateType,
		Changes:        testprovider.LinkLambdaDynamoDBChangesOutput().Changes,
		LinkContext:    testutils.CreateTestLinkContext(),
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
			TemplateName:               "processOrderFunction",
			InstanceID:                 testInstance1ID,
			Status:                     core.ResourceStatusCreated,
			PreciseStatus:              core.PreciseResourceStatusCreated,
			LastStatusUpdateTimestamp:  1742389743,
			LastDeployedTimestamp:      1742389743,
			LastDeployAttemptTimestamp: 1742389743,
			Description:                "Lambda function for processing orders",
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"arn":          core.MappingNodeFromString("arn:aws:lambda:us-west-2:123456789012:function:processOrderFunction_0"),
					"functionName": core.MappingNodeFromString("Process-Order-Function-0"),
				},
			},
			Metadata: &state.ResourceMetadataState{
				DisplayName: "Process Order Function",
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
			Description: core.MappingNodeFromString("Lambda function for processing orders"),
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"functionName": core.MappingNodeFromString("Process-Order-Function-0--Updated"),
				},
			},
			Metadata: &provider.ResolvedResourceMetadata{
				DisplayName: core.MappingNodeFromString("Process Order Function"),
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
