// Resource implementations for tests.

package internal

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type DynamoDBTableResource struct {
	// A stub for the resource state value to return when requesting the state
	// of the resource in the external provider.
	ExternalState                            *core.MappingNode
	FallbackToStateContainerForExternalState bool
	StateContainer                           state.Container
	// Tracks the number of stabilise calls have been made for a resource ID.
	// Unlike the test lambda resource, this is not used to test polling behaviour,
	// this is used to test that transient failures are handled correctly by the
	// resource deployer.
	CurrentStabiliseCalls map[string]int
	mu                    sync.Mutex
}

func (r *DynamoDBTableResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/dynamodb/stream", "aws/lambda/function"},
	}, nil
}

func (r *DynamoDBTableResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (r *DynamoDBTableResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: true,
	}, nil
}

func (r *DynamoDBTableResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/dynamodb/table",
	}, nil
}

func (r *DynamoDBTableResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "",
		MarkdownDescription:  "# DynamoDB Table\n\nA table in DynamoDB.",
	}, nil
}

func (r *DynamoDBTableResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *DynamoDBTableResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *DynamoDBTableResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	defaultGlobal := false
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"tableName": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"region": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"global": {
						Type: provider.ResourceDefinitionsSchemaTypeBoolean,
						Default: &core.MappingNode{
							Scalar: &core.ScalarValue{
								BoolValue: &defaultGlobal,
							},
						},
					},
				},
			},
		},
	}, nil
}

func (r *DynamoDBTableResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{}, nil
}

func (r *DynamoDBTableResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.CurrentStabiliseCalls != nil {
		attemptCount, exists := r.CurrentStabiliseCalls[input.ResourceID]
		if !exists {
			attemptCount = 0
		}
		attemptCount += 1
		r.CurrentStabiliseCalls[input.ResourceID] = attemptCount

		// Provider retry policy allows for a maximum of 3 attempts before failing.
		if attemptCount < 3 {
			return nil, &provider.RetryableError{
				ChildError: errors.New("stabilisation check failed due to transient error"),
			}
		}
	}

	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *DynamoDBTableResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	if r.ExternalState == nil && r.FallbackToStateContainerForExternalState {
		resource, err := r.StateContainer.Resources().Get(
			ctx,
			input.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetExternalStateOutput{
			ResourceSpecState: resource.SpecData,
		}, nil
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: r.ExternalState,
	}, nil
}

func (r *DynamoDBTableResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

type DynamoDBStreamResource struct {
	// A stub for the resource state value to return when requesting the state
	// of the resource in the external provider.
	ExternalState                            *core.MappingNode
	FallbackToStateContainerForExternalState bool
	StateContainer                           state.Container
}

func (r *DynamoDBStreamResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/lambda/function"},
	}, nil
}

func (r *DynamoDBStreamResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (r *DynamoDBStreamResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *DynamoDBStreamResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/dynamodb/stream",
	}, nil
}

func (r *DynamoDBStreamResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "",
		MarkdownDescription:  "# DynamoDB Stream\n\nA table event stream in DynamoDB.",
	}, nil
}

func (r *DynamoDBStreamResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *DynamoDBStreamResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *DynamoDBStreamResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"label": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"region": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *DynamoDBStreamResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{}, nil
}

func (r *DynamoDBStreamResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *DynamoDBStreamResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	if r.ExternalState == nil && r.FallbackToStateContainerForExternalState {
		resource, err := r.StateContainer.Resources().Get(
			ctx,
			input.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetExternalStateOutput{
			ResourceSpecState: resource.SpecData,
		}, nil
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: r.ExternalState,
	}, nil
}

func (r *DynamoDBStreamResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

type LambdaFunctionResource struct {
	// Tracks the number of destroy attempts for each unique resource ID.
	// This is used to emulate transient failures when destroying resources,
	// the blueprint container will retry destroying the resource until the
	// destroy attempt count exceeds the max destroy attempts.
	CurrentDestroyAttempts map[string]int
	// Tracks the number of deploy attempts for each unique resource ID.
	// This is used to emulate transient failures when deploying resources,
	// the blueprint container will retry deploying the resource until the
	// deploy attempt count exceeds the max deploy attempts.
	CurrentDeployAttemps map[string]int
	// Tracks the number of get external state attempts for each unique resource ID.
	// This is used to emulate transient failures when getting the external state
	// of resources, the drift checker will retry getting the external state
	// until the get external state attempt count exceeds the max get external state attempts.
	CurrentGetExternalStateAttemps map[string]int
	// Resource IDs for which the lambda function resource implementation
	// should fail with a terminal error.
	FailResourceIDs []string
	// A mapping of resource IDs to their respective stub resource stabilisation
	// configuration.
	StabiliseResourceIDs map[string]*StubResourceStabilisationConfig
	// Override stabilisation config to always report the resource as stabilised
	// on first check.
	AlwaysStabilise bool
	// Tracks the number of stabilise calls have been made for a resource ID.
	CurrentStabiliseCalls map[string]int
	// A list of instance IDs for which retry failures should be skipped.
	SkipRetryFailuresForInstances []string
	// A stub for the resource state value to return when requesting the state
	// of the resource in the external provider.
	ExternalState                            *core.MappingNode
	FallbackToStateContainerForExternalState bool
	StateContainer                           state.Container
	mu                                       sync.Mutex
}

func (r *LambdaFunctionResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/dynamodb/table", "aws/lambda/function", "aws/lambda2/function"},
	}, nil
}

func (r *LambdaFunctionResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (r *LambdaFunctionResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: true,
	}, nil
}

func (r *LambdaFunctionResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/lambda/function",
	}, nil
}

