package plugintestutils

import (
	"errors"
	"reflect"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ResourceTestRunnerSuite struct {
	suite.Suite

	providerCtx provider.Context
	configStore *mockConfigStore
}

func (s *ResourceTestRunnerSuite) SetupTest() {
	s.providerCtx = NewTestProviderContext(
		"testProvider",
		map[string]*core.ScalarValue{
			"configKey1": core.ScalarFromString("configValue1"),
			"configKey2": core.ScalarFromString("configValue2"),
		},
		map[string]*core.ScalarValue{},
	)
	s.configStore = &mockConfigStore{
		config: &mockConfig{},
	}
}

func (s *ResourceTestRunnerSuite) Test_get_external_state_suite_runner() {

	testCases := []ResourceGetExternalStateTestCase[*mockConfig, *mockService]{
		{
			Name: "gets external state for resource",
			ServiceFactory: newMockServiceFactory(
				withGetMockResourceOutput(
					&getMockResourceOutput{
						ID:           "test-id",
						Name:         "test-name",
						DebugMode:    true,
						Param1:       4032,
						Param2:       "test-param2",
						Param3:       403.23029,
						InlineCode:   "test-inline-code",
						CodeLocation: "s3://test-bucket/test-key",
						Tags: []mockTag{
							{
								CustomKey:   "test-key",
								CustomValue: "test-value",
							},
							{
								CustomKey:   "test-key2",
								CustomValue: "test-value2",
							},
							{
								CustomKey:   "test-key3",
								CustomValue: "test-value3",
							},
						},
					},
				),
			),
			ConfigStore: s.configStore,
			Input: &provider.ResourceGetExternalStateInput{
				ProviderContext: s.providerCtx,
			},
			CheckTags: true,
			TagObjectFieldNames: &TagFieldNames{
				KeyFieldName:   "customKey",
				ValueFieldName: "customValue",
			},
			ExpectedOutput: &provider.ResourceGetExternalStateOutput{
				ResourceSpecState: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"id":           core.MappingNodeFromString("test-id"),
						"name":         core.MappingNodeFromString("test-name"),
						"debugMode":    core.MappingNodeFromBool(true),
						"param1":       core.MappingNodeFromInt(4032),
						"param2":       core.MappingNodeFromString("test-param2"),
						"param3":       core.MappingNodeFromFloat(403.23029),
						"inlineCode":   core.MappingNodeFromString("test-inline-code"),
						"codeLocation": core.MappingNodeFromString("s3://test-bucket/test-key"),
						"tags": {
							Items: []*core.MappingNode{
								{
									Fields: map[string]*core.MappingNode{
										"customKey":   core.MappingNodeFromString("test-key"),
										"customValue": core.MappingNodeFromString("test-value"),
									},
								},
								{
									Fields: map[string]*core.MappingNode{
										"customKey":   core.MappingNodeFromString("test-key2"),
										"customValue": core.MappingNodeFromString("test-value2"),
									},
								},
								{
									Fields: map[string]*core.MappingNode{
										"customKey":   core.MappingNodeFromString("test-key3"),
										"customValue": core.MappingNodeFromString("test-value3"),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "returns error when service fails",
			ServiceFactory: newMockServiceFactory(
				withGetMockResourceError(
					errors.New("service error"),
				),
			),
			ConfigStore: s.configStore,
			Input: &provider.ResourceGetExternalStateInput{
				ProviderContext: s.providerCtx,
			},
			ExpectError: true,
		},
	}

	RunResourceGetExternalStateTestCases(
		testCases,
		newMockResource,
		&s.Suite,
	)
}

func (s *ResourceTestRunnerSuite) Test_has_stabilised_suite_runner() {

	testCases := []ResourceHasStabilisedTestCase[*mockConfig, *mockService]{
		{
			Name: "returns true when resource has stabilised",
			ServiceFactory: newMockServiceFactory(
				withGetMockResourceOutput(
					&getMockResourceOutput{
						Status: "active",
					},
				),
			),
			ConfigStore: s.configStore,
			Input: &provider.ResourceHasStabilisedInput{
				ProviderContext: s.providerCtx,
			},
			ExpectedOutput: &provider.ResourceHasStabilisedOutput{
				Stabilised: true,
			},
		},
		{
			Name: "returns false when resource has not stabilised",
			ServiceFactory: newMockServiceFactory(
				withGetMockResourceOutput(
					&getMockResourceOutput{
						Status: "pending",
					},
				),
			),
			ConfigStore: s.configStore,
			Input: &provider.ResourceHasStabilisedInput{
				ProviderContext: s.providerCtx,
			},
			ExpectedOutput: &provider.ResourceHasStabilisedOutput{
				Stabilised: false,
			},
		},
		{
			Name: "returns error when service fails",
			ServiceFactory: newMockServiceFactory(
				withGetMockResourceError(
					errors.New("service error"),
				),
			),
			ConfigStore: s.configStore,
			Input: &provider.ResourceHasStabilisedInput{
				ProviderContext: s.providerCtx,
			},
			ExpectError: true,
		},
	}

	RunResourceHasStabilisedTestCases(
		testCases,
		newMockResource,
		&s.Suite,
	)
}

func (s *ResourceTestRunnerSuite) Test_deploy_suite_runner() {

	testCases := []ResourceDeployTestCase[*mockConfig, *mockService]{
		s.createMockResourceUpdateTestCase(),
		s.createMockResourceSaveNewTestCase(),
		s.createMockResourceSaveRealTestCase(),
		s.createMockResourceDeployErrorTestCase(),
	}

	RunResourceDeployTestCases(
		testCases,
		newMockResource,
		&s.Suite,
	)
}

func (s *ResourceTestRunnerSuite) Test_destroy_suite_runner() {
	testCases := []ResourceDestroyTestCase[*mockConfig, *mockService]{
		s.createMockResourceDestroyTestCase(),
		s.createMockResourceDestroyRealTestCase(),
		s.createMockResourceDestroyErrorTestCase(),
	}

	RunResourceDestroyTestCases(
		testCases,
		newMockResource,
		&s.Suite,
	)
}

func (s *ResourceTestRunnerSuite) createMockResourceDestroyTestCase() ResourceDestroyTestCase[*mockConfig, *mockService] {
	service := newMockService(
		withDeleteMockResourceCodeOutput(
			&deleteMockResourceCodeOutput{
				ID: "test-id",
			},
		),
		withDeleteMockResourceConfigOutput(
			&deleteMockResourceConfigOutput{
				ID: "test-id",
			},
		),
	)

	return ResourceDestroyTestCase[*mockConfig, *mockService]{
		Name: "destroys resource successfully",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		ServiceMockCalls: &service.MockCalls,
		ConfigStore:      s.configStore,
		Input: &provider.ResourceDestroyInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			ResourceState: &state.ResourceState{
				ResourceID: "test-resource-id",
				Name:       "TestResource",
				InstanceID: "test-instance-id",
				// SpecData is not used by the mock resource implementation
				// a pre-determined stub is returned by the underlying mock service.
			},
		},
		DestroyActionsCalled: map[string]any{
			"DeleteResourceConfig": &deleteMockResourceConfigInput{},
			"DeleteResourceCode":   &deleteMockResourceCodeInput{},
		},
	}
}

func (s *ResourceTestRunnerSuite) createMockResourceDestroyRealTestCase() ResourceDestroyTestCase[*mockConfig, *mockService] {
	// This test case provides an example of using the `ResourceDestroyTestCase` helper
	// for integration testing with a real service.
	// The assertion will be that no errors occur.
	// A separate call to the third party service will usually be made
	// to verify that the resource has been destroyed, this is not
	// covered by the `ResourceDestroyTestCase` helper.

	service := newMockService(
		withDeleteMockResourceCodeOutput(
			&deleteMockResourceCodeOutput{
				ID: "test-id",
			},
		),
		withDeleteMockResourceConfigOutput(
			&deleteMockResourceConfigOutput{
				ID: "test-id",
			},
		),
	)

	return ResourceDestroyTestCase[*mockConfig, *mockService]{
		Name: "destroys resource successfully",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		ServiceMockCalls: &service.MockCalls,
		ConfigStore:      s.configStore,
		Input: &provider.ResourceDestroyInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			ResourceState: &state.ResourceState{
				ResourceID: "test-resource-id",
				Name:       "TestResource",
				InstanceID: "test-instance-id",
				// SpecData is not used by the mock resource implementation
				// a pre-determined stub is returned by the underlying mock service.
			},
		},
		ExpectError: false,
	}
}

func (s *ResourceTestRunnerSuite) createMockResourceDestroyErrorTestCase() ResourceDestroyTestCase[*mockConfig, *mockService] {
	service := newMockService(
		withDeleteMockResourceConfigError(
			errors.New("failed to delete resource config"),
		),
	)

	return ResourceDestroyTestCase[*mockConfig, *mockService]{
		Name: "fails to destroy resource when service returns error",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		ServiceMockCalls: &service.MockCalls,
		ConfigStore:      s.configStore,
		Input: &provider.ResourceDestroyInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			ResourceState: &state.ResourceState{
				ResourceID: "test-resource-id",
				Name:       "TestResource",
				InstanceID: "test-instance-id",
				// SpecData is not used by the mock resource implementation
				// a pre-determined stub is returned by the underlying mock service.
			},
		},
		ExpectError: true,
	}
}

func (s *ResourceTestRunnerSuite) createMockResourceUpdateTestCase() ResourceDeployTestCase[*mockConfig, *mockService] {
	service := newMockService(
		withUpdateMockResourceConfigOutput(
			&updateMockResourceConfigOutput{
				ID: "test-id",
				updateMockResourceConfigInput: updateMockResourceConfigInput{
					Name:      "test-resource",
					DebugMode: true,
					Param1:    4032,
					Param2:    "test-param2",
					Param3:    403.23029,
				},
			},
		),
		withUpdateMockResourceCodeOutput(
			&updateMockResourceCodeOutput{
				ID: "test-id",
				updateMockResourceCodeInput: updateMockResourceCodeInput{
					InlineCode:   "test-inline-code",
					CodeLocation: "s3://test-bucket/test-key",
				},
			},
		),
	)

	return ResourceDeployTestCase[*mockConfig, *mockService]{
		Name: "deploys resource successfully for update to existing resource",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		ServiceMockCalls: &service.MockCalls,
		ConfigStore:      s.configStore,
		Input: &provider.ResourceDeployInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			Changes: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceID:   "test-function-id",
					ResourceName: "TestResource",
					InstanceID:   "test-instance-id",
					CurrentResourceState: &state.ResourceState{
						ResourceID: "test-resource-id",
						Name:       "TestResource",
						InstanceID: "test-instance-id",
						// SpecData is not used by the mock resource implementation
						// a pre-determined stub is returned by the underlying mock service.
					},
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Type: &schema.ResourceTypeWrapper{
							Value: "aws/lambda/function",
						},
						// Spec is not used by the mock resource implementation
						// a pre-determined stub is returned by the underlying mock service.
					},
				},
				ModifiedFields: []provider.FieldChange{},
			},
		},
		ExpectedOutput: &provider.ResourceDeployOutput{
			ComputedFieldValues: map[string]*core.MappingNode{
				"spec.id": core.MappingNodeFromString("test-id"),
			},
		},
		SaveActionsCalled: map[string]any{
			"UpdateConfig": []any{
				&updateMockResourceConfigInput{},
				func(input any) bool {
					updateInput, ok := input.(*updateMockResourceConfigInput)
					if !ok {
						return false
					}
					return reflect.DeepEqual(
						updateInput,
						&updateMockResourceConfigInput{
							Name:      "Second Update Name",
							DebugMode: true,
						})
				},
			},
			"UpdateCode": &updateMockResourceCodeInput{},
		},
		SaveActionsNotCalled: []string{"SaveResource"},
	}
}

