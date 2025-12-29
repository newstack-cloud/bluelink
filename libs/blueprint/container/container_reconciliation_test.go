package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/drift"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

const (
	testReconciliationInstanceID   = "test-reconciliation-instance"
	testReconciliationInstanceName = "TestReconciliationInstance"
)

type ContainerReconciliationTestSuite struct {
	suite.Suite
	stateContainer state.Container
	driftChecker   *mockDriftChecker
	container      *defaultBlueprintContainer
}

func (s *ContainerReconciliationTestSuite) SetupTest() {
	s.stateContainer = memstate.NewMemoryStateContainer()
	s.driftChecker = &mockDriftChecker{
		checkInterruptedResults: []drift.ReconcileResult{},
		checkDriftResults:       map[string]*state.ResourceDriftState{},
	}

	// Create a minimal container with just the dependencies needed for reconciliation
	s.container = &defaultBlueprintContainer{
		stateContainer: s.stateContainer,
		driftChecker:   s.driftChecker,
		clock:          core.SystemClock{},
		logger:         core.NewNopLogger(),
	}
}

func (s *ContainerReconciliationTestSuite) populateTestState(
	resources map[string]*state.ResourceState,
	links map[string]*state.LinkState,
) error {
	// Create instance
	err := s.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:   testReconciliationInstanceID,
			InstanceName: testReconciliationInstanceName,
			Status:       core.InstanceStatusUpdated,
			Resources:    resources,
			Links:        links,
		},
	)
	if err != nil {
		return err
	}

	// Save individual resources
	for _, r := range resources {
		err = s.stateContainer.Resources().Save(context.Background(), *r)
		if err != nil {
			return err
		}
	}

	// Save individual links
	for _, l := range links {
		err = s.stateContainer.Links().Save(context.Background(), *l)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_returns_error_when_input_is_nil() {
	_, err := s.container.CheckReconciliation(
		context.Background(),
		nil,
		nil,
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "input is required")
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_returns_error_when_instance_id_is_empty() {
	_, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: "",
		},
		nil,
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "instance ID is required")
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_returns_empty_when_no_interrupted_resources() {
	// Setup state with no interrupted resources
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		nil,
	)
	s.Require().NoError(err)

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(testReconciliationInstanceID, result.InstanceID)
	s.Empty(result.Resources)
	s.Empty(result.Links)
	s.False(result.HasInterrupted)
	s.False(result.HasDrift)
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_returns_interrupted_resources() {
	testValue := "test-value"
	// Setup drift checker to return interrupted resource results
	s.driftChecker.checkInterruptedResults = []drift.ReconcileResult{
		{
			ResourceID:   "resource-1",
			ResourceName: "testResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
			ExternalState: &core.MappingNode{
				Scalar: &core.ScalarValue{StringValue: &testValue},
			},
		},
	}

	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
	)
	s.Require().NoError(err)

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Len(result.Resources, 1)
	s.True(result.HasInterrupted)
	s.False(result.HasDrift)

	resource := result.Resources[0]
	s.Equal("resource-1", resource.ResourceID)
	s.Equal("testResource1", resource.ResourceName)
	s.Equal(ReconciliationTypeInterrupted, resource.Type)
	s.Equal(core.PreciseResourceStatusCreateInterrupted, resource.OldStatus)
	s.Equal(core.PreciseResourceStatusCreated, resource.NewStatus)
	s.True(resource.ResourceExists)
	s.Equal(ReconciliationActionAcceptExternal, resource.RecommendedAction)
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_returns_interrupted_links() {
	// Setup state with interrupted link
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreating,
				PreciseStatus: core.PreciseLinkStatusResourceAUpdateInterrupted,
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Len(result.Links, 1)
	s.True(result.HasInterrupted)

	link := result.Links[0]
	s.Equal("link-1", link.LinkID)
	s.Equal("resourceA::resourceB", link.LinkName)
	s.Equal(ReconciliationTypeInterrupted, link.Type)
	s.Equal(core.PreciseLinkStatusResourceAUpdateInterrupted, link.OldStatus)
	// Since both resources are in Created state, link should be marked as succeeded
	s.Equal(core.PreciseLinkStatusResourceAUpdated, link.NewStatus)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_returns_error_when_input_is_nil() {
	_, err := s.container.ApplyReconciliation(
		context.Background(),
		nil,
		nil,
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "input is required")
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_returns_error_when_instance_id_is_empty() {
	_, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: "",
		},
		nil,
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "instance ID is required")
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_requires_external_state_for_accept_external() {
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "resource-1",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: nil, // Missing external state
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().Len(result.Errors, 1)
	s.Contains(result.Errors[0].Error, "external state is required")
	s.Equal("resource-1", result.Errors[0].ElementID)
	s.Equal("resource", result.Errors[0].ElementType)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_updates_resource_status() {
	// Setup state with interrupted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID: "resource-1",
					Action:     ReconciliationActionUpdateStatus,
					NewStatus:  core.PreciseResourceStatusCreated,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the resource state was updated
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "resource-1")
	s.Require().NoError(err)
	s.Equal(core.ResourceStatusCreated, resourceState.Status)
	s.Equal(core.PreciseResourceStatusCreated, resourceState.PreciseStatus)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_accepts_external_state() {
	oldValue := "old-value"
	// Setup state with interrupted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
				SpecData: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &oldValue},
				},
			},
		},
		nil,
	)
	s.Require().NoError(err)

	newValue := "new-external-value"
	externalState := &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &newValue},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "resource-1",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: externalState,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the resource state was updated with external state
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "resource-1")
	s.Require().NoError(err)
	s.Equal(core.ResourceStatusCreated, resourceState.Status)
	s.Equal(core.PreciseResourceStatusCreated, resourceState.PreciseStatus)
	s.Equal("new-external-value", *resourceState.SpecData.Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_marks_resource_failed() {
	// Setup state with interrupted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID: "resource-1",
					Action:     ReconciliationActionManualCleanupRequired,
					NewStatus:  core.PreciseResourceStatusCreateFailed,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the resource state was updated
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "resource-1")
	s.Require().NoError(err)
	s.Equal(core.ResourceStatusCreateFailed, resourceState.Status)
	s.Equal(core.PreciseResourceStatusCreateFailed, resourceState.PreciseStatus)
	s.NotEmpty(resourceState.FailureReasons)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_updates_link_status() {
	// Setup state with interrupted link
	err := s.populateTestState(
		map[string]*state.ResourceState{},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreating,
				PreciseStatus: core.PreciseLinkStatusResourceAUpdateInterrupted,
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionUpdateStatus,
					NewStatus: core.PreciseLinkStatusResourceAUpdated,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)
	s.Empty(result.Errors)

	// Verify the link state was updated
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Equal(core.LinkStatusCreated, linkState.Status)
	s.Equal(core.PreciseLinkStatusResourceAUpdated, linkState.PreciseStatus)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_updates_link_data() {
	oldValue := "old-handler-value"
	newValue := "new-handler-value"

	// Setup state with link that has existing Data
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				Drifted:       true,
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldValue}},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionAcceptExternal,
					NewStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
					LinkDataUpdates: map[string]*core.MappingNode{
						"resourceA.handler": {Scalar: &core.ScalarValue{StringValue: &newValue}},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)
	s.Empty(result.Errors)

	// Verify the link state was updated with new Data values
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Equal(core.LinkStatusCreated, linkState.Status)
	s.Equal(core.PreciseLinkStatusIntermediaryResourcesUpdated, linkState.PreciseStatus)
	s.False(linkState.Drifted) // Drifted should be cleared

	// Verify the Data was updated
	s.Require().NotNil(linkState.Data)
	s.Require().NotNil(linkState.Data["resourceA"])
	s.Require().NotNil(linkState.Data["resourceA"].Fields["handler"])
	s.Equal(newValue, *linkState.Data["resourceA"].Fields["handler"].Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_updates_intermediary_state() {
	oldValue := "old-value"
	// Setup state with link that has intermediary resources
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
					{
						ResourceID:    "intermediary-1",
						ResourceType:  "test/intermediary",
						InstanceID:    testReconciliationInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
						ResourceSpecData: &core.MappingNode{
							Scalar: &core.ScalarValue{StringValue: &oldValue},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	newValue := "new-external-value"
	externalState := &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &newValue},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionUpdateStatus,
					NewStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
					IntermediaryActions: map[string]*IntermediaryReconcileAction{
						"intermediary-1": {
							IntermediaryID: "intermediary-1",
							Action:         ReconciliationActionAcceptExternal,
							ExternalState:  externalState,
							NewStatus:      core.PreciseResourceStatusUpdated,
						},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)
	s.Empty(result.Errors)

	// Verify the link state was updated with intermediary changes
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Require().Len(linkState.IntermediaryResourceStates, 1)
	s.Equal(core.ResourceStatusUpdated, linkState.IntermediaryResourceStates[0].Status)
	s.Equal(core.PreciseResourceStatusUpdated, linkState.IntermediaryResourceStates[0].PreciseStatus)
	s.Equal("new-external-value", *linkState.IntermediaryResourceStates[0].ResourceSpecData.Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_marks_intermediary_failed() {
	oldValue := "old-value"
	// Setup state with link that has intermediary resources
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
					{
						ResourceID:    "intermediary-1",
						ResourceType:  "test/intermediary",
						InstanceID:    testReconciliationInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
						ResourceSpecData: &core.MappingNode{
							Scalar: &core.ScalarValue{StringValue: &oldValue},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionUpdateStatus,
					NewStatus: core.PreciseLinkStatusIntermediaryResourceUpdateFailed,
					IntermediaryActions: map[string]*IntermediaryReconcileAction{
						"intermediary-1": {
							IntermediaryID: "intermediary-1",
							Action:         ReconciliationActionManualCleanupRequired,
							NewStatus:      core.PreciseResourceStatusCreateFailed,
						},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)
	s.Empty(result.Errors)

	// Verify the link and intermediary states were updated
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Equal(core.LinkStatusCreateFailed, linkState.Status)
	s.Equal(core.PreciseLinkStatusIntermediaryResourceUpdateFailed, linkState.PreciseStatus)
	s.Require().Len(linkState.IntermediaryResourceStates, 1)
	s.Equal(core.ResourceStatusCreateFailed, linkState.IntermediaryResourceStates[0].Status)
	s.Equal(core.PreciseResourceStatusCreateFailed, linkState.IntermediaryResourceStates[0].PreciseStatus)
	s.NotEmpty(linkState.IntermediaryResourceStates[0].FailureReasons)
}

func (s *ContainerReconciliationTestSuite) Test_apply_resource_reconciliation_updates_affected_link_data() {
	oldHandlerValue := "old-handler-arn"
	newHandlerValue := "new-external-handler-arn"

	// Setup state with:
	// 1. A resource that has drifted
	// 2. A link that has ResourceDataMappings pointing to that resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"handler": {Scalar: &core.ScalarValue{StringValue: &oldHandlerValue}},
					},
				},
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				// Link has stored the old handler value
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldHandlerValue}},
						},
					},
				},
				// ResourceDataMappings: resource field path -> link data path
				ResourceDataMappings: map[string]string{
					"resourceA::handler": "resourceA.handler",
				},
			},
		},
	)
	s.Require().NoError(err)

	// Apply resource reconciliation with AcceptExternal to update SpecData
	externalState := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"handler": {Scalar: &core.ScalarValue{StringValue: &newHandlerValue}},
		},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "resource-a",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: externalState,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the resource state was updated
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "resource-a")
	s.Require().NoError(err)
	s.Equal(newHandlerValue, *resourceState.SpecData.Fields["handler"].Scalar.StringValue)

	// Verify the link.Data was ALSO updated to maintain consistency
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Require().NotNil(linkState.Data)
	s.Require().NotNil(linkState.Data["resourceA"])
	s.Require().NotNil(linkState.Data["resourceA"].Fields["handler"])
	s.Equal(
		newHandlerValue,
		*linkState.Data["resourceA"].Fields["handler"].Scalar.StringValue,
		"link.Data should be updated with external value from ResourceDataMappings",
	)
}

