package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linkhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	ddbTableResourceType       = "aws/dynamodb/table"
	iamRoleResourceType        = "aws/iam/role"
	lambdaFunctionResourceType = "aws/lambda/function"
	ec2VPCResourceType         = "aws/ec2/vpc"
)

func createParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

func newTestAWSProvider(
	alwaysStabilise bool,
	skipRetryFailuresForLinkNames []string,
	stateContainer state.Container,
) provider.Provider {
	return &internal.ProviderMock{
		NamespaceValue: "aws",
		Resources: map[string]provider.Resource{
			ddbTableResourceType: &internal.DynamoDBTableResource{
				FallbackToStateContainerForExternalState: true,
				StateContainer:                           stateContainer,
				// Used to emulate transient failures when checking if DynamoDB
				// table has stabilised.
				CurrentStabiliseCalls: map[string]int{},
			},
			"aws/dynamodb/stream": &internal.DynamoDBStreamResource{
				FallbackToStateContainerForExternalState: true,
				StateContainer:                           stateContainer,
			},
			lambdaFunctionResourceType: &internal.LambdaFunctionResource{
				CurrentDestroyAttempts:         map[string]int{},
				CurrentDeployAttemps:           map[string]int{},
				CurrentGetExternalStateAttemps: map[string]int{},
				FailResourceIDs: []string{
					"test-failing-order-function-id",
					"test-failing-update-order-function-id",
				},
				StabiliseResourceIDs: map[string]*internal.StubResourceStabilisationConfig{
					"test-resource-id": {
						StabilisesAfterAttempts: 3,
					},
					"process-order-function": {
						StabilisesAfterAttempts: 2,
					},
					"get-order-function": {
						StabilisesAfterAttempts: 1,
					},
					"update-order-function": {
						StabilisesAfterAttempts: 1,
					},
					"list-orders-function": {
						// This function will never stabilise.
						StabilisesAfterAttempts: -1,
					},
				},
				AlwaysStabilise:       alwaysStabilise,
				CurrentStabiliseCalls: map[string]int{},
				SkipRetryFailuresForInstances: []string{
					"resource-deploy-test--blueprint-instance-2",
					"resource-deploy-test--blueprint-instance-3",
					"resource-deploy-test--blueprint-instance-4",
					"resource-deploy-test--blueprint-instance-5",
					"resource-deploy-test--blueprint-instance-6",
				},
				FallbackToStateContainerForExternalState: true,
				StateContainer:                           stateContainer,
			},
			"aws/lambda2/function": &internal.Lambda2FunctionResource{
				FallbackToStateContainerForExternalState: true,
				StateContainer:                           stateContainer,
			},
			iamRoleResourceType: &internal.IAMRoleResource{},
		},
		Links: map[string]provider.Link{
			"aws/apigateway/api::aws/lambda/function": &testApiGatewayLambdaLink{},
			"aws/lambda/function::aws/dynamodb/table": &testLambdaDynamoDBTableLink{
				resourceAUpdateAttempts: map[string]int{},
				failResourceANames:      []string{},
				failResourceBNames: []string{
					"ordersTableFailingLink_0",
				},
				failIntermediariesUpdateLinkNames: []string{},
				skipRetryFailuresForInstance:      []string{},
				skipRetryFailuresForLinkNames:     skipRetryFailuresForLinkNames,
			},
			"aws/dynamodb/table::aws/dynamodb/stream":   &testDynamoDBTableStreamLink{},
			"aws/dynamodb/stream::aws/lambda/function":  &testDynamoDBStreamLambdaLink{},
			"aws/lambda/function::aws/lambda/function":  &testLambdaLambdaLink{},
			"aws/lambda/function::aws/lambda2/function": &testLambdaLambda2Link{},
			"aws/dynamodb/table::aws/lambda/function":   &testDynamoDBTableLambdaLink{},
			"aws/ec2/subnet::aws/ec2/vpc":               &testSubnetVPCLink{},
			"aws/ec2/securityGroup::aws/ec2/vpc":        &testSecurityGroupVPCLink{},
			"aws/ec2/routeTable::aws/ec2/vpc":           &testRouteTableVPCLink{},
			"aws/ec2/route::aws/ec2/routeTable":         &testRouteRouteTableLink{},
			"aws/ec2/route::aws/ec2/internetGateway":    &testRouteInternetGatewayLink{},
		},
		CustomVariableTypes: map[string]provider.CustomVariableType{
			"aws/ec2/instanceType": &internal.InstanceTypeCustomVariableType{},
		},
		DataSources: map[string]provider.DataSource{
			"aws/vpc": &internal.VPCDataSource{},
		},
		ProviderRetryPolicy: &provider.RetryPolicy{
			MaxRetries: 3,
			// The first retry delay is 1 millisecond
			FirstRetryDelay: 0.001,
			// The maximum delay between retries is 10 milliseconds.
			MaxDelay:      0.01,
			BackoffFactor: 0.5,
			// Make the retry behaviour more deterministic for tests by disabling jitter.
			Jitter: false,
		},
	}
}

