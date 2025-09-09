package plugintestutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/stretchr/testify/suite"
)

// ResourceGetExternalStateTestCase defines a test case for `GetExternalState` method
// of a resource implementation in a provider plugin.
type ResourceGetExternalStateTestCase[ServiceConfig any, Service any] struct {
	// Name is the name of the test case that will be used when running a test suite.
	Name string
	// ServiceFactory is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context.
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactory pluginutils.ServiceFactory[ServiceConfig, Service]
	// ConfigStore is a store that generates service-specific configuration
	// for the service factory to create an instance of the service.
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if your service factory requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStore pluginutils.ServiceConfigStore[ServiceConfig]
	// Input is the input passed into the `GetExternalState` method.
	Input *provider.ResourceGetExternalStateInput
	// ExpectedOutput is the expected output from the `GetExternalState` method.
	ExpectedOutput *provider.ResourceGetExternalStateOutput
	// ExpectedOutputMatcher is a function that takes the actual output
	// from the `GetExternalState` method and returns a prepared actual
	// value along with the expected value to be used in an equality check.
	ExpectedOutputMatcher func(
		actual *provider.ResourceGetExternalStateOutput,
	) (EqualityCheckValues, error)
	// CheckTags should be set to true if the subject resource holds tags
	// as an array of key-value pair objects.
	// The default behaviour is to not check tags.
	CheckTags bool
	// TagsFieldName is the name of the field in the resource spec state
	// that holds the tags as an array of key-value pair objects.
	// This field is used only if CheckTags is `true`.
	// The default value is "tags".
	TagsFieldName string
	// TagObjectFieldNames is used to specify the field names for keys and values
	// for tags in the resource spec state.
	// The default field names are "key" and "value", this will only
	// be used if CheckTags is `true`.
	TagObjectFieldNames *TagFieldNames
	// ExpectError indicates whether the test case expects an error
	// to be returned from the `GetExternalState` method.
	ExpectError bool
}

// RunResourceGetExternalStateTestCases runs a set of test cases for the `GetExternalState` method
// of a resource implementation in a provider plugin.
func RunResourceGetExternalStateTestCases[ServiceConfig any, Service any](
	testCases []ResourceGetExternalStateTestCase[ServiceConfig, Service],
	createResource func(
		serviceFactory pluginutils.ServiceFactory[ServiceConfig, Service],
		configStore pluginutils.ServiceConfigStore[ServiceConfig],
	) provider.Resource,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			resource := createResource(
				tc.ServiceFactory,
				tc.ConfigStore,
			)

			output, err := resource.GetExternalState(context.Background(), tc.Input)
			if tc.ExpectError {
				testSuite.Error(err)
				return
			}

			testSuite.NoError(err)

			// Special handling for tags: compare as sets (order-independent).
			if tc.CheckTags {
				tagsFieldName := tc.TagsFieldName
				if tagsFieldName == "" {
					tagsFieldName = "tags"
				}

				expectedTags := tc.ExpectedOutput.ResourceSpecState.Fields[tagsFieldName].Items
				actualTags := output.ResourceSpecState.Fields[tagsFieldName].Items
				CompareTags(testSuite.T(), expectedTags, actualTags, tc.TagObjectFieldNames)

				testSuite.Equal(
					pluginutils.ShallowCopy(
						tc.ExpectedOutput.ResourceSpecState.Fields,
						tagsFieldName,
					),
					pluginutils.ShallowCopy(
						output.ResourceSpecState.Fields,
						tagsFieldName,
					),
				)
			} else {
				assertExpectedOutput(
					tc.ExpectedOutput,
					tc.ExpectedOutputMatcher,
					output,
					testSuite,
				)
			}
		})
	}
}

func assertExpectedOutput[Output any](
	expectedOutput Output,
	expectedOutputMatcher func(actual Output) (EqualityCheckValues, error),
	actualOutput Output,
	testSuite *suite.Suite,
) {
	if expectedOutputMatcher == nil {
		testSuite.Equal(expectedOutput, actualOutput)
		return
	}

	checkValues, err := expectedOutputMatcher(actualOutput)
	if err != nil {
		testSuite.Fail(
			"Expected output matcher failed",
			"Error: %v",
			err,
		)
		return
	}
	testSuite.Equal(
		checkValues.Expected,
		checkValues.Actual,
		"Expected output does not match actual output",
	)
}