func (s *ContainerReconciliationTestSuite) Test_apply_resource_reconciliation_updates_multiple_link_data_paths() {
	oldHandler := "old-handler"
	oldTimeout := "30"
	newHandler := "new-handler"
	newTimeout := "60"

	// Setup state with resource and link that has multiple ResourceDataMappings
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"handler": {Scalar: &core.ScalarValue{StringValue: &oldHandler}},
						"config": {
							Fields: map[string]*core.MappingNode{
								"timeout": {Scalar: &core.ScalarValue{StringValue: &oldTimeout}},
							},
						},
					},
				},
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldHandler}},
							"timeout": {Scalar: &core.ScalarValue{StringValue: &oldTimeout}},
						},
					},
				},
				// Multiple mappings from resource fields to link data
				ResourceDataMappings: map[string]string{
					"resourceA::handler":        "resourceA.handler",
					"resourceA::config.timeout": "resourceA.timeout",
				},
			},
		},
	)
	s.Require().NoError(err)

	// Apply resource reconciliation with new external state
	externalState := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"handler": {Scalar: &core.ScalarValue{StringValue: &newHandler}},
			"config": {
				Fields: map[string]*core.MappingNode{
					"timeout": {Scalar: &core.ScalarValue{StringValue: &newTimeout}},
				},
			},
		},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "resource-a",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: externalState,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify both link.Data paths were updated
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Require().NotNil(linkState.Data["resourceA"])
	s.Equal(newHandler, *linkState.Data["resourceA"].Fields["handler"].Scalar.StringValue)
	s.Equal(newTimeout, *linkState.Data["resourceA"].Fields["timeout"].Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_clears_resource_drifted_flag() {
	oldValue := "old-value"
	driftTimestamp := 1234567890
	// Setup state with drifted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:                 "resource-1",
				Name:                       "testResource1",
				Type:                       "test/resource",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.ResourceStatusCreated,
				PreciseStatus:              core.PreciseResourceStatusCreated,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				SpecData: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &oldValue},
				},
			},
		},
		nil,
	)
	s.Require().NoError(err)

	newValue := "new-external-value"
	externalState := &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &newValue},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "resource-1",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: externalState,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the resource state was updated and Drifted flag was cleared
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "resource-1")
	s.Require().NoError(err)
	s.False(resourceState.Drifted, "Drifted flag should be cleared after accepting external state")
	s.Nil(resourceState.LastDriftDetectedTimestamp, "LastDriftDetectedTimestamp should be cleared")
	s.Equal("new-external-value", *resourceState.SpecData.Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_removes_resource_drift_state() {
	oldValue := "old-value"
	newValue := "new-external-value"
	driftTimestamp := 1234567890

	// Setup state with drifted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:                 "resource-1",
				Name:                       "testResource1",
				Type:                       "test/resource",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.ResourceStatusCreated,
				PreciseStatus:              core.PreciseResourceStatusCreated,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				SpecData: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &oldValue},
				},
			},
		},
		nil,
	)
	s.Require().NoError(err)

	// Also save drift state to the state container
	err = s.stateContainer.Resources().SaveDrift(context.Background(), state.ResourceDriftState{
		ResourceID:   "resource-1",
		ResourceName: "testResource1",
		SpecData: &core.MappingNode{
			Scalar: &core.ScalarValue{StringValue: &newValue},
		},
		Difference: &state.ResourceDriftChanges{
			ModifiedFields: []*state.ResourceDriftFieldChange{
				{
					FieldPath:    "value",
					StateValue:   &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldValue}},
					DriftedValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newValue}},
				},
			},
		},
		Timestamp: &driftTimestamp,
	})
	s.Require().NoError(err)

	// Verify drift state exists before reconciliation
	driftState, err := s.stateContainer.Resources().GetDrift(context.Background(), "resource-1")
	s.Require().NoError(err)
	s.Equal("resource-1", driftState.ResourceID)

	// Apply reconciliation
	externalState := &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &newValue},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "resource-1",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: externalState,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)

	// Verify drift state was removed - memstate returns empty struct without error
	driftState, err = s.stateContainer.Resources().GetDrift(context.Background(), "resource-1")
	s.Require().NoError(err)
	s.Empty(driftState.ResourceID, "drift state should be removed after reconciliation")
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_removes_link_drift_state() {
	oldValue := "old-handler"
	newValue := "new-handler"
	driftTimestamp := 1234567890

	// Setup state with drifted link
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:                     "link-1",
				Name:                       "resourceA::resourceB",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.LinkStatusCreated,
				PreciseStatus:              core.PreciseLinkStatusIntermediaryResourcesUpdated,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldValue}},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Also save link drift state to the state container
	err = s.stateContainer.Links().SaveDrift(context.Background(), state.LinkDriftState{
		LinkID:   "link-1",
		LinkName: "resourceA::resourceB",
		ResourceADrift: &state.LinkResourceDrift{
			ResourceID:   "resource-a",
			ResourceName: "resourceA",
			MappedFieldChanges: []*state.LinkDriftFieldChange{
				{
					ResourceFieldPath: "resourceA::handler",
					LinkDataPath:      "resourceA.handler",
					LinkDataValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldValue}},
					ExternalValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newValue}},
				},
			},
		},
		Timestamp: &driftTimestamp,
	})
	s.Require().NoError(err)

	// Verify link drift state exists before reconciliation
	linkDriftState, err := s.stateContainer.Links().GetDrift(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Equal("link-1", linkDriftState.LinkID)

	// Apply reconciliation
	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionAcceptExternal,
					NewStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
					LinkDataUpdates: map[string]*core.MappingNode{
						"resourceA.handler": {Scalar: &core.ScalarValue{StringValue: &newValue}},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)

	// Verify link drift state was removed - memstate returns empty struct without error
	linkDriftState, err = s.stateContainer.Links().GetDrift(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Empty(linkDriftState.LinkID, "link drift state should be removed after reconciliation")

	// Also verify the link Drifted flag was cleared
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.False(linkState.Drifted, "Drifted flag should be cleared after reconciliation")
	s.Nil(linkState.LastDriftDetectedTimestamp, "LastDriftDetectedTimestamp should be cleared")
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_populates_external_state_and_changes_for_drifted_resources() {
	persistedValue := "persisted-value"
	externalValue := "external-drifted-value"
	driftTimestamp := 1234567890

	// Setup state with drifted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:                 "resource-1",
				Name:                       "testResource1",
				Type:                       "test/resource",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.ResourceStatusCreated,
				PreciseStatus:              core.PreciseResourceStatusCreated,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"value": {Scalar: &core.ScalarValue{StringValue: &persistedValue}},
					},
				},
			},
		},
		nil,
	)
	s.Require().NoError(err)

	// Configure mock drift checker to return drift results
	// (CheckReconciliation calls driftChecker.CheckDriftWithState, not the state container)
	mockDriftCheckerWithDrift := &mockDriftChecker{
		checkDriftResults: map[string]*state.ResourceDriftState{
			"resource-1": {
				ResourceID:   "resource-1",
				ResourceName: "testResource1",
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"value": {Scalar: &core.ScalarValue{StringValue: &externalValue}},
					},
				},
				Difference: &state.ResourceDriftChanges{
					ModifiedFields: []*state.ResourceDriftFieldChange{
						{
							FieldPath:    "value",
							StateValue:   &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &persistedValue}},
							DriftedValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &externalValue}},
						},
					},
				},
				Timestamp: &driftTimestamp,
			},
		},
	}

	containerWithMockDrift := &defaultBlueprintContainer{
		stateContainer: s.stateContainer,
		driftChecker:   mockDriftCheckerWithDrift,
		clock:          core.SystemClock{},
		logger:         core.NewNopLogger(),
	}

	result, err := containerWithMockDrift.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeAll,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasDrift)
	s.Len(result.Resources, 1)

	resource := result.Resources[0]
	s.Equal("resource-1", resource.ResourceID)
	s.Equal("testResource1", resource.ResourceName)
	s.Equal(ReconciliationTypeDrift, resource.Type)

	// Verify ExternalState is populated with the drifted value
	s.Require().NotNil(resource.ExternalState, "ExternalState should be populated")
	s.Require().NotNil(resource.ExternalState.Fields["value"])
	s.Equal(externalValue, *resource.ExternalState.Fields["value"].Scalar.StringValue)

	// Verify PersistedState is populated with the original persisted value
	s.Require().NotNil(resource.PersistedState, "PersistedState should be populated")
	s.Require().NotNil(resource.PersistedState.Fields["value"])
	s.Equal(persistedValue, *resource.PersistedState.Fields["value"].Scalar.StringValue)

	// Verify Changes is populated with the drift information
	s.Require().NotNil(resource.Changes, "Changes should be populated")
	s.Len(resource.Changes.ModifiedFields, 1)
	s.Equal("value", resource.Changes.ModifiedFields[0].FieldPath)
	s.Equal(persistedValue, *resource.Changes.ModifiedFields[0].PrevValue.Scalar.StringValue)
	s.Equal(externalValue, *resource.Changes.ModifiedFields[0].NewValue.Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_assigns_persisted_state_from_resource() {
	// This test verifies that PersistedState correctly reflects what's in the resource SpecData
	// (the stored state) rather than the external/drifted state from ResourceDriftState.SpecData
	persistedValue := "original-persisted-value"
	externalValue := "drifted-external-value"
	driftTimestamp := 1234567890

	// Setup state with drifted resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:                 "resource-1",
				Name:                       "testResource1",
				Type:                       "test/resource",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.ResourceStatusCreated,
				PreciseStatus:              core.PreciseResourceStatusCreated,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				// This is the PERSISTED state - what we last deployed
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"name": {Scalar: &core.ScalarValue{StringValue: &persistedValue}},
					},
				},
			},
		},
		nil,
	)
	s.Require().NoError(err)

	// Configure mock drift checker to return drift results
	// (CheckReconciliation calls driftChecker.CheckDriftWithState, not the state container)
	// The key point: driftState.SpecData contains EXTERNAL state, resource.SpecData contains PERSISTED state
	mockDriftCheckerWithDrift := &mockDriftChecker{
		checkDriftResults: map[string]*state.ResourceDriftState{
			"resource-1": {
				ResourceID:   "resource-1",
				ResourceName: "testResource1",
				// This is the EXTERNAL state - what we found in the cloud
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"name": {Scalar: &core.ScalarValue{StringValue: &externalValue}},
					},
				},
				Difference: &state.ResourceDriftChanges{
					ModifiedFields: []*state.ResourceDriftFieldChange{
						{
							FieldPath:    "name",
							StateValue:   &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &persistedValue}},
							DriftedValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &externalValue}},
						},
					},
				},
				Timestamp: &driftTimestamp,
			},
		},
	}

	containerWithMockDrift := &defaultBlueprintContainer{
		stateContainer: s.stateContainer,
		driftChecker:   mockDriftCheckerWithDrift,
		clock:          core.SystemClock{},
		logger:         core.NewNopLogger(),
	}

	result, err := containerWithMockDrift.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeAll,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Len(result.Resources, 1)

	resource := result.Resources[0]

	// ExternalState should be the DRIFTED value (from drift state SpecData)
	s.Require().NotNil(resource.ExternalState)
	s.Equal(externalValue, *resource.ExternalState.Fields["name"].Scalar.StringValue,
		"ExternalState should contain the drifted/external value")

	// PersistedState should be the ORIGINAL value (from resource SpecData)
	s.Require().NotNil(resource.PersistedState)
	s.Equal(persistedValue, *resource.PersistedState.Fields["name"].Scalar.StringValue,
		"PersistedState should contain the original persisted value, NOT the external value")

	// They should be different values
	s.NotEqual(
		*resource.PersistedState.Fields["name"].Scalar.StringValue,
		*resource.ExternalState.Fields["name"].Scalar.StringValue,
		"PersistedState and ExternalState should have different values for drifted resource",
	)
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_detects_intermediary_drift() {
	persistedValue := "persisted-value"
	externalValue := "external-drifted-value"

	// Setup state with link that has intermediary resources
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
					{
						ResourceID:    "intermediary-1",
						ResourceType:  "test/intermediary",
						InstanceID:    testReconciliationInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
						ResourceSpecData: &core.MappingNode{
							Scalar: &core.ScalarValue{StringValue: &persistedValue},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Create a mock drift checker that returns the expected link drift state
	// with intermediary drift
	mockDriftCheckerWithLinkDrift := &mockDriftChecker{
		checkLinkDriftState: &state.LinkDriftState{
			LinkID:   "link-1",
			LinkName: "resourceA::resourceB",
			IntermediaryDrift: map[string]*state.IntermediaryDriftState{
				"intermediary-1": {
					ResourceID:   "intermediary-1",
					ResourceType: "test/intermediary",
					PersistedState: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"value": {Scalar: &core.ScalarValue{StringValue: &persistedValue}},
						},
					},
					ExternalState: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"value": {Scalar: &core.ScalarValue{StringValue: &externalValue}},
						},
					},
					Changes: &state.IntermediaryDriftChanges{
						ModifiedFields: []state.IntermediaryFieldChange{
							{
								FieldPath: "value",
								PrevValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &persistedValue}},
								NewValue:  &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &externalValue}},
							},
						},
					},
					Exists: true,
				},
			},
		},
	}

	// Create container with the mock drift checker
	containerWithMockDriftChecker := &defaultBlueprintContainer{
		stateContainer: s.stateContainer,
		driftChecker:   mockDriftCheckerWithLinkDrift,
		clock:          core.SystemClock{},
		logger:         core.NewNopLogger(),
	}

	result, err := containerWithMockDriftChecker.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeAll,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasDrift)
	s.Len(result.Links, 1)

	link := result.Links[0]
	s.Equal("link-1", link.LinkID)
	s.Equal("resourceA::resourceB", link.LinkName)
	s.Equal(ReconciliationTypeDrift, link.Type)
	s.Equal(ReconciliationActionAcceptExternal, link.RecommendedAction)

	// Verify intermediary changes were detected
	s.Require().Len(link.IntermediaryChanges, 1)
	intermediaryResult, ok := link.IntermediaryChanges["intermediary-1"]
	s.Require().True(ok)
	s.Equal("intermediary-1", intermediaryResult.Name)
	s.Equal("test/intermediary", intermediaryResult.Type)
	s.True(intermediaryResult.Exists)
	s.Equal(persistedValue, *intermediaryResult.PersistedState.Fields["value"].Scalar.StringValue)
	s.Equal(externalValue, *intermediaryResult.ExternalState.Fields["value"].Scalar.StringValue)

	// Verify Changes is populated and converted to provider.Changes
	s.Require().NotNil(intermediaryResult.Changes)
	s.Len(intermediaryResult.Changes.ModifiedFields, 1)
	s.Equal("value", intermediaryResult.Changes.ModifiedFields[0].FieldPath)
	s.Equal(persistedValue, *intermediaryResult.Changes.ModifiedFields[0].PrevValue.Scalar.StringValue)
	s.Equal(externalValue, *intermediaryResult.Changes.ModifiedFields[0].NewValue.Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_link_reconciliation_update_status_preserves_drift_state() {
	oldValue := "old-handler"
	driftTimestamp := 1234567890

	// Setup state with drifted link
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:                     "link-1",
				Name:                       "resourceA::resourceB",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.LinkStatusCreating,
				PreciseStatus:              core.PreciseLinkStatusResourceAUpdateInterrupted,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldValue}},
						},
					},
				},
				IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
					{
						ResourceID:   "intermediary-1",
						ResourceType: "test/intermediary",
						InstanceID:   testReconciliationInstanceID,
						Status:       core.ResourceStatusCreated,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Save link drift state
	newValue := "new-external-handler"
	err = s.stateContainer.Links().SaveDrift(context.Background(), state.LinkDriftState{
		LinkID:   "link-1",
		LinkName: "resourceA::resourceB",
		ResourceADrift: &state.LinkResourceDrift{
			ResourceID:   "resource-a",
			ResourceName: "resourceA",
			MappedFieldChanges: []*state.LinkDriftFieldChange{
				{
					ResourceFieldPath: "resourceA::handler",
					LinkDataPath:      "resourceA.handler",
					LinkDataValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldValue}},
					ExternalValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newValue}},
				},
			},
		},
		Timestamp: &driftTimestamp,
	})
	s.Require().NoError(err)

	// Apply UpdateStatus action (NOT AcceptExternal)
	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionUpdateStatus,
					NewStatus: core.PreciseLinkStatusResourceAUpdated,
					// Needs IntermediaryActions to trigger full save path
					IntermediaryActions: map[string]*IntermediaryReconcileAction{
						"intermediary-1": {
							IntermediaryID: "intermediary-1",
							Action:         ReconciliationActionUpdateStatus,
							NewStatus:      core.PreciseResourceStatusCreated,
						},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)

	// Verify link drift state was NOT removed
	linkDriftState, err := s.stateContainer.Links().GetDrift(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Equal("link-1", linkDriftState.LinkID, "drift state should be preserved for UpdateStatus action")

	// Verify link Drifted flag was NOT cleared
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.True(linkState.Drifted, "Drifted flag should be preserved for UpdateStatus action")
	s.NotNil(linkState.LastDriftDetectedTimestamp, "LastDriftDetectedTimestamp should be preserved")
}

