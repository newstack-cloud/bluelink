package drift

import (
	"context"
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type DriftCheckerTestSuite struct {
	stateContainer state.Container
	driftChecker   Checker
	suite.Suite
}

const (
	instance1ID           = "blueprint-instance-1"
	ordersTableID         = "orders-table"
	saveOrderFunctionID   = "save-order-function"
	ordersTableName       = "ordersTable"
	saveOrderFunctionName = "saveOrderFunction"
)

func (s *DriftCheckerTestSuite) SetupTest() {
	s.stateContainer = memstate.NewMemoryStateContainer()
	err := s.populateCurrentState()
	s.Require().NoError(err)
	s.driftChecker = NewDefaultChecker(
		s.stateContainer,
		map[string]provider.Provider{
			"aws": newTestAWSProvider(
				s.dynamoDBTableExternalState(),
				s.lambdaFunctionExternalState(),
			),
		},
		changes.NewDefaultResourceChangeGenerator(),
		core.SystemClock{},
		core.NewNopLogger(),
	)
}

func (s *DriftCheckerTestSuite) Test_checks_drift_for_resources_in_blueprint() {
	driftStateMap, err := s.driftChecker.CheckDrift(
		context.Background(),
		instance1ID,
		createParams(),
	)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(normaliseResourceDriftStateMap(driftStateMap))
	s.Require().NoError(err)

	resources := s.stateContainer.Resources()
	resourceIDs := []string{saveOrderFunctionID, ordersTableID}
	for _, resourceID := range resourceIDs {
		stateAfterCheck, err := resources.Get(
			context.Background(),
			resourceID,
		)
		s.Require().NoError(err)

		s.Assert().True(stateAfterCheck.Drifted)
		s.Assert().NotNil(stateAfterCheck.LastDriftDetectedTimestamp)
		s.Assert().Greater(*stateAfterCheck.LastDriftDetectedTimestamp, 0)

		persistedDriftState, err := resources.GetDrift(
			context.Background(),
			resourceID,
		)
		s.Require().NoError(err)
		s.Assert().NotNil(persistedDriftState)
		s.Assert().Equal(driftStateMap[resourceID], &persistedDriftState)
	}
}

func (s *DriftCheckerTestSuite) Test_checks_drift_for_a_single_resource() {
	driftState, err := s.driftChecker.CheckResourceDrift(
		context.Background(),
		instance1ID,
		instance1ID,
		saveOrderFunctionID,
		createParams(),
	)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(normaliseResourceDriftState(driftState))
	s.Require().NoError(err)

	resources := s.stateContainer.Resources()

	stateAfterCheck, err := resources.Get(
		context.Background(),
		saveOrderFunctionID,
	)
	s.Require().NoError(err)

	s.Assert().True(stateAfterCheck.Drifted)
	s.Assert().NotNil(stateAfterCheck.LastDriftDetectedTimestamp)
	s.Assert().Greater(*stateAfterCheck.LastDriftDetectedTimestamp, 0)

	persistedDriftState, err := resources.GetDrift(
		context.Background(),
		saveOrderFunctionID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(persistedDriftState)
	s.Assert().Equal(driftState, &persistedDriftState)
}

func (s *DriftCheckerTestSuite) populateCurrentState() error {
	return s.stateContainer.Instances().Save(
		context.Background(),
		state.InstanceState{
			InstanceID: instance1ID,
			Status:     core.InstanceStatusDeployed,
			ResourceIDs: map[string]string{
				saveOrderFunctionName: saveOrderFunctionID,
				ordersTableName:       ordersTableID,
			},
			Resources: map[string]*state.ResourceState{
				saveOrderFunctionID: {
					ResourceID:    saveOrderFunctionID,
					Name:          saveOrderFunctionName,
					Type:          "aws/lambda/function",
					InstanceID:    instance1ID,
					Status:        core.ResourceStatusCreated,
					PreciseStatus: core.PreciseResourceStatusCreated,
					SpecData: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"id": core.MappingNodeFromString(
								"arn:aws:lambda:us-east-1:123456789012:function:save-order-function",
							),
							"handler": core.MappingNodeFromString("saveOrderFunction.handler"),
						},
					},
					Drifted: false,
				},
				ordersTableID: {
					ResourceID:    ordersTableID,
					Name:          ordersTableName,
					Type:          "aws/dynamodb/table",
					InstanceID:    instance1ID,
					Status:        core.ResourceStatusCreated,
					PreciseStatus: core.PreciseResourceStatusCreated,
					SpecData: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"tableName": core.MappingNodeFromString("ORDERS_TABLE"),
							"region":    core.MappingNodeFromString("us-east-1"),
						},
					},
					Drifted: false,
				},
			},
		},
	)
}

func (s *DriftCheckerTestSuite) dynamoDBTableExternalState() *core.MappingNode {
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"tableName": core.MappingNodeFromString("ORDERS_TABLE_2"),
			"region":    core.MappingNodeFromString("us-west-1"),
		},
	}
}

func (s *DriftCheckerTestSuite) lambdaFunctionExternalState() *core.MappingNode {
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id": core.MappingNodeFromString(
				"arn:aws:lambda:us-west-1:124856789012:function:save-order-function-2",
			),
			"handler": core.MappingNodeFromString("orders.saveOrder"),
		},
	}
}

func createParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

func normaliseResourceDriftStateMap(
	driftState map[string]*state.ResourceDriftState,
) map[string]*state.ResourceDriftState {
	normalised := map[string]*state.ResourceDriftState{}
	for k, v := range driftState {
		normalised[k] = normaliseResourceDriftState(v)
	}
	return normalised
}

func normaliseResourceDriftState(
	driftState *state.ResourceDriftState,
) *state.ResourceDriftState {
	replacementTimestamp := -1
	return &state.ResourceDriftState{
		ResourceID:   driftState.ResourceID,
		ResourceName: driftState.ResourceName,
		SpecData:     driftState.SpecData,
		Difference:   normaliseResourceDriftDifference(driftState.Difference),
		Timestamp:    &replacementTimestamp,
	}
}

func normaliseResourceDriftDifference(
	difference *state.ResourceDriftChanges,
) *state.ResourceDriftChanges {
	return &state.ResourceDriftChanges{
		ModifiedFields: orderResourceDriftFieldChanges(difference.ModifiedFields),
		NewFields:      orderResourceDriftFieldChanges(difference.NewFields),
		RemovedFields:  internal.OrderStringSlice(difference.RemovedFields),
		UnchangedFields: internal.OrderStringSlice(
			difference.UnchangedFields,
		),
	}
}

func orderResourceDriftFieldChanges(
	fieldChanges []*state.ResourceDriftFieldChange,
) []*state.ResourceDriftFieldChange {
	orderedFieldChanges := make([]*state.ResourceDriftFieldChange, len(fieldChanges))
	copy(orderedFieldChanges, fieldChanges)
	slices.SortFunc(orderedFieldChanges, func(a, b *state.ResourceDriftFieldChange) int {
		if a.FieldPath < b.FieldPath {
			return -1
		}

		if a.FieldPath > b.FieldPath {
			return 1
		}

		return 0
	})
	return orderedFieldChanges
}

func TestDriftCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(DriftCheckerTestSuite))
}
