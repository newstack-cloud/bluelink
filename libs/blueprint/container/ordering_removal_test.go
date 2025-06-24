package container

import (
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/stretchr/testify/suite"
)

type OrderingForRemovalTestSuite struct {
	orderRemovalFixture1 orderRemovalFixture
	suite.Suite
}

type orderRemovalFixture struct {
	elementsToRemove *CollectedElements
	currentState     *state.InstanceState
	// All the elements that are expected to be present
	// in the ordered list of elements.
	// The combination of LogicalName() and Type() is used to check the identity of elements.
	expectedPresent []state.Element
	// A two-dimensional slice of elements that must come in the provided order,
	// The order of the expectations does not matter, only the order of the elements
	// in each expectation "tuple".
	// The combination of LogicalName() and Type()
	// is used to check the identity of elements.
	orderExpected [][]state.Element
	// A map of composite identifiers of an element to its dependencies.
	// The composite identifier is the combination of LogicalName() and Type()
	// in the form of "{LogicalName}__{Type}".
	expectedDependencies map[string][]state.Element
}

func (s *OrderingForRemovalTestSuite) SetupSuite() {
	fixture1, err := orderRemovalFixture1()
	if err != nil {
		s.FailNow(err.Error())
	}
	s.orderRemovalFixture1 = fixture1
}

func (s *OrderingForRemovalTestSuite) Test_orders_elements_for_removal() {
	orderedElements, err := OrderElementsForRemoval(
		s.orderRemovalFixture1.elementsToRemove,
		s.orderRemovalFixture1.currentState,
	)
	s.Require().NoError(err)
	s.Assert().Len(orderedElements, len(s.orderRemovalFixture1.expectedPresent))
	s.Assert().Len(
		core.Filter(
			orderedElements,
			inExpectedElements(s.orderRemovalFixture1.expectedPresent),
		),
		len(s.orderRemovalFixture1.expectedPresent),
	)

	for _, orderedExpectedSet := range s.orderRemovalFixture1.orderExpected {
		s.assertOrderExpected(orderedElements, orderedExpectedSet)
	}
}

func (s *OrderingForRemovalTestSuite) assertOrderExpected(actual []*ElementWithAllDeps, orderExpected []state.Element) {
	expectedItemsInOrder := getActualInExpectedOrder(actual, orderExpected)
	inOrder := true
	i := 0
	var elemA *ElementWithAllDeps
	var elemB *ElementWithAllDeps

	for inOrder && i < len(expectedItemsInOrder) {
		if i+1 < len(expectedItemsInOrder) {
			elemA = expectedItemsInOrder[i]
			elemB = expectedItemsInOrder[i+1]
			inOrder = elemA.Element.LogicalName() == orderExpected[i].LogicalName() &&
				elemA.Element.Kind() == orderExpected[i].Kind() &&
				elemB.Element.LogicalName() == orderExpected[i+1].LogicalName() &&
				elemB.Element.Kind() == orderExpected[i+1].Kind()
		}
		i += 2
	}

	if !inOrder {
		s.Failf("incorrect order", "expected \"%s\" to come before \"%s\"", elemB.Element.ID(), elemA.Element.ID())
	}
}

func orderRemovalFixture1() (orderRemovalFixture, error) {
	currentState, err := internal.LoadInstanceState(
		"__testdata/ordering-removal/fixture1-state.json",
	)
	if err != nil {
		return orderRemovalFixture{}, err
	}

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

	// These elements are purposefully arranged in a way that requires
	// sorting.
	elementsToRemove := &CollectedElements{
		Resources: []*ResourceIDInfo{
			preprocessOrderFunction,
			ordersTable0,
			ordersTable1,
			saveOrderFunction,
			invoicesTable,
		},
		Links: []*LinkIDInfo{
			testLink1,
			testLink2,
		},
		Children: []*ChildBlueprintIDInfo{
			networking,
			coreInfra,
		},
	}

	expectedPresent := []state.Element{
		ordersTable0,
		ordersTable1,
		saveOrderFunction,
		invoicesTable,
		preprocessOrderFunction,
		testLink1,
		testLink2,
		coreInfra,
		networking,
	}

	expectedDependencies := map[string][]state.Element{
		"saveOrderFunction::ordersTable_0__link": {
			saveOrderFunction,
			ordersTable0,
			preprocessOrderFunction,
			coreInfra,
			networking,
		},
		"saveOrderFunction::ordersTable_1__link": {
			saveOrderFunction,
			ordersTable1,
			preprocessOrderFunction,
			coreInfra,
			networking,
		},
		"saveOrderFunction__resource": {
			preprocessOrderFunction,
			coreInfra,
			networking,
		},
		"ordersTable_0__resource": {
			// Orders table 0 does not have any dependencies.
		},
		"ordersTable_1__resource": {
			// Orders table 1 does not have any dependencies.
		},
		"preprocessOrderFunction__resource": {
			// Preprocess order function does not have any dependencies.
		},
		"invoicesTable__resource": {
			// Invoices table does not have any dependencies.
		},
		"coreInfra__child": {
			networking,
			preprocessOrderFunction,
		},
		"networking__child": {
			// Networking child blueprint does not have any dependencies.
		},
	}

	orderExpected := [][]state.Element{
		{
			// Test link 1 is expected to be removed before the save order function.
			testLink1,
			saveOrderFunction,
		},
		{
			// Test link 1 is expected to be removed before orders table 0.
			testLink1,
			ordersTable0,
		},
		{
			// Test link 2 is expected to be removed before the save order function.
			testLink2,
			saveOrderFunction,
		},
		{
			// Test link 2 is expected to be removed before orders table 1.
			testLink2,
			ordersTable1,
		},
		{
			// Save order function is expected to be removed before preprocess order function.
			saveOrderFunction,
			preprocessOrderFunction,
		},
		{
			// Save order function is expected to be removed before core infra.
			saveOrderFunction,
			coreInfra,
		},
		{
			// Core infra is expected to be removed before networking.
			coreInfra,
			networking,
		},
		{
			// Core infra is expected to be removed before preprocess order function.
			coreInfra,
			preprocessOrderFunction,
		},
	}

	return orderRemovalFixture{
		elementsToRemove:     elementsToRemove,
		currentState:         currentState,
		expectedPresent:      expectedPresent,
		orderExpected:        orderExpected,
		expectedDependencies: expectedDependencies,
	}, nil
}

func getActualInExpectedOrder(actual []*ElementWithAllDeps, orderExpected []state.Element) []*ElementWithAllDeps {
	actualInOrder := []*ElementWithAllDeps{}
	for _, expectedElement := range orderExpected {
		actualFiltered := core.Filter(actual, func(current *ElementWithAllDeps, index int) bool {
			return current.Element.LogicalName() == expectedElement.LogicalName() &&
				current.Element.Kind() == expectedElement.Kind()
		})
		if len(actualFiltered) > 0 {
			actualInOrder = append(actualInOrder, actualFiltered[0])
		}
	}
	return actualInOrder
}

func inExpectedElements(expectedElements []state.Element) func(*ElementWithAllDeps, int) bool {
	return func(current *ElementWithAllDeps, index int) bool {
		return slices.ContainsFunc(expectedElements, func(expected state.Element) bool {
			return current.Element.LogicalName() == expected.LogicalName() &&
				current.Element.Kind() == expected.Kind()
		})
	}
}

func TestOrderingForRemovalTestSuite(t *testing.T) {
	suite.Run(t, new(OrderingForRemovalTestSuite))
}
