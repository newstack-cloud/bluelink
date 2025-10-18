package serialisation

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schemapb"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"google.golang.org/protobuf/proto"
)

func (s *ProtobufExpandedBlueprintSerialiser) Marshal(blueprint *schema.Blueprint) ([]byte, error) {
	schemaPB, err := ToSchemaPB(blueprint)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(schemaPB)
}

// ToSchemaPB converts a core blueprint to a schemapb.Blueprint
// that can be stored and transmitted as a protobuf message.
func ToSchemaPB(blueprint *schema.Blueprint) (*schemapb.Blueprint, error) {
	variables, err := toVariablesPB(blueprint.Variables)
	if err != nil {
		return nil, err
	}

	values, err := toValuesPB(blueprint.Values)
	if err != nil {
		return nil, err
	}

	includes, err := toIncludesPB(blueprint.Include)
	if err != nil {
		return nil, err
	}

	resources, err := toResourcesPB(blueprint.Resources)
	if err != nil {
		return nil, err
	}

	dataSources, err := toDataSourcesPB(blueprint.DataSources)
	if err != nil {
		return nil, err
	}

	exports, err := toExportsPB(blueprint.Exports)
	if err != nil {
		return nil, err
	}

	metadata, err := ToMappingNodePB(blueprint.Metadata, true)
	if err != nil {
		return nil, err
	}

	version, err := ToScalarValuePB(blueprint.Version, false)
	if err != nil {
		return nil, err
	}

	transform := []string{}
	if blueprint.Transform != nil {
		transform = blueprint.Transform.Values
	}

	return &schemapb.Blueprint{
		Version:     version,
		Transform:   transform,
		Variables:   variables,
		Values:      values,
		Include:     includes,
		Resources:   resources,
		DataSources: dataSources,
		Exports:     exports,
		Metadata:    metadata,
	}, nil
}

func toVariablesPB(variables *schema.VariableMap) (map[string]*schemapb.Variable, error) {
	if variables == nil {
		return nil, nil
	}

	var variablesPB = make(map[string]*schemapb.Variable)
	for k, v := range variables.Values {
		defaultValue, err := ToScalarValuePB(v.Default, true)
		if err != nil {
			return nil, err
		}

		allowedValues, err := toScalarValuesPB(v.AllowedValues)
		if err != nil {
			return nil, err
		}

		description, err := ToScalarValuePB(v.Description, true)
		if err != nil {
			return nil, err
		}

		secret, err := ToScalarValuePB(v.Secret, true)
		if err != nil {
			return nil, err
		}

		variablesPB[k] = &schemapb.Variable{
			Type:          string(v.Type.Value),
			Description:   description,
			Secret:        secret,
			Default:       defaultValue,
			AllowedValues: allowedValues,
		}
	}
	return variablesPB, nil
}

func toValuesPB(values *schema.ValueMap) (map[string]*schemapb.Value, error) {
	if values == nil {
		return nil, nil
	}

	var valuesPB = make(map[string]*schemapb.Value)
	for k, v := range values.Values {

		valuePB, err := ToMappingNodePB(v.Value, false)
		if err != nil {
			return nil, err
		}

		descriptionPB, err := toStringOrSubstitutionsPB(v.Description, true)
		if err != nil {
			return nil, err
		}

		secretPB, err := ToScalarValuePB(v.Secret, true)
		if err != nil {
			return nil, err
		}

		valuesPB[k] = &schemapb.Value{
			Type:        string(v.Type.Value),
			Value:       valuePB,
			Description: descriptionPB,
			Secret:      secretPB,
		}
	}

	return valuesPB, nil
}

func toIncludesPB(includes *schema.IncludeMap) (map[string]*schemapb.Include, error) {
	if includes == nil {
		return nil, nil
	}

	var includesPB = make(map[string]*schemapb.Include)
	for k, v := range includes.Values {
		path, err := toStringOrSubstitutionsPB(v.Path, false)
		if err != nil {
			return nil, err
		}

		variablesPB, err := ToMappingNodePB(v.Variables, false)
		if err != nil {
			return nil, err
		}

		metadataPB, err := ToMappingNodePB(v.Metadata, false)
		if err != nil {
			return nil, err
		}

		descriptionPB, err := toStringOrSubstitutionsPB(v.Description, true)
		if err != nil {
			return nil, err
		}

		includesPB[k] = &schemapb.Include{
			Path:        path,
			Variables:   variablesPB,
			Metadata:    metadataPB,
			Description: descriptionPB,
		}
	}

	return includesPB, nil
}