func newTestExampleProvider() provider.Provider {
	return &internal.ProviderMock{
		NamespaceValue: "example",
		Resources: map[string]provider.Resource{
			"example/complex": &internal.ExampleComplexResource{},
		},
		Links:               map[string]provider.Link{},
		CustomVariableTypes: map[string]provider.CustomVariableType{},
		DataSources:         map[string]provider.DataSource{},
	}
}

type testApiGatewayLambdaLink struct{}

func (l *testApiGatewayLambdaLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testApiGatewayLambdaLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: lambdaFunctionResourceType,
	}, nil
}

func (l *testApiGatewayLambdaLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/apigateway/api::aws/lambda/function",
	}, nil
}

func (l *testApiGatewayLambdaLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testApiGatewayLambdaLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindSoft,
	}, nil
}

func (l *testApiGatewayLambdaLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testApiGatewayLambdaLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testApiGatewayLambdaLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testApiGatewayLambdaLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testApiGatewayLambdaLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testLambdaDynamoDBTableLink struct {
	// Tracks the number of resource A update attempts for each unique link name.
	// This is used to emulate transient failures when updating links,
	// the blueprint container will retry updating resource A for the link until the
	// update attempt count exceeds the max update attempts.
	resourceAUpdateAttempts map[string]int
	// A list of logical resource names that should fail
	// with a terminal error when updating resource A.
	failResourceANames []string
	// A list of logical resource names that should fail
	// with a terminal error when updating resource B.
	failResourceBNames []string
	// A list of logical link names that should fail
	// with a terminal error when updating intermediary resources.
	failIntermediariesUpdateLinkNames []string
	// Instance IDs
	skipRetryFailuresForInstance []string
	// Logical link names (resourceAName::resourceBName) for which behaviour to
	// emulate transient failures should be skipped.
	skipRetryFailuresForLinkNames []string
	mu                            sync.Mutex
}

func (l *testLambdaDynamoDBTableLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	changes := &provider.LinkChanges{}

	functionResourceName := linkhelpers.GetResourceNameFromChanges(input.ResourceAChanges)
	tableResourceName := linkhelpers.GetResourceNameFromChanges(input.ResourceBChanges)

	currentLinkData := linkhelpers.GetLinkDataFromState(input.CurrentLinkState)

	tableFieldPath := fmt.Sprintf(
		"$.%s.environmentVariables.TABLE_NAME_%s",
		functionResourceName,
		tableResourceName,
	)
	err := linkhelpers.CollectChanges(
		"$.spec.tableName",
		tableFieldPath,
		currentLinkData,
		input.ResourceBChanges,
		changes,
	)
	if err != nil {
		return nil, err
	}

	regionFieldPath := fmt.Sprintf(
		"$.%s.environmentVariables.TABLE_REGION_%s",
		functionResourceName,
		tableResourceName,
	)
	err = linkhelpers.CollectChanges(
		"$.spec.region",
		regionFieldPath,
		currentLinkData,
		input.ResourceBChanges,
		changes,
	)
	if err != nil {
		return nil, err
	}

	accessType := linkhelpers.GetAnnotation(
		input.ResourceAChanges,
		"aws.lambda.dynamodb.accessType",
		core.MappingNodeFromString("read"),
	)
	actionArray := l.policyStatementActionFromAccessType(accessType)
	actionFieldPath := fmt.Sprintf(
		"$.%s[\"iam.policyStatements\"][0].action",
		functionResourceName,
	)
	err = linkhelpers.CollectLinkDataChanges(
		actionFieldPath,
		currentLinkData,
		changes,
		actionArray,
	)
	if err != nil {
		return nil, err
	}

	effect := core.MappingNodeFromString("Allow")
	effectFieldPath := fmt.Sprintf(
		"$.%s[\"iam.policyStatements\"][0].effect",
		functionResourceName,
	)
	err = linkhelpers.CollectLinkDataChanges(
		effectFieldPath,
		currentLinkData,
		changes,
		effect,
	)
	if err != nil {
		return nil, err
	}

	resourceARNFieldPath := fmt.Sprintf(
		"$.%s[\"iam.policyStatements\"][0].resource",
		functionResourceName,
	)
	err = linkhelpers.CollectChanges(
		"$.spec.id",
		resourceARNFieldPath,
		currentLinkData,
		input.ResourceBChanges,
		changes,
	)
	if err != nil {
		return nil, err
	}

	return &provider.LinkStageChangesOutput{
		Changes: changes,
	}, nil
}

func (l *testLambdaDynamoDBTableLink) policyStatementActionFromAccessType(accessType *core.MappingNode) *core.MappingNode {
	switch *accessType.Scalar.StringValue {
	case "write":
		return &core.MappingNode{
			Items: []*core.MappingNode{
				core.MappingNodeFromString("dynamodb:PutItem"),
				core.MappingNodeFromString("dynamodb:DeleteItem"),
				core.MappingNodeFromString("dynamodb:UpdateItem"),
			},
		}
	case "readwrite":
		return &core.MappingNode{
			Items: []*core.MappingNode{
				core.MappingNodeFromString("dynamodb:GetItem"),
				core.MappingNodeFromString("dynamodb:PutItem"),
				core.MappingNodeFromString("dynamodb:DeleteItem"),
				core.MappingNodeFromString("dynamodb:UpdateItem"),
			},
		}
	default:
		return &core.MappingNode{
			Items: []*core.MappingNode{
				core.MappingNodeFromString("dynamodb:GetItem"),
			},
		}
	}
}

func (l *testLambdaDynamoDBTableLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: ddbTableResourceType,
	}, nil
}

func (l *testLambdaDynamoDBTableLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/lambda/function::aws/dynamodb/table",
	}, nil
}

func (l *testLambdaDynamoDBTableLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testLambdaDynamoDBTableLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testLambdaDynamoDBTableLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testLambdaDynamoDBTableLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if slices.Contains(l.failResourceANames, input.ResourceInfo.ResourceName) {
		return nil, &provider.LinkUpdateResourceAError{
			FailureReasons: []string{"resource A update failed due to terminal error"},
		}
	}

	logicalLinkName := core.LogicalLinkName(
		input.ResourceInfo.ResourceName,
		input.OtherResourceInfo.ResourceName,
	)
	attemptCount, exists := l.resourceAUpdateAttempts[logicalLinkName]
	if !exists {
		attemptCount = 0
	}
	attemptCount += 1
	l.resourceAUpdateAttempts[logicalLinkName] = attemptCount

	// Provider retry policy allows for a maximum of 3 attempts before failing.
	if attemptCount < 3 &&
		!slices.Contains(l.skipRetryFailuresForInstance, input.ResourceInfo.InstanceID) &&
		!slices.Contains(l.skipRetryFailuresForLinkNames, logicalLinkName) {
		return nil, &provider.RetryableError{
			ChildError: errors.New("resource A update failed due to transient error"),
		}
	}

	tableNameEnvVar := fmt.Sprintf("TABLE_NAME_%s", input.OtherResourceInfo.ResourceName)
	tableRegionEnvVar := fmt.Sprintf("TABLE_REGION_%s", input.OtherResourceInfo.ResourceName)

	resourceEnvVarTableNameFieldPath := fmt.Sprintf(
		"%s::spec.environment.variables.%s",
		input.ResourceInfo.ResourceName,
		tableNameEnvVar,
	)
	linkEnvVarTableNameFieldPath := fmt.Sprintf(
		"%s.environmentVariables.%s",
		input.ResourceInfo.ResourceName,
		tableNameEnvVar,
	)
	resourceEnvVarTableRegionFieldPath := fmt.Sprintf(
		"%s::spec.environment.variables.%s",
		input.ResourceInfo.ResourceName,
		tableRegionEnvVar,
	)
	linkEnvVarTableRegionFieldPath := fmt.Sprintf(
		"%s.environmentVariables.%s",
		input.ResourceInfo.ResourceName,
		tableRegionEnvVar,
	)

	return &provider.LinkUpdateResourceOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				input.ResourceInfo.ResourceName: {
					Fields: map[string]*core.MappingNode{
						"environmentVariables": {
							Fields: map[string]*core.MappingNode{
								tableNameEnvVar:   core.MappingNodeFromString("production-orders"),
								tableRegionEnvVar: core.MappingNodeFromString("eu-west-2"),
							},
						},
					},
				},
			},
		},
		ResourceDataMappings: map[string]string{
			resourceEnvVarTableNameFieldPath:   linkEnvVarTableNameFieldPath,
			resourceEnvVarTableRegionFieldPath: linkEnvVarTableRegionFieldPath,
		},
	}, nil
}

func (l *testLambdaDynamoDBTableLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if slices.Contains(l.failResourceBNames, input.ResourceInfo.ResourceName) {
		return nil, &provider.LinkUpdateResourceBError{
			FailureReasons: []string{"resource B update failed due to terminal error"},
		}
	}

	return &provider.LinkUpdateResourceOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				input.ResourceInfo.ResourceName: core.MappingNodeFromString("testResourceBValue"),
			},
		},
	}, nil
}

func (l *testLambdaDynamoDBTableLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	linkName := fmt.Sprintf(
		"%s::%s",
		input.ResourceAInfo.ResourceName,
		input.ResourceBInfo.ResourceName,
	)
	if slices.Contains(l.failIntermediariesUpdateLinkNames, linkName) {
		return nil, &provider.LinkUpdateIntermediaryResourcesError{
			FailureReasons: []string{"intermediary resources update failed due to terminal error"},
		}
	}

	deployInput := &provider.ResourceDeployInput{
		InstanceID: input.ResourceAInfo.InstanceID,
		ResourceID: "testRoleID",
		Changes: &provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				ResourceID:   "testRoleID",
				ResourceName: "testRole",
				InstanceID:   input.ResourceAInfo.InstanceID,
			},
		},
		ProviderContext: provider.NewProviderContextFromLinkContext(
			input.LinkContext,
			"aws",
		),
	}
	roleDeployOutput, err := input.ResourceService.Deploy(
		ctx,
		iamRoleResourceType,
		&provider.ResourceDeployServiceInput{
			DeployInput:     deployInput,
			WaitUntilStable: false,
		},
	)
	if err != nil {
		return nil, err
	}

	resolvedResource := &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: iamRoleResourceType,
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}
	roleSpecState, err := specmerge.MergeResourceSpec(
		resolvedResource,
		"testRoleID",
		roleDeployOutput.ComputedFieldValues,
		[]string{"spec.id"},
	)
	if err != nil {
		return nil, err
	}

	return &provider.LinkUpdateIntermediaryResourcesOutput{
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
			{
				ResourceID:       "testRoleID",
				ResourceSpecData: roleSpecState,
			},
		},
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"testIntermediaryResource": core.MappingNodeFromString("testIntermediaryResourceValue"),
			},
		},
	}, nil
}

func (l *testLambdaDynamoDBTableLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testDynamoDBTableStreamLink struct{}

func (l *testDynamoDBTableStreamLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testDynamoDBTableStreamLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceA,
		PriorityResourceType: ddbTableResourceType,
	}, nil
}

func (l *testDynamoDBTableStreamLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/dynamodb/table::aws/dynamodb/stream",
	}, nil
}

func (l *testDynamoDBTableStreamLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testDynamoDBTableStreamLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testDynamoDBTableStreamLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testDynamoDBTableStreamLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testDynamoDBTableStreamLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testDynamoDBTableStreamLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testDynamoDBTableStreamLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testDynamoDBStreamLambdaLink struct{}

func (l *testDynamoDBStreamLambdaLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testDynamoDBStreamLambdaLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: lambdaFunctionResourceType,
	}, nil
}

func (l *testDynamoDBStreamLambdaLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/dynamodb/stream::aws/lambda/function",
	}, nil
}

func (l *testDynamoDBStreamLambdaLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testDynamoDBStreamLambdaLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindSoft,
	}, nil
}

func (l *testDynamoDBStreamLambdaLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testDynamoDBStreamLambdaLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testDynamoDBStreamLambdaLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testDynamoDBStreamLambdaLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testDynamoDBStreamLambdaLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testDynamoDBTableLambdaLink struct{}

func (l *testDynamoDBTableLambdaLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testDynamoDBTableLambdaLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceA,
		PriorityResourceType: ddbTableResourceType,
	}, nil
}

func (l *testDynamoDBTableLambdaLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/dynamodb/table::aws/lambda/function",
	}, nil
}

func (l *testDynamoDBTableLambdaLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testDynamoDBTableLambdaLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testDynamoDBTableLambdaLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testDynamoDBTableLambdaLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testDynamoDBTableLambdaLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testDynamoDBTableLambdaLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testDynamoDBTableLambdaLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testLambdaLambdaLink struct{}

func (l *testLambdaLambdaLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testLambdaLambdaLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceNone,
		PriorityResourceType: lambdaFunctionResourceType,
	}, nil
}

func (l *testLambdaLambdaLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/lambda/function::aws/lambda/function",
	}, nil
}

func (l *testLambdaLambdaLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testLambdaLambdaLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testLambdaLambdaLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testLambdaLambdaLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testLambdaLambdaLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testLambdaLambdaLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testLambdaLambdaLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testLambdaLambda2Link struct{}

func (l *testLambdaLambda2Link) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testLambdaLambda2Link) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		// Lambda -> Lambda2 exists so there can be a relationship between
		// 2 lambda functions where one has priority for certain test cases.
		PriorityResource:     provider.LinkPriorityResourceA,
		PriorityResourceType: lambdaFunctionResourceType,
	}, nil
}

func (l *testLambdaLambda2Link) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/lambda/function::aws/lambda2/function",
	}, nil
}

func (l *testLambdaLambda2Link) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testLambdaLambda2Link) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testLambdaLambda2Link) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testLambdaLambda2Link) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testLambdaLambda2Link) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testLambdaLambda2Link) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testLambdaLambda2Link) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testSubnetVPCLink struct{}

func (l *testSubnetVPCLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testSubnetVPCLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: ec2VPCResourceType,
	}, nil
}

func (l *testSubnetVPCLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/ec2/subnet::aws/ec2/vpc",
	}, nil
}

func (l *testSubnetVPCLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testSubnetVPCLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testSubnetVPCLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testSubnetVPCLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testSubnetVPCLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testSubnetVPCLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testSubnetVPCLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testSecurityGroupVPCLink struct{}

func (l *testSecurityGroupVPCLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testSecurityGroupVPCLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: ec2VPCResourceType,
	}, nil
}

func (l *testSecurityGroupVPCLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/ec2/securityGroup::aws/ec2/vpc",
	}, nil
}

func (l *testSecurityGroupVPCLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testSecurityGroupVPCLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testSecurityGroupVPCLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testSecurityGroupVPCLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testSecurityGroupVPCLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testSecurityGroupVPCLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testSecurityGroupVPCLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testRouteTableVPCLink struct{}

func (l *testRouteTableVPCLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testRouteTableVPCLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: ec2VPCResourceType,
	}, nil
}

func (l *testRouteTableVPCLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/ec2/routeTable::aws/ec2/vpc",
	}, nil
}

func (l *testRouteTableVPCLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testRouteTableVPCLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testRouteTableVPCLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testRouteTableVPCLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testRouteTableVPCLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testRouteTableVPCLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testRouteTableVPCLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testRouteRouteTableLink struct{}

func (l *testRouteRouteTableLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testRouteRouteTableLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: "aws/ec2/routeTable",
	}, nil
}

func (l *testRouteRouteTableLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/ec2/route::aws/ec2/routeTable",
	}, nil
}

func (l *testRouteRouteTableLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testRouteRouteTableLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testRouteRouteTableLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testRouteRouteTableLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testRouteRouteTableLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testRouteRouteTableLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testRouteRouteTableLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

type testRouteInternetGatewayLink struct{}

func (l *testRouteInternetGatewayLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testRouteInternetGatewayLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: "aws/ec2/internetGateway",
	}, nil
}

func (l *testRouteInternetGatewayLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/ec2/route::aws/ec2/internetGateway",
	}, nil
}

func (l *testRouteInternetGatewayLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testRouteInternetGatewayLink) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testRouteInternetGatewayLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testRouteInternetGatewayLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testRouteInternetGatewayLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testRouteInternetGatewayLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testRouteInternetGatewayLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}