func (r *LambdaFunctionResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "",
		MarkdownDescription:  "# AWS Lambda\n\nA Lambda function in AWS.",
	}, nil
}

func (r *LambdaFunctionResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *LambdaFunctionResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *LambdaFunctionResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"handler": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *LambdaFunctionResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if slices.Contains(r.FailResourceIDs, input.ResourceID) {
		return nil, &provider.ResourceDeployError{
			FailureReasons: []string{"deploy failed due to terminal error"},
		}
	}

	if !slices.Contains(r.SkipRetryFailuresForInstances, input.InstanceID) {
		attemptCount, exists := r.CurrentDeployAttemps[input.ResourceID]
		if !exists {
			attemptCount = 0
		}
		attemptCount += 1
		r.CurrentDeployAttemps[input.ResourceID] = attemptCount

		// Provider retry policy allows for a maximum of 3 attempts before failing.
		if attemptCount < 3 {
			return nil, &provider.RetryableError{
				ChildError: errors.New("deploy failed due to transient error"),
			}
		}
	}

	id := fmt.Sprintf(
		"arn:aws:lambda:us-east-1:123456789012:function:%s",
		input.Changes.AppliedResourceInfo.ResourceName,
	)

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString(id),
		},
	}, nil
}

func (r *LambdaFunctionResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	if r.AlwaysStabilise {
		return &provider.ResourceHasStabilisedOutput{
			Stabilised: true,
		}, nil
	}

	if r.StabiliseResourceIDs == nil || r.CurrentStabiliseCalls == nil {
		return &provider.ResourceHasStabilisedOutput{
			Stabilised: false,
		}, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stubConfig, exists := r.StabiliseResourceIDs[input.ResourceID]
	if !exists {
		return &provider.ResourceHasStabilisedOutput{
			Stabilised: false,
		}, nil
	}

	stabiliseCalls, exists := r.CurrentStabiliseCalls[input.ResourceID]
	if !exists {
		stabiliseCalls = 0
	}

	stabiliseCalls += 1
	r.CurrentStabiliseCalls[input.ResourceID] = stabiliseCalls

	if stabiliseCalls < stubConfig.StabilisesAfterAttempts ||
		stubConfig.StabilisesAfterAttempts == -1 {
		return &provider.ResourceHasStabilisedOutput{
			Stabilised: false,
		}, nil
	}

	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *LambdaFunctionResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	attemptCount, exists := r.CurrentGetExternalStateAttemps[input.ResourceID]
	if !exists {
		attemptCount = 0
	}
	attemptCount += 1
	r.CurrentGetExternalStateAttemps[input.ResourceID] = attemptCount

	// Provider retry policy allows for a maximum of 3 attempts before failing.
	if attemptCount < 3 {
		return nil, &provider.RetryableError{
			ChildError: errors.New("get external state failed due to transient error"),
		}
	}

	if r.ExternalState == nil && r.FallbackToStateContainerForExternalState {
		resource, err := r.StateContainer.Resources().Get(
			ctx,
			input.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetExternalStateOutput{
			ResourceSpecState: resource.SpecData,
		}, nil
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: r.ExternalState,
	}, nil
}

func (r *LambdaFunctionResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if slices.Contains(r.FailResourceIDs, input.ResourceID) {
		return &provider.ResourceDestroyError{
			FailureReasons: []string{"destroy failed due to terminal error"},
		}
	}

	attemptCount, exists := r.CurrentDestroyAttempts[input.ResourceID]
	if !exists {
		attemptCount = 0
	}
	attemptCount += 1
	r.CurrentDestroyAttempts[input.ResourceID] = attemptCount

	// Provider retry policy allows for a maximum of 3 attempts before failing.
	if attemptCount < 3 {
		return &provider.RetryableError{
			ChildError: errors.New("destroy failed due to transient error"),
		}
	}

	return nil
}

type Lambda2FunctionResource struct {
	// A stub for the resource state value to return when requesting the state
	// of the resource in the external provider.
	ExternalState                            *core.MappingNode
	FallbackToStateContainerForExternalState bool
	StateContainer                           state.Container
}

func (r *Lambda2FunctionResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{"aws/lambda/function"},
	}, nil
}

func (r *Lambda2FunctionResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (r *Lambda2FunctionResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: true,
	}, nil
}

func (r *Lambda2FunctionResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/lambda2/function",
	}, nil
}