func (s *ResourceTestRunnerSuite) createMockResourceSaveNewTestCase() ResourceDeployTestCase[*mockConfig, *mockService] {
	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID:           "new-resource-id",
				Name:         "new-resource",
				DebugMode:    true,
				Param1:       4032,
				Param2:       "new-param2",
				Param3:       403.23029,
				InlineCode:   "new-inline-code",
				CodeLocation: "s3://new-bucket/new-key",
			},
		),
	)

	return ResourceDeployTestCase[*mockConfig, *mockService]{
		Name: "deploys resource successfully for creation of new resource",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		ServiceMockCalls: &service.MockCalls,
		ConfigStore:      s.configStore,
		Input: &provider.ResourceDeployInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			Changes: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceID:   "test-function-id",
					ResourceName: "TestResource",
					InstanceID:   "test-instance-id",
					// No current resource state for a new resource.
					CurrentResourceState: nil,
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Type: &schema.ResourceTypeWrapper{
							Value: "aws/lambda/function",
						},
						// Spec is not used by the mock resource implementation
						// a pre-determined stub is returned by the underlying mock service.
					},
				},
				ModifiedFields: []provider.FieldChange{},
			},
		},
		ExpectedOutputMatcher: func(actual *provider.ResourceDeployOutput) (EqualityCheckValues, error) {
			return EqualityCheckValues{
				Expected: &provider.ResourceDeployOutput{
					ComputedFieldValues: map[string]*core.MappingNode{
						"spec.id": core.MappingNodeFromString("new-resource-id"),
					},
				},
				Actual: actual,
			}, nil
		},
		SaveActionsCalled: map[string]any{
			"SaveResource": &saveMockResourceInput{},
		},
		SaveActionsNotCalled: []string{"UpdateConfig", "UpdateCode"},
	}
}

