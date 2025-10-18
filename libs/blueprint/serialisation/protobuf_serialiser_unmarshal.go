package serialisation

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schemapb"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"google.golang.org/protobuf/proto"
)

func (s *ProtobufExpandedBlueprintSerialiser) Unmarshal(data []byte) (*schema.Blueprint, error) {
	blueprintPB := &schemapb.Blueprint{}
	err := proto.Unmarshal(data, blueprintPB)
	if err != nil {
		return nil, err
	}

	return FromSchemaPB(blueprintPB)
}

// FromSchemaPB converts a Blueprint protobuf message to a schema.Blueprint struct
// to be used with the blueprint framework.
func FromSchemaPB(blueprintPB *schemapb.Blueprint) (*schema.Blueprint, error) {
	variables, err := fromVariablesPB(blueprintPB.Variables)
	if err != nil {
		return nil, err
	}

	values, err := fromValuesPB(blueprintPB.Values)
	if err != nil {
		return nil, err
	}

	includes, err := fromIncludesPB(blueprintPB.Include)
	if err != nil {
		return nil, err
	}

	resources, err := fromResourcesPB(blueprintPB.Resources)
	if err != nil {
		return nil, err
	}

	dataSources, err := fromDataSourcesPB(blueprintPB.DataSources)
	if err != nil {
		return nil, err
	}

	exports, err := fromExportsPB(blueprintPB.Exports)
	if err != nil {
		return nil, err
	}

	metadata, err := FromMappingNodePB(blueprintPB.Metadata, true)
	if err != nil {
		return nil, err
	}

	version, err := FromScalarValuePB(blueprintPB.Version, false)
	if err != nil {
		return nil, err
	}

	transform := (*schema.TransformValueWrapper)(nil)
	if len(blueprintPB.Transform) > 0 {
		transform = &schema.TransformValueWrapper{
			StringList: schema.StringList{Values: blueprintPB.Transform},
		}
	}

	return &schema.Blueprint{
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

func fromVariablesPB(variablesPB map[string]*schemapb.Variable) (*schema.VariableMap, error) {
	if variablesPB == nil {
		return nil, nil
	}

	var variables = make(map[string]*schema.Variable)
	for k, v := range variablesPB {
		defaultValue, err := FromScalarValuePB(v.Default, true)
		if err != nil {
			return nil, err
		}

		allowedValues, err := fromScalarValuesPB(v.AllowedValues)
		if err != nil {
			return nil, err
		}

		description, err := FromScalarValuePB(v.Description, true)
		if err != nil {
			return nil, err
		}

		secret, err := FromScalarValuePB(v.Secret, true)
		if err != nil {
			return nil, err
		}

		variables[k] = &schema.Variable{
			Type:          &schema.VariableTypeWrapper{Value: schema.VariableType(v.Type)},
			Description:   description,
			Secret:        secret,
			Default:       defaultValue,
			AllowedValues: allowedValues,
		}
	}

	return &schema.VariableMap{
		Values: variables,
	}, nil
}

func fromValuesPB(valuesPB map[string]*schemapb.Value) (*schema.ValueMap, error) {
	if valuesPB == nil {
		return nil, nil
	}

	var values = make(map[string]*schema.Value)
	for k, v := range valuesPB {

		value, err := FromMappingNodePB(v.Value, false)
		if err != nil {
			return nil, err
		}

		description, err := fromStringOrSubstitutionsPB(v.Description, true)
		if err != nil {
			return nil, err
		}

		secret, err := FromScalarValuePB(v.Secret, true)
		if err != nil {
			return nil, err
		}

		values[k] = &schema.Value{
			Type:        &schema.ValueTypeWrapper{Value: schema.ValueType(v.Type)},
			Value:       value,
			Description: description,
			Secret:      secret,
		}
	}

	return &schema.ValueMap{
		Values: values,
	}, nil
}

func fromIncludesPB(includesPB map[string]*schemapb.Include) (*schema.IncludeMap, error) {
	if includesPB == nil {
		return nil, nil
	}

	var includes = make(map[string]*schema.Include)
	for k, v := range includesPB {
		path, err := fromStringOrSubstitutionsPB(v.Path, false)
		if err != nil {
			return nil, err
		}

		variables, err := FromMappingNodePB(v.Variables, false)
		if err != nil {
			return nil, err
		}

		metadata, err := FromMappingNodePB(v.Metadata, false)
		if err != nil {
			return nil, err
		}

		description, err := fromStringOrSubstitutionsPB(v.Description, true)
		if err != nil {
			return nil, err
		}

		includes[k] = &schema.Include{
			Path:        path,
			Variables:   variables,
			Metadata:    metadata,
			Description: description,
		}
	}

	return &schema.IncludeMap{
		Values: includes,
	}, nil
}

func fromResourcesPB(resourcesPB map[string]*schemapb.Resource) (*schema.ResourceMap, error) {
	if resourcesPB == nil {
		return nil, nil
	}

	resources := make(map[string]*schema.Resource)
	for k, v := range resourcesPB {
		resource, err := FromResourcePB(v)
		if err != nil {
			return nil, err
		}

		resources[k] = resource
	}

	return &schema.ResourceMap{
		Values: resources,
	}, nil
}

func fromDataSourcesPB(dataSourcesPB map[string]*schemapb.DataSource) (*schema.DataSourceMap, error) {
	if dataSourcesPB == nil {
		return nil, nil
	}

	dataSources := make(map[string]*schema.DataSource)
	for k, v := range dataSourcesPB {
		dataSource, err := FromDataSourcePB(v)
		if err != nil {
			return nil, err
		}

		dataSources[k] = dataSource
	}

	return &schema.DataSourceMap{
		Values: dataSources,
	}, nil
}

// FromDataSourcePB converts a DataSource protobuf message to a schema.DataSource struct
// to be used with the blueprint framework.
func FromDataSourcePB(dataSourcePB *schemapb.DataSource) (*schema.DataSource, error) {
	description, err := fromStringOrSubstitutionsPB(dataSourcePB.Description, true)
	if err != nil {
		return nil, err
	}

	metadata, err := fromDataSourceMetadataPB(dataSourcePB.Metadata)
	if err != nil {
		return nil, err
	}

	filter, err := fromDataSourceFiltersPB(dataSourcePB.Filter)
	if err != nil {
		return nil, err
	}

	exports, err := fromDataSourceFieldExports(
		dataSourcePB,
	)
	if err != nil {
		return nil, err
	}

	return &schema.DataSource{
		Type:               &schema.DataSourceTypeWrapper{Value: dataSourcePB.Type},
		DataSourceMetadata: metadata,
		Filter:             filter,
		Exports:            exports,
		Description:        description,
	}, nil
}

func fromDataSourceMetadataPB(metadataPB *schemapb.DataSourceMetadata) (*schema.DataSourceMetadata, error) {
	displayName, err := fromStringOrSubstitutionsPB(metadataPB.DisplayName, true)
	if err != nil {
		return nil, err
	}

	annotations, err := fromAnnotationsPB(metadataPB.Annotations)
	if err != nil {
		return nil, err
	}

	custom, err := FromMappingNodePB(metadataPB.Custom, true)
	if err != nil {
		return nil, err
	}

	return &schema.DataSourceMetadata{
		DisplayName: displayName,
		Annotations: annotations,
		Custom:      custom,
	}, nil
}

func fromDataSourceFiltersPB(
	filtersPB []*schemapb.DataSourceFilter,
) (*schema.DataSourceFilters, error) {
	if filtersPB == nil {
		return nil, nil
	}
	filters := make([]*schema.DataSourceFilter, len(filtersPB))
	for i, filterPB := range filtersPB {
		filter, err := fromDataSourceFilterPB(filterPB)
		if err != nil {
			return nil, err
		}

		filters[i] = filter
	}
	return &schema.DataSourceFilters{
		Filters: filters,
	}, nil
}

func fromDataSourceFilterPB(filterPB *schemapb.DataSourceFilter) (*schema.DataSourceFilter, error) {
	search, err := fromDataSourceFilterSearchPB(filterPB.Search)
	if err != nil {
		return nil, err
	}

	field, err := FromScalarValuePB(filterPB.Field, false)
	if err != nil {
		return nil, err
	}

	return &schema.DataSourceFilter{
		Field: field,
		Operator: &schema.DataSourceFilterOperatorWrapper{
			Value: schema.DataSourceFilterOperator(filterPB.Operator),
		},
		Search: search,
	}, nil
}

func fromDataSourceFilterSearchPB(
	searchPB *schemapb.DataSourceFilterSearch,
) (*schema.DataSourceFilterSearch, error) {
	values := make([]*substitutions.StringOrSubstitutions, len(searchPB.Values))
	for i, v := range searchPB.Values {
		value, err := fromStringOrSubstitutionsPB(v, false)
		if err != nil {
			return nil, err
		}

		values[i] = value
	}

	return &schema.DataSourceFilterSearch{
		Values: values,
	}, nil
}

func fromDataSourceFieldExports(
	dataSourcePB *schemapb.DataSource,
) (*schema.DataSourceFieldExportMap, error) {
	if dataSourcePB.ExportAllFields {
		return &schema.DataSourceFieldExportMap{
			ExportAll: true,
			Values:    map[string]*schema.DataSourceFieldExport{},
		}, nil
	}
	if dataSourcePB.Exports == nil {
		return nil, nil
	}

	exports := make(map[string]*schema.DataSourceFieldExport)
	for k, v := range dataSourcePB.Exports {
		export, err := fromDataSourceFieldExportPB(v)
		if err != nil {
			return nil, err
		}

		exports[k] = export
	}

	return &schema.DataSourceFieldExportMap{
		Values: exports,
	}, nil
}

func fromDataSourceFieldExportPB(
	exportPB *schemapb.DataSourceFieldExport,
) (*schema.DataSourceFieldExport, error) {
	description, err := fromStringOrSubstitutionsPB(exportPB.Description, true)
	if err != nil {
		return nil, err
	}

	aliasFor, err := FromScalarValuePB(exportPB.AliasFor, true)
	if err != nil {
		return nil, err
	}

	return &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{
			Value: schema.DataSourceFieldType(exportPB.Type),
		},
		AliasFor:    aliasFor,
		Description: description,
	}, nil
}

// FromResourcePB converts a Resource protobuf message to a schema.Resource struct
// to be used with the blueprint framework.
func FromResourcePB(resourcePB *schemapb.Resource) (*schema.Resource, error) {
	description, err := fromStringOrSubstitutionsPB(resourcePB.Description, true)
	if err != nil {
		return nil, err
	}

	each, err := fromStringOrSubstitutionsPB(resourcePB.Each, true)
	if err != nil {
		return nil, err
	}

	condition, err := fromConditionPB(resourcePB.Condition)
	if err != nil {
		return nil, err
	}

	dependsOn := (*schema.DependsOnList)(nil)
	if resourcePB.DependsOn != nil {
		dependsOn = &schema.DependsOnList{
			StringList: schema.StringList{
				Values: resourcePB.DependsOn,
			},
		}
	}

	resourceMetadata, err := fromResourceMetadataPB(resourcePB.Metadata)
	if err != nil {
		return nil, err
	}

	spec, err := FromMappingNodePB(resourcePB.Spec, false)
	if err != nil {
		return nil, err
	}

	return &schema.Resource{
		Type:         &schema.ResourceTypeWrapper{Value: resourcePB.Type},
		Description:  description,
		Each:         each,
		Condition:    condition,
		DependsOn:    dependsOn,
		Metadata:     resourceMetadata,
		LinkSelector: FromLinkSelectorPB(resourcePB.LinkSelector),
		Spec:         spec,
	}, nil
}

func fromConditionPB(conditionPB *schemapb.ResourceCondition) (*schema.Condition, error) {
	if conditionPB == nil {
		return nil, nil
	}

	stringVal, err := fromStringOrSubstitutionsPB(conditionPB.StringValue, true)
	if err != nil {
		return nil, err
	}

	and, err := fromConditionsPB(conditionPB.And)
	if err != nil {
		return nil, err
	}

	or, err := fromConditionsPB(conditionPB.Or)
	if err != nil {
		return nil, err
	}

	not, err := fromConditionPB(conditionPB.Not)
	if err != nil {
		return nil, err
	}
	return &schema.Condition{
		StringValue: stringVal,
		And:         and,
		Or:          or,
		Not:         not,
	}, nil
}

func fromConditionsPB(conditionsPB []*schemapb.ResourceCondition) ([]*schema.Condition, error) {
	var conditions = make([]*schema.Condition, len(conditionsPB))
	for i, v := range conditionsPB {
		condition, err := fromConditionPB(v)
		if err != nil {
			return nil, err
		}

		conditions[i] = condition
	}
	return conditions, nil
}

func fromResourceMetadataPB(metadataPB *schemapb.ResourceMetadata) (*schema.Metadata, error) {
	if metadataPB == nil {
		return nil, nil
	}

	displayName, err := fromStringOrSubstitutionsPB(metadataPB.DisplayName, true)
	if err != nil {
		return nil, err
	}

	annotations, err := fromAnnotationsPB(metadataPB.Annotations)
	if err != nil {
		return nil, err
	}

	custom, err := FromMappingNodePB(metadataPB.Custom, true)
	if err != nil {
		return nil, err
	}

	labels := (*schema.StringMap)(nil)
	if metadataPB.Labels != nil {
		labels = &schema.StringMap{
			Values: metadataPB.Labels,
		}
	}

	return &schema.Metadata{
		DisplayName: displayName,
		Annotations: annotations,
		Labels:      labels,
		Custom:      custom,
	}, nil
}

func fromAnnotationsPB(
	annotationsPB map[string]*schemapb.StringOrSubstitutions,
) (*schema.StringOrSubstitutionsMap, error) {
	if annotationsPB == nil {
		return nil, nil
	}

	annotations := make(map[string]*substitutions.StringOrSubstitutions)
	for k, v := range annotationsPB {
		stringOrSubs, err := fromStringOrSubstitutionsPB(v, false)
		if err != nil {
			return nil, err
		}

		annotations[k] = stringOrSubs
	}

	return &schema.StringOrSubstitutionsMap{
		Values: annotations,
	}, nil
}

func fromExportsPB(exportsPB map[string]*schemapb.Export) (*schema.ExportMap, error) {
	if exportsPB == nil {
		return nil, nil
	}

	exports := make(map[string]*schema.Export)
	for k, v := range exportsPB {
		export, err := fromExportPB(v)
		if err != nil {
			return nil, err
		}

		exports[k] = export
	}

	return &schema.ExportMap{
		Values: exports,
	}, nil
}

// FromLinkSelectorPB converts a LinkSelector protobuf message to a schema.LinkSelector struct
// to be used with the blueprint framework.
func FromLinkSelectorPB(linkSelectorPB *schemapb.LinkSelector) *schema.LinkSelector {
	if linkSelectorPB == nil {
		return nil
	}

	return &schema.LinkSelector{
		ByLabel: &schema.StringMap{
			Values: linkSelectorPB.ByLabel,
		},
	}
}

func fromExportPB(exportPB *schemapb.Export) (*schema.Export, error) {
	description, err := fromStringOrSubstitutionsPB(exportPB.Description, true)
	if err != nil {
		return nil, err
	}

	field, err := FromScalarValuePB(exportPB.Field, false)
	if err != nil {
		return nil, err
	}

	return &schema.Export{
		Type:        &schema.ExportTypeWrapper{Value: schema.ExportType(exportPB.Type)},
		Field:       field,
		Description: description,
	}, nil
}

// FromMappingNodePB converts a MappingNode protobuf message to a core.MappingNode struct
// to be used with the blueprint framework.
func FromMappingNodePB(mappingNodePB *schemapb.MappingNode, optional bool) (*core.MappingNode, error) {
	if optional && mappingNodePBIsNil(mappingNodePB) {
		return nil, nil
	}

	if !optional && mappingNodePB == nil {
		return nil, errMappingNodeIsNil()
	}

	if mappingNodePB.Scalar != nil {
		scalar, err := FromScalarValuePB(mappingNodePB.Scalar, false)
		if err != nil {
			return nil, err
		}

		return &core.MappingNode{
			Scalar: scalar,
		}, nil
	}

	if mappingNodePB.Fields != nil {
		return fromMappingNodeFieldsPB(mappingNodePB.Fields)
	}

	if mappingNodePB.Items != nil {
		return fromMappingNodeItemsPB(mappingNodePB.Items)
	}

	if mappingNodePB.StringWithSubstitutions != nil {
		stringOrSubs, err := fromStringOrSubstitutionsPB(mappingNodePB.StringWithSubstitutions, false)
		if err != nil {
			return nil, err
		}

		return &core.MappingNode{
			StringWithSubstitutions: stringOrSubs,
		}, nil
	}

	return nil, errMissingMappingNodeValue()
}

func fromMappingNodeFieldsPB(fieldsPB map[string]*schemapb.MappingNode) (*core.MappingNode, error) {
	fields := make(map[string]*core.MappingNode)
	for k, v := range fieldsPB {
		mappingNode, err := FromMappingNodePB(v, true)
		if err != nil {
			return nil, err
		}

		fields[k] = mappingNode
	}

	return &core.MappingNode{
		Fields: fields,
	}, nil
}

func fromMappingNodeItemsPB(itemsPB []*schemapb.MappingNode) (*core.MappingNode, error) {
	items := make([]*core.MappingNode, len(itemsPB))
	for i, item := range itemsPB {
		mappingNode, err := FromMappingNodePB(item, true)
		if err != nil {
			return nil, err
		}

		items[i] = mappingNode
	}

	return &core.MappingNode{
		Items: items,
	}, nil
}

func fromStringOrSubstitutionsPB(
	stringOrSubstitutionsPB *schemapb.StringOrSubstitutions,
	optional bool,
) (*substitutions.StringOrSubstitutions, error) {
	if optional && stringOrSubstitutionsPB == nil {
		return nil, nil
	}

	if !optional && stringOrSubstitutionsPB == nil {
		return nil, errStringOrSubstitutionsIsNil()
	}

	stringOrSubs := make([]*substitutions.StringOrSubstitution, len(stringOrSubstitutionsPB.Values))
	for i, stringOrSub := range stringOrSubstitutionsPB.Values {
		stringOrSub, err := fromStringOrSubstitutionPB(stringOrSub)
		if err != nil {
			return nil, err
		}
		stringOrSubs[i] = stringOrSub
	}
	return &substitutions.StringOrSubstitutions{
		Values: stringOrSubs,
	}, nil
}

func fromStringOrSubstitutionPB(
	stringOrSubstitutionPB *schemapb.StringOrSubstitution,
) (*substitutions.StringOrSubstitution, error) {
	if strVal, isStr := stringOrSubstitutionPB.Value.(*schemapb.StringOrSubstitution_StringValue); isStr {
		return &substitutions.StringOrSubstitution{
			StringValue: &strVal.StringValue,
		}, nil
	}

	if subVal, isSub := stringOrSubstitutionPB.Value.(*schemapb.StringOrSubstitution_SubstitutionValue); isSub {
		substitution, err := fromSubstitutionPB(subVal.SubstitutionValue)
		if err != nil {
			return nil, err
		}

		return &substitutions.StringOrSubstitution{
			SubstitutionValue: substitution,
		}, nil
	}

	return nil, errMissingStringOrSubstitutionValue()
}

func fromSubstitutionPB(substitution *schemapb.Substitution) (*substitutions.Substitution, error) {
	if funcVal, isFunc := substitution.Sub.(*schemapb.Substitution_FunctionExpr); isFunc {
		return fromSubstitutionFunctionPB(funcVal.FunctionExpr)
	}

	if varVal, isVar := substitution.Sub.(*schemapb.Substitution_Variable); isVar {
		return fromSubstitutionVariablePB(varVal.Variable)
	}

	if val, isVal := substitution.Sub.(*schemapb.Substitution_Value); isVal {
		return fromSubstitutionValuePB(val.Value)
	}

	if elem, isElem := substitution.Sub.(*schemapb.Substitution_Elem); isElem {
		return fromSubstitutionElemPB(elem.Elem)
	}

	if _, isElemIndex := substitution.Sub.(*schemapb.Substitution_ElemIndex); isElemIndex {
		return fromSubstitutionElemIndexPB()
	}

	if dataSourceProp, isDataSourceProp := substitution.Sub.(*schemapb.Substitution_DataSourceProperty); isDataSourceProp {
		return fromSubstitutionDataSourcePropertyPB(dataSourceProp.DataSourceProperty)
	}

	if resourceProp, isResourceProp := substitution.Sub.(*schemapb.Substitution_ResourceProperty); isResourceProp {
		return fromSubstitutionResourcePropertyPB(resourceProp.ResourceProperty)
	}

	if child, isChild := substitution.Sub.(*schemapb.Substitution_Child); isChild {
		return fromSubstitutionChildPB(child.Child)
	}

	if strVal, isStr := substitution.Sub.(*schemapb.Substitution_StringValue); isStr {
		return &substitutions.Substitution{
			StringValue: &strVal.StringValue,
		}, nil
	}

	if intVal, isInt := substitution.Sub.(*schemapb.Substitution_IntValue); isInt {
		return &substitutions.Substitution{
			IntValue: &intVal.IntValue,
		}, nil
	}

	if floatVal, isFloat := substitution.Sub.(*schemapb.Substitution_FloatValue); isFloat {
		return &substitutions.Substitution{
			FloatValue: &floatVal.FloatValue,
		}, nil
	}

	if boolVal, isBool := substitution.Sub.(*schemapb.Substitution_BoolValue); isBool {
		return &substitutions.Substitution{
			BoolValue: &boolVal.BoolValue,
		}, nil
	}

	return nil, errMissingSubstitutionValue()
}

func fromSubstitutionVariablePB(
	substitutionVariablePB *schemapb.SubstitutionVariable,
) (*substitutions.Substitution, error) {
	return &substitutions.Substitution{
		Variable: &substitutions.SubstitutionVariable{
			VariableName: substitutionVariablePB.VariableName,
		},
	}, nil
}

func fromSubstitutionValuePB(
	substitutionValuePB *schemapb.SubstitutionValue,
) (*substitutions.Substitution, error) {
	path, err := fromSubstitutionPathItemsPB(substitutionValuePB.Path)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		ValueReference: &substitutions.SubstitutionValueReference{
			ValueName: substitutionValuePB.ValueName,
			Path:      path,
		},
	}, nil
}

