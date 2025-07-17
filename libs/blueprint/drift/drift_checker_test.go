package drift

import (
	"context"
	"fmt"
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
	complexResourceID     = "complex-resource"
	complexResourceName   = "complexResource"
)

func (s *DriftCheckerTestSuite) SetupTest() {
	s.stateContainer = memstate.NewMemoryStateContainer()
	err := s.populateCurrentState( /* includeLinkData */ false)
	s.Require().NoError(err)
	s.driftChecker = NewDefaultChecker(
		s.stateContainer,
		map[string]provider.Provider{
			"aws": newTestAWSProvider(
				s.dynamoDBTableExternalState(),
				s.lambdaFunctionExternalState(),
			),
			"example": newTestExampleProvider(
				s.exampleComplexResourceExternalState(),
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

func (s *DriftCheckerTestSuite) Test_checks_drift_for_a_single_resource_where_link_changes_are_taken_into_account() {
	// Link changes should not be treated as drift, the implementation should overlay
	// them on top of the existing resource state to create a derived state to compare
	// with what is in the external system.

	// Re-populate the current state with link data included.
	err := s.populateCurrentState( /* includeLinkData */ true)
	s.Require().NoError(err)

	driftState, err := s.driftChecker.CheckResourceDrift(
		context.Background(),
		instance1ID,
		instance1ID,
		saveOrderFunctionID,
		createParams(),
	)
	s.Require().NoError(err)
	s.Assert().Nil(driftState)

	resources := s.stateContainer.Resources()

	stateAfterCheck, err := resources.Get(
		context.Background(),
		saveOrderFunctionID,
	)
	s.Require().NoError(err)

	s.Assert().False(stateAfterCheck.Drifted)

	persistedDriftState, err := resources.GetDrift(
		context.Background(),
		saveOrderFunctionID,
	)
	s.Require().NoError(err)
	// Empty fields indicate that there was no drift detected.
	s.Assert().Equal(persistedDriftState.ResourceID, "")
	s.Assert().Equal(persistedDriftState.ResourceName, "")
	s.Assert().Nil(persistedDriftState.SpecData)
	s.Assert().Nil(persistedDriftState.Difference)
	s.Assert().Nil(persistedDriftState.Timestamp)
}

func (s *DriftCheckerTestSuite) populateCurrentState(includeLinkData bool) error {
	instanceState := state.InstanceState{
		InstanceID: instance1ID,
		Status:     core.InstanceStatusDeployed,
		ResourceIDs: map[string]string{
			saveOrderFunctionName: saveOrderFunctionID,
			ordersTableName:       ordersTableID,
			complexResourceName:   complexResourceID,
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
			complexResourceID: {
				ResourceID:    complexResourceID,
				Name:          complexResourceName,
				Type:          "example/complexResource",
				InstanceID:    instance1ID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"itemConfig": {
							Fields: map[string]*core.MappingNode{
								"endpoints": {
									Items: []*core.MappingNode{
										core.MappingNodeFromString("https://api.example.com/node/1"),
										core.MappingNodeFromString("https://api.example.com/node/2"),
									},
								},
								"primaryPort": core.MappingNodeFromInt(8080),
								"score":       core.MappingNodeFromFloat(13.5),
								"ipv4":        core.MappingNodeFromBool(true),
								"metadata": {
									Fields: map[string]*core.MappingNode{
										"sampleKey": core.MappingNodeFromString("sampleValue"),
									},
								},
							},
						},
						"otherItemConfig": {
							Fields: map[string]*core.MappingNode{
								"item1": {
									Fields: map[string]*core.MappingNode{
										"value1": core.MappingNodeFromString("Value 1"),
										"value2": core.MappingNodeFromString("Value 2"),
									},
								},
							},
						},
						"vendorTags": {
							Items: []*core.MappingNode{
								core.MappingNodeFromString("tag1"),
								core.MappingNodeFromString("tag2"),
								core.MappingNodeFromString("tag3"),
							},
						},
						"vendorConfig": {
							Items: []*core.MappingNode{
								{
									Fields: map[string]*core.MappingNode{
										"vendorNamespace": core.MappingNodeFromString("vendor1"),
										"vendorId":        core.MappingNodeFromString("vendor1-id"),
									},
								},
							},
						},
					},
				},
				Drifted: false,
			},
		},
	}

	if includeLinkData {
		saveOrderDataMappingFieldPath := fmt.Sprintf("%s::spec.handler", saveOrderFunctionName)
		instanceState.Links = map[string]*state.LinkState{
			"saveOrderFunction::ordersTable": {
				LinkID:     "test-link-1",
				Name:       "saveOrderFunction::ordersTable",
				InstanceID: instance1ID,
				Data: map[string]*core.MappingNode{
					"saveOrderFunction": {
						Fields: map[string]*core.MappingNode{
							// The same value as the external system, so when the link is applied
							// in the check, it should not cause any drift.
							"handler": core.MappingNodeFromString("orders.saveOrder"),
						},
					},
				},
				ResourceDataMappings: map[string]string{
					saveOrderDataMappingFieldPath: "saveOrderFunction.handler",
				},
			},
		}
	}

	return s.stateContainer.Instances().Save(
		context.Background(),
		instanceState,
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

func (s *DriftCheckerTestSuite) exampleComplexResourceExternalState() *core.MappingNode {
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"itemConfig": {
				Fields: map[string]*core.MappingNode{
					"endpoints": {
						Items: []*core.MappingNode{
							core.MappingNodeFromString("https://api2.example.com/node/1"),
						},
					},
					"primaryPort": core.MappingNodeFromInt(8181),
					"score":       core.MappingNodeFromFloat(15.5),
					"ipv4":        core.MappingNodeFromBool(true),
					"metadata": {
						Fields: map[string]*core.MappingNode{
							"sampleKey": core.MappingNodeFromString("sampleValue2"),
						},
					},
				},
			},
			"otherItemConfig": {
				Fields: map[string]*core.MappingNode{
					"item1": {
						Fields: map[string]*core.MappingNode{
							"value1": core.MappingNodeFromString("Value 1 Updated"),
							"value2": core.MappingNodeFromString("Value 2"),
						},
					},
				},
			},
			"vendorTags": {
				Items: []*core.MappingNode{
					core.MappingNodeFromString("tag1--a"),
					core.MappingNodeFromString("tag2--b"),
					core.MappingNodeFromString("tag3--c"),
				},
			},
			"vendorConfig": {
				Items: []*core.MappingNode{
					{
						Fields: map[string]*core.MappingNode{
							"vendorNamespace": core.MappingNodeFromString("vendor1"),
							"vendorId":        core.MappingNodeFromString("vendor1-id-new"),
						},
					},
				},
			},
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