func toResourcesPB(resources *schema.ResourceMap) (map[string]*schemapb.Resource, error) {
	if resources == nil {
		return nil, nil
	}

	resourcesPB := make(map[string]*schemapb.Resource)
	for k, v := range resources.Values {
		resourcePB, err := ToResourcePB(v)
		if err != nil {
			return nil, err
		}

		resourcesPB[k] = resourcePB
	}
	return resourcesPB, nil
}

func toDataSourcesPB(dataSources *schema.DataSourceMap) (map[string]*schemapb.DataSource, error) {
	if dataSources == nil {
		return nil, nil
	}

	dataSourcesPB := make(map[string]*schemapb.DataSource)
	for k, v := range dataSources.Values {
		dataSourcePB, err := ToDataSourcePB(v)
		if err != nil {
			return nil, err
		}

		dataSourcesPB[k] = dataSourcePB
	}
	return dataSourcesPB, nil
}

// ToDataSourcePB converts a core data source type to a schemapb.DataSource
// that can be stored and transmitted as a protobuf message.
func ToDataSourcePB(dataSource *schema.DataSource) (*schemapb.DataSource, error) {
	descriptionPB, err := toStringOrSubstitutionsPB(dataSource.Description, true)
	if err != nil {
		return nil, err
	}

	metadataPB, err := toDataSourceMetadataPB(dataSource.DataSourceMetadata)
	if err != nil {
		return nil, err
	}

	filtersPB, err := toDataSourceFiltersPB(dataSource.Filter)
	if err != nil {
		return nil, err
	}

	exportsPB, err := toDataSourceFieldExports(dataSource.Exports)
	if err != nil {
		return nil, err
	}

	return &schemapb.DataSource{
		Type:            string(dataSource.Type.Value),
		Metadata:        metadataPB,
		Filter:          filtersPB,
		ExportAllFields: getDataSourceExportAll(dataSource),
		Exports:         exportsPB,
		Description:     descriptionPB,
	}, nil
}

func getDataSourceExportAll(dataSource *schema.DataSource) bool {
	if dataSource.Exports == nil {
		return false
	}

	return dataSource.Exports.ExportAll
}

func toDataSourceMetadataPB(metadata *schema.DataSourceMetadata) (*schemapb.DataSourceMetadata, error) {
	displayNamePB, err := toStringOrSubstitutionsPB(metadata.DisplayName, true)
	if err != nil {
		return nil, err
	}

	annotationsPB, err := toAnnotationsPB(metadata.Annotations)
	if err != nil {
		return nil, err
	}

	customPB, err := ToMappingNodePB(metadata.Custom, true)
	if err != nil {
		return nil, err
	}

	return &schemapb.DataSourceMetadata{
		DisplayName: displayNamePB,
		Annotations: annotationsPB,
		Custom:      customPB,
	}, nil
}

func toDataSourceFiltersPB(filters *schema.DataSourceFilters) ([]*schemapb.DataSourceFilter, error) {
	if filters == nil {
		return nil, nil
	}

	filtersPB := make([]*schemapb.DataSourceFilter, len(filters.Filters))
	for i, filter := range filters.Filters {
		filterPB, err := toDataSourceFilterPB(filter)
		if err != nil {
			return nil, err
		}

		filtersPB[i] = filterPB
	}

	return filtersPB, nil
}

func toDataSourceFilterPB(filter *schema.DataSourceFilter) (*schemapb.DataSourceFilter, error) {
	searchPB, err := toDataSourceFilterSearchPB(filter.Search)
	if err != nil {
		return nil, err
	}

	fieldPB, err := ToScalarValuePB(filter.Field, false)
	if err != nil {
		return nil, err
	}

	return &schemapb.DataSourceFilter{
		Field:    fieldPB,
		Operator: string(filter.Operator.Value),
		Search:   searchPB,
	}, nil
}