func (r *Lambda2FunctionResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "",
		MarkdownDescription:  "# AWS Lambda\n\nA Lambda function in AWS.",
	}, nil
}

func (r *Lambda2FunctionResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *Lambda2FunctionResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *Lambda2FunctionResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"handler": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *Lambda2FunctionResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	id := fmt.Sprintf(
		"arn:aws:lambda:us-east-1:123456789012:function:%s",
		input.Changes.AppliedResourceInfo.ResourceName,
	)

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString(id),
		},
	}, nil
}

func (r *Lambda2FunctionResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *Lambda2FunctionResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	if r.ExternalState == nil && r.FallbackToStateContainerForExternalState {
		resource, err := r.StateContainer.Resources().Get(
			ctx,
			input.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetExternalStateOutput{
			ResourceSpecState: resource.SpecData,
		}, nil
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: r.ExternalState,
	}, nil
}

func (r *Lambda2FunctionResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

type ExampleComplexResource struct {
	// A stub for the resource state value to return when requesting the state
	// of the resource in the external provider.
	ExternalState                            *core.MappingNode
	FallbackToStateContainerForExternalState bool
	StateContainer                           state.Container
}

func (r *ExampleComplexResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{},
	}, nil
}

func (r *ExampleComplexResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (r *ExampleComplexResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: true,
	}, nil
}

func (r *ExampleComplexResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "example/complex",
	}, nil
}

func (r *ExampleComplexResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "",
		MarkdownDescription:  "# An example resource with a complex specification",
	}, nil
}

