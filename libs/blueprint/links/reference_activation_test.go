package links

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