func toDataSourceFilterSearchPB(
	search *schema.DataSourceFilterSearch,
) (*schemapb.DataSourceFilterSearch, error) {
	valuesPB := make([]*schemapb.StringOrSubstitutions, len(search.Values))
	for i, v := range search.Values {
		valuePB, err := toStringOrSubstitutionsPB(v, false)
		if err != nil {
			return nil, err
		}

		valuesPB[i] = valuePB
	}

	return &schemapb.DataSourceFilterSearch{
		Values: valuesPB,
	}, nil
}

func toDataSourceFieldExports(
	exports *schema.DataSourceFieldExportMap,
) (map[string]*schemapb.DataSourceFieldExport, error) {
	if exports == nil {
		return nil, nil
	}

	exportsPB := make(map[string]*schemapb.DataSourceFieldExport)
	for k, v := range exports.Values {
		exportPB, err := toDataSourceFieldExportPB(v)
		if err != nil {
			return nil, err
		}

		exportsPB[k] = exportPB
	}

	return exportsPB, nil
}

func toDataSourceFieldExportPB(
	export *schema.DataSourceFieldExport,
) (*schemapb.DataSourceFieldExport, error) {
	descriptionPB, err := toStringOrSubstitutionsPB(export.Description, true)
	if err != nil {
		return nil, err
	}

	aliasForPB, err := ToScalarValuePB(export.AliasFor, true)
	if err != nil {
		return nil, err
	}

	return &schemapb.DataSourceFieldExport{
		Type:        string(export.Type.Value),
		AliasFor:    aliasForPB,
		Description: descriptionPB,
	}, nil
}

// ToResourcePB converts a schema.Resource to a schemapb.Resource
// that can be stored and transmitted as a protobuf message.
func ToResourcePB(resource *schema.Resource) (*schemapb.Resource, error) {
	conditionPB, err := toConditionPB(resource.Condition)
	if err != nil {
		return nil, err
	}

	eachPB, err := toStringOrSubstitutionsPB(resource.Each, true)
	if err != nil {
		return nil, err
	}

	descriptionPB, err := toStringOrSubstitutionsPB(resource.Description, true)
	if err != nil {
		return nil, err
	}

	dependsOn := []string{}
	if resource.DependsOn != nil {
		dependsOn = resource.DependsOn.Values
	}

	resourceMetadataPB, err := toResourceMetadataPB(resource.Metadata)
	if err != nil {
		return nil, err
	}

	specPB, err := ToMappingNodePB(resource.Spec, false)
	if err != nil {
		return nil, err
	}

	return &schemapb.Resource{
		Type:         string(resource.Type.Value),
		Description:  descriptionPB,
		Condition:    conditionPB,
		Each:         eachPB,
		Metadata:     resourceMetadataPB,
		DependsOn:    dependsOn,
		LinkSelector: ToLinkSelectorPB(resource.LinkSelector),
		Spec:         specPB,
	}, nil
}

func toExportsPB(exports *schema.ExportMap) (map[string]*schemapb.Export, error) {
	if exports == nil {
		return nil, nil
	}

	exportsPB := make(map[string]*schemapb.Export)
	for k, v := range exports.Values {
		exportPB, err := toExportPB(v)
		if err != nil {
			return nil, err
		}

		exportsPB[k] = exportPB
	}

	return exportsPB, nil
}

func toExportPB(export *schema.Export) (*schemapb.Export, error) {
	descriptionPB, err := toStringOrSubstitutionsPB(export.Description, true)
	if err != nil {
		return nil, err
	}

	field, err := ToScalarValuePB(export.Field, false)
	if err != nil {
		return nil, err
	}

	return &schemapb.Export{
		Type:        string(export.Type.Value),
		Field:       field,
		Description: descriptionPB,
	}, nil
}

