package schema

import (
	"fmt"
	"slices"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// TreeNode is a tree representation of a blueprint
// schema that is in sequential source order, from left to right.
// As a blueprint is made up of an augmentation of a host document
// language (e.g. YAML) and the embedded substitution language, there
// is no natural AST or AST-like representation of a blueprint.
//
// TreeNode is derived from a parsed schema to provide a sequential representation
// of elements in a blueprint that makes it easier to determine accurate ranges
// for diagnostics, completions and other language server features.
//
// The unordered maps used in the schema structs are not suitable for
// efficiently determining where elements start and end which is essential for tools
// such as language servers.
//
// A tree should only be used when source location information is made available
// by the host document language parser. (e.g. YAML)
type TreeNode struct {
	// Label is the label of the tree node.
	// For a node that represents a named field in a schema element,
	// the label is the name of the field. (e.g. "variables" or "resources")
	// For a node that represents an element in a list, the label is the index
	// of the element in the list.
	// For a node that represents an element in a mapping, the label is the key
	// of the element in the mapping.
	Label string
	// Path contains the path to the tree node relative to the root.
	// For example, the path to a variable type node could be
	// "/variables/myVar/type".
	// "/" is used as the separator as "." can be in names of elements.
	// The path is made up of the labels of the nodes in the tree.
	Path string
	// Type is the type of the tree node.
	Type TreeNodeType
	// Children is a list of child nodes in the tree.
	Children []*TreeNode
	// SchemaElement is the schema element that this tree node represents.
	SchemaElement interface{}
	// Range is the source range of the tree node in the blueprint source document.
	Range *source.Range
}

// SetRangeEnd sets the range end of a tree node and the last child.
// This is applied recursively to the last child of the last child and so on.
func (n *TreeNode) SetRangeEnd(end *source.Meta) {
	n.Range = &source.Range{
		Start: n.Range.Start,
		End:   &end.Position,
	}

	if len(n.Children) > 0 {
		n.Children[len(n.Children)-1].SetRangeEnd(end)
	}
}

// TreeNodeType is the type of a tree node.
type TreeNodeType int

const (
	// TreeNodeTypeNonTerminal is a non-terminal node of the tree
	// of a blueprint.
	TreeNodeTypeNonTerminal TreeNodeType = iota
	// TreeNodeTypeLeaf is a leaf node of the tree
	// of a blueprint.
	TreeNodeTypeLeaf
)

// SchemaToTree converts a blueprint schema to a tree representation.
func SchemaToTree(blueprint *Blueprint) *TreeNode {
	if blueprint == nil {
		return nil
	}

	root := &TreeNode{
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: blueprint,
	}

	children := []*TreeNode{}

	versionNode := versionToTreeNode(blueprint.Version, root.Path)
	if versionNode != nil {
		children = append(children, versionNode)
	}

	transformNode := transformToTreeNode(blueprint.Transform, root.Path)
	if transformNode != nil {
		children = append(children, transformNode)
	}

	variablesNode := variablesToTreeNode(blueprint.Variables, root.Path)
	if variablesNode != nil {
		children = append(children, variablesNode)
	}

	valuesNode := valuesToTreeNode(blueprint.Values, root.Path)
	if valuesNode != nil {
		children = append(children, valuesNode)
	}

	includesNode := includesToTreeNode(blueprint.Include, root.Path)
	if includesNode != nil {
		children = append(children, includesNode)
	}

	resourcesNode := resourcesToTreeNode(blueprint.Resources, root.Path)
	if resourcesNode != nil {
		children = append(children, resourcesNode)
	}

	dataSourcesNode := dataSourcesToTreeNode(blueprint.DataSources, root.Path)
	if dataSourcesNode != nil {
		children = append(children, dataSourcesNode)
	}

	exportsNode := exportsToTreeNode(blueprint.Exports, root.Path)
	if exportsNode != nil {
		children = append(children, exportsNode)
	}

	metadataNode := mappingNodeToTreeNode("metadata", blueprint.Metadata, root.Path, nil)
	if metadataNode != nil {
		children = append(children, metadataNode)
	}

	sortTreeNodes(children)
	root.Children = children

	root.Range = &source.Range{
		Start: &source.Position{
			Line:   1,
			Column: 1,
		},
		// Root node doesn't have an end location, the range for the very last leaf node
		// will extend to the end of the document.
	}

	return root
}

func versionToTreeNode(version *bpcore.ScalarValue, parentPath string) *TreeNode {
	if version == nil {
		return nil
	}

	versionNode := &TreeNode{
		Label:         "version",
		Path:          fmt.Sprintf("%s/version", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: version,
		Range: &source.Range{
			Start: &version.SourceMeta.Position,
			End:   version.SourceMeta.EndPosition,
		},
	}

	return versionNode
}

func transformToTreeNode(transform *TransformValueWrapper, parentPath string) *TreeNode {
	if transform == nil || len(transform.Values) == 0 {
		return nil
	}

	transformNode := &TreeNode{
		Label:         "transform",
		Path:          fmt.Sprintf("%s/transform", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: transform,
		Range: &source.Range{
			Start: &transform.SourceMeta[0].Position,
		},
	}

	children := make([]*TreeNode, len(transform.Values))
	for i, value := range transform.Values {
		child := &TreeNode{
			Label:         fmt.Sprintf("%d", i),
			Path:          fmt.Sprintf("%s/transform/%d", parentPath, i),
			Type:          TreeNodeTypeLeaf,
			SchemaElement: value,
			Range: &source.Range{
				Start: &transform.SourceMeta[i].Position,
				End:   transform.SourceMeta[i].EndPosition,
			},
		}
		children[i] = child
	}
	sortTreeNodes(children)
	transformNode.Children = children
	transformNode.Range.End = children[len(children)-1].Range.End

	return transformNode
}

func variablesToTreeNode(variables *VariableMap, parentPath string) *TreeNode {
	if variables == nil || len(variables.Values) == 0 {
		return nil
	}

	variablesNode := &TreeNode{
		Label:         "variables",
		Path:          fmt.Sprintf("%s/variables", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: variables,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(variables.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for varName, variable := range variables.Values {
		variableNode := variableToTreeNode(
			varName,
			variable,
			variablesNode.Path,
			variables.SourceMeta[varName],
		)
		if variableNode != nil {
			children = append(children, variableNode)
		}
	}
	sortTreeNodes(children)
	variablesNode.Children = children
	variablesNode.Range.End = children[len(children)-1].Range.End

	return variablesNode
}

func variableToTreeNode(varName string, variable *Variable, parentPath string, location *source.Meta) *TreeNode {
	variableNode := &TreeNode{
		Label:         varName,
		Path:          fmt.Sprintf("%s/%s", parentPath, varName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: variable,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	varTypeNode := variableTypeToTreeNode(variable.Type, variableNode.Path)
	if varTypeNode != nil {
		children = append(children, varTypeNode)
	}

	descriptionNode := scalarToTreeNode(
		"description",
		variable.Description,
		variableNode.Path,
	)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	secretNode := scalarToTreeNode(
		"secret",
		variable.Secret,
		variableNode.Path,
	)
	if secretNode != nil {
		children = append(children, secretNode)
	}

	defaultNode := scalarToTreeNode(
		"default",
		variable.Default,
		variableNode.Path,
	)
	if defaultNode != nil {
		children = append(children, defaultNode)
	}

	allowedValuesNode := scalarsToTreeNode(
		"allowedValues",
		variable.AllowedValues,
		variableNode.Path,
	)
	if allowedValuesNode != nil {
		children = append(children, allowedValuesNode)
	}

	sortTreeNodes(children)
	variableNode.Children = children
	variableNode.Range.End = children[len(children)-1].Range.End

	return variableNode
}

func variableTypeToTreeNode(varType *VariableTypeWrapper, parentPath string) *TreeNode {
	if varType == nil {
		return nil
	}

	return &TreeNode{
		Label:         "type",
		Path:          fmt.Sprintf("%s/type", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: varType,
		Range: &source.Range{
			Start: &varType.SourceMeta.Position,
			End:   varType.SourceMeta.EndPosition,
		},
	}
}

func valuesToTreeNode(values *ValueMap, parentPath string) *TreeNode {
	if values == nil || len(values.Values) == 0 {
		return nil
	}

	valuesNode := &TreeNode{
		Label:         "values",
		Path:          fmt.Sprintf("%s/values", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: values,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(values.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for valName, val := range values.Values {
		valNode := valueToTreeNode(
			valName,
			val,
			valuesNode.Path,
			values.SourceMeta[valName],
		)
		if valNode != nil {
			children = append(children, valNode)
		}
	}
	sortTreeNodes(children)
	valuesNode.Children = children
	valuesNode.Range.End = children[len(children)-1].Range.End

	return valuesNode
}

func valueToTreeNode(valName string, value *Value, parentPath string, location *source.Meta) *TreeNode {
	valueNode := &TreeNode{
		Label:         valName,
		Path:          fmt.Sprintf("%s/%s", parentPath, valName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: value,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	valTypeNode := valueTypToTreeNode(value.Type, valueNode.Path)
	if valTypeNode != nil {
		children = append(children, valTypeNode)
	}

	contentNode := mappingNodeToTreeNode(
		"value",
		value.Value,
		valueNode.Path,
		nil,
	)
	if contentNode != nil {
		children = append(children, contentNode)
	}

	descriptionNode := stringSubsToTreeNode(
		"description",
		value.Description,
		valueNode.Path,
	)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	secretNode := scalarToTreeNode(
		"secret",
		value.Secret,
		valueNode.Path,
	)
	if secretNode != nil {
		children = append(children, secretNode)
	}

	sortTreeNodes(children)
	valueNode.Children = children
	valueNode.Range.End = children[len(children)-1].Range.End

	return valueNode
}

func valueTypToTreeNode(valType *ValueTypeWrapper, parentPath string) *TreeNode {
	if valType == nil {
		return nil
	}

	return &TreeNode{
		Label:         "type",
		Path:          fmt.Sprintf("%s/type", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: valType,
		Range: &source.Range{
			Start: &valType.SourceMeta.Position,
			End:   valType.SourceMeta.EndPosition,
		},
	}
}

func includesToTreeNode(includes *IncludeMap, parentPath string) *TreeNode {
	if includes == nil || len(includes.Values) == 0 {
		return nil
	}

	includesNode := &TreeNode{
		Label:         "includes",
		Path:          fmt.Sprintf("%s/includes", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: includes,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(includes.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for includeName, val := range includes.Values {
		includeNode := includeToTreeNode(
			includeName,
			val,
			includesNode.Path,
			includes.SourceMeta[includeName],
		)
		if includeNode != nil {
			children = append(children, includeNode)
		}
	}
	sortTreeNodes(children)
	includesNode.Children = children
	includesNode.Range.End = children[len(children)-1].Range.End

	return includesNode
}

func includeToTreeNode(includeName string, include *Include, parentPath string, location *source.Meta) *TreeNode {
	includeNode := &TreeNode{
		Label:         includeName,
		Path:          fmt.Sprintf("%s/%s", parentPath, includeName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: include,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	pathNode := stringSubsToTreeNode("path", include.Path, includeNode.Path)
	if pathNode != nil {
		children = append(children, pathNode)
	}

	variablesNode := mappingNodeToTreeNode("variables", include.Variables, includeNode.Path, nil)
	if variablesNode != nil {
		children = append(children, variablesNode)
	}

	metadataNode := mappingNodeToTreeNode("metadata", include.Metadata, includeNode.Path, nil)
	if metadataNode != nil {
		children = append(children, metadataNode)
	}

	descriptionNode := stringSubsToTreeNode("description", include.Description, includeNode.Path)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	sortTreeNodes(children)
	includeNode.Children = children
	includeNode.Range.End = children[len(children)-1].Range.End

	return includeNode
}

func resourcesToTreeNode(resources *ResourceMap, parentPath string) *TreeNode {
	if resources == nil || len(resources.Values) == 0 {
		return nil
	}

	resourcesNode := &TreeNode{
		Label:         "resources",
		Path:          fmt.Sprintf("%s/resources", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: resources,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(resources.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for resourceName, val := range resources.Values {
		resourceNode := resourceToTreeNode(
			resourceName,
			val,
			resourcesNode.Path,
			resources.SourceMeta[resourceName],
		)
		if resourceNode != nil {
			children = append(children, resourceNode)
		}
	}
	sortTreeNodes(children)
	resourcesNode.Children = children
	resourcesNode.Range.End = children[len(children)-1].Range.End

	return resourcesNode
}

func resourceToTreeNode(resourceName string, resource *Resource, parentPath string, location *source.Meta) *TreeNode {
	resourceNode := &TreeNode{
		Label:         resourceName,
		Path:          fmt.Sprintf("%s/%s", parentPath, resourceName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: resource,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	resourceTypeNode := resourceTypeToTreeNode(resource.Type, resourceNode.Path)
	if resourceTypeNode != nil {
		children = append(children, resourceTypeNode)
	}

	descriptionNode := stringSubsToTreeNode("description", resource.Description, resourceNode.Path)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	metadataNode := resourceMetadataToTreeNode(resource.Metadata, resourceNode.Path)
	if metadataNode != nil {
		children = append(children, metadataNode)
	}

	conditionNode := resourceConditionToTreeNode("condition", resource.Condition, resourceNode.Path)
	if conditionNode != nil {
		children = append(children, conditionNode)
	}

	eachNode := stringSubsToTreeNode("each", resource.Each, resourceNode.Path)
	if eachNode != nil {
		children = append(children, eachNode)
	}

	linkSelectorNode := resourceLinkSelectorToTreeNode(resource.LinkSelector, resourceNode.Path)
	if linkSelectorNode != nil {
		children = append(children, linkSelectorNode)
	}

	specNode := mappingNodeToTreeNode("spec", resource.Spec, resourceNode.Path, nil)
	if specNode != nil {
		children = append(children, specNode)
	}

	sortTreeNodes(children)
	resourceNode.Children = children
	resourceNode.Range.End = children[len(children)-1].Range.End

	return resourceNode
}

func resourceLinkSelectorToTreeNode(linkSelector *LinkSelector, parentPath string) *TreeNode {
	if linkSelector == nil {
		return nil
	}

	linkSelectorNode := &TreeNode{
		Label:         "linkSelector",
		Path:          fmt.Sprintf("%s/linkSelector", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: linkSelector,
		Range: &source.Range{
			Start: &linkSelector.SourceMeta.Position,
		},
	}

	if linkSelector.ByLabel == nil {
		linkSelectorNode.Type = TreeNodeTypeLeaf
		return linkSelectorNode
	}

	byLabelNode := stringMapToTreeNode("byLabel", linkSelector.ByLabel, linkSelectorNode.Path)
	if byLabelNode != nil {
		linkSelectorNode.Children = []*TreeNode{byLabelNode}
		linkSelectorNode.Range.End = byLabelNode.Range.End
	} else {
		linkSelectorNode.Type = TreeNodeTypeLeaf
	}

	return linkSelectorNode
}

func resourceConditionToTreeNode(label string, condition *Condition, parentPath string) *TreeNode {
	if condition == nil {
		return nil
	}

	conditionNode := &TreeNode{
		Label:         label,
		Path:          fmt.Sprintf("%s/%s", parentPath, label),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: condition,
		Range: &source.Range{
			Start: &condition.SourceMeta.Position,
		},
	}

	children := []*TreeNode{}

	if condition.StringValue != nil {
		exprNode := stringSubsToTreeNode("expr", condition.StringValue, conditionNode.Path)
		if exprNode != nil {
			children = append(children, exprNode)
		}
	}

	if condition.And != nil {
		andNode := conditionsListToTreeNode("and", condition.And, conditionNode.Path)
		if andNode != nil {
			children = append(children, andNode)
		}
	}

	if condition.Or != nil {
		orNode := conditionsListToTreeNode("or", condition.Or, conditionNode.Path)
		if orNode != nil {
			children = append(children, orNode)
		}
	}

	if condition.Not != nil {
		notNode := notConditionToTreeNode(condition.Not, conditionNode.Path)
		if notNode != nil {
			children = append(children, notNode)
		}
	}

	sortTreeNodes(children)
	conditionNode.Children = children
	conditionNode.Range.End = children[len(children)-1].Range.End

	return conditionNode
}

func conditionsListToTreeNode(label string, conditions []*Condition, parentPath string) *TreeNode {
	if len(conditions) == 0 {
		return nil
	}

	condListNode := &TreeNode{
		Label:         label,
		Path:          fmt.Sprintf("%s/%s", parentPath, label),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: conditions,
		Range: &source.Range{
			Start: &conditions[0].SourceMeta.Position,
		},
	}

	children := make([]*TreeNode, len(conditions))
	for i, cond := range conditions {
		child := resourceConditionToTreeNode(fmt.Sprintf("%d", i), cond, condListNode.Path)
		if child != nil {
			children[i] = child
		}
	}

	sortTreeNodes(children)
	condListNode.Children = children
	condListNode.Range.End = children[len(children)-1].Range.End

	return condListNode
}

func notConditionToTreeNode(toNegate *Condition, parentPath string) *TreeNode {
	if toNegate == nil {
		return nil
	}

	notNode := &TreeNode{
		Label:         "not",
		Path:          fmt.Sprintf("%s/not", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: toNegate,
		Range: &source.Range{
			Start: &toNegate.SourceMeta.Position,
		},
	}

	child := resourceConditionToTreeNode("0", toNegate, notNode.Path)
	if child != nil {
		notNode.Children = []*TreeNode{child}
		notNode.Range.End = child.Range.End
	}

	return notNode
}

func resourceTypeToTreeNode(resType *ResourceTypeWrapper, parentPath string) *TreeNode {
	if resType == nil {
		return nil
	}

	return &TreeNode{
		Label:         "type",
		Path:          fmt.Sprintf("%s/type", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: resType,
		Range: &source.Range{
			Start: &resType.SourceMeta.Position,
			End:   resType.SourceMeta.EndPosition,
		},
	}
}

func resourceMetadataToTreeNode(metadata *Metadata, parentPath string) *TreeNode {
	if metadata == nil {
		return nil
	}

	metadataNode := &TreeNode{
		Label:         "metadata",
		Path:          fmt.Sprintf("%s/metadata", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: metadata,
		Range: &source.Range{
			Start: &metadata.SourceMeta.Position,
		},
	}

	children := []*TreeNode{}

	displayNameNode := stringSubsToTreeNode("displayName", metadata.DisplayName, metadataNode.Path)
	if displayNameNode != nil {
		children = append(children, displayNameNode)
	}

	annotationsNode := stringSubsMapToTreeNode("annotations", metadata.Annotations, metadataNode.Path)
	if annotationsNode != nil {
		children = append(children, annotationsNode)
	}

	labelsNode := stringMapToTreeNode("labels", metadata.Labels, metadataNode.Path)
	if labelsNode != nil {
		children = append(children, labelsNode)
	}

	customNode := mappingNodeToTreeNode("custom", metadata.Custom, metadataNode.Path, nil)
	if customNode != nil {
		children = append(children, customNode)
	}

	sortTreeNodes(children)
	metadataNode.Children = children
	metadataNode.Range.End = children[len(children)-1].Range.End

	return metadataNode
}

func dataSourcesToTreeNode(dataSources *DataSourceMap, parentPath string) *TreeNode {
	if dataSources == nil || len(dataSources.Values) == 0 {
		return nil
	}

	dataSourcesNode := &TreeNode{
		Label:         "datasources",
		Path:          fmt.Sprintf("%s/datasources", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: dataSources,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(dataSources.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for dataSourceName, val := range dataSources.Values {
		dataSourceNode := dataSourceToTreeNode(
			dataSourceName,
			val,
			dataSourcesNode.Path,
			dataSources.SourceMeta[dataSourceName],
		)
		if dataSourceNode != nil {
			children = append(children, dataSourceNode)
		}
	}
	sortTreeNodes(children)
	dataSourcesNode.Children = children
	dataSourcesNode.Range.End = children[len(children)-1].Range.End

	return dataSourcesNode
}

func dataSourceToTreeNode(dataSourceName string, dataSource *DataSource, parentPath string, location *source.Meta) *TreeNode {
	dataSourceNode := &TreeNode{
		Label:         dataSourceName,
		Path:          fmt.Sprintf("%s/%s", parentPath, dataSourceName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: dataSource,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	dataSourceTypeNode := dataSourceTypeToTreeNode(dataSource.Type, dataSourceNode.Path)
	if dataSourceTypeNode != nil {
		children = append(children, dataSourceTypeNode)
	}

	dataSourceMetadataNode := dataSourceMetadataToTreeNode(dataSource.DataSourceMetadata, dataSourceNode.Path)
	if dataSourceMetadataNode != nil {
		children = append(children, dataSourceMetadataNode)
	}

	dataSourceFilterNode := dataSourceFiltersToTreeNode(dataSource.Filter, dataSourceNode.Path)
	if dataSourceFilterNode != nil {
		children = append(children, dataSourceFilterNode)
	}

	dataSourceFieldExportsNode := dataSourceFieldExportsToTreeNode(dataSource.Exports, dataSourceNode.Path)
	if dataSourceFieldExportsNode != nil {
		children = append(children, dataSourceFieldExportsNode)
	}

	descriptionNode := stringSubsToTreeNode("description", dataSource.Description, dataSourceNode.Path)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	sortTreeNodes(children)
	dataSourceNode.Children = children
	dataSourceNode.Range.End = children[len(children)-1].Range.End

	return dataSourceNode
}

func dataSourceTypeToTreeNode(dataSourceType *DataSourceTypeWrapper, parentPath string) *TreeNode {
	if dataSourceType == nil {
		return nil
	}

	return &TreeNode{
		Label:         "type",
		Path:          fmt.Sprintf("%s/type", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: dataSourceType,
		Range: &source.Range{
			Start: &dataSourceType.SourceMeta.Position,
			End:   dataSourceType.SourceMeta.EndPosition,
		},
	}
}

func dataSourceMetadataToTreeNode(metadata *DataSourceMetadata, parentPath string) *TreeNode {
	if metadata == nil {
		return nil
	}

	metadataNode := &TreeNode{
		Label:         "metadata",
		Path:          fmt.Sprintf("%s/metadata", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: metadata,
		Range: &source.Range{
			Start: &metadata.SourceMeta.Position,
		},
	}

	children := []*TreeNode{}

	displayNameNode := stringSubsToTreeNode("displayName", metadata.DisplayName, metadataNode.Path)
	if displayNameNode != nil {
		children = append(children, displayNameNode)
	}

	annotationsNode := stringSubsMapToTreeNode("annotations", metadata.Annotations, metadataNode.Path)
	if annotationsNode != nil {
		children = append(children, annotationsNode)
	}

	customNode := mappingNodeToTreeNode("custom", metadata.Custom, metadataNode.Path, nil)
	if customNode != nil {
		children = append(children, customNode)
	}

	sortTreeNodes(children)
	metadataNode.Children = children
	metadataNode.Range.End = children[len(children)-1].Range.End

	return metadataNode
}

func dataSourceFiltersToTreeNode(filters *DataSourceFilters, parentPath string) *TreeNode {
	if filters == nil || len(filters.Filters) == 0 {
		return nil
	}

	filtersNode := &TreeNode{
		Label:         "filters",
		Path:          fmt.Sprintf("%s/filters", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: filters,
		Range: &source.Range{
			Start: &filters.Filters[0].SourceMeta.Position,
		},
	}

	children := []*TreeNode{}

	for i, filter := range filters.Filters {
		child := dataSourceFilterToTreeNode(filter, fmt.Sprintf("%s/%d", filtersNode.Path, i))
		if child != nil {
			children = append(children, child)
		}
	}

	sortTreeNodes(children)
	filtersNode.Children = children
	filtersNode.Range.End = children[len(children)-1].Range.End

	return filtersNode
}

func dataSourceFilterToTreeNode(filter *DataSourceFilter, parentPath string) *TreeNode {
	if filter == nil {
		return nil
	}

	filterNode := &TreeNode{
		Label:         "filter",
		Path:          fmt.Sprintf("%s/filter", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: filter,
		Range: &source.Range{
			Start: &filter.SourceMeta.Position,
		},
	}

	children := []*TreeNode{}

	fieldNode := scalarToTreeNode("field", filter.Field, filterNode.Path)
	if fieldNode != nil {
		children = append(children, fieldNode)
	}

	opNode := filterOperatorToTreeNode(filter.Operator, filterNode.Path)
	if opNode != nil {
		children = append(children, opNode)
	}

	searchNode := filterSearchToTreeNode(filter.Search, filterNode.Path)
	if searchNode != nil {
		children = append(children, searchNode)
	}

	sortTreeNodes(children)
	filterNode.Children = children
	filterNode.Range.End = children[len(children)-1].Range.End

	return filterNode
}

func filterOperatorToTreeNode(operator *DataSourceFilterOperatorWrapper, parentPath string) *TreeNode {
	if operator == nil {
		return nil
	}

	opNode := &TreeNode{
		Label:         "operator",
		Path:          fmt.Sprintf("%s/operator", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: operator,
		Range: &source.Range{
			Start: &operator.SourceMeta.Position,
			End:   operator.SourceMeta.EndPosition,
		},
		Children: []*TreeNode{
			{
				Label:         string(operator.Value),
				Path:          fmt.Sprintf("%s/operator/%s", parentPath, operator.Value),
				Type:          TreeNodeTypeLeaf,
				SchemaElement: operator.Value,
				Range: &source.Range{
					Start: &operator.SourceMeta.Position,
					End:   operator.SourceMeta.EndPosition,
				},
			},
		},
	}

	return opNode
}

func filterSearchToTreeNode(search *DataSourceFilterSearch, parentPath string) *TreeNode {
	if search == nil {
		return nil
	}

	searchNode := &TreeNode{
		Label:         "search",
		Path:          fmt.Sprintf("%s/search", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: search,
		Range: &source.Range{
			Start: &search.SourceMeta.Position,
		},
	}

	if len(search.Values) == 0 {
		searchNode.Type = TreeNodeTypeLeaf
		return searchNode
	}

	children := make([]*TreeNode, len(search.Values))
	for i, val := range search.Values {
		child := stringSubsToTreeNode(fmt.Sprintf("%d", i), val, searchNode.Path)
		if child != nil {
			children[i] = child
		}
	}

	sortTreeNodes(children)
	searchNode.Children = children
	searchNode.Range.End = children[len(children)-1].Range.End

	return searchNode
}

func dataSourceFieldExportsToTreeNode(exports *DataSourceFieldExportMap, parentPath string) *TreeNode {
	if exports == nil || len(exports.Values) == 0 {
		return nil
	}

	exportsNode := &TreeNode{
		Label:         "exports",
		Path:          fmt.Sprintf("%s/exports", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: exports,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(exports.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for exportName, val := range exports.Values {
		exportNode := dataSourceFieldExportToTreeNode(
			exportName,
			val,
			parentPath,
			exports.SourceMeta[exportName],
		)
		if exportNode != nil {
			children = append(children, exportNode)
		}
	}

	sortTreeNodes(children)
	exportsNode.Children = children
	exportsNode.Range.End = children[len(children)-1].Range.End

	return exportsNode
}

func dataSourceFieldExportToTreeNode(exportName string, export *DataSourceFieldExport, parentPath string, location *source.Meta) *TreeNode {
	if export == nil {
		return nil
	}

	fieldExportNode := &TreeNode{
		Label:         exportName,
		Path:          fmt.Sprintf("%s/%s", parentPath, exportName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: export,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	exportTypeNode := dataSourceFieldExportTypeToTreeNode(export.Type, fieldExportNode.Path)
	if exportTypeNode != nil {
		children = append(children, exportTypeNode)
	}

	aliasForNode := scalarToTreeNode("aliasFor", export.AliasFor, fieldExportNode.Path)
	if aliasForNode != nil {
		children = append(children, aliasForNode)
	}

	descriptionNode := stringSubsToTreeNode("description", export.Description, fieldExportNode.Path)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	sortTreeNodes(children)
	fieldExportNode.Children = children
	fieldExportNode.Range.End = children[len(children)-1].Range.End

	return fieldExportNode
}

func dataSourceFieldExportTypeToTreeNode(dataSourceType *DataSourceFieldTypeWrapper, parentPath string) *TreeNode {
	if dataSourceType == nil {
		return nil
	}

	return &TreeNode{
		Label:         "type",
		Path:          fmt.Sprintf("%s/type", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: dataSourceType,
		Range: &source.Range{
			Start: &dataSourceType.SourceMeta.Position,
			End:   dataSourceType.SourceMeta.EndPosition,
		},
	}
}

func exportsToTreeNode(exports *ExportMap, parentPath string) *TreeNode {
	if exports == nil || len(exports.Values) == 0 {
		return nil
	}

	exportsNode := &TreeNode{
		Label:         "exports",
		Path:          fmt.Sprintf("%s/exports", parentPath),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: exports,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(exports.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for exportName, val := range exports.Values {
		exportNode := exportToTreeNode(
			exportName,
			val,
			exportsNode.Path,
			exports.SourceMeta[exportName],
		)
		if exportNode != nil {
			children = append(children, exportNode)
		}
	}
	sortTreeNodes(children)
	exportsNode.Children = children
	exportsNode.Range.End = children[len(children)-1].Range.End

	return exportsNode
}

func exportToTreeNode(exportName string, export *Export, parentPath string, location *source.Meta) *TreeNode {
	exportNode := &TreeNode{
		Label:         exportName,
		Path:          fmt.Sprintf("%s/%s", parentPath, exportName),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: export,
		Range: &source.Range{
			Start: &location.Position,
		},
	}

	children := []*TreeNode{}

	exportTypeNode := exportTypeToTreeNode(export.Type, exportNode.Path)
	if exportTypeNode != nil {
		children = append(children, exportTypeNode)
	}

	fieldNode := scalarToTreeNode("field", export.Field, exportNode.Path)
	if fieldNode != nil {
		children = append(children, fieldNode)
	}

	descriptionNode := stringSubsToTreeNode("description", export.Description, exportNode.Path)
	if descriptionNode != nil {
		children = append(children, descriptionNode)
	}

	sortTreeNodes(children)
	exportNode.Children = children
	exportNode.Range.End = children[len(children)-1].Range.End

	return exportNode
}

func exportTypeToTreeNode(exportType *ExportTypeWrapper, parentPath string) *TreeNode {
	if exportType == nil {
		return nil
	}

	return &TreeNode{
		Label:         "type",
		Path:          fmt.Sprintf("%s/type", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: exportType,
		Range: &source.Range{
			Start: &exportType.SourceMeta.Position,
			End:   exportType.SourceMeta.EndPosition,
		},
	}
}

func stringSubsMapToTreeNode(label string, subsMap *StringOrSubstitutionsMap, parentPath string) *TreeNode {
	if subsMap == nil || len(subsMap.Values) == 0 {
		return nil
	}

	subsMapNode := &TreeNode{
		Label:         label,
		Path:          fmt.Sprintf("%s/%s", parentPath, label),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: subsMap,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(subsMap.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for key, value := range subsMap.Values {
		child := stringSubsToTreeNode(
			key,
			value,
			subsMapNode.Path,
		)
		if child != nil {
			children = append(children, child)
		}
	}

	sortTreeNodes(children)
	subsMapNode.Children = children
	subsMapNode.Range.End = children[len(children)-1].Range.End

	return subsMapNode
}

func stringMapToTreeNode(label string, stringMap *StringMap, parentPath string) *TreeNode {
	if stringMap == nil || len(stringMap.Values) == 0 {
		return nil
	}

	subsMapNode := &TreeNode{
		Label:         label,
		Path:          fmt.Sprintf("%s/%s", parentPath, label),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: stringMap,
		Range: &source.Range{
			Start: minPosition(core.MapToSlice(stringMap.SourceMeta)),
		},
	}

	children := []*TreeNode{}
	for key, value := range stringMap.Values {
		child := stringToTreeNode(
			value,
			stringMap.SourceMeta[key],
			subsMapNode.Path,
		)
		if child != nil {
			children = append(children, child)
		}
	}

	sortTreeNodes(children)
	subsMapNode.Children = children
	subsMapNode.Range.End = children[len(children)-1].Range.End

	return subsMapNode
}

func mappingNodeToTreeNode(
	label string,
	mappingNode *bpcore.MappingNode,
	parentPath string,
	// This is expected to be provided for elements in `MappingNode.Fields` to capture
	// the start location of the key/attribute name in the source document
	// as the start location.
	startPosition *source.Meta,
) *TreeNode {
	if bpcore.IsNilMappingNode(mappingNode) {
		return nil
	}

	mappingNodePath := fmt.Sprintf("%s/%s", parentPath, label)
	finalStartPosition := startPosition
	if finalStartPosition == nil {
		finalStartPosition = mappingNode.SourceMeta
	}
	mappingTreeNode := &TreeNode{
		Label:         label,
		Path:          mappingNodePath,
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: mappingNode,
		Range: &source.Range{
			Start: &finalStartPosition.Position,
		},
	}

	if mappingNode.Scalar != nil {
		literalNode := scalarToTreeNode("scalar", mappingNode.Scalar, mappingNodePath)
		if literalNode != nil {
			mappingTreeNode.Children = []*TreeNode{literalNode}
			mappingTreeNode.Range.End = literalNode.Range.End
		}
	}

	if mappingNode.StringWithSubstitutions != nil {
		stringSubsNode := stringSubsToTreeNode(
			"stringSubs",
			mappingNode.StringWithSubstitutions,
			mappingNodePath,
		)
		if stringSubsNode != nil {
			mappingTreeNode.Children = []*TreeNode{stringSubsNode}
			mappingTreeNode.Range.End = stringSubsNode.Range.End
		}
	}

	if mappingNode.Fields != nil {
		children := []*TreeNode{}
		for key, value := range mappingNode.Fields {
			node := mappingNodeToTreeNode(
				key, value, mappingNodePath, mappingNode.FieldsSourceMeta[key],
			)
			if node != nil {
				children = append(children, node)
			}
		}

		sortTreeNodes(children)
		mappingTreeNode.Children = children
		mappingTreeNode.Range.End = children[len(children)-1].Range.End
	}

	if mappingNode.Items != nil {
		children := make([]*TreeNode, len(mappingNode.Items))
		for i, item := range mappingNode.Items {
			node := mappingNodeToTreeNode(
				fmt.Sprintf("%d", i), item, mappingNodePath, nil,
			)
			if node != nil {
				children[i] = node
			}
		}

		sortTreeNodes(children)
		mappingTreeNode.Children = children
		mappingTreeNode.Range.End = children[len(children)-1].Range.End
	}

	return mappingTreeNode
}

func scalarsToTreeNode(label string, scalars []*bpcore.ScalarValue, parentPath string) *TreeNode {
	if len(scalars) == 0 {
		return nil
	}

	scalarsPath := fmt.Sprintf("%s/%s", parentPath, label)
	scalarsNode := &TreeNode{
		Label: label,
		Path:  scalarsPath,
		Type:  TreeNodeTypeNonTerminal,
		Range: &source.Range{
			Start: minPosition(core.Map(
				scalars,
				func(scalar *bpcore.ScalarValue, _ int) *source.Meta {
					if scalar == nil {
						return nil
					}

					return scalar.SourceMeta
				}),
			),
		},
	}

	children := make([]*TreeNode, len(scalars))
	for i, scalar := range scalars {
		scalarNode := scalarToTreeNode(
			fmt.Sprintf("%d", i),
			scalar,
			scalarsPath,
		)
		if scalarNode != nil {
			children[i] = scalarNode
		}
	}

	sortTreeNodes(children)
	scalarsNode.Children = children
	lastChild := children[len(children)-1]
	if lastChild != nil {
		scalarsNode.Range.End = lastChild.Range.End
	}

	return scalarsNode
}

func stringSubsToTreeNode(label string, subs *substitutions.StringOrSubstitutions, parentPath string) *TreeNode {
	if subs == nil {
		return nil
	}

	path := buildStringSubsNodePath(label, parentPath)

	stringSubsNode := &TreeNode{
		Label:         label,
		Path:          path,
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: subs,
		Range: &source.Range{
			Start: &subs.SourceMeta.Position,
		},
	}

	children := make([]*TreeNode, len(subs.Values))
	for i, value := range subs.Values {
		var child *TreeNode
		if value.StringValue != nil {
			child = stringToTreeNode(*value.StringValue, value.SourceMeta, stringSubsNode.Path)
		} else if value.SubstitutionValue != nil {
			child = substitutionToTreeNode(value.SubstitutionValue, stringSubsNode.Path)
		}

		if child != nil {
			children[i] = child
		}
	}

	sortTreeNodes(children)
	stringSubsNode.Children = children
	stringSubsNode.Range.End = children[len(children)-1].Range.End

	return stringSubsNode
}

func substitutionToTreeNode(value *substitutions.Substitution, parentPath string) *TreeNode {

	if value.StringValue != nil {
		return stringToTreeNode(
			*value.StringValue,
			value.SourceMeta,
			parentPath,
		)
	}

	if value.IntValue != nil {
		return subIntToTreeNode(
			*value.IntValue,
			value.SourceMeta,
			parentPath,
		)
	}

	if value.BoolValue != nil {
		return subBoolToTreeNode(
			*value.BoolValue,
			value.SourceMeta,
			parentPath,
		)
	}

	if value.FloatValue != nil {
		return subFloatToTreeNode(
			*value.FloatValue,
			value.SourceMeta,
			parentPath,
		)
	}

	if value.Child != nil {
		return subChildToTreeNode(
			value.Child,
			parentPath,
		)
	}

	if value.Variable != nil {
		return subVariableToTreeNode(
			value.Variable,
			parentPath,
		)
	}

	if value.ValueReference != nil {
		return subValueRefToTreeNode(
			value.ValueReference,
			parentPath,
		)
	}

	if value.ResourceProperty != nil {
		return subResourcePropToTreeNode(
			value.ResourceProperty,
			parentPath,
		)
	}

	if value.DataSourceProperty != nil {
		return subDataSourcePropToTreeNode(
			value.DataSourceProperty,
			parentPath,
		)
	}

	if value.ElemReference != nil {
		return subElemRefToTreeNode(
			value.ElemReference,
			parentPath,
		)
	}

	if value.ElemIndexReference != nil {
		return subElemIndexRefToTreeNode(
			value.ElemIndexReference,
			parentPath,
		)
	}

	if value.Function != nil {
		return subFunctionToTreeNode(
			value.Function,
			parentPath,
		)
	}

	return nil
}

func subFunctionToTreeNode(value *substitutions.SubstitutionFunctionExpr, parentPath string) *TreeNode {

	treeNodeType := TreeNodeTypeNonTerminal
	if len(value.Arguments) == 0 {
		treeNodeType = TreeNodeTypeLeaf
	}

	root := &TreeNode{
		Label:         string(value.FunctionName),
		Path:          fmt.Sprintf("%s/functionCall/%s", parentPath, value.FunctionName),
		Type:          treeNodeType,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}

	children := make([]*TreeNode, len(value.Arguments))
	for i, arg := range value.Arguments {
		child := subFunctionArgToTreeNode(arg, i, root.Path)
		if child != nil {
			children[i] = child
		}
	}

	// Args are already in order so sorting is not needed.
	root.Children = children

	return root
}

func subFunctionArgToTreeNode(value *substitutions.SubstitutionFunctionArg, index int, parentPath string) *TreeNode {
	if value == nil {
		return nil
	}

	return &TreeNode{
		Label:         fmt.Sprintf("%d", index),
		Path:          fmt.Sprintf("%s/arg/%d", parentPath, index),
		Type:          TreeNodeTypeNonTerminal,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
		Children: []*TreeNode{
			substitutionToTreeNode(value.Value, parentPath),
		},
	}
}

func subChildToTreeNode(value *substitutions.SubstitutionChild, parentPath string) *TreeNode {

	return &TreeNode{
		Label:         value.ChildName,
		Path:          fmt.Sprintf("%s/childRef/%s", parentPath, value.ChildName),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subVariableToTreeNode(value *substitutions.SubstitutionVariable, parentPath string) *TreeNode {

	return &TreeNode{
		Label:         value.VariableName,
		Path:          fmt.Sprintf("%s/varRef/%s", parentPath, value.VariableName),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subValueRefToTreeNode(
	value *substitutions.SubstitutionValueReference,
	parentPath string,
) *TreeNode {

	return &TreeNode{
		Label:         value.ValueName,
		Path:          fmt.Sprintf("%s/valRef/%s", parentPath, value.ValueName),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subResourcePropToTreeNode(
	value *substitutions.SubstitutionResourceProperty,
	parentPath string,
) *TreeNode {

	return &TreeNode{
		Label:         value.ResourceName,
		Path:          fmt.Sprintf("%s/resourceRef/%s", parentPath, value.ResourceName),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subDataSourcePropToTreeNode(
	value *substitutions.SubstitutionDataSourceProperty,
	parentPath string,
) *TreeNode {

	return &TreeNode{
		Label:         value.DataSourceName,
		Path:          fmt.Sprintf("%s/datasourceRef/%s", parentPath, value.DataSourceName),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subElemRefToTreeNode(
	value *substitutions.SubstitutionElemReference,
	parentPath string,
) *TreeNode {

	return &TreeNode{
		Label:         "elemRef",
		Path:          fmt.Sprintf("%s/elemRef", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subElemIndexRefToTreeNode(
	value *substitutions.SubstitutionElemIndexReference,
	parentPath string,
) *TreeNode {

	return &TreeNode{
		Label:         "elemIndexRef",
		Path:          fmt.Sprintf("%s/elemIndexRef", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &value.SourceMeta.Position,
			End:   value.SourceMeta.EndPosition,
		},
	}
}

func subIntToTreeNode(value int64, location *source.Meta, parentPath string) *TreeNode {

	return &TreeNode{
		Label:         "int",
		Path:          fmt.Sprintf("%s/int", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &location.Position,
			End:   location.EndPosition,
		},
	}
}

func subFloatToTreeNode(value float64, location *source.Meta, parentPath string) *TreeNode {

	return &TreeNode{
		Label:         "float",
		Path:          fmt.Sprintf("%s/float", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &location.Position,
			End:   location.EndPosition,
		},
	}
}

func subBoolToTreeNode(value bool, location *source.Meta, parentPath string) *TreeNode {

	return &TreeNode{
		Label:         "bool",
		Path:          fmt.Sprintf("%s/bool", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &location.Position,
			End:   location.EndPosition,
		},
	}
}

func stringToTreeNode(value string, location *source.Meta, parentPath string) *TreeNode {

	return &TreeNode{
		Label:         "string",
		Path:          fmt.Sprintf("%s/string", parentPath),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: value,
		Range: &source.Range{
			Start: &location.Position,
			End:   location.EndPosition,
		},
	}
}

func scalarToTreeNode(label string, scalar *bpcore.ScalarValue, parentPath string) *TreeNode {
	if bpcore.IsScalarNil(scalar) {
		return nil
	}

	return &TreeNode{
		Label:         label,
		Path:          fmt.Sprintf("%s/%s", parentPath, label),
		Type:          TreeNodeTypeLeaf,
		SchemaElement: scalar,
		Range: &source.Range{
			Start: &scalar.SourceMeta.Position,
			End:   scalar.SourceMeta.EndPosition,
		},
	}
}

func sortTreeNodes(nodes []*TreeNode) {
	if len(nodes) == 0 {
		return
	}

	slices.SortFunc(nodes, func(a, b *TreeNode) int {
		if a == nil || b == nil || a.Range == nil ||
			b.Range == nil || a.Range.Start == nil ||
			b.Range.Start == nil {
			return 0
		}

		if a.Range.Start.Line < b.Range.Start.Line {
			return -1
		}

		if a.Range.Start.Line == b.Range.Start.Line &&
			a.Range.Start.Column < b.Range.Start.Column {
			return -1
		}

		if a.Range.Start.Line == b.Range.Start.Line &&
			a.Range.Start.Column == b.Range.Start.Column {
			return 0
		}

		return 1
	})
}

func minPosition(positions []*source.Meta) *source.Position {
	if len(positions) == 0 {
		return nil
	}

	min := positions[0]
	for _, pos := range positions {
		if pos == nil {
			continue
		}

		if pos.Line < min.Line || (pos.Line == min.Line && pos.Column < min.Column) {
			min = pos
		}
	}

	return &min.Position
}

func buildStringSubsNodePath(label, parentPath string) string {
	if label == "stringSubs" {
		return fmt.Sprintf("%s/%s", parentPath, label)
	}
	return fmt.Sprintf("%s/%s/stringSubs", parentPath, label)
}
