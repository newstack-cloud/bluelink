package container

import (
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/stretchr/testify/suite"
)

type GroupOrderedElementsForRemovalTestSuite struct {
	groupFixture1 groupRemovalElementFixture
	suite.Suite
}

type groupRemovalElementFixture struct {
	orderedElements []*ElementWithAllDeps
	// All the elements that are expected to be in each group.
	// The order of the groups matter but the order of the resources, child blueprints
	// and links in each group doesn't.
	expectedPresent [][]state.Element
}

func (s *GroupOrderedElementsForRemovalTestSuite) SetupSuite() {
	groupFixture1 := groupRemovalFixture1()
	s.groupFixture1 = groupFixture1

}

func (s *GroupOrderedElementsForRemovalTestSuite) Test_groups_elements_to_be_removed() {
	groups := GroupOrderedElementsForRemoval(
		s.groupFixture1.orderedElements,
	)
	s.Assert().Len(groups, len(s.groupFixture1.expectedPresent))

	s.assertExpectedGroups(groups, s.groupFixture1.expectedPresent)
}

func (s *GroupOrderedElementsForRemovalTestSuite) assertExpectedGroups(
	groups [][]state.Element,
	expectedPresent [][]state.Element,
) {
	for i, group := range groups {
		expectedGroup := expectedPresent[i]
		actualGroupIDs := core.Map(group, func(element state.Element, _ int) string {
			return element.ID()
		})
		expectedGroupIDs := []string{}
		for _, element := range expectedGroup {
			expectedGroupIDs = append(expectedGroupIDs, element.ID())
		}
		slices.Sort(actualGroupIDs)
		slices.Sort(expectedGroupIDs)
		s.Assert().Equal(expectedGroupIDs, actualGroupIDs)
	}
}

func groupRemovalFixture1() groupRemovalElementFixture {
	ordersTable0 := &ResourceIDInfo{
		ResourceID:   "test-orders-table-0-id",
		ResourceName: "ordersTable_0",
	}
	ordersTable1 := &ResourceIDInfo{
		ResourceID:   "test-orders-table-1-id",
		ResourceName: "ordersTable_1",
	}
	saveOrderFunction := &ResourceIDInfo{
		ResourceID:   "test-save-order-function-id",
		ResourceName: "saveOrderFunction",
	}
	invoicesTable := &ResourceIDInfo{
		ResourceID:   "test-invoices-table-id",
		ResourceName: "invoicesTable",
	}
	preprocessOrderFunction := &ResourceIDInfo{
		ResourceID:   "test-preprocess-order-function-id",
		ResourceName: "preprocessOrderFunction",
	}
	testLink1 := &LinkIDInfo{
		LinkID:   "test-link-1",
		LinkName: "saveOrderFunction::ordersTable_0",
	}
	testLink2 := &LinkIDInfo{
		LinkID:   "test-link-2",
		LinkName: "saveOrderFunction::ordersTable_1",
	}
	coreInfra := &ChildBlueprintIDInfo{
		ChildInstanceID: "blueprint-instance-child-core-infra",
		ChildName:       "coreInfra",
	}
	networking := &ChildBlueprintIDInfo{
		ChildInstanceID: "blueprint-instance-child-networking",
		ChildName:       "networking",
	}

	return groupRemovalElementFixture{
		orderedElements: []*ElementWithAllDeps{
			{
				Element: testLink1,
				AllDependencies: []state.Element{
					saveOrderFunction,
					preprocessOrderFunction,
					coreInfra,
					networking,
					ordersTable0,
				},
			},
			{
				Element: testLink2,
				AllDependencies: []state.Element{
					saveOrderFunction,
					preprocessOrderFunction,
					coreInfra,
					ordersTable1,
				},
			},
			{
				Element: ordersTable0,
			},
			{
				Element: ordersTable1,
			},
			{
				Element: saveOrderFunction,
				AllDependencies: []state.Element{
					preprocessOrderFunction,
					coreInfra,
					networking,
				},
			},
			{
				Element: invoicesTable,
			},
			{
				Element: coreInfra,
				AllDependencies: []state.Element{
					networking,
					preprocessOrderFunction,
				},
			},
			{
				Element: networking,
			},
			{
				Element: preprocessOrderFunction,
			},
		},
		expectedPresent: [][]state.Element{
			{
				testLink1,
				testLink2,
			},
			{
				ordersTable0,
				ordersTable1,
				saveOrderFunction,
				invoicesTable,
			},
			{
				coreInfra,
			},
			{
				networking,
				preprocessOrderFunction,
			},
		},
	}
}

func TestGroupOrderedElementsForRemovalTestSuite(t *testing.T) {
	suite.Run(t, new(GroupOrderedElementsForRemovalTestSuite))
}
