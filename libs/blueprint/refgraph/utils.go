package refgraph

import "github.com/newstack-cloud/bluelink/libs/common/core"

// DeepCopyReferenceChains deep copies a slice of reference chains.
// This is useful when you want to modify the chains without affecting the original
// (e.g. in a topological sort algorithm).
func DeepCopyReferenceChains(chains []*ReferenceChainNode) []*ReferenceChainNode {
	visited := []*ReferenceChainNode{}
	return deepCopyReferenceChains(chains, &visited)
}

func deepCopyReferenceChains(
	chains []*ReferenceChainNode,
	visited *[]*ReferenceChainNode,
) []*ReferenceChainNode {
	if chains == nil {
		return nil
	}
	copied := make([]*ReferenceChainNode, len(chains))
	for i, node := range chains {
		visitedNode := core.Find(
			*visited,
			func(current *ReferenceChainNode, _ int) bool {
				return current.ElementName == node.ElementName
			},
		)
		if visitedNode != nil {
			// If the node was already visited, copy the reference to the visited node.
			// The visited node will be a copy of the original node, so must be used here
			// instead of the original to ensure no references to the original are kept.
			copied[i] = visitedNode
		} else {
			nodeCopy := &ReferenceChainNode{}
			*visited = append(*visited, nodeCopy)
			deepCopyReferenceChain(node, nodeCopy, visited)
			copied[i] = nodeCopy
		}
	}
	return copied
}

func deepCopyReferenceChain(
	src *ReferenceChainNode,
	dest *ReferenceChainNode,
	visited *[]*ReferenceChainNode,
) {
	if src == nil {
		return
	}
	dest.ElementName = src.ElementName
	dest.Element = src.Element
	dest.References = deepCopyReferenceChains(src.References, visited)
	dest.ReferencedBy = deepCopyReferenceChains(src.ReferencedBy, visited)
	dest.Paths = src.Paths
	dest.Tags = src.Tags
}
