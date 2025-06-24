package container

import (
	"context"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// OrderItemsForDeployment deals with creating a flat ordered
// slice of chain link nodes and child blueprints for change staging and deployments.
// Ordering is determined by the priority resource type specified
// in each link implementation and usage of references between resources and child blueprints.
// A reference is treated as a hard link where the priority resource or child blueprint
// is the one being referenced.
//
// It is a requirement for the input chains and child blueprints to not have any direct
// or transitive circular hard links.
// A hard link is when one resource type requires the other in a link
// relationship to be deployed first or where a reference is made to a
// resource or child blueprint.
//
// For the following set of chains and child blueprints:
//
// (lt)                = the linked to resource is the priority resource type.
// (lf)                = the linked from resource is the priority resource type.
// (rb:{referencedBy}) = referenced by other item.
//
// *All the links in the example below are hard links.
//
// Chain 1
// ├── ResourceA1
// │	 ├── ResourceA2 (lf)
// │	 │   ├── ResourceA4 (lt)
// │	 │   └── ResourceA5 (lt)
// │	 └── ResourceA3 (lf)
// │	 	 └── ResourceA6 (lf)
//
// Chain 2
// ├── ResourceB1
// │	 ├── ResourceB2 (lt)
// │	 │   ├── ResourceB4 (lt) (rb:ResourceA5)
// │     │   │   └── ResourceA6 (lt)
// │	 │   └── ResourceB5 (lt) (rb:Child2)
// │	 └── ResourceB3 (lf)
// │	 	 └── ResourceB6 (lt)
//
// Child1 (rb:ResourceA4)
//
// Child2 (rb:ResourceA5)
//
// We will want output like:
// [
//
//		Child1,
//		ResourceA4,
//		ResourceA1,
//		ResourceA3,
//		ResourceA6,
//		ResourceB4,
//		ResourceB5,
//	 	Child2,
//		ResourceA5,
//		ResourceA2,
//		ResourceB2,
//		ResourceB1,
//		ResourceB6,
//		ResourceB3
//
// ]
//
// What matters in the output is that resources are ordered by the priority
// definition of the links and based on references (treated the same as hard links),
// the order of items that have no direct or transitive relationship are irrelevant.
func OrderItemsForDeployment(
	ctx context.Context,
	chains []*links.ChainLinkNode,
	children []*refgraph.ReferenceChainNode,
	refChainCollector refgraph.RefChainCollector,
	params bpcore.BlueprintParams,
) ([]*DeploymentNode, error) {
	flattened := flattenChains(chains, []*links.ChainLinkNode{})
	combined := combineChainsAndChildren(flattened, children)
	refChains := refChainCollector.ChainsByDependencies()
	refChainsDeepCopy := refgraph.DeepCopyReferenceChains(refChains)

	return refgraph.TopologicalSortReferences(
		refChainsDeepCopy,
		combined,
		refgraph.ReferenceSortDirectionReferencedBy,
		/* empty */ nil,
	)
}

func flattenChains(chains []*links.ChainLinkNode, flattenedAccum []*links.ChainLinkNode) []*links.ChainLinkNode {
	flattened := append([]*links.ChainLinkNode{}, flattenedAccum...)
	for _, chain := range chains {
		if !core.SliceContains(flattened, chain) {
			flattened = append(flattened, chain)
			if len(chain.LinksTo) > 0 {
				flattened = flattenChains(chain.LinksTo, flattened)
			}
		}
	}
	return flattened
}

func combineChainsAndChildren(
	chains []*links.ChainLinkNode,
	children []*refgraph.ReferenceChainNode,
) []*DeploymentNode {
	deploymentNodes := []*DeploymentNode{}
	for _, chain := range chains {
		deploymentNodes = append(deploymentNodes, &DeploymentNode{
			ChainLinkNode:      chain,
			DirectDependencies: []*DeploymentNode{},
		})
	}
	for _, child := range children {
		deploymentNodes = append(deploymentNodes, &DeploymentNode{
			ChildNode:          child,
			DirectDependencies: []*DeploymentNode{},
		})
	}
	return deploymentNodes
}

// DeploymentNode is a node that represents a resource or a child blueprint
// to be deployed (or staged for deployment).
type DeploymentNode struct {
	ChainLinkNode *links.ChainLinkNode
	ChildNode     *refgraph.ReferenceChainNode
	// DirectDependencies holds the direct dependencies of the given deployment
	// node.
	// This isn't populated upon creation of the deployment node,
	// as ordering of the nodes does not compare every node,
	// the dependencies list would be incomplete.
	// This is primarily used for the deployment process where the container
	// will populate the direct dependencies of each node as a part of the
	// preparation phase.
	DirectDependencies []*DeploymentNode
}

func (d *DeploymentNode) Name() string {
	if d.ChainLinkNode != nil {
		return bpcore.ResourceElementID(d.ChainLinkNode.ResourceName)
	}
	return d.ChildNode.ElementName
}

func (d *DeploymentNode) Type() DeploymentNodeType {
	if d.ChainLinkNode != nil {
		return DeploymentNodeTypeResource
	}

	if d.ChildNode != nil {
		return DeploymentNodeTypeChild
	}

	return ""
}

// DeploymentNodeType is the type of a deployment node extracted
// from a source blueprint.
type DeploymentNodeType string

const (
	// DeploymentNodeTypeResource is a deployment node that represents a resource
	// to be deployed.
	DeploymentNodeTypeResource DeploymentNodeType = "resource"

	// DeploymentNodeTypeChild is a deployment node that represents a child blueprint
	// to be deployed.
	DeploymentNodeTypeChild DeploymentNodeType = "child"
)