func fromSubstitutionElemPB(
	substitutionElemPB *schemapb.SubstitutionElem,
) (*substitutions.Substitution, error) {
	path, err := fromSubstitutionPathItemsPB(substitutionElemPB.Path)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		ElemReference: &substitutions.SubstitutionElemReference{
			Path: path,
		},
	}, nil
}

func fromSubstitutionElemIndexPB() (*substitutions.Substitution, error) {
	return &substitutions.Substitution{
		ElemIndexReference: &substitutions.SubstitutionElemIndexReference{},
	}, nil
}

func fromSubstitutionResourcePropertyPB(
	substitutionResourcePropertyPB *schemapb.SubstitutionResourceProperty,
) (*substitutions.Substitution, error) {
	path, err := fromSubstitutionPathItemsPB(substitutionResourcePropertyPB.Path)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		ResourceProperty: &substitutions.SubstitutionResourceProperty{
			ResourceName:              substitutionResourcePropertyPB.ResourceName,
			ResourceEachTemplateIndex: substitutionResourcePropertyPB.EachTemplateIndex,
			Path:                      path,
		},
	}, nil
}

func fromSubstitutionDataSourcePropertyPB(
	substitutionDataSourcePropertyPB *schemapb.SubstitutionDataSourceProperty,
) (*substitutions.Substitution, error) {
	return &substitutions.Substitution{
		DataSourceProperty: &substitutions.SubstitutionDataSourceProperty{
			DataSourceName:    substitutionDataSourcePropertyPB.DataSourceName,
			FieldName:         substitutionDataSourcePropertyPB.FieldName,
			PrimitiveArrIndex: substitutionDataSourcePropertyPB.PrimitiveArrIndex,
		},
	}, nil
}

