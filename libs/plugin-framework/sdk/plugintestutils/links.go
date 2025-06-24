package plugintestutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/stretchr/testify/suite"
)

// LinkChangeStagingTestCase defines a test case for link change staging.
type LinkChangeStagingTestCase[
	ResourceAServiceConfig any,
	ResourceAService any,
	ResourceBServiceConfig any,
	ResourceBService any,
] struct {
	// The name of the test case used for errors and debugging.
	Name string
	// ServiceFactoryA is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context
	// for the first resource (resource A).
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactoryA pluginutils.ServiceFactory[ResourceAServiceConfig, ResourceAService]
	// ServiceFactoryB is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context
	// for the second resource (resource B).
	//
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for
	// AWS services and the `Service` type parameter could be a `lambda.Service`
	// interface that implements API methods for the AWS Lambda service.
	ServiceFactoryB pluginutils.ServiceFactory[ResourceBServiceConfig, ResourceBService]
	// ConfigStoreA is a store that generates service-specific configuration
	// for the service factory to create an instance of the service
	// for the first resource (resource A).
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if your service factory requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStoreA pluginutils.ServiceConfigStore[ResourceAServiceConfig]
	// ConfigStoreB is a store that generates service-specific configuration
	// for the service factory to create an instance of the service
	// for the second resource (resource B).
	ConfigStoreB pluginutils.ServiceConfigStore[ResourceBServiceConfig]
	// Input for the link stage changing operation.
	Input *provider.LinkStageChangesInput
	// Expected output from the link stage changing operation.
	ExpectedOutput *provider.LinkStageChangesOutput
	// If true, the test case expects an error to be returned.
	ExpectError bool
	// The expected error message if an error is expected.
	// This doesn't have to be an exact match, the error message
	// just needs to contain the provided string.
	ExpectedErrorMessage string
}

func RunLinkChangeStagingTestCases[
	ResourceAServiceConfig any,
	ResourceAService any,
	ResourceBServiceConfig any,
	ResourceBService any,
](
	testCases []LinkChangeStagingTestCase[
		ResourceAServiceConfig,
		ResourceAService,
		ResourceBServiceConfig,
		ResourceBService,
	],
	createLink func(
		linkServiceDeps pluginutils.LinkServiceDeps[
			ResourceAServiceConfig,
			ResourceAService,
			ResourceBServiceConfig,
			ResourceBService,
		],
	) provider.Link,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			link := createLink(
				pluginutils.LinkServiceDeps[
					ResourceAServiceConfig,
					ResourceAService,
					ResourceBServiceConfig,
					ResourceBService,
				]{
					ResourceAService: pluginutils.ServiceWithConfigStore[
						ResourceAServiceConfig,
						ResourceAService,
					]{
						ServiceFactory: tc.ServiceFactoryA,
						ConfigStore:    tc.ConfigStoreA,
					},
					ResourceBService: pluginutils.ServiceWithConfigStore[
						ResourceBServiceConfig,
						ResourceBService,
					]{
						ServiceFactory: tc.ServiceFactoryB,
						ConfigStore:    tc.ConfigStoreB,
					},
				},
			)

			changes, err := link.StageChanges(context.Background(), tc.Input)
			if tc.ExpectError {
				testSuite.ErrorContains(err, tc.ExpectedErrorMessage)
			} else {
				testSuite.NoError(err)
				testSuite.Equal(tc.ExpectedOutput, changes)
			}
		})
	}
}

// LinkUpdateResourceTestCase defines a test case for link resource update operations.
type LinkUpdateResourceTestCase[
	ResourceAServiceConfig any,
	ResourceAService any,
	ResourceBServiceConfig any,
	ResourceBService any,
] struct {
	// The name of the test case used for errors and debugging.
	Name string
	// ServiceFactoryA is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context
	// for the first resource (resource A).
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactoryA pluginutils.ServiceFactory[ResourceAServiceConfig, ResourceAService]
	// ServiceFactoryB is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context
	// for the second resource (resource B).
	//
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for
	// AWS services and the `Service` type parameter could be a `lambda.Service`
	// interface that implements API methods for the AWS Lambda service.
	ServiceFactoryB pluginutils.ServiceFactory[ResourceBServiceConfig, ResourceBService]
	// ConfigStoreA is a store that generates service-specific configuration
	// for the service factory to create an instance of the service
	// for the first resource (resource A).
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if the service factory for resource A requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStoreA pluginutils.ServiceConfigStore[ResourceAServiceConfig]
	// ConfigStoreB is a store that generates service-specific configuration
	// for the service factory to create an instance of the service
	// for the second resource (resource B).
	// This is only needed if the service factory for resource B requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStoreB pluginutils.ServiceConfigStore[ResourceBServiceConfig]
	// CurrentServiceMockCalls is a mock calls tracker that is expected to be embedded
	// into a mock implementation of the service interface for carrying out
	// the update operation via the provider service APIs for the resource
	// under test.
	CurrentServiceMockCalls *MockCalls
	// Input for the link resource update operation.
	Input *provider.LinkUpdateResourceInput
	// Expected output from the link resource update operation.
	ExpectedOutput *provider.LinkUpdateResourceOutput
	// Resource defines the resource in the link relationship that should be updated.
	Resource LinkUpdateResource
	// UpdateActionsCalled is a mapping of method name to the
	// expected second argument for the method.
	// When the value is a slice of any, it is expected that the method
	// is called multiple times with different arguments in the provided order.
	// This will usually be something like a `*Input` or `*Request` struct
	// that service library functions take after a context argument.
	UpdateActionsCalled map[string]any
	// UpdateActionsNotCalled is a list of method names
	// that are not expected to be called as a part
	// of the update operation.
	UpdateActionsNotCalled []string
	// If true, the test case expects an error to be returned.
	ExpectError bool
	// The expected error message if an error is expected.
	// This doesn't have to be an exact match, the error message
	// just needs to contain the provided string.
	ExpectedErrorMessage string
}