// ToLinkSelectorPB converts a schema.LinkSelector to a schemapb.LinkSelector
// that can be stored and transmitted as a protobuf message.
func ToLinkSelectorPB(linkSelector *schema.LinkSelector) *schemapb.LinkSelector {
	if linkSelector == nil || linkSelector.ByLabel == nil {
		return nil
	}

	return &schemapb.LinkSelector{
		ByLabel: linkSelector.ByLabel.Values,
	}
}

func toConditionsPB(conditions []*schema.Condition) ([]*schemapb.ResourceCondition, error) {
	conditionsPB := make([]*schemapb.ResourceCondition, len(conditions))
	for i, condition := range conditions {
		conditionPB, err := toConditionPB(condition)
		if err != nil {
			return nil, err
		}

		conditionsPB[i] = conditionPB
	}

	return conditionsPB, nil
}

func toConditionPB(condition *schema.Condition) (*schemapb.ResourceCondition, error) {
	if condition == nil {
		return nil, nil
	}

	stringValPB, err := toStringOrSubstitutionsPB(condition.StringValue, true)
	if err != nil {
		return nil, err
	}

	andPB, err := toConditionsPB(condition.And)
	if err != nil {
		return nil, err
	}

	orPB, err := toConditionsPB(condition.Or)
	if err != nil {
		return nil, err
	}

	notPB, err := toConditionPB(condition.Not)
	if err != nil {
		return nil, err
	}

	return &schemapb.ResourceCondition{
		And:         andPB,
		Or:          orPB,
		Not:         notPB,
		StringValue: stringValPB,
	}, nil
}

func toResourceMetadataPB(metadata *schema.Metadata) (*schemapb.ResourceMetadata, error) {
	if metadata == nil {
		return nil, nil
	}

	displayNamePB, err := toStringOrSubstitutionsPB(metadata.DisplayName, true)
	if err != nil {
		return nil, err
	}

	annotationsPB, err := toAnnotationsPB(metadata.Annotations)
	if err != nil {
		return nil, err
	}

	customPB, err := ToMappingNodePB(metadata.Custom, true)
	if err != nil {
		return nil, err
	}

	labels := (map[string]string)(nil)
	if metadata.Labels != nil {
		labels = metadata.Labels.Values
	}

	return &schemapb.ResourceMetadata{
		DisplayName: displayNamePB,
		Annotations: annotationsPB,
		Labels:      labels,
		Custom:      customPB,
	}, nil
}

func toAnnotationsPB(
	annotations *schema.StringOrSubstitutionsMap,
) (map[string]*schemapb.StringOrSubstitutions, error) {
	if annotations == nil {
		return nil, nil
	}

	annotationsPB := make(map[string]*schemapb.StringOrSubstitutions)
	for k, v := range annotations.Values {
		stringOrSubsPB, err := toStringOrSubstitutionsPB(v, false)
		if err != nil {
			return nil, err
		}

		annotationsPB[k] = stringOrSubsPB
	}

	return annotationsPB, nil
}

// ToMappingNodePB converts a core.MappingNode to a schemapb.MappingNode
// that can be stored and transmitted as a protobuf message.
func ToMappingNodePB(mappingNode *core.MappingNode, optional bool) (*schemapb.MappingNode, error) {
	if optional && mappingNode == nil {
		return nil, nil
	}

	if !optional && mappingNode == nil {
		return nil, errMappingNodeIsNil()
	}

	if mappingNode.Scalar != nil {
		scalarPB, err := ToScalarValuePB(mappingNode.Scalar, false)
		if err != nil {
			return nil, err
		}

		return &schemapb.MappingNode{
			Scalar: scalarPB,
		}, nil
	}

	if mappingNode.Fields != nil {
		return toMappingNodeFieldsPB(mappingNode.Fields)
	}

	if mappingNode.Items != nil {
		return toMappingNodeItemsPB(mappingNode.Items)
	}

	if mappingNode.StringWithSubstitutions != nil {
		stringOrSubsPB, err := toStringOrSubstitutionsPB(mappingNode.StringWithSubstitutions, false)
		if err != nil {
			return nil, err
		}

		return &schemapb.MappingNode{
			StringWithSubstitutions: stringOrSubsPB,
		}, nil
	}

	return nil, errMissingMappingNodeValue()
}