func fromSubstitutionChildPB(
	substitutionChildPB *schemapb.SubstitutionChild,
) (*substitutions.Substitution, error) {
	path, err := fromSubstitutionPathItemsPB(substitutionChildPB.Path)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		Child: &substitutions.SubstitutionChild{
			ChildName: substitutionChildPB.ChildName,
			Path:      path,
		},
	}, nil
}

func fromSubstitutionPathItemsPB(
	pathItemsPB []*schemapb.SubstitutionPathItem,
) ([]*substitutions.SubstitutionPathItem, error) {
	var pathItems = make([]*substitutions.SubstitutionPathItem, len(pathItemsPB))
	for i, pathItemPB := range pathItemsPB {
		pathItem, err := fromSubstitutionPathItemPB(pathItemPB)
		if err != nil {
			return nil, err
		}

		pathItems[i] = pathItem
	}
	return pathItems, nil
}

func fromSubstitutionPathItemPB(
	pathItemPB *schemapb.SubstitutionPathItem,
) (*substitutions.SubstitutionPathItem, error) {
	if fieldNameVal, isFieldName := pathItemPB.Item.(*schemapb.SubstitutionPathItem_FieldName); isFieldName {
		return &substitutions.SubstitutionPathItem{
			FieldName: fieldNameVal.FieldName,
		}, nil
	}

	if indexVal, isIndex := pathItemPB.Item.(*schemapb.SubstitutionPathItem_ArrayIndex); isIndex {
		return &substitutions.SubstitutionPathItem{
			ArrayIndex: &indexVal.ArrayIndex,
		}, nil
	}

	return nil, errMissingSubstitutionPathItemValue()
}