// LinkUpdateResource is a type that represents a resource
// that should be updated in a link update operation.
type LinkUpdateResource string

const (
	// LinkUpdateResourceA is used to select "resource A" in a link
	// update operation for testing purposes.
	LinkUpdateResourceA LinkUpdateResource = "resourceA"
	// LinkUpdateResourceB is used to select "resource B" in a link
	// update operation for testing purposes.
	LinkUpdateResourceB LinkUpdateResource = "resourceB"
)

// RunLinkUpdateResourceTestCases runs a set of test cases for the `UpdateResource(A|B)` methods
// of a link implementation in a provider plugin.
func RunLinkUpdateResourceTestCases[
	ResourceAServiceConfig any,
	ResourceAService any,
	ResourceBServiceConfig any,
	ResourceBService any,
](
	testCases []LinkUpdateResourceTestCase[
		ResourceAServiceConfig,
		ResourceAService,
		ResourceBServiceConfig,
		ResourceBService,
	],
	createLink func(
		linkServiceDeps pluginutils.LinkServiceDeps[
			ResourceAServiceConfig,
			ResourceAService,
			ResourceBServiceConfig,
			ResourceBService,
		],
	) provider.Link,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			link := createLink(
				pluginutils.LinkServiceDeps[
					ResourceAServiceConfig,
					ResourceAService,
					ResourceBServiceConfig,
					ResourceBService,
				]{
					ResourceAService: pluginutils.ServiceWithConfigStore[
						ResourceAServiceConfig,
						ResourceAService,
					]{
						ServiceFactory: tc.ServiceFactoryA,
						ConfigStore:    tc.ConfigStoreA,
					},
					ResourceBService: pluginutils.ServiceWithConfigStore[
						ResourceBServiceConfig,
						ResourceBService,
					]{
						ServiceFactory: tc.ServiceFactoryB,
						ConfigStore:    tc.ConfigStoreB,
					},
				},
			)

			var output *provider.LinkUpdateResourceOutput
			var err error
			switch tc.Resource {
			case LinkUpdateResourceA:
				output, err = link.UpdateResourceA(
					context.Background(),
					tc.Input,
				)
			case LinkUpdateResourceB:
				output, err = link.UpdateResourceB(
					context.Background(),
					tc.Input,
				)
			default:
				testSuite.Failf("Invalid link resource", "Resource %s is not supported", tc.Resource)
				return
			}

			if tc.ExpectError {
				testSuite.Error(err)
				testSuite.ErrorContains(err, tc.ExpectedErrorMessage)
				return
			}

			testSuite.NoError(err)
			testSuite.Equal(tc.ExpectedOutput, output)

			assertActionsCalled(testSuite, tc.CurrentServiceMockCalls, tc.UpdateActionsCalled)
			assertActionsNotCalled(testSuite, tc.CurrentServiceMockCalls, tc.UpdateActionsNotCalled)
		})
	}
}