func (s *ResourceTestRunnerSuite) createMockResourceSaveRealTestCase() ResourceDeployTestCase[*mockConfig, *mockService] {
	// This test case provides an example of using the `ResourceDeployTestCase` helper
	// for integration testing with a real service.
	// The assertion will be purely on outputs and no mock calls are expected.

	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID:           "new-resource-id",
				Name:         "new-resource",
				DebugMode:    true,
				Param1:       4032,
				Param2:       "new-param2",
				Param3:       403.23029,
				InlineCode:   "new-inline-code",
				CodeLocation: "s3://new-bucket/new-key",
			},
		),
	)

	return ResourceDeployTestCase[*mockConfig, *mockService]{
		Name: "deploys resource successfully for creation of new resource",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		// No service mock calls for a real service.
		ConfigStore: s.configStore,
		Input: &provider.ResourceDeployInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			Changes: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceID:   "test-function-id",
					ResourceName: "TestResource",
					InstanceID:   "test-instance-id",
					// No current resource state for a new resource.
					CurrentResourceState: nil,
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Type: &schema.ResourceTypeWrapper{
							Value: "aws/lambda/function",
						},
						// Spec is not used by the mock resource implementation
						// a pre-determined stub is returned by the underlying mock service.
					},
				},
				ModifiedFields: []provider.FieldChange{},
			},
		},
		ExpectedOutputMatcher: func(actual *provider.ResourceDeployOutput) (EqualityCheckValues, error) {
			return EqualityCheckValues{
				Expected: &provider.ResourceDeployOutput{
					ComputedFieldValues: map[string]*core.MappingNode{
						"spec.id": core.MappingNodeFromString("new-resource-id"),
					},
				},
				Actual: actual,
			}, nil
		},
	}
}