func fromSubstitutionFunctionPB(
	substitutionFunctionPB *schemapb.SubstitutionFunctionExpr,
) (*substitutions.Substitution, error) {
	arguments, err := fromSubstitutionFunctionArgsPB(substitutionFunctionPB.Arguments)
	if err != nil {
		return nil, err
	}

	return &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: substitutions.SubstitutionFunctionName(
				substitutionFunctionPB.FunctionName,
			),
			Arguments: arguments,
		},
	}, nil
}

func fromSubstitutionFunctionArgsPB(
	argsPB []*schemapb.SubstitutionFunctionArg,
) ([]*substitutions.SubstitutionFunctionArg, error) {
	var args = make([]*substitutions.SubstitutionFunctionArg, len(argsPB))
	for i, argPB := range argsPB {
		arg, err := fromSubstitutionFunctionArgPB(argPB)
		if err != nil {
			return nil, err
		}

		args[i] = arg
	}
	return args, nil
}

func fromSubstitutionFunctionArgPB(
	argPB *schemapb.SubstitutionFunctionArg,
) (*substitutions.SubstitutionFunctionArg, error) {
	if argPB.Value == nil {
		return nil, errMissingSubstitutionFunctionArgValue()
	}

	val, err := fromSubstitutionPB(argPB.Value)
	if err != nil {
		return nil, err
	}

	return &substitutions.SubstitutionFunctionArg{
		Name:  *argPB.Name,
		Value: val,
	}, nil
}