func (r *ExampleComplexResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *ExampleComplexResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *ExampleComplexResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	defaultPrimaryPort := 80
	defaultOtherItemConfigValue1 := "Contents of value 1"
	defaultOtherItemConfigValue2 := "Contents of value 2"
	defaultVendorId := "default-vendor-id"

	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"itemConfig": {
						Type: provider.ResourceDefinitionsSchemaTypeUnion,
						OneOf: []*provider.ResourceDefinitionsSchema{
							{
								Type: provider.ResourceDefinitionsSchemaTypeString,
							},
							{
								Type: provider.ResourceDefinitionsSchemaTypeObject,
								Attributes: map[string]*provider.ResourceDefinitionsSchema{
									"endpoints": {
										Type: provider.ResourceDefinitionsSchemaTypeArray,
										Items: &provider.ResourceDefinitionsSchema{
											Type: provider.ResourceDefinitionsSchemaTypeString,
										},
									},
									"primaryPort": {
										Type: provider.ResourceDefinitionsSchemaTypeInteger,
										Default: &core.MappingNode{
											Scalar: &core.ScalarValue{
												IntValue: &defaultPrimaryPort,
											},
										},
										// In reality a port number would not be sensitive,
										// this is just to test the sensitive field logic for field
										// change values without significantly changing the resource
										// schema.
										Sensitive: true,
									},
									"score": {
										Type:     provider.ResourceDefinitionsSchemaTypeFloat,
										Nullable: true,
									},
									"ipv4": {
										Type: provider.ResourceDefinitionsSchemaTypeBoolean,
									},
									// Deep config exists to test out the max depth logic where changes
									// beyond a certain depth should not be staged.
									// See core.MappingNodeMaxTraverseDepth for the max depth.
									"deepConfig": {
										Type:       provider.ResourceDefinitionsSchemaTypeObject,
										Attributes: createDeepObjectSchema(25, "item"),
										Nullable:   true,
									},
									"metadata": {
										Type: provider.ResourceDefinitionsSchemaTypeMap,
										MapValues: &provider.ResourceDefinitionsSchema{
											Type:        provider.ResourceDefinitionsSchemaTypeString,
											IgnoreDrift: true,
										},
									},
								},
							},
						},
						MustRecreate: true,
					},
					"otherItemConfig": {
						Type: provider.ResourceDefinitionsSchemaTypeUnion,
						OneOf: []*provider.ResourceDefinitionsSchema{
							{
								Type: provider.ResourceDefinitionsSchemaTypeString,
							},
							{
								Type: provider.ResourceDefinitionsSchemaTypeMap,
								MapValues: &provider.ResourceDefinitionsSchema{
									Type: provider.ResourceDefinitionsSchemaTypeObject,
									Attributes: map[string]*provider.ResourceDefinitionsSchema{
										"value1": {
											Type:        provider.ResourceDefinitionsSchemaTypeString,
											IgnoreDrift: true,
										},
										"value2": {
											Type: provider.ResourceDefinitionsSchemaTypeString,
										},
									},
								},
							},
						},
						Default: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"default": {
									Fields: map[string]*core.MappingNode{
										"value1": {
											Scalar: &core.ScalarValue{
												StringValue: &defaultOtherItemConfigValue1,
											},
										},
										"value2": {
											Scalar: &core.ScalarValue{
												StringValue: &defaultOtherItemConfigValue2,
											},
										},
									},
								},
							},
						},
					},
					"vendorTags": {
						Type: provider.ResourceDefinitionsSchemaTypeUnion,
						OneOf: []*provider.ResourceDefinitionsSchema{
							{
								Type: provider.ResourceDefinitionsSchemaTypeString,
							},
							{
								Type: provider.ResourceDefinitionsSchemaTypeArray,
								Items: &provider.ResourceDefinitionsSchema{
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
							},
						},
					},
					"vendorConfig": {
						Type: provider.ResourceDefinitionsSchemaTypeUnion,
						OneOf: []*provider.ResourceDefinitionsSchema{
							{
								Type: provider.ResourceDefinitionsSchemaTypeString,
							},
							{
								Type: provider.ResourceDefinitionsSchemaTypeArray,
								Items: &provider.ResourceDefinitionsSchema{
									Type: provider.ResourceDefinitionsSchemaTypeObject,
									Attributes: map[string]*provider.ResourceDefinitionsSchema{
										"vendorNamespace": {
											Type: provider.ResourceDefinitionsSchemaTypeString,
										},
										"vendorId": {
											Type: provider.ResourceDefinitionsSchemaTypeUnion,
											OneOf: []*provider.ResourceDefinitionsSchema{
												{
													Type: provider.ResourceDefinitionsSchemaTypeString,
												},
												{
													Type: provider.ResourceDefinitionsSchemaTypeInteger,
												},
											},
											IgnoreDrift: true,
											Default: &core.MappingNode{
												Scalar: &core.ScalarValue{
													StringValue: &defaultVendorId,
												},
											},
										},
									},
								},
							},
						},
						Nullable: true,
					},
					"computedFieldTrackDrift": {
						Type:       provider.ResourceDefinitionsSchemaTypeString,
						Computed:   true,
						TrackDrift: true,
					},
				},
			},
		},
	}, nil
}

func createDeepObjectSchema(depth int, fieldPrefix string) map[string]*provider.ResourceDefinitionsSchema {
	if depth == 0 {
		return nil
	}

	fieldName := fmt.Sprintf("%s%d", fieldPrefix, depth)
	return map[string]*provider.ResourceDefinitionsSchema{
		fieldName: {
			Type:       provider.ResourceDefinitionsSchemaTypeObject,
			Attributes: createDeepObjectSchema(depth-1, fieldPrefix),
		},
	}
}

func (r *ExampleComplexResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{}, nil
}

func (r *ExampleComplexResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *ExampleComplexResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	if r.ExternalState == nil && r.FallbackToStateContainerForExternalState {
		resource, err := r.StateContainer.Resources().Get(
			ctx,
			input.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResourceGetExternalStateOutput{
			ResourceSpecState: resource.SpecData,
		}, nil
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: r.ExternalState,
	}, nil
}

func (r *ExampleComplexResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

// IAMRoleResource is a stub implementation of a resource that represents an IAM role.
// This has been prepared primarily to be used in tests for intermediary resource
// deployment as a part of a link plugin implementation.
type IAMRoleResource struct{}

func (r *IAMRoleResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{},
	}, nil
}

func (r *IAMRoleResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (r *IAMRoleResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *IAMRoleResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "aws/iam/role",
	}, nil
}

func (r *IAMRoleResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "",
		MarkdownDescription:  "# AWS IAM Role\n\nAn IAM role for managing access to resources in AWS.",
	}, nil
}

func (r *IAMRoleResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *IAMRoleResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *IAMRoleResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
				},
			},
		},
	}, nil
}

func (r *IAMRoleResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	id := fmt.Sprintf(
		"arn:aws:iam:us-east-1:123456789012:role:%s",
		input.Changes.AppliedResourceInfo.ResourceName,
	)

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString(id),
		},
	}, nil
}

func (r *IAMRoleResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *IAMRoleResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: nil,
	}, nil
}

func (r *IAMRoleResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}