func (s *ResourceTestRunnerSuite) createMockResourceDeployErrorTestCase() ResourceDeployTestCase[*mockConfig, *mockService] {
	service := newMockService(
		withSaveMockResourceError(
			errors.New("failed to save resource"),
		),
	)

	return ResourceDeployTestCase[*mockConfig, *mockService]{
		Name: "fails to deploy resource when service returns error",
		ServiceFactory: func(serviceConfig *mockConfig, providerContext provider.Context) *mockService {
			return service
		},
		ServiceMockCalls: &service.MockCalls,
		ConfigStore:      s.configStore,
		Input: &provider.ResourceDeployInput{
			ProviderContext: s.providerCtx,
			InstanceID:      "test-instance-id",
			ResourceID:      "test-resource-id",
			Changes: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceID:   "test-function-id",
					ResourceName: "TestResource",
					InstanceID:   "test-instance-id",
					// No current resource state for a new resource.
					CurrentResourceState: nil,
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Type: &schema.ResourceTypeWrapper{
							Value: "aws/lambda/function",
						},
						// Spec is not used by the mock resource implementation
						// a pre-determined stub is returned by the underlying mock service.
					},
				},
				ModifiedFields: []provider.FieldChange{},
			},
		},
		ExpectError: true,
	}
}

func TestResourceTestUtilsSuite(t *testing.T) {
	suite.Run(t, new(ResourceTestRunnerSuite))
}