func (s *ContainerReconciliationTestSuite) Test_apply_link_reconciliation_manual_cleanup_preserves_drift_state() {
	oldValue := "old-handler"
	driftTimestamp := 1234567890

	// Setup state with drifted link
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:                     "link-1",
				Name:                       "resourceA::resourceB",
				InstanceID:                 testReconciliationInstanceID,
				Status:                     core.LinkStatusCreating,
				PreciseStatus:              core.PreciseLinkStatusResourceAUpdateInterrupted,
				Drifted:                    true,
				LastDriftDetectedTimestamp: &driftTimestamp,
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldValue}},
						},
					},
				},
				IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
					{
						ResourceID:   "intermediary-1",
						ResourceType: "test/intermediary",
						InstanceID:   testReconciliationInstanceID,
						Status:       core.ResourceStatusCreated,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Save link drift state
	newValue := "new-external-handler"
	err = s.stateContainer.Links().SaveDrift(context.Background(), state.LinkDriftState{
		LinkID:   "link-1",
		LinkName: "resourceA::resourceB",
		ResourceADrift: &state.LinkResourceDrift{
			ResourceID:   "resource-a",
			ResourceName: "resourceA",
			MappedFieldChanges: []*state.LinkDriftFieldChange{
				{
					ResourceFieldPath: "resourceA::handler",
					LinkDataPath:      "resourceA.handler",
					LinkDataValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldValue}},
					ExternalValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newValue}},
				},
			},
		},
		Timestamp: &driftTimestamp,
	})
	s.Require().NoError(err)

	// Apply ManualCleanupRequired action
	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionManualCleanupRequired,
					NewStatus: core.PreciseLinkStatusResourceAUpdateFailed,
					// Needs IntermediaryActions to trigger full save path
					IntermediaryActions: map[string]*IntermediaryReconcileAction{
						"intermediary-1": {
							IntermediaryID: "intermediary-1",
							Action:         ReconciliationActionManualCleanupRequired,
							NewStatus:      core.PreciseResourceStatusCreateFailed,
						},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)

	// Verify link drift state was NOT removed
	linkDriftState, err := s.stateContainer.Links().GetDrift(context.Background(), "link-1")
	s.Require().NoError(err)
	s.Equal("link-1", linkDriftState.LinkID, "drift state should be preserved for ManualCleanupRequired action")

	// Verify link Drifted flag was NOT cleared
	linkState, err := s.stateContainer.Links().Get(context.Background(), "link-1")
	s.Require().NoError(err)
	s.True(linkState.Drifted, "Drifted flag should be preserved for ManualCleanupRequired action")
	s.NotNil(linkState.LastDriftDetectedTimestamp, "LastDriftDetectedTimestamp should be preserved")

	// Verify failure reasons were added
	s.NotEmpty(linkState.FailureReasons, "failure reasons should be added for ManualCleanupRequired action")
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_populates_element_name_in_error() {
	// Setup state with resource
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-1": {
				ResourceID:    "resource-1",
				Name:          "testResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
					{
						ResourceID:   "intermediary-1",
						ResourceType: "test/intermediary",
						InstanceID:   testReconciliationInstanceID,
						Status:       core.ResourceStatusCreated,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Apply reconciliation with action on non-existent intermediary to trigger error
	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "link-1",
					Action:    ReconciliationActionUpdateStatus,
					NewStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
					IntermediaryActions: map[string]*IntermediaryReconcileAction{
						"non-existent-intermediary": {
							IntermediaryID: "non-existent-intermediary",
							Action:         ReconciliationActionUpdateStatus,
							NewStatus:      core.PreciseResourceStatusCreated,
						},
					},
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Len(result.Errors, 1)

	reconcileError := result.Errors[0]
	s.Equal("link-1", reconcileError.ElementID)
	s.Equal("resourceA::resourceB", reconcileError.ElementName, "ElementName should be populated")
	s.Equal("link", reconcileError.ElementType)
	s.Contains(reconcileError.Error, "non-existent-intermediary")
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_populates_link_data_updates_for_drift() {
	oldValue := "old-handler"
	newValue := "new-external-handler"

	// Setup state with link
	err := s.populateTestState(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreated,
				PreciseStatus: core.PreciseLinkStatusIntermediaryResourcesUpdated,
				Drifted:       true,
				Data: map[string]*core.MappingNode{
					"resourceA": {
						Fields: map[string]*core.MappingNode{
							"handler": {Scalar: &core.ScalarValue{StringValue: &oldValue}},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Configure mock drift checker to return link drift with mapped field changes
	mockDriftCheckerWithLinkDrift := &mockDriftChecker{
		checkLinkDriftState: &state.LinkDriftState{
			LinkID:   "link-1",
			LinkName: "resourceA::resourceB",
			ResourceADrift: &state.LinkResourceDrift{
				ResourceID:   "resource-a",
				ResourceName: "resourceA",
				MappedFieldChanges: []*state.LinkDriftFieldChange{
					{
						ResourceFieldPath: "resourceA::handler",
						LinkDataPath:      "resourceA.handler",
						LinkDataValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldValue}},
						ExternalValue:     &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newValue}},
					},
				},
			},
		},
	}

	containerWithMockDrift := &defaultBlueprintContainer{
		stateContainer: s.stateContainer,
		driftChecker:   mockDriftCheckerWithLinkDrift,
		clock:          core.SystemClock{},
		logger:         core.NewNopLogger(),
	}

	result, err := containerWithMockDrift.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeAll,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasDrift)
	s.Len(result.Links, 1)

	linkResult := result.Links[0]
	s.Equal("link-1", linkResult.LinkID)
	s.Equal(ReconciliationTypeDrift, linkResult.Type)

	// Verify LinkDataUpdates is pre-computed with the external values
	s.Require().NotNil(linkResult.LinkDataUpdates, "LinkDataUpdates should be populated")
	s.Len(linkResult.LinkDataUpdates, 1)
	s.Require().NotNil(linkResult.LinkDataUpdates["resourceA.handler"])
	s.Equal(newValue, *linkResult.LinkDataUpdates["resourceA.handler"].Scalar.StringValue,
		"LinkDataUpdates should contain the external value for reconciliation")
}

func TestContainerReconciliationTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerReconciliationTestSuite))
}