// ResourceHasStabilisedTestCase defines a test case for `HasStabilised` method
// of a resource implementation in a provider plugin.
type ResourceHasStabilisedTestCase[ServiceConfig any, Service any] struct {
	// Name is the name of the test case that will be used when running a test suite.
	Name string
	// ServiceFactory is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context.
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactory pluginutils.ServiceFactory[ServiceConfig, Service]
	// ConfigStore is a store that generates service-specific configuration
	// for the service factory to create an instance of the service.
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if your service factory requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStore pluginutils.ServiceConfigStore[ServiceConfig]
	// Input is the input passed into the `HasStabilised` method.
	Input *provider.ResourceHasStabilisedInput
	// ExpectedOutput is the expected output from the `HasStabilised` method.
	ExpectedOutput *provider.ResourceHasStabilisedOutput
	// ExpectedOutputMatcher is a function that takes the actual output
	// from the `HasStabilised` method and returns a prepared actual
	// value along with the expected value to be used in an equality check.
	ExpectedOutputMatcher func(
		actual *provider.ResourceHasStabilisedOutput,
	) (EqualityCheckValues, error)
	// ExpectError indicates whether the test case expects an error
	// to be returned from the `GetExternalState` method.
	ExpectError bool
}

// RunResourceHasStabilisedTestCases runs a set of test cases for the `HasStabilised` method
// of a resource implementation in a provider plugin.
func RunResourceHasStabilisedTestCases[ServiceConfig any, Service any](
	testCases []ResourceHasStabilisedTestCase[ServiceConfig, Service],
	createResource func(
		serviceFactory pluginutils.ServiceFactory[ServiceConfig, Service],
		configStore pluginutils.ServiceConfigStore[ServiceConfig],
	) provider.Resource,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			resource := createResource(
				tc.ServiceFactory,
				tc.ConfigStore,
			)

			output, err := resource.HasStabilised(context.Background(), tc.Input)
			if tc.ExpectError {
				testSuite.Error(err)
				return
			}

			testSuite.NoError(err)
			assertExpectedOutput(
				tc.ExpectedOutput,
				tc.ExpectedOutputMatcher,
				output,
				testSuite,
			)
		})
	}
}

// ResourceDeployTestCase defines a test case for the `Deploy` method
// of a resource implementation in a provider plugin.
// This is used to test both creation and update operations
// which may be two separate methods depending on the provider plugin implementation.
//
// For resources defined with the `providerv1.ResourceDefinition` helper struct,
// this test case will be used to test both `Create` and `Update` methods
// as the `Deploy` method in the helper struct will determine the appropriate
// action to take based on the deploy input.
type ResourceDeployTestCase[ServiceConfig any, Service any] struct {
	// Name is the name of the test case that will be used when running a test suite.
	Name string
	// ServiceFactory is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context.
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactory pluginutils.ServiceFactory[ServiceConfig, Service]
	// ServiceMockCalls is a mock calls tracker that is expected to be embedded
	// into a mock implementation of the service interface for carrying out
	// the save operation via the provider service APIs.
	//
	// This is optional and only needs to be set if the test case
	// uses mocks instead of a real service.
	ServiceMockCalls *MockCalls
	// ConfigStore is a store that generates service-specific configuration
	// for the service factory to create an instance of the service.
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if your service factory requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStore pluginutils.ServiceConfigStore[ServiceConfig]
	// Input is passed into the `Deploy` method of the resource implementation.
	Input *provider.ResourceDeployInput
	// SaveActionsCalled is a mapping of method name to the
	// expected second argument for the method.
	// When the value is a slice of any, it is expected that the method
	// is called multiple times with different arguments in the provided order.
	// This will usually be something like a `*Input` or `*Request` struct
	// that service library functions take after a context argument.
	//
	// This should only be set if the case uses mocks instead of a real service
	// (i.e. when `ServiceMockCalls` is set).
	SaveActionsCalled map[string]any
	// SaveActionsNotCalled is a list of method names
	// that are not expected to be called as a part
	// of the save operation.
	//
	// This should only be set if the case uses mocks instead of a real service
	// (i.e. when `ServiceMockCalls` is set).
	SaveActionsNotCalled []string
	// ExpectedOutput is the expected output from the `Deploy` method.
	ExpectedOutput *provider.ResourceDeployOutput
	// ExpectedOutputMatcher is a function that takes the actual output
	// from the `Deploy` method and returns a prepared actual
	// value along with the expected value to be used in an equality check.
	ExpectedOutputMatcher func(
		actual *provider.ResourceDeployOutput,
	) (EqualityCheckValues, error)
	// ExpectError indicates whether the test case expects an error
	// to be returned from the `Deploy` method.
	ExpectError bool
	// Cleanup is a function that is called after the test case has run.
	// This is used to clean up any resources that were created during the test case.
	// This is useful for integration tests with real services where the resource
	// needs to be cleaned up after the test case has run regardless of whether
	// the test case passes or fails.
	Cleanup func(ctx context.Context, output *provider.ResourceDeployOutput) error
}