func toMappingNodeFieldsPB(fields map[string]*core.MappingNode) (*schemapb.MappingNode, error) {
	fieldsPB := make(map[string]*schemapb.MappingNode)
	for k, v := range fields {
		mappingNodePB, err := ToMappingNodePB(v, true)
		if err != nil {
			return nil, err
		}

		fieldsPB[k] = mappingNodePB
	}

	return &schemapb.MappingNode{
		Fields: fieldsPB,
	}, nil
}

func toMappingNodeItemsPB(items []*core.MappingNode) (*schemapb.MappingNode, error) {
	itemsPB := make([]*schemapb.MappingNode, len(items))
	for i, item := range items {
		mappingNodePB, err := ToMappingNodePB(item, true)
		if err != nil {
			return nil, err
		}

		itemsPB[i] = mappingNodePB
	}

	return &schemapb.MappingNode{
		Items: itemsPB,
	}, nil
}

func toStringOrSubstitutionsPB(
	stringOrSubstitutions *substitutions.StringOrSubstitutions,
	optional bool,
) (*schemapb.StringOrSubstitutions, error) {
	if optional && stringOrSubstitutions == nil {
		return nil, nil
	}

	if !optional && stringOrSubstitutions == nil {
		return nil, errStringOrSubstitutionsIsNil()
	}

	stringOrSubsPB := make([]*schemapb.StringOrSubstitution, len(stringOrSubstitutions.Values))
	for i, stringOrSub := range stringOrSubstitutions.Values {
		stringOrSubPB, err := toStringOrSubstitutionPB(stringOrSub)
		if err != nil {
			return nil, err
		}
		stringOrSubsPB[i] = stringOrSubPB
	}
	return &schemapb.StringOrSubstitutions{
		Values: stringOrSubsPB,
	}, nil
}

func toStringOrSubstitutionPB(
	stringOrSubstitution *substitutions.StringOrSubstitution,
) (*schemapb.StringOrSubstitution, error) {
	if stringOrSubstitution.StringValue != nil {
		return &schemapb.StringOrSubstitution{
			Value: &schemapb.StringOrSubstitution_StringValue{
				StringValue: *stringOrSubstitution.StringValue,
			},
		}, nil
	}

	if stringOrSubstitution.SubstitutionValue != nil {
		substitutionPB, err := toSubstitutionPB(stringOrSubstitution.SubstitutionValue)
		if err != nil {
			return nil, err
		}

		return &schemapb.StringOrSubstitution{
			Value: &schemapb.StringOrSubstitution_SubstitutionValue{
				SubstitutionValue: substitutionPB,
			},
		}, nil
	}

	return nil, errMissingStringOrSubstitutionValue()
}

func toSubstitutionPB(substitution *substitutions.Substitution) (*schemapb.Substitution, error) {
	if substitution.Function != nil {
		return toSubstitutionFunctionPB(substitution.Function)
	}

	if substitution.Variable != nil {
		return toSubstitutionVariablePB(substitution.Variable)
	}

	if substitution.ValueReference != nil {
		return toSubstitutionValuePB(substitution.ValueReference)
	}

	if substitution.ElemReference != nil {
		return toSubstitutionElemPB(substitution.ElemReference)
	}

	if substitution.ElemIndexReference != nil {
		return toSubstitutionElemIndexPB()
	}

	if substitution.DataSourceProperty != nil {
		return toSubstitutionDataSourcePropertyPB(substitution.DataSourceProperty)
	}

	if substitution.ResourceProperty != nil {
		return toSubstitutionResourcePropertyPB(substitution.ResourceProperty)
	}

	if substitution.Child != nil {
		return toSubstitutionChildPB(substitution.Child)
	}

	if substitution.StringValue != nil {
		return &schemapb.Substitution{
			Sub: &schemapb.Substitution_StringValue{
				StringValue: *substitution.StringValue,
			},
		}, nil
	}

	if substitution.IntValue != nil {
		return &schemapb.Substitution{
			Sub: &schemapb.Substitution_IntValue{
				IntValue: *substitution.IntValue,
			},
		}, nil
	}

	if substitution.FloatValue != nil {
		return &schemapb.Substitution{
			Sub: &schemapb.Substitution_FloatValue{
				FloatValue: *substitution.FloatValue,
			},
		}, nil
	}

	if substitution.BoolValue != nil {
		return &schemapb.Substitution{
			Sub: &schemapb.Substitution_BoolValue{
				BoolValue: *substitution.BoolValue,
			},
		}, nil
	}

	return nil, errMissingSubstitutionValue()
}

