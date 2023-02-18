package links

import (
	"reflect"

	"github.com/freshwebio/celerity/libs/blueprint/pkg/schema"
	. "gopkg.in/check.v1"
)

type LinkUtilsTestSuite struct{}

var _ = Suite(&LinkUtilsTestSuite{})

func (s *LinkUtilsTestSuite) Test_group_resources_by_selector_with_labels_produces_correct_grouping(c *C) {
	groupedResources := GroupResourcesBySelector(&testBlueprintSpec{schema: testBlueprintSchema1})
	assertSelectGroupsEquals(c, groupedResources, expectedGroupedResources1)
}

func (s *LinkUtilsTestSuite) Test_group_resources_by_selector_without_labels_produces_empty_map_with_no_errors(c *C) {
	groupedResources := GroupResourcesBySelector(&testBlueprintSpec{schema: testBlueprintSchema2})
	c.Assert(len(groupedResources), Equals, 0)
}

func assertSelectGroupsEquals(c *C, obtained map[string]*SelectGroup, expected map[string]*SelectGroup) {
	c.Assert(len(obtained), Equals, len(expected))
	for key, obtainedGroup := range obtained {
		expectedGroup, inExpected := expected[key]
		c.Assert(inExpected, Equals, true)
		assertResourcesMatchIgnoreOrder(c, obtainedGroup.SelectorResources, expectedGroup.SelectorResources)
		assertResourcesMatchIgnoreOrder(c, obtainedGroup.CandidateResourcesForSelection, expectedGroup.CandidateResourcesForSelection)
	}
}

func assertResourcesMatchIgnoreOrder(c *C, obtained []*ResourceWithNameAndSelectors, expected []*ResourceWithNameAndSelectors) {
	for _, obtainedResource := range obtained {
		inExpected := false
		i := 0
		for !inExpected && i < len(expected) {

			nameMatches := expected[i].Name == obtainedResource.Name
			selectorsMatch := reflect.DeepEqual(obtainedResource.Selectors, expected[i].Selectors)
			// We are comparing pointers in this case,
			// not the deep structures for resource.Resource!
			inExpected = expected[i].Resource == obtainedResource.Resource && nameMatches && selectorsMatch
			i += 1
		}
		c.Assert(inExpected, Equals, true)
	}
}

var testBlueprintSchema1 = &schema.Blueprint{
	Resources: map[string]*schema.Resource{
		"orderApi": {
			Type: "aws/apigateway",
			Metadata: &schema.Metadata{
				Labels: map[string]string{
					"app": "orderApi",
				},
			},
			LinkSelector: &schema.LinkSelector{
				ByLabel: map[string]string{
					"app": "orderApi",
				},
			},
		},
		"orderQueue": {
			Type: "aws/sqs/queue",
			Metadata: &schema.Metadata{
				Labels: map[string]string{
					"app": "orderWorkflow",
				},
			},
			LinkSelector: &schema.LinkSelector{
				ByLabel: map[string]string{
					"app": "orderWorkflow",
				},
			},
		},
		"processOrdersFunction": {
			Type: "aws/lambda/function",
			Metadata: &schema.Metadata{
				Labels: map[string]string{
					"app": "orderWorkflow",
				},
			},
			LinkSelector: &schema.LinkSelector{
				ByLabel: map[string]string{
					"system": "orders",
				},
			},
		},
		"createOrderFunction": {
			Type: "aws/lambda/function",
			Metadata: &schema.Metadata{
				Labels: map[string]string{
					"app": "orderApi",
				},
			},
			LinkSelector: &schema.LinkSelector{
				ByLabel: map[string]string{
					"system": "orders",
				},
			},
		},
		"getOrdersFunction": {
			Type: "aws/lambda/function",
			Metadata: &schema.Metadata{
				Labels: map[string]string{
					"app": "orderApi",
				},
			},
			LinkSelector: &schema.LinkSelector{
				ByLabel: map[string]string{
					"system": "orders",
				},
			},
		},
		"ordersTable": {
			Type: "aws/dynamodb/table",
			Metadata: &schema.Metadata{
				Labels: map[string]string{
					"system": "orders",
				},
			},
		},
	},
}

var expectedGroupedResources1 = map[string]*SelectGroup{
	"label::app:orderApi": {
		SelectorResources: []*ResourceWithNameAndSelectors{
			{
				Name:      "orderApi",
				Resource:  testBlueprintSchema1.Resources["orderApi"],
				Selectors: []string{"label::app:orderApi"},
			},
		},
		CandidateResourcesForSelection: []*ResourceWithNameAndSelectors{
			// orderApi is also a candidate for selection as it has the app:orderApi label.
			{
				Name:      "orderApi",
				Resource:  testBlueprintSchema1.Resources["orderApi"],
				Selectors: []string{"label::app:orderApi"},
			},
			{
				Name:      "createOrderFunction",
				Resource:  testBlueprintSchema1.Resources["createOrderFunction"],
				Selectors: []string{"label::system:orders"},
			},
			{
				Name:      "getOrdersFunction",
				Resource:  testBlueprintSchema1.Resources["getOrdersFunction"],
				Selectors: []string{"label::system:orders"},
			},
		},
	},
	"label::app:orderWorkflow": {
		SelectorResources: []*ResourceWithNameAndSelectors{
			{
				Name:      "orderQueue",
				Resource:  testBlueprintSchema1.Resources["orderQueue"],
				Selectors: []string{"label::app:orderWorkflow"},
			},
		},
		CandidateResourcesForSelection: []*ResourceWithNameAndSelectors{
			// orderQueue is also a candidate for selection as it has the app:orderWorkflow label.
			{
				Name:      "orderQueue",
				Resource:  testBlueprintSchema1.Resources["orderQueue"],
				Selectors: []string{"label::app:orderWorkflow"},
			},
			{
				Name:      "processOrdersFunction",
				Resource:  testBlueprintSchema1.Resources["processOrdersFunction"],
				Selectors: []string{"label::system:orders"},
			},
		},
	},
	"label::system:orders": {
		SelectorResources: []*ResourceWithNameAndSelectors{
			{
				Name:      "createOrderFunction",
				Resource:  testBlueprintSchema1.Resources["createOrderFunction"],
				Selectors: []string{"label::system:orders"},
			},
			{
				Name:      "getOrdersFunction",
				Resource:  testBlueprintSchema1.Resources["getOrdersFunction"],
				Selectors: []string{"label::system:orders"},
			},
			{
				Name:      "processOrdersFunction",
				Resource:  testBlueprintSchema1.Resources["processOrdersFunction"],
				Selectors: []string{"label::system:orders"},
			},
		},
		CandidateResourcesForSelection: []*ResourceWithNameAndSelectors{
			{
				Name:      "ordersTable",
				Resource:  testBlueprintSchema1.Resources["ordersTable"],
				Selectors: []string{},
			},
		},
	},
}

var testBlueprintSchema2 = &schema.Blueprint{
	Resources: map[string]*schema.Resource{
		"orderApi": {
			Type: "aws/apigateway",
		},
		"orderQueue": {
			Type: "aws/sqs/queue",
		},
		"processOrdersFunction": {
			Type: "aws/lambda/function",
		},
		"createOrderFunction": {
			Type: "aws/lambda/function",
		},
		"getOrdersFunction": {
			Type: "aws/lambda/function",
		},
		"ordersTable": {
			Type: "aws/dynamodb/table",
		},
	},
}
