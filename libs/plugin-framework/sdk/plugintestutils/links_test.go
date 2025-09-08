package plugintestutils

import (
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type LinkTestRunnerSuite struct {
	suite.Suite

	providerCtx provider.Context
	configStore *mockConfigStore
}

func (s *LinkTestRunnerSuite) SetupTest() {
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

func (s *LinkTestRunnerSuite) Test_stage_changes_suite_runner() {
	service := newMockService()
	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	testCases := []LinkChangeStagingTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		{
			Name:            "Stage Changes A",
			Input:           &provider.LinkStageChangesInput{},
			ServiceFactoryA: serviceFactory,
			ConfigStoreA:    s.configStore,
			ServiceFactoryB: serviceFactory,
			ConfigStoreB:    s.configStore,
			ExpectedOutput: &provider.LinkStageChangesOutput{
				Changes: &provider.LinkChanges{
					ModifiedFields: []*provider.FieldChange{
						{
							FieldPath: "testField",
							NewValue:  core.MappingNodeFromString("newValue"),
						},
					},
				},
			},
		},
	}

	RunLinkChangeStagingTestCases(
		testCases,
		newMockLink,
		&s.Suite,
	)
}

func (s *LinkTestRunnerSuite) Test_update_link_resource_suite_runner() {
	testCases := []LinkUpdateResourceTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		s.createMockLinkUpdateResourceATestCase(),
		s.createMockLinkUpdateResourceRealTestCase(),
		s.createMockLinkUpdateResourceBTestCase(),
		s.createMockLinkUpdateErrorTestCase(),
	}

	RunLinkUpdateResourceTestCases(
		testCases,
		newMockLink,
		&s.Suite,
	)
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateResourceATestCase() LinkUpdateResourceTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID: "new-resource-id",
			},
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateResourceTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:                    "Update Resource A",
		Input:                   &provider.LinkUpdateResourceInput{},
		Resource:                LinkUpdateResourceA,
		ServiceFactoryA:         serviceFactory,
		ConfigStoreA:            s.configStore,
		ServiceFactoryB:         serviceFactory,
		ConfigStoreB:            s.configStore,
		CurrentServiceMockCalls: &service.MockCalls,
		ExpectedOutputMatcher: func(actual *provider.LinkUpdateResourceOutput) (EqualityCheckValues, error) {
			return EqualityCheckValues{
				Expected: &provider.LinkUpdateResourceOutput{
					LinkData: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"resourceAId": core.MappingNodeFromString("new-resource-id"),
						},
					},
				},
				Actual: actual,
			}, nil
		},
		UpdateActionsCalled: map[string]any{
			"SaveResource": &saveMockResourceInput{},
		},
	}
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateResourceRealTestCase() LinkUpdateResourceTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	// This test case provides an example of using the `LinkUpdateResourceTestCase` helper
	// for integration testing with a real service.
	// The assertion will be purely on outputs and no mock calls are expected.

	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID: "new-resource-id",
			},
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateResourceTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:            "Update Resource A",
		Input:           &provider.LinkUpdateResourceInput{},
		Resource:        LinkUpdateResourceA,
		ServiceFactoryA: serviceFactory,
		ConfigStoreA:    s.configStore,
		ServiceFactoryB: serviceFactory,
		ConfigStoreB:    s.configStore,
		// No service mock calls for a real service.
		ExpectedOutputMatcher: func(actual *provider.LinkUpdateResourceOutput) (EqualityCheckValues, error) {
			return EqualityCheckValues{
				Expected: &provider.LinkUpdateResourceOutput{
					LinkData: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"resourceAId": core.MappingNodeFromString("new-resource-id"),
						},
					},
				},
				Actual: actual,
			}, nil
		},
	}
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateResourceBTestCase() LinkUpdateResourceTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID: "new-resource-id-b",
			},
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateResourceTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:                    "Update Resource B",
		Input:                   &provider.LinkUpdateResourceInput{},
		Resource:                LinkUpdateResourceB,
		ServiceFactoryA:         serviceFactory,
		ConfigStoreA:            s.configStore,
		ServiceFactoryB:         serviceFactory,
		ConfigStoreB:            s.configStore,
		CurrentServiceMockCalls: &service.MockCalls,
		ExpectedOutput: &provider.LinkUpdateResourceOutput{
			LinkData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"resourceBId": core.MappingNodeFromString("new-resource-id-b"),
				},
			},
		},
		UpdateActionsCalled: map[string]any{
			"SaveResource": &saveMockResourceInput{},
		},
	}
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateErrorTestCase() LinkUpdateResourceTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	service := newMockService(
		withSaveMockResourceError(
			errors.New("Failed to save resource"),
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateResourceTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:                    "Update Resource Error",
		Input:                   &provider.LinkUpdateResourceInput{},
		Resource:                LinkUpdateResourceA,
		ServiceFactoryA:         serviceFactory,
		ConfigStoreA:            s.configStore,
		ServiceFactoryB:         serviceFactory,
		ConfigStoreB:            s.configStore,
		CurrentServiceMockCalls: &service.MockCalls,
		ExpectError:             true,
		ExpectedErrorMessage:    "Failed to save resource",
	}
}