func toSubstitutionVariablePB(
	substitutionVariable *substitutions.SubstitutionVariable,
) (*schemapb.Substitution, error) {
	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_Variable{
			Variable: &schemapb.SubstitutionVariable{
				VariableName: substitutionVariable.VariableName,
			},
		},
	}, nil
}

func toSubstitutionValuePB(
	substitutionValue *substitutions.SubstitutionValueReference,
) (*schemapb.Substitution, error) {
	pathPB, err := toSubstitutionPathItemsPB(substitutionValue.Path)
	if err != nil {
		return nil, err
	}

	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_Value{
			Value: &schemapb.SubstitutionValue{
				ValueName: substitutionValue.ValueName,
				Path:      pathPB,
			},
		},
	}, nil
}

func toSubstitutionElemPB(
	substitutionValue *substitutions.SubstitutionElemReference,
) (*schemapb.Substitution, error) {
	pathPB, err := toSubstitutionPathItemsPB(substitutionValue.Path)
	if err != nil {
		return nil, err
	}

	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_Elem{
			Elem: &schemapb.SubstitutionElem{
				Path: pathPB,
			},
		},
	}, nil
}

func toSubstitutionElemIndexPB() (*schemapb.Substitution, error) {
	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_ElemIndex{
			ElemIndex: &schemapb.SubstitutionElemIndex{
				IsIndex: true,
			},
		},
	}, nil
}

func toSubstitutionResourcePropertyPB(
	substitutionResourceProperty *substitutions.SubstitutionResourceProperty,
) (*schemapb.Substitution, error) {
	path, err := toSubstitutionPathItemsPB(substitutionResourceProperty.Path)
	if err != nil {
		return nil, err
	}

	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_ResourceProperty{
			ResourceProperty: &schemapb.SubstitutionResourceProperty{
				ResourceName:      substitutionResourceProperty.ResourceName,
				EachTemplateIndex: substitutionResourceProperty.ResourceEachTemplateIndex,
				Path:              path,
			},
		},
	}, nil
}

func toSubstitutionPathItemsPB(
	pathItems []*substitutions.SubstitutionPathItem,
) ([]*schemapb.SubstitutionPathItem, error) {
	var pathItemsPB = make([]*schemapb.SubstitutionPathItem, len(pathItems))
	for i, pathItem := range pathItems {
		pathItemPB, err := toSubstitutionPathItemPB(pathItem)
		if err != nil {
			return nil, err
		}

		pathItemsPB[i] = pathItemPB
	}
	return pathItemsPB, nil
}

func toSubstitutionPathItemPB(
	pathItem *substitutions.SubstitutionPathItem,
) (*schemapb.SubstitutionPathItem, error) {
	if pathItem.FieldName != "" {
		return &schemapb.SubstitutionPathItem{
			Item: &schemapb.SubstitutionPathItem_FieldName{
				FieldName: pathItem.FieldName,
			},
		}, nil
	}

	if pathItem.ArrayIndex != nil {
		return &schemapb.SubstitutionPathItem{
			Item: &schemapb.SubstitutionPathItem_ArrayIndex{
				ArrayIndex: *pathItem.ArrayIndex,
			},
		}, nil
	}

	return nil, errMissingSubstitutionPathItemValue()
}

func toSubstitutionDataSourcePropertyPB(
	substitutionDataSourceProperty *substitutions.SubstitutionDataSourceProperty,
) (*schemapb.Substitution, error) {
	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_DataSourceProperty{
			DataSourceProperty: &schemapb.SubstitutionDataSourceProperty{
				DataSourceName:    substitutionDataSourceProperty.DataSourceName,
				FieldName:         substitutionDataSourceProperty.FieldName,
				PrimitiveArrIndex: substitutionDataSourceProperty.PrimitiveArrIndex,
			},
		},
	}, nil
}