// mockDriftChecker is a test mock for the drift.Checker interface
type mockDriftChecker struct {
	checkDriftResults        map[string]*state.ResourceDriftState
	checkDriftError          error
	checkResourceDriftState  *state.ResourceDriftState
	checkResourceDriftError  error
	checkInterruptedResults  []drift.ReconcileResult
	checkInterruptedError    error
	applyReconcileError      error
	checkLinkDriftState      *state.LinkDriftState
	checkLinkDriftError      error
	checkAllLinkDriftResults map[string]*state.LinkDriftState
	checkAllLinkDriftError   error
}

func (m *mockDriftChecker) CheckDrift(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (map[string]*state.ResourceDriftState, error) {
	return m.checkDriftResults, m.checkDriftError
}

func (m *mockDriftChecker) CheckResourceDrift(
	ctx context.Context,
	instanceID string,
	instanceName string,
	resourceID string,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (*state.ResourceDriftState, error) {
	return m.checkResourceDriftState, m.checkResourceDriftError
}

func (m *mockDriftChecker) CheckInterruptedResources(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) ([]drift.ReconcileResult, error) {
	return m.checkInterruptedResults, m.checkInterruptedError
}

func (m *mockDriftChecker) ApplyReconciliation(
	ctx context.Context,
	results []drift.ReconcileResult,
) error {
	return m.applyReconcileError
}

func (m *mockDriftChecker) CheckLinkDrift(
	ctx context.Context,
	instanceID string,
	linkID string,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (*state.LinkDriftState, error) {
	return m.checkLinkDriftState, m.checkLinkDriftError
}

func (m *mockDriftChecker) CheckAllLinkDrift(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (map[string]*state.LinkDriftState, error) {
	return m.checkAllLinkDriftResults, m.checkAllLinkDriftError
}

func (m *mockDriftChecker) CheckDriftWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (map[string]*state.ResourceDriftState, error) {
	return m.checkDriftResults, m.checkDriftError
}

func (m *mockDriftChecker) CheckInterruptedResourcesWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) ([]drift.ReconcileResult, error) {
	return m.checkInterruptedResults, m.checkInterruptedError
}

func (m *mockDriftChecker) CheckAllLinkDriftWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	taggingConfig *provider.TaggingConfig,
) (map[string]*state.LinkDriftState, error) {
	return m.checkAllLinkDriftResults, m.checkAllLinkDriftError
}

// ============================================================================
// Child Blueprint Reconciliation Tests
// ============================================================================

const (
	testChildInstanceID   = "test-child-instance"
	testChildInstanceName = "TestChildInstance"
	testNestedChildID     = "test-nested-child-instance"
	testNestedChildName   = "TestNestedChildInstance"
)

// populateTestStateWithChildren creates instance state with nested child blueprints
func (s *ContainerReconciliationTestSuite) populateTestStateWithChildren(
	parentResources map[string]*state.ResourceState,
	parentLinks map[string]*state.LinkState,
	childBlueprints map[string]*state.InstanceState,
) error {
	// Create parent instance with children
	err := s.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID:      testReconciliationInstanceID,
			InstanceName:    testReconciliationInstanceName,
			Status:          core.InstanceStatusUpdated,
			Resources:       parentResources,
			Links:           parentLinks,
			ChildBlueprints: childBlueprints,
		},
	)
	if err != nil {
		return err
	}

	// Save individual parent resources
	if err := s.saveResources(parentResources); err != nil {
		return err
	}

	// Save individual parent links
	if err := s.saveLinks(parentLinks); err != nil {
		return err
	}

	// Save child instance resources and links
	for _, child := range childBlueprints {
		if err := s.saveChildInstanceState(child); err != nil {
			return err
		}
	}

	return nil
}