func (s *LinkTestRunnerSuite) Test_update_link_intermediary_resources_suite_runner() {
	testCases := []LinkUpdateIntermediaryResourcesTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		s.createMockLinkUpdateIntermediaryResourcesTestCase(),
		s.createMockLinkUpdateIntermediaryResourcesRealTestCase(),
		s.createMockLinkUpdateIntermediaryResourcesErrorTestCase(),
	}

	RunLinkUpdateIntermediaryResourcesTestCases(
		testCases,
		newMockLink,
		&s.Suite,
	)
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateIntermediaryResourcesTestCase() LinkUpdateIntermediaryResourcesTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID: "intermediary-resource-id",
			},
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateIntermediaryResourcesTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:                           "Update Intermediary Resources",
		Input:                          &provider.LinkUpdateIntermediaryResourcesInput{},
		ServiceFactoryA:                serviceFactory,
		ConfigStoreA:                   s.configStore,
		ServiceFactoryB:                serviceFactory,
		ConfigStoreB:                   s.configStore,
		IntermediariesServiceMockCalls: &service.MockCalls,
		ExpectedOutput: &provider.LinkUpdateIntermediaryResourcesOutput{
			IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
				{
					ResourceID: "intermediary-resource-id",
				},
			},
			LinkData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"intermediaryResourceId": core.MappingNodeFromString("intermediary-resource-id"),
				},
			},
		},
		UpdateActionsCalled: map[string]any{
			"SaveResource": &saveMockResourceInput{},
		},
	}
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateIntermediaryResourcesRealTestCase() LinkUpdateIntermediaryResourcesTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	// This test case provides an example of using the `LinkUpdateIntermediaryResourcesTestCase` helper
	// for integration testing with a real service.
	// The assertion will be purely on outputs and no mock calls are expected.

	service := newMockService(
		withSaveMockResourceOutput(
			&saveMockResourceOutput{
				ID: "intermediary-resource-id",
			},
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateIntermediaryResourcesTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:            "Update Intermediary Resources",
		Input:           &provider.LinkUpdateIntermediaryResourcesInput{},
		ServiceFactoryA: serviceFactory,
		ConfigStoreA:    s.configStore,
		ServiceFactoryB: serviceFactory,
		ConfigStoreB:    s.configStore,
		// No service mock calls for a real service.
		ExpectedOutput: &provider.LinkUpdateIntermediaryResourcesOutput{
			IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
				{
					ResourceID: "intermediary-resource-id",
				},
			},
			LinkData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"intermediaryResourceId": core.MappingNodeFromString("intermediary-resource-id"),
				},
			},
		},
	}
}

func (s *LinkTestRunnerSuite) createMockLinkUpdateIntermediaryResourcesErrorTestCase() LinkUpdateIntermediaryResourcesTestCase[
	*mockConfig,
	*mockService,
	*mockConfig,
	*mockService,
] {
	service := newMockService(
		withSaveMockResourceError(
			errors.New("Failed to save intermediary resource"),
		),
	)

	serviceFactory := func(_ *mockConfig, _ provider.Context) *mockService {
		return service
	}

	return LinkUpdateIntermediaryResourcesTestCase[
		*mockConfig,
		*mockService,
		*mockConfig,
		*mockService,
	]{
		Name:                           "Update Intermediary Resources Error",
		Input:                          &provider.LinkUpdateIntermediaryResourcesInput{},
		ServiceFactoryA:                serviceFactory,
		ConfigStoreA:                   s.configStore,
		ServiceFactoryB:                serviceFactory,
		ConfigStoreB:                   s.configStore,
		IntermediariesServiceMockCalls: &service.MockCalls,
		ExpectError:                    true,
		ExpectedErrorMessage:           "Failed to save intermediary resource",
	}
}

func TestLinkTestRunnerSuite(t *testing.T) {
	suite.Run(t, new(LinkTestRunnerSuite))
}