// LinkUpdateIntermediaryResourcesTestCase defines a test case for
// updating intermediary resources in a link.
type LinkUpdateIntermediaryResourcesTestCase[
	ResourceAServiceConfig any,
	ResourceAService any,
	ResourceBServiceConfig any,
	ResourceBService any,
] struct {
	// The name of the test case used for errors and debugging.
	Name string
	// ServiceFactoryA is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context
	// for the first resource (resource A).
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	//
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for AWS services
	// and the `Service` type parameter could be a `lambda.Service` interface that implements
	// API methods for the AWS Lambda service.
	// The `ServiceConfigStore` type parameter would be an optional store
	ServiceFactoryA pluginutils.ServiceFactory[ResourceAServiceConfig, ResourceAService]
	// ServiceFactoryB is a function that creates an instance of the service
	// using the provided service-specific configuration and a provider context
	// for the second resource (resource B).
	//
	// serviceConfig can be nil for services that do not take a specific configuration
	// to create client instances.
	// For example the `ServiceConfig` type parameter would be `*aws.Config` for
	// AWS services and the `Service` type parameter could be a `lambda.Service`
	// interface that implements API methods for the AWS Lambda service.
	ServiceFactoryB pluginutils.ServiceFactory[ResourceBServiceConfig, ResourceBService]
	// ConfigStoreA is a store that generates service-specific configuration
	// for the service factory to create an instance of the service
	// for the first resource (resource A).
	// This could be something like a cache for `*aws.Config` for AWS services
	// based on a session ID.
	// This is only needed if the service factory for resource A requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStoreA pluginutils.ServiceConfigStore[ResourceAServiceConfig]
	// ConfigStoreB is a store that generates service-specific configuration
	// for the service factory to create an instance of the service
	// for the second resource (resource B).
	// This is only needed if the service factory for resource B requires a specific
	// configuration struct to be able to create an instance of the service.
	ConfigStoreB pluginutils.ServiceConfigStore[ResourceBServiceConfig]
	// IntermediariesServiceMockCalls is a mock calls tracker that is expected to be embedded
	// into a mock implementation of the service interface for carrying out
	// the update operation via the provider service APIs for intermediary resources.
	IntermediariesServiceMockCalls *MockCalls
	// Input for the link intermediary resources update operation.
	Input *provider.LinkUpdateIntermediaryResourcesInput
	// Expected output from the link intermediary resources update operation.
	ExpectedOutput *provider.LinkUpdateIntermediaryResourcesOutput
	// UpdateActionsCalled is a mapping of method name to the
	// expected second argument for the method.
	// When the value is a slice of any, it is expected that the method
	// is called multiple times with different arguments in the provided order.
	// This will usually be something like a `*Input` or `*Request` struct
	// that service library functions take after a context argument.
	UpdateActionsCalled map[string]any
	// UpdateActionsNotCalled is a list of method names
	// that are not expected to be called as a part
	// of the update operation.
	UpdateActionsNotCalled []string
	// If true, the test case expects an error to be returned.
	ExpectError bool
	// The expected error message if an error is expected.
	// This doesn't have to be an exact match, the error message
	// just needs to contain the provided string.
	ExpectedErrorMessage string
}

// RunLinkUpdateIntermediaryResourcesTestCases runs a set of test cases
// for the `UpdateIntermediaryResources“ methods of a link implementation
// in a provider plugin.
func RunLinkUpdateIntermediaryResourcesTestCases[
	ResourceAServiceConfig any,
	ResourceAService any,
	ResourceBServiceConfig any,
	ResourceBService any,
](
	testCases []LinkUpdateIntermediaryResourcesTestCase[
		ResourceAServiceConfig,
		ResourceAService,
		ResourceBServiceConfig,
		ResourceBService,
	],
	createLink func(
		linkServiceDeps pluginutils.LinkServiceDeps[
			ResourceAServiceConfig,
			ResourceAService,
			ResourceBServiceConfig,
			ResourceBService,
		],
	) provider.Link,
	testSuite *suite.Suite,
) {
	for _, tc := range testCases {
		testSuite.Run(tc.Name, func() {
			link := createLink(
				pluginutils.LinkServiceDeps[
					ResourceAServiceConfig,
					ResourceAService,
					ResourceBServiceConfig,
					ResourceBService,
				]{
					ResourceAService: pluginutils.ServiceWithConfigStore[
						ResourceAServiceConfig,
						ResourceAService,
					]{
						ServiceFactory: tc.ServiceFactoryA,
						ConfigStore:    tc.ConfigStoreA,
					},
					ResourceBService: pluginutils.ServiceWithConfigStore[
						ResourceBServiceConfig,
						ResourceBService,
					]{
						ServiceFactory: tc.ServiceFactoryB,
						ConfigStore:    tc.ConfigStoreB,
					},
				},
			)

			output, err := link.UpdateIntermediaryResources(
				context.Background(),
				tc.Input,
			)
			if tc.ExpectError {
				testSuite.Error(err)
				testSuite.ErrorContains(err, tc.ExpectedErrorMessage)
				return
			}

			testSuite.NoError(err)
			testSuite.Equal(tc.ExpectedOutput, output)

			assertActionsCalled(testSuite, tc.IntermediariesServiceMockCalls, tc.UpdateActionsCalled)
			assertActionsNotCalled(testSuite, tc.IntermediariesServiceMockCalls, tc.UpdateActionsNotCalled)
		})
	}
}
