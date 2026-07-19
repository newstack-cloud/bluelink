package container

import (
	"slices"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
)

// GroupOrderedNodes deals with grouping ordered deployment nodes
// for change staging and deployments to make the process
// more efficient by concurrently staging and deploying unrelated resources
// and child blueprints.
// The input is expected to be an ordered list of deployment nodes.
// The output is a list of groups of deployment nodes that can be staged or deployed
// concurrently, maintaining the order of the provided list for nodes that are
// connected.
func GroupOrderedNodes(
	orderedNodes []*DeploymentNode,
	refChainCollector refgraph.RefChainCollector,
) ([][]*DeploymentNode, error) {
	if len(orderedNodes) == 0 {
		return [][]*DeploymentNode{}, nil
	}

	currentGroupIndex := 0
	groups := [][]*DeploymentNode{{}}
	nodeGroupMap := map[string]int{}

	for _, node := range orderedNodes {
		hasReferenceInCurrentGroup := hasReferenceInGroup(
			node,
			refChainCollector,
			nodeGroupMap,
			currentGroupIndex,
		)

		hasLinkInCurrentGroup := false
		if node.Type() == DeploymentNodeTypeResource {
			hasLinkInCurrentGroup = hasLinkInGroup(
				node.ChainLinkNode,
				nodeGroupMap,
				currentGroupIndex,
			)
		}

		if hasReferenceInCurrentGroup || hasLinkInCurrentGroup {
			currentGroupIndex += 1
			newGroup := []*DeploymentNode{node}
			groups = append(groups, newGroup)
		} else {
			groups[currentGroupIndex] = append(groups[currentGroupIndex], node)
		}

		nodeGroupMap[node.Name()] = currentGroupIndex
	}

	return groups, nil
}

func hasReferenceInGroup(
	node *DeploymentNode,
	refChainCollector refgraph.RefChainCollector,
	nodeGroupMap map[string]int,
	currentGroupIndex int,
) bool {
	refChainNode := refChainCollector.Chain(node.Name())
	if refChainNode == nil {
		return false
	}

	return hasReferenceInGroupTransitively(
		refChainNode,
		nodeGroupMap,
		currentGroupIndex,
		map[string]bool{},
	)
}

// Checks references to deployment nodes in the
// current group both directly and through elements that are not deployed
// themselves (values and data sources). A resource referencing a derived
// value that is defined with a reference to another resource must not share
// a group with that resource, otherwise both are deployed concurrently and
// the derived value resolves before the resource it depends on has been
// deployed.
func hasReferenceInGroupTransitively(
	refChainNode *refgraph.ReferenceChainNode,
	nodeGroupMap map[string]int,
	currentGroupIndex int,
	visited map[string]bool,
) bool {
	for _, reference := range refChainNode.References {
		if visited[reference.ElementName] {
			continue
		}
		visited[reference.ElementName] = true

		referencedByCurrent := slices.ContainsFunc(reference.Tags, func(tag string) bool {
			return tag == validation.CreateSubRefTag(refChainNode.ElementName) ||
				tag == validation.CreateDependencyRefTag(refChainNode.ElementName)
		})
		if !referencedByCurrent {
			continue
		}

		if groupIndex, ok := nodeGroupMap[reference.ElementName]; ok {
			if groupIndex == currentGroupIndex {
				return true
			}
			continue
		}

		// Deployment nodes that are not in the group map yet cannot be
		// traversed through, ordering guarantees they are only encountered
		// here when they are not part of the current deployment.
		if isDeploymentElementName(reference.ElementName) {
			continue
		}

		if hasReferenceInGroupTransitively(
			reference,
			nodeGroupMap,
			currentGroupIndex,
			visited,
		) {
			return true
		}
	}

	return false
}

func hasLinkInGroup(
	node *links.ChainLinkNode,
	nodeGroupMap map[string]int,
	currentGroupIndex int,
) bool {
	linkInGroup := false
	relatedNodes := append(node.LinkedFrom, node.LinksTo...)
	i := 0
	for !linkInGroup && i < len(relatedNodes) {
		relatedNode := relatedNodes[i]
		relatedElementName := bpcore.ResourceElementID(relatedNode.ResourceName)
		if groupIndex, ok := nodeGroupMap[relatedElementName]; ok {
			// Originally, the idea was to only check for hard links in the grouping logic,
			// however, this can create issues where a link is being resolved in staging state
			// prior to the resource changes being applied to the state as the link resolving functionality
			// obtains a lock on the staging state before the resource changes are applied.
			// In change staging, this creates an incorrect set of link changes being reported.
			// To make the process more predictable and less error prone, we have to make sure that
			// two resources that are linked are never in the same group regardless of the link type.
			linkInGroup = groupIndex == currentGroupIndex
		}
		i += 1
	}

	return linkInGroup
}
