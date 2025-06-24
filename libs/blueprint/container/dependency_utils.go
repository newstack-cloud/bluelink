package container

import (
	"context"

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
			return linkedToResourceHasPriority(
				ctx,
				node.ChainLinkNode,
				dependsOnResourceName,
				provider.LinkPriorityResourceB,
				params,
			)
		}

		linkedFrom := node.ChainLinkNode.LinkedFrom
		linkedFromDependencyNode := core.Find(
			linkedFrom,
			func(node *links.ChainLinkNode, _ int) bool {
				return node.ResourceName == dependsOnResourceName
			},
		)
		if linkedFromDependencyNode != nil {
			return linkedToResourceHasPriority(
				ctx,
				linkedFromDependencyNode,
				node.ChainLinkNode.ResourceName,
				provider.LinkPriorityResourceA,
				params,
			)
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

	if refChainNode != nil {
		referencedChainNode := core.Find(
			refChainNode.References,
			func(node *refgraph.ReferenceChainNode, _ int) bool {
				return node.ElementName == dependsOnElementName
			},
		)
		if referencedChainNode != nil {
			return true, nil
		}
	}

	return false, nil
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
