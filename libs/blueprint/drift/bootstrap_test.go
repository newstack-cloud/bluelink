package drift

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

func newTestAWSProvider(
	dynamoDBTableExternalState *core.MappingNode,
	lambdaFunctionExternalState *core.MappingNode,
) provider.Provider {
	return &internal.ProviderMock{
		NamespaceValue: "aws",
		Resources: map[string]provider.Resource{
			"aws/dynamodb/table": &internal.DynamoDBTableResource{
				ExternalState: dynamoDBTableExternalState,
			},
			"aws/lambda/function": &internal.LambdaFunctionResource{
				CurrentDestroyAttempts:         map[string]int{},
				CurrentDeployAttemps:           map[string]int{},
				CurrentGetExternalStateAttemps: map[string]int{},
				FailResourceIDs:                []string{},
				StabiliseResourceIDs:           map[string]*internal.StubResourceStabilisationConfig{},
				CurrentStabiliseCalls:          map[string]int{},
				SkipRetryFailuresForInstances:  []string{},
				ExternalState:                  lambdaFunctionExternalState,
			},
		},
		Links:               map[string]provider.Link{},
		CustomVariableTypes: map[string]provider.CustomVariableType{},
		DataSources:         map[string]provider.DataSource{},
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

func newTestExampleProvider(
	exampleComplexResourceExternalState *core.MappingNode,
) provider.Provider {
	return &internal.ProviderMock{
		NamespaceValue: "example",
		Resources: map[string]provider.Resource{
			"example/complexResource": &internal.ExampleComplexResource{
				ExternalState: exampleComplexResourceExternalState,
			},
		},
		Links:               map[string]provider.Link{},
		CustomVariableTypes: map[string]provider.CustomVariableType{},
		DataSources:         map[string]provider.DataSource{},
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