// FromScalarValuePB converts a ScalarValue protobuf message to a core.ScalarValue struct
// to be used with the blueprint framework.
func FromScalarValuePB(scalarValue *schemapb.ScalarValue, optional bool) (*core.ScalarValue, error) {
	if optional && scalarValue == nil {
		return nil, nil
	}

	if stringVal, isString := scalarValue.Value.(*schemapb.ScalarValue_StringValue); isString {
		return &core.ScalarValue{
			StringValue: &stringVal.StringValue,
		}, nil
	}

	if bytesVal, isBytes := scalarValue.Value.(*schemapb.ScalarValue_BytesValue); isBytes {
		return &core.ScalarValue{
			BytesValue: &bytesVal.BytesValue,
		}, nil
	}

	if intWrapper, isInt := scalarValue.Value.(*schemapb.ScalarValue_IntValue); isInt {
		intVal := int(intWrapper.IntValue)
		return &core.ScalarValue{
			IntValue: &intVal,
		}, nil
	}

	if floatVal, isFloat := scalarValue.Value.(*schemapb.ScalarValue_FloatValue); isFloat {
		return &core.ScalarValue{
			FloatValue: &floatVal.FloatValue,
		}, nil
	}

	if boolVal, isBool := scalarValue.Value.(*schemapb.ScalarValue_BoolValue); isBool {
		return &core.ScalarValue{
			BoolValue: &boolVal.BoolValue,
		}, nil
	}

	return nil, errMissingScalarValue()
}

func fromScalarValuesPB(scalarValuesPB []*schemapb.ScalarValue) ([]*core.ScalarValue, error) {
	var scalarValues = make([]*core.ScalarValue, len(scalarValuesPB))
	for i, v := range scalarValuesPB {
		scalarValue, err := FromScalarValuePB(v, false)
		if err != nil {
			return nil, err
		}

		scalarValues[i] = scalarValue
	}
	return scalarValues, nil
}

func mappingNodePBIsNil(
	mappingNodePB *schemapb.MappingNode,
) bool {
	return mappingNodePB == nil ||
		(mappingNodePB.Scalar == nil &&
			mappingNodePB.Fields == nil &&
			mappingNodePB.Items == nil &&
			mappingNodePB.StringWithSubstitutions == nil)
}