func toSubstitutionChildPB(
	substitutionChild *substitutions.SubstitutionChild,
) (*schemapb.Substitution, error) {
	path, err := toSubstitutionPathItemsPB(substitutionChild.Path)
	if err != nil {
		return nil, err
	}

	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_Child{
			Child: &schemapb.SubstitutionChild{
				ChildName: substitutionChild.ChildName,
				Path:      path,
			},
		},
	}, nil
}

func toSubstitutionFunctionPB(
	substitutionFunction *substitutions.SubstitutionFunctionExpr,
) (*schemapb.Substitution, error) {
	arguments, err := toSubstitutionFunctionArgsPB(substitutionFunction.Arguments)
	if err != nil {
		return nil, err
	}

	return &schemapb.Substitution{
		Sub: &schemapb.Substitution_FunctionExpr{
			FunctionExpr: &schemapb.SubstitutionFunctionExpr{
				FunctionName: string(substitutionFunction.FunctionName),
				Arguments:    arguments,
			},
		},
	}, nil
}

func toSubstitutionFunctionArgsPB(
	substitutionFunctionArgs []*substitutions.SubstitutionFunctionArg,
) ([]*schemapb.SubstitutionFunctionArg, error) {
	var substitutionFunctionArgsPB = make([]*schemapb.SubstitutionFunctionArg, len(substitutionFunctionArgs))
	for i, arg := range substitutionFunctionArgs {
		argPB, err := toSubstitutionFunctionArgPB(arg)
		if err != nil {
			return nil, err
		}

		substitutionFunctionArgsPB[i] = argPB
	}
	return substitutionFunctionArgsPB, nil
}

func toSubstitutionFunctionArgPB(
	substitutionFunctionArg *substitutions.SubstitutionFunctionArg,
) (*schemapb.SubstitutionFunctionArg, error) {
	valuePB, err := toSubstitutionPB(substitutionFunctionArg.Value)
	if err != nil {
		return nil, err
	}

	return &schemapb.SubstitutionFunctionArg{
		Name:  &substitutionFunctionArg.Name,
		Value: valuePB,
	}, nil
}

func toScalarValuesPB(scalarValues []*core.ScalarValue) ([]*schemapb.ScalarValue, error) {
	var scalarValuesPB = make([]*schemapb.ScalarValue, len(scalarValues))
	for i, v := range scalarValues {
		scalarValuePB, err := ToScalarValuePB(v, false)
		if err != nil {
			return nil, err
		}

		scalarValuesPB[i] = scalarValuePB
	}
	return scalarValuesPB, nil
}

// ToScalarValuePB converts a core.ScalarValue to a schemapb.ScalarValue
// that can be stored and transmitted as a protobuf message.
func ToScalarValuePB(scalarValue *core.ScalarValue, optional bool) (*schemapb.ScalarValue, error) {
	if optional && scalarValue == nil {
		return nil, nil
	}

	if scalarValue.StringValue != nil {
		return &schemapb.ScalarValue{
			Value: &schemapb.ScalarValue_StringValue{StringValue: *scalarValue.StringValue},
		}, nil
	}

	if scalarValue.BytesValue != nil {
		return &schemapb.ScalarValue{
			Value: &schemapb.ScalarValue_BytesValue{BytesValue: *scalarValue.BytesValue},
		}, nil
	}

	if scalarValue.IntValue != nil {
		return &schemapb.ScalarValue{
			Value: &schemapb.ScalarValue_IntValue{IntValue: int64(*scalarValue.IntValue)},
		}, nil
	}

	if scalarValue.FloatValue != nil {
		return &schemapb.ScalarValue{
			Value: &schemapb.ScalarValue_FloatValue{FloatValue: *scalarValue.FloatValue},
		}, nil
	}

	if scalarValue.BoolValue != nil {
		return &schemapb.ScalarValue{
			Value: &schemapb.ScalarValue_BoolValue{BoolValue: *scalarValue.BoolValue},
		}, nil
	}

	return nil, errMissingScalarValue()
}
