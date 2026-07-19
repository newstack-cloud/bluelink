package container

import (
	"context"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// PopulateDirectDependencies checks the relationships between deployment nodes
// and populates the dependencies between them.
// This should only need to be computed once per blueprint deployment
// where the dependency information can then be used to determine
// which elements to be deploy after others have completed.
// This only populates direct dependencies between nodes as the nodes are expected
// to be ordered and grouped in pools that also take transitive dependencies into account
// in the deployment process.
func PopulateDirectDependencies(
	ctx context.Context,
	allNodes []*DeploymentNode,
	refChainCollector refgraph.RefChainCollector,
	params bpcore.BlueprintParams,
) error {
	for _, possibleDependency := range allNodes {
		for _, node := range allNodes {
			if possibleDependency.Name() != node.Name() {
				dependsOn, err := checkDependency(
					ctx,
					node,
					possibleDependency,
					refChainCollector,
					params,
				)
				if err != nil {
					return err
				}

				if dependsOn {
					node.DirectDependencies = append(node.DirectDependencies, possibleDependency)
				}
			}
		}
	}

	return nil
}

func checkDependency(
	ctx context.Context,
	dependent *DeploymentNode,
	possibleDependency *DeploymentNode,
	refChainCollector refgraph.RefChainCollector,
	params bpcore.BlueprintParams,
) (bool, error) {
	if possibleDependency.Type() == DeploymentNodeTypeResource {
		return checkHasDependencyOnResource(
			ctx,
			dependent,
			possibleDependency.ChainLinkNode.ResourceName,
			refChainCollector,
			params,
		)
	}

	return checkHasDependencyOnChildBlueprint(
		dependent,
		bpcore.ToLogicalChildName(possibleDependency.Name()),
		refChainCollector,
	)
}

func checkHasDependencyOnResource(
	ctx context.Context,
	node *DeploymentNode,
	dependsOnResourceName string,
	refChainCollector refgraph.RefChainCollector,
	params bpcore.BlueprintParams,
) (bool, error) {
	if node.Type() == DeploymentNodeTypeResource {
		linksTo := node.ChainLinkNode.LinksTo
		linksToDependencyNode := core.Find(
			linksTo,
			func(node *links.ChainLinkNode, _ int) bool {
				return node.ResourceName == dependsOnResourceName
			},
		)
		if linksToDependencyNode != nil {
			hasPriority, err := linkedToResourceHasPriority(
				ctx,
				node.ChainLinkNode,
				dependsOnResourceName,
				provider.LinkPriorityResourceB,
				params,
			)
			if err != nil || hasPriority {
				return hasPriority, err
			}
			// A link relationship without the dependency as the priority
			// resource does not determine ordering on its own, fall through
			// to the reference check as the node may still reference the
			// linked resource (e.g. a link activated by a reference to a
			// property of the linked resource).
		}

		linkedFrom := node.ChainLinkNode.LinkedFrom
		linkedFromDependencyNode := core.Find(
			linkedFrom,
			func(node *links.ChainLinkNode, _ int) bool {
				return node.ResourceName == dependsOnResourceName
			},
		)
		if linkedFromDependencyNode != nil {
			hasPriority, err := linkedToResourceHasPriority(
				ctx,
				linkedFromDependencyNode,
				node.ChainLinkNode.ResourceName,
				provider.LinkPriorityResourceA,
				params,
			)
			if err != nil || hasPriority {
				return hasPriority, err
			}
			// Same as above, fall through to the reference check when the
			// link priority does not make the linked resource a dependency.
		}
	}

	dependsOnElementName := bpcore.ResourceElementID(dependsOnResourceName)
	return nodeReferencesElement(node, refChainCollector, dependsOnElementName)
}

func checkHasDependencyOnChildBlueprint(
	node *DeploymentNode,
	dependsOnChildName string,
	refChainCollector refgraph.RefChainCollector,
) (bool, error) {
	dependsOnElementName := bpcore.ChildElementID(dependsOnChildName)
	return nodeReferencesElement(node, refChainCollector, dependsOnElementName)
}

func nodeReferencesElement(
	node *DeploymentNode,
	refChainCollector refgraph.RefChainCollector,
	dependsOnElementName string,
) (bool, error) {
	refChainNode := getRefChainNode(node, refChainCollector)
	if refChainNode == nil {
		return false, nil
	}

	return referencesElementThroughDerivedValues(
		refChainNode,
		dependsOnElementName,
		map[string]bool{},
	), nil
}

func referencesElementThroughDerivedValues(
	refChainNode *refgraph.ReferenceChainNode,
	dependsOnElementName string,
	visited map[string]bool,
) bool {
	for _, reference := range refChainNode.References {
		if reference.ElementName == dependsOnElementName {
			return true
		}
		if visited[reference.ElementName] {
			continue
		}
		visited[reference.ElementName] = true

		// References to other resources or child blueprints are dependency
		// edges in their own right, ordering through them is handled by
		// their own dependency relationships.
		if isDeploymentElementName(reference.ElementName) {
			continue
		}
		if referencesElementThroughDerivedValues(reference, dependsOnElementName, visited) {
			return true
		}
	}

	return false
}

func isDeploymentElementName(elementName string) bool {
	return strings.HasPrefix(elementName, "resources.") ||
		strings.HasPrefix(elementName, "children.")
}

func getRefChainNode(
	node *DeploymentNode,
	refChainCollector refgraph.RefChainCollector,
) *refgraph.ReferenceChainNode {
	if node.Type() == DeploymentNodeTypeChild {
		return node.ChildNode
	}

	return refChainCollector.Chain(node.Name())
}

func linkedToResourceHasPriority(
	ctx context.Context,
	chainLinkNode *links.ChainLinkNode,
	linksToResourceName string,
	dependencyResourcePriority provider.LinkPriorityResource,
	params bpcore.BlueprintParams,
) (bool, error) {
	linkImpl, hasLinkImpl := chainLinkNode.LinkImplementations[linksToResourceName]
	if hasLinkImpl {
		linkCtx := provider.NewLinkContextFromParams(params)
		priorityOutput, err := linkImpl.GetPriorityResource(
			ctx,
			&provider.LinkGetPriorityResourceInput{
				LinkContext: linkCtx,
			},
		)
		if err != nil {
			return false, err
		}

		return priorityOutput.PriorityResource == dependencyResourcePriority, nil
	}

	return false, nil
}