func (s *ContainerReconciliationTestSuite) saveResources(resources map[string]*state.ResourceState) error {
	for _, r := range resources {
		if err := s.stateContainer.Resources().Save(context.Background(), *r); err != nil {
			return err
		}
	}
	return nil
}

func (s *ContainerReconciliationTestSuite) saveLinks(links map[string]*state.LinkState) error {
	for _, l := range links {
		if err := s.stateContainer.Links().Save(context.Background(), *l); err != nil {
			return err
		}
	}
	return nil
}

func (s *ContainerReconciliationTestSuite) saveChildInstanceState(child *state.InstanceState) error {
	if child == nil {
		return nil
	}

	if err := s.saveResources(child.Resources); err != nil {
		return err
	}

	if err := s.saveLinks(child.Links); err != nil {
		return err
	}

	// Recursively save nested children
	for _, nested := range child.ChildBlueprints {
		if err := s.saveChildInstanceState(nested); err != nil {
			return err
		}
	}

	return nil
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_includes_child_blueprint_resources() {
	// Setup state with parent and child resources
	err := s.populateTestStateWithChildren(
		map[string]*state.ResourceState{
			"parent-resource-1": {
				ResourceID:    "parent-resource-1",
				Name:          "parentResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Setup drift checker to return interrupted resource results for both parent and child
	s.driftChecker.checkInterruptedResults = []drift.ReconcileResult{
		{
			ResourceID:   "parent-resource-1",
			ResourceName: "parentResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
		{
			ResourceID:   "child-resource-1",
			ResourceName: "childResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
	}

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasInterrupted)
	s.True(result.HasChildIssues, "HasChildIssues should be true when child has issues")

	// Should have results from both parent and child
	s.Require().Len(result.Resources, 2)

	// Find parent and child results
	parentResult := s.findResourceResult(result.Resources, "parent-resource-1")
	childResult := s.findResourceResult(result.Resources, "child-resource-1")

	s.Require().NotNil(parentResult, "should have parent resource result")
	s.Equal("", parentResult.ChildPath, "parent resource should have empty ChildPath")
	s.Equal("parentResource1", parentResult.ResourceName)

	s.Require().NotNil(childResult, "should have child resource result")
	s.Equal("childA", childResult.ChildPath, "child resource should have ChildPath set")
	s.Equal("childResource1", childResult.ResourceName)
}

func (s *ContainerReconciliationTestSuite) findResourceResult(
	results []ResourceReconcileResult,
	resourceID string,
) *ResourceReconcileResult {
	for i := range results {
		if results[i].ResourceID == resourceID {
			return &results[i]
		}
	}
	return nil
}

func (s *ContainerReconciliationTestSuite) findLinkResult(
	results []LinkReconcileResult,
	linkID string,
) *LinkReconcileResult {
	for i := range results {
		if results[i].LinkID == linkID {
			return &results[i]
		}
	}
	return nil
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_includes_nested_child_blueprint_resources() {
	// Setup state with parent, child, and nested child resources
	err := s.populateTestStateWithChildren(
		map[string]*state.ResourceState{
			"parent-resource-1": {
				ResourceID:    "parent-resource-1",
				Name:          "parentResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
					},
				},
				ChildBlueprints: map[string]*state.InstanceState{
					"nestedChild": {
						InstanceID:   testNestedChildID,
						InstanceName: testNestedChildName,
						Status:       core.InstanceStatusUpdated,
						Resources: map[string]*state.ResourceState{
							"nested-resource-1": {
								ResourceID:    "nested-resource-1",
								Name:          "nestedResource1",
								Type:          "test/resource",
								InstanceID:    testNestedChildID,
								Status:        core.ResourceStatusCreating,
								PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
							},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Setup drift checker to return interrupted resource results
	s.driftChecker.checkInterruptedResults = []drift.ReconcileResult{
		{
			ResourceID:   "parent-resource-1",
			ResourceName: "parentResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
		{
			ResourceID:   "child-resource-1",
			ResourceName: "childResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
		{
			ResourceID:   "nested-resource-1",
			ResourceName: "nestedResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
	}

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasChildIssues)

	// Should have results from parent, child, and nested child
	s.Require().Len(result.Resources, 3)

	// Find nested child result
	nestedResult := s.findResourceResult(result.Resources, "nested-resource-1")

	s.Require().NotNil(nestedResult, "should have nested child resource result")
	s.Equal("childA.nestedChild", nestedResult.ChildPath, "nested resource should have hierarchical ChildPath")
	s.Equal("nestedResource1", nestedResult.ResourceName)
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_excludes_children_when_include_children_false() {
	// Setup state with parent and child resources
	err := s.populateTestStateWithChildren(
		map[string]*state.ResourceState{
			"parent-resource-1": {
				ResourceID:    "parent-resource-1",
				Name:          "parentResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Setup drift checker to return interrupted resource results
	s.driftChecker.checkInterruptedResults = []drift.ReconcileResult{
		{
			ResourceID:   "parent-resource-1",
			ResourceName: "parentResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
	}

	includeChildren := false
	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID:      testReconciliationInstanceID,
			Scope:           ReconciliationScopeInterrupted,
			IncludeChildren: &includeChildren,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Should only have parent result when IncludeChildren is false
	s.Len(result.Resources, 1)
	s.Equal("parent-resource-1", result.Resources[0].ResourceID)
	s.Equal("", result.Resources[0].ChildPath)
	s.False(result.HasChildIssues, "HasChildIssues should be false when children are excluded")
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_filters_by_child_path() {
	// Setup state with parent and multiple child resources
	err := s.populateTestStateWithChildren(
		map[string]*state.ResourceState{
			"parent-resource-1": {
				ResourceID:    "parent-resource-1",
				Name:          "parentResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
					},
				},
			},
			"childB": {
				InstanceID:   "test-child-b-instance",
				InstanceName: "TestChildBInstance",
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-b-resource-1": {
						ResourceID:    "child-b-resource-1",
						Name:          "childBResource1",
						Type:          "test/resource",
						InstanceID:    "test-child-b-instance",
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Setup drift checker to return interrupted resource results
	s.driftChecker.checkInterruptedResults = []drift.ReconcileResult{
		{
			ResourceID:   "child-resource-1",
			ResourceName: "childResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
	}

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
			ChildPath:  "childA", // Only check resources in childA
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Should only have childA result when filtered by ChildPath
	s.Len(result.Resources, 1)
	s.Equal("child-resource-1", result.Resources[0].ResourceID)
	s.Equal("childA", result.Resources[0].ChildPath)
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_includes_child_blueprint_links() {
	// Setup state with parent and child links
	err := s.populateTestStateWithChildren(
		map[string]*state.ResourceState{
			"resource-a": {
				ResourceID:    "resource-a",
				Name:          "resourceA",
				Type:          "test/resourceA",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
			"resource-b": {
				ResourceID:    "resource-b",
				Name:          "resourceB",
				Type:          "test/resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
		map[string]*state.LinkState{
			"resourceA::resourceB": {
				LinkID:        "parent-link-1",
				Name:          "resourceA::resourceB",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.LinkStatusCreating,
				PreciseStatus: core.PreciseLinkStatusResourceAUpdateInterrupted,
			},
		},
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-a": {
						ResourceID:    "child-resource-a",
						Name:          "childResourceA",
						Type:          "test/resourceA",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
					},
					"child-resource-b": {
						ResourceID:    "child-resource-b",
						Name:          "childResourceB",
						Type:          "test/resourceB",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
					},
				},
				Links: map[string]*state.LinkState{
					"childResourceA::childResourceB": {
						LinkID:        "child-link-1",
						Name:          "childResourceA::childResourceB",
						InstanceID:    testChildInstanceID,
						Status:        core.LinkStatusCreating,
						PreciseStatus: core.PreciseLinkStatusResourceAUpdateInterrupted,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasInterrupted)
	s.True(result.HasChildIssues)

	// Should have links from both parent and child
	s.Require().Len(result.Links, 2)

	// Find parent and child link results
	parentLink := s.findLinkResult(result.Links, "parent-link-1")
	childLink := s.findLinkResult(result.Links, "child-link-1")

	s.Require().NotNil(parentLink, "should have parent link result")
	s.Equal("", parentLink.ChildPath, "parent link should have empty ChildPath")
	s.Equal("resourceA::resourceB", parentLink.LinkName)

	s.Require().NotNil(childLink, "should have child link result")
	s.Equal("childA", childLink.ChildPath, "child link should have ChildPath set")
	s.Equal("childResourceA::childResourceB", childLink.LinkName)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_updates_child_resource() {
	// Setup state with child resource
	err := s.populateTestStateWithChildren(
		nil,
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID: "child-resource-1",
					ChildPath:  "childA",
					Action:     ReconciliationActionUpdateStatus,
					NewStatus:  core.PreciseResourceStatusCreated,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the child resource state was updated
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "child-resource-1")
	s.Require().NoError(err)
	s.Equal(core.ResourceStatusCreated, resourceState.Status)
	s.Equal(core.PreciseResourceStatusCreated, resourceState.PreciseStatus)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_accepts_external_state_for_child_resource() {
	oldValue := "old-child-value"
	// Setup state with child resource
	err := s.populateTestStateWithChildren(
		nil,
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreating,
						PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
						SpecData: &core.MappingNode{
							Scalar: &core.ScalarValue{StringValue: &oldValue},
						},
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	newValue := "new-external-child-value"
	externalState := &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &newValue},
	}

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "child-resource-1",
					ChildPath:     "childA",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: externalState,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.ResourcesUpdated)
	s.Empty(result.Errors)

	// Verify the child resource state was updated with external state
	resourceState, err := s.stateContainer.Resources().Get(context.Background(), "child-resource-1")
	s.Require().NoError(err)
	s.Equal(core.PreciseResourceStatusCreated, resourceState.PreciseStatus)
	s.Equal("new-external-child-value", *resourceState.SpecData.Scalar.StringValue)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_updates_child_link() {
	// Setup state with child link
	err := s.populateTestStateWithChildren(
		nil,
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-a": {
						ResourceID:    "child-resource-a",
						Name:          "childResourceA",
						Type:          "test/resourceA",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
					},
					"child-resource-b": {
						ResourceID:    "child-resource-b",
						Name:          "childResourceB",
						Type:          "test/resourceB",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
					},
				},
				Links: map[string]*state.LinkState{
					"childResourceA::childResourceB": {
						LinkID:        "child-link-1",
						Name:          "childResourceA::childResourceB",
						InstanceID:    testChildInstanceID,
						Status:        core.LinkStatusCreating,
						PreciseStatus: core.PreciseLinkStatusResourceAUpdateInterrupted,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			LinkActions: []LinkReconcileAction{
				{
					LinkID:    "child-link-1",
					ChildPath: "childA",
					Action:    ReconciliationActionUpdateStatus,
					NewStatus: core.PreciseLinkStatusResourceAUpdated,
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(1, result.LinksUpdated)
	s.Empty(result.Errors)

	// Verify the child link state was updated
	linkState, err := s.stateContainer.Links().Get(context.Background(), "child-link-1")
	s.Require().NoError(err)
	s.Equal(core.LinkStatusCreated, linkState.Status)
	s.Equal(core.PreciseLinkStatusResourceAUpdated, linkState.PreciseStatus)
}

func (s *ContainerReconciliationTestSuite) Test_apply_reconciliation_populates_child_path_in_error() {
	// Setup state with child resource
	err := s.populateTestStateWithChildren(
		nil,
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreated,
						PreciseStatus: core.PreciseResourceStatusCreated,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Apply reconciliation with AcceptExternal but missing external state
	result, err := s.container.ApplyReconciliation(
		context.Background(),
		&ApplyReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			ResourceActions: []ResourceReconcileAction{
				{
					ResourceID:    "child-resource-1",
					ChildPath:     "childA",
					Action:        ReconciliationActionAcceptExternal,
					NewStatus:     core.PreciseResourceStatusCreated,
					ExternalState: nil, // Missing external state will cause error
				},
			},
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Len(result.Errors, 1)

	reconcileError := result.Errors[0]
	s.Equal("child-resource-1", reconcileError.ElementID)
	s.Equal("childA.childResource1", reconcileError.ElementName, "ElementName should include ChildPath")
	s.Equal("resource", reconcileError.ElementType)
	s.Contains(reconcileError.Error, "external state is required")
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_returns_error_for_max_depth_exceeded() {
	// Build a deeply nested structure that exceeds MaxBlueprintDepth (5)
	deeplyNestedChild := s.buildDeeplyNestedChild(6)

	// Setup with the deeply nested structure
	err := s.populateTestStateWithChildren(
		nil,
		nil,
		map[string]*state.InstanceState{
			"child": deeplyNestedChild,
		},
	)
	s.Require().NoError(err)

	_, err = s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().Error(err)
	s.Contains(err.Error(), "max blueprint depth exceeded")
}

// buildDeeplyNestedChild creates a chain of nested child instances up to the given depth
func (s *ContainerReconciliationTestSuite) buildDeeplyNestedChild(depth int) *state.InstanceState {
	if depth <= 0 {
		return nil
	}

	// Create the deepest child first
	current := &state.InstanceState{
		InstanceID:   "depth-1-instance",
		InstanceName: "Depth1Instance",
		Status:       core.InstanceStatusUpdated,
		Resources: map[string]*state.ResourceState{
			"deep-resource": {
				ResourceID:    "deep-resource",
				Name:          "deepResource",
				Type:          "test/resource",
				InstanceID:    "depth-1-instance",
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
	}

	// Wrap it in successive parent layers
	for i := 2; i <= depth; i++ {
		current = &state.InstanceState{
			InstanceID:   "depth-" + string(rune('0'+i)) + "-instance",
			InstanceName: "Depth" + string(rune('0'+i)) + "Instance",
			Status:       core.InstanceStatusUpdated,
			ChildBlueprints: map[string]*state.InstanceState{
				"child": current,
			},
		}
	}

	return current
}

func (s *ContainerReconciliationTestSuite) Test_check_reconciliation_has_child_issues_only_when_child_has_issues() {
	// Setup state with parent issue but no child issues
	err := s.populateTestStateWithChildren(
		map[string]*state.ResourceState{
			"parent-resource-1": {
				ResourceID:    "parent-resource-1",
				Name:          "parentResource1",
				Type:          "test/resource",
				InstanceID:    testReconciliationInstanceID,
				Status:        core.ResourceStatusCreating,
				PreciseStatus: core.PreciseResourceStatusCreateInterrupted,
			},
		},
		nil,
		map[string]*state.InstanceState{
			"childA": {
				InstanceID:   testChildInstanceID,
				InstanceName: testChildInstanceName,
				Status:       core.InstanceStatusUpdated,
				Resources: map[string]*state.ResourceState{
					"child-resource-1": {
						ResourceID:    "child-resource-1",
						Name:          "childResource1",
						Type:          "test/resource",
						InstanceID:    testChildInstanceID,
						Status:        core.ResourceStatusCreated, // Not interrupted
						PreciseStatus: core.PreciseResourceStatusCreated,
					},
				},
			},
		},
	)
	s.Require().NoError(err)

	// Setup drift checker to return only parent interrupted result
	s.driftChecker.checkInterruptedResults = []drift.ReconcileResult{
		{
			ResourceID:   "parent-resource-1",
			ResourceName: "parentResource1",
			ResourceType: "test/resource",
			OldStatus:    core.PreciseResourceStatusCreateInterrupted,
			NewStatus:    core.PreciseResourceStatusCreated,
		},
	}

	result, err := s.container.CheckReconciliation(
		context.Background(),
		&CheckReconciliationInput{
			InstanceID: testReconciliationInstanceID,
			Scope:      ReconciliationScopeInterrupted,
		},
		nil,
	)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.True(result.HasInterrupted)
	s.False(result.HasChildIssues, "HasChildIssues should be false when only parent has issues")
	s.Len(result.Resources, 1)
	s.Equal("", result.Resources[0].ChildPath)
}
