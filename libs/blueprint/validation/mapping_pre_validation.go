package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func preValidateMappingNode(
	ctx context.Context,
	node *core.MappingNode,
	nodeParentType string,
	nodeParentName string,
) []error {
	errors := []error{}
	preValidateMappingNodeRecursive(ctx, node, nodeParentType, nodeParentName, 0, &errors)
	return errors
}

func preValidateMappingNodeRecursive(
	ctx context.Context,
	node *core.MappingNode,
	nodeParentType string,
	nodeParentName string,
	depth int,
	errors *[]error,
) {
	if depth > core.MappingNodeMaxTraverseDepth {
		return
	}

	if node.Fields != nil {
		for key, value := range node.Fields {
			if substitutions.ContainsSubstitution(key) {
				*errors = append(
					*errors,
					errMappingNodeKeyContainsSubstitution(
						key,
						nodeParentType,
						nodeParentName,
						node.SourceMeta,
					),
				)
			}

			if value != nil && (value.Fields != nil || value.Items != nil) {
				preValidateMappingNodeRecursive(
					ctx, value, nodeParentType, nodeParentName, depth+1, errors,
				)
			}
		}
	}

	if node.Items != nil {
		for _, item := range node.Items {
			if item != nil && (item.Fields != nil || item.Items != nil) {
				preValidateMappingNodeRecursive(
					ctx, item, nodeParentType, nodeParentName, depth+1, errors,
				)
			}
		}
	}
}