// RunResourceDeployTestCases runs a set of test cases for the `Deploy` method
// of a resource implementation in a provider plugin.
func RunResourceDeployTestCases[ServiceConfig any, Service any](
	testCases []ResourceDeployTestCase[ServiceConfig, Service],
	createResource func(
		serviceFactory pluginutils.ServiceFactory[ServiceConfig, Service],
		configStore pluginutils.ServiceConfigStore[ServiceConfig],
	) provider.Resource,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			resource := createResource(
				tc.ServiceFactory,
				tc.ConfigStore,
			)

			output, err := resource.Deploy(context.Background(), tc.Input)
			if tc.ExpectError {
				testSuite.Error(err)
				return
			}

			if tc.Cleanup != nil {
				defer func() {
					err := tc.Cleanup(context.Background(), output)
					if err != nil {
						testSuite.Fail(
							"Cleanup function failed",
							"Error: %v",
							err,
						)
					}
				}()
			}

			testSuite.NoError(err)
			assertExpectedOutput(
				tc.ExpectedOutput,
				tc.ExpectedOutputMatcher,
				output,
				testSuite,
			)

			assertActionsCalled(testSuite, tc.ServiceMockCalls, tc.SaveActionsCalled)
			assertActionsNotCalled(testSuite, tc.ServiceMockCalls, tc.SaveActionsNotCalled)
		})
	}
}

// ResourceDestroyTestCase defines a test case for the `Destroy` method
// of a resource implementation in a provider plugin.
// This is used to test both creation and update operations
// which may be two separate methods depending on the provider plugin implementation.
type ResourceDestroyTestCase[ServiceConfig any, Service any] struct {
	// Name is the name of the test case that will be used when running a test suite.
	Name string
	// ServiceFactory is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context.
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactory pluginutils.ServiceFactory[ServiceConfig, Service]
	// ServiceMockCalls is a mock calls tracker that is expected to be embedded
	// into a mock implementation of the service interface for carrying out
	// the destroy operation via the provider service APIs.
	//
	// This is optional and only needs to be set if the test case
	// uses mocks instead of a real service.
	ServiceMockCalls *MockCalls
	// ConfigStore is a store that generates service-specific configuration
	// for the service factory to create an instance of the service.
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if your service factory requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStore pluginutils.ServiceConfigStore[ServiceConfig]
	// Input is passed into the `Destroy` method of the resource implementation.
	Input *provider.ResourceDestroyInput
	// DestroyActionsCalled is a mapping of method name to the
	// expected second argument for the method.
	// When the value is a slice of any, it is expected that the method
	// is called multiple times with different arguments in the provided order.
	// This will usually be something like a `*Input` or `*Request` struct
	// that service library functions take after a context argument.
	//
	// This should only be set if the case uses mocks instead of a real service
	// (i.e. when `ServiceMockCalls` is set).
	DestroyActionsCalled map[string]any
	// SaveActionsNotCalled is a list of method names
	// that are not expected to be called as a part
	// of the save operation.
	//
	// This should only be set if the case uses mocks instead of a real service
	// (i.e. when `ServiceMockCalls` is set).
	DestroyActionsNotCalled []string
	// ExpectError indicates whether the test case expects an error
	// to be returned from the `Destroy` method.
	ExpectError bool
}

// RunResourceDestroyTestCases runs a set of test cases for the `Destroy` method
// of a resource implementation in a provider plugin.
func RunResourceDestroyTestCases[ServiceConfig any, Service any](
	testCases []ResourceDestroyTestCase[ServiceConfig, Service],
	createResource func(
		serviceFactory pluginutils.ServiceFactory[ServiceConfig, Service],
		configStore pluginutils.ServiceConfigStore[ServiceConfig],
	) provider.Resource,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			resource := createResource(
				tc.ServiceFactory,
				tc.ConfigStore,
			)

			err := resource.Destroy(context.Background(), tc.Input)
			if tc.ExpectError {
				testSuite.Error(err)
				return
			}

			testSuite.NoError(err)

			assertActionsCalled(testSuite, tc.ServiceMockCalls, tc.DestroyActionsCalled)
			assertActionsNotCalled(testSuite, tc.ServiceMockCalls, tc.DestroyActionsNotCalled)
		})
	}
}

func assertActionsCalled(
	s *suite.Suite,
	serviceMockCalls *MockCalls,
	expected map[string]any,
) {
	if serviceMockCalls == nil {
		return
	}

	for methodName, expectedInput := range expected {
		if expectedSlice, ok := expectedInput.([]any); ok {
			// []any indicates that the method is expected to be called multiple times
			// with different inputs, in the given order.
			for i, input := range expectedSlice {
				serviceMockCalls.AssertCalledWith(s, methodName, i, Any, input)
			}
			return
		}

		serviceMockCalls.AssertCalledWith(s, methodName, 0, Any, expectedInput)
	}
}

func assertActionsNotCalled(
	s *suite.Suite,
	serviceMockCalls *MockCalls,
	notCalled []string,
) {
	if serviceMockCalls == nil {
		return
	}

	for _, methodName := range notCalled {
		serviceMockCalls.AssertNotCalled(s, methodName)
	}
}
