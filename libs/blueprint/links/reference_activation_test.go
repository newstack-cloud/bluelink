package links

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	. "gopkg.in/check.v1"
)

func queueFunctionBlueprint(queueLinkSelector bool) *schema.Blueprint {
	queue := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/sqs/queue"},
	}
	function := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	if queueLinkSelector {
		function.Metadata = &schema.Metadata{
			Labels: &schema.StringMap{Values: map[string]string{"app": "orders"}},
		}
		queue.LinkSelector = &schema.LinkSelector{
			ByLabel: &schema.StringMap{Values: map[string]string{"app": "orders"}},
		}
	}

	return &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"ordersQueue":          queue,
				"processOrderFunction": function,
			},
		},
	}
}

// Mirrors the tags the validation phase attaches when fromResource references
// toResource at spec.<slotPath>.
func collectorWithReference(blueprint *schema.Blueprint, fromName, toName, slotPath string) refgraph.RefChainCollector {
	collector := refgraph.NewRefChainCollector()
	fromElementID := core.ResourceElementID(fromName)
	collector.Collect(
		core.ResourceElementID(toName),
		blueprint.Resources.Values[toName],
		fromElementID,
		[]string{
			fmt.Sprintf("subRef:%s", fromElementID),
			fmt.Sprintf("subRefProp:%s:%s.spec.%s", fromElementID, fromElementID, slotPath),
		},
	)
	return collector
}

func findChainNode(chains []*ChainLinkNode, name string) *ChainLinkNode {
	for _, chain := range chains {
		if found := searchChainNode(chain, name, map[string]bool{}); found != nil {
			return found
		}
	}
	return nil
}

func searchChainNode(node *ChainLinkNode, name string, seen map[string]bool) *ChainLinkNode {
	if node == nil || seen[node.ResourceName] {
		return nil
	}
	seen[node.ResourceName] = true
	if node.ResourceName == name {
		return node
	}
	for _, child := range node.LinksTo {
		if found := searchChainNode(child, name, seen); found != nil {
			return found
		}
	}
	return nil
}

func (s *SpecLinkInfoTestSuite) linksFor(
	c *C,
	blueprint *schema.Blueprint,
	collector refgraph.RefChainCollector,
) []*ChainLinkNode {
	specLinkInfo, err := NewDefaultLinkInfoProvider(
		s.resourceProviders, s.linkRegistry, &testBlueprintSpec{schema: blueprint}, nil, collector,
	)
	if err != nil {
		c.Fatal(err)
	}
	chains, err := specLinkInfo.Links(context.Background())
	if err != nil {
		c.Fatal(err)
	}
	return chains
}

func (s *SpecLinkInfoTestSuite) Test_activates_link_from_reference_at_wiring_slot(c *C) {
	blueprint := queueFunctionBlueprint(false)
	collector := collectorWithReference(blueprint, "ordersQueue", "processOrderFunction", "targetFunctionArn")

	chains := s.linksFor(c, blueprint, collector)

	queue := findChainNode(chains, "ordersQueue")
	c.Assert(queue, NotNil)
	c.Assert(queue.LinksTo, HasLen, 1)
	c.Assert(queue.LinksTo[0].ResourceName, Equals, "processOrderFunction")
	c.Assert(queue.LinkImplementations["processOrderFunction"], NotNil)
}

func (s *SpecLinkInfoTestSuite) Test_does_not_activate_from_reference_at_non_wiring_slot(c *C) {
	blueprint := queueFunctionBlueprint(false)
	// plainValue is not marked with ActivatesLinkOnReference.
	collector := collectorWithReference(blueprint, "ordersQueue", "processOrderFunction", "plainValue")

	chains := s.linksFor(c, blueprint, collector)

	queue := findChainNode(chains, "ordersQueue")
	c.Assert(queue, NotNil)
	c.Assert(queue.LinksTo, HasLen, 0)
}

func (s *SpecLinkInfoTestSuite) Test_does_not_double_activate_with_selector_and_reference(c *C) {
	blueprint := queueFunctionBlueprint(true)
	collector := collectorWithReference(blueprint, "ordersQueue", "processOrderFunction", "targetFunctionArn")

	chains := s.linksFor(c, blueprint, collector)

	queue := findChainNode(chains, "ordersQueue")
	c.Assert(queue, NotNil)
	c.Assert(queue.LinksTo, HasLen, 1)
	c.Assert(queue.LinksTo[0].ResourceName, Equals, "processOrderFunction")
}

func (s *SpecLinkInfoTestSuite) Test_does_not_activate_without_a_registered_link(c *C) {
	// The queue cannot link to a table (no registered link), so a reference at a
	// wiring slot doesn't activate a link.
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"ordersQueue": {Type: &schema.ResourceTypeWrapper{Value: "aws/sqs/queue"}},
				"ordersTable": {Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"}},
			},
		},
	}
	collector := collectorWithReference(blueprint, "ordersQueue", "ordersTable", "targetFunctionArn")

	chains := s.linksFor(c, blueprint, collector)

	queue := findChainNode(chains, "ordersQueue")
	c.Assert(queue, NotNil)
	c.Assert(queue.LinksTo, HasLen, 0)
}

// testRecursiveSpecResource exposes a self-referential "any JSON value" style
// spec definition schema, mirroring transformer resources that accept
// arbitrarily nested input, to guard against unbounded recursion when
// collecting activating slots.
type testRecursiveSpecResource struct {
	testSQSQueueResource
}

func (r *testRecursiveSpecResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	anyValue := &provider.ResourceDefinitionsSchema{}
	anyValue.OneOf = []*provider.ResourceDefinitionsSchema{
		{Type: provider.ResourceDefinitionsSchemaTypeString},
		{
			Type:  provider.ResourceDefinitionsSchemaTypeArray,
			Items: anyValue,
		},
		{
			Type: provider.ResourceDefinitionsSchemaTypeObject,
			Attributes: map[string]*provider.ResourceDefinitionsSchema{
				"value": anyValue,
			},
		},
	}

	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"input": anyValue,
				},
			},
		},
	}, nil
}

func (s *SpecLinkInfoTestSuite) Test_link_loading_terminates_for_self_referential_resource_schema(c *C) {
	awsProvider := &testAWSProvider{
		resources: map[string]provider.Resource{
			"aws/test/recursiveStructure": &testRecursiveSpecResource{},
		},
	}
	resourceProviders := map[string]provider.Provider{
		"aws/test/recursiveStructure": awsProvider,
	}
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"recursiveResource": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/test/recursiveStructure"},
				},
			},
		},
	}

	specLinkInfo, err := NewDefaultLinkInfoProvider(
		resourceProviders,
		provider.NewLinkRegistry(resourceProviders),
		&testBlueprintSpec{schema: blueprint},
		nil,
		refgraph.NewRefChainCollector(),
	)
	if err != nil {
		c.Fatal(err)
	}

	chains, err := specLinkInfo.Links(context.Background())
	c.Assert(err, IsNil)
	c.Assert(chains, HasLen, 1)
	c.Assert(chains[0].ResourceName, Equals, "recursiveResource")
	c.Assert(chains[0].LinksTo, HasLen, 0)
}
