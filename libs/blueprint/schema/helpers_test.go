package schema

import (
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/substitutions"
)

// NormaliseSchema normalises a schema to make it comparable
// in a way that ignores line and column numbers of the input source
// the schema was parsed from.
func NormaliseSchema(bpSchema *Blueprint) {
	if bpSchema == nil {
		return
	}

	if bpSchema.Transform != nil {
		NormaliseTransform(bpSchema.Transform)
	}

	if bpSchema.Variables != nil && bpSchema.Variables.Values != nil {
		for _, variable := range bpSchema.Variables.Values {
			NormaliseVariable(variable)
		}
		bpSchema.Variables.SourceMeta = nil
	}

	if bpSchema.Include != nil && bpSchema.Include.Values != nil {
		for _, include := range bpSchema.Include.Values {
			NormaliseInclude(include)
		}
		bpSchema.Include.SourceMeta = nil
	}

	if bpSchema.Resources != nil && bpSchema.Resources.Values != nil {
		for _, resource := range bpSchema.Resources.Values {
			NormaliseResource(resource)
		}
		bpSchema.Resources.SourceMeta = nil
	}

	if bpSchema.DataSources != nil && bpSchema.DataSources.Values != nil {
		for _, dataSource := range bpSchema.DataSources.Values {
			NormaliseDataSource(dataSource)
		}
		bpSchema.DataSources.SourceMeta = nil
	}

	if bpSchema.Exports != nil && bpSchema.Exports.Values != nil {
		for _, export := range bpSchema.Exports.Values {
			NormaliseExport(export)
		}
		bpSchema.Exports.SourceMeta = nil
	}

	NormaliseMappingNode(bpSchema.Metadata)
}

// NormaliseTransform strips the source meta information from a transform
// to make it comparable in a way that ignores line and column numbers.
func NormaliseTransform(transform *TransformValueWrapper) {
	if transform == nil {
		return
	}

	transform.SourceMeta = nil
}

// NormaliseVariable strips the source meta information from a variable
// to make it comparable in a way that ignores line and column numbers.
func NormaliseVariable(variable *Variable) {
	if variable == nil {
		return
	}

	NormaliseScalarValue(variable.Default)
	NormaliseScalarValues(variable.AllowedValues)
	variable.SourceMeta = nil
}

// NormaliseInclude strips the source meta information from an include
// to make it comparable in a way that ignores line and column numbers.
func NormaliseInclude(include *Include) {
	if include == nil {
		return
	}

	NormaliseStringOrSubstitutions(include.Path)
	NormaliseMappingNode(include.Variables)
	NormaliseMappingNode(include.Metadata)
	NormaliseStringOrSubstitutions(include.Description)
	include.SourceMeta = nil
}

// NormaliseResource strips the source meta information from a resource
// to make it comparable in a way that ignores line and column numbers.
func NormaliseResource(resource *Resource) {
	if resource == nil {
		return
	}

	NormaliseStringOrSubstitutions(resource.Description)
	NormaliseResourceMetadata(resource.Metadata)
	NormaliseLinkSelector(resource.LinkSelector)
	NormaliseMappingNode(resource.Spec)
	resource.SourceMeta = nil
}

// NormaliseResourceMetadata strips the source meta information from a resource metadata
// to make it comparable in a way that ignores line and column numbers.
func NormaliseResourceMetadata(metadata *Metadata) {
	if metadata == nil {
		return
	}

	NormaliseStringOrSubstitutions(metadata.DisplayName)

	if metadata.Annotations != nil && metadata.Annotations.Values != nil {
		for _, annotation := range metadata.Annotations.Values {
			NormaliseStringOrSubstitutions(annotation)
		}
	}

	NormaliseMappingNode(metadata.Custom)
	metadata.SourceMeta = nil
}

// NormaliseLinkSelector strips the source meta information from a link selector
// to make it comparable in a way that ignores line and column numbers.
func NormaliseLinkSelector(linkSelector *LinkSelector) {
	if linkSelector == nil {
		return
	}

	linkSelector.SourceMeta = nil
}

// NormaliseDataSource strips the source meta information from a data source
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSource(dataSource *DataSource) {
	if dataSource == nil {
		return
	}

	NormaliseDataSourceMetadata(dataSource.DataSourceMetadata)
	NormaliseDataSourceFilter(dataSource.Filter)

	if dataSource.Exports != nil && dataSource.Exports.Values != nil {
		for _, export := range dataSource.Exports.Values {
			NormaliseDataSourceFieldExport(export)
		}
	}

	NormaliseStringOrSubstitutions(dataSource.Description)
	dataSource.SourceMeta = nil
}

// NormaliseDataSourceMetadata strips the source meta information from a data source metadata
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSourceMetadata(dataSourceMetadata *DataSourceMetadata) {
	if dataSourceMetadata == nil {
		return
	}

	NormaliseStringOrSubstitutions(dataSourceMetadata.DisplayName)

	if dataSourceMetadata.Annotations != nil && dataSourceMetadata.Annotations.Values != nil {
		for _, export := range dataSourceMetadata.Annotations.Values {
			NormaliseStringOrSubstitutions(export)
		}
	}

	NormaliseMappingNode(dataSourceMetadata.Custom)
	dataSourceMetadata.SourceMeta = nil
}

// NormaliseDataSourceFilter strips the source meta information from a data source filter
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSourceFilter(filter *DataSourceFilter) {
	if filter == nil {
		return
	}

	NormaliseDataSourceFilterOperator(filter.Operator)
	NormaliseDataSourceFilterSearch(filter.Search)
	filter.SourceMeta = nil
}

// NormaliseDataSourceFilterOperator strips the source meta information from a data source filter operator
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSourceFilterOperator(operator *DataSourceFilterOperatorWrapper) {
	if operator == nil {
		return
	}

	operator.SourceMeta = nil
}

// NormaliseDataSourceFilterSearch strips the source meta information from a data source filter search
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSourceFilterSearch(search *DataSourceFilterSearch) {
	if search == nil {
		return
	}

	for _, value := range search.Values {
		NormaliseStringOrSubstitutions(value)
	}
	search.SourceMeta = nil
}

// NormaliseDataSourceFieldExport strips the source meta information from a data source field export
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSourceFieldExport(export *DataSourceFieldExport) {
	if export == nil {
		return
	}

	NormaliseDataSourceFieldType(export.Type)
	NormaliseStringOrSubstitutions(export.Description)
	export.SourceMeta = nil
}

// NormaliseDataSourceFieldType strips the source meta information from a data source field type
// to make it comparable in a way that ignores line and column numbers.
func NormaliseDataSourceFieldType(fieldType *DataSourceFieldTypeWrapper) {
	if fieldType == nil {
		return
	}

	fieldType.SourceMeta = nil
}

// NormaliseExport strips the source meta information from an export
// to make it comparable in a way that ignores line and column numbers.
func NormaliseExport(export *Export) {
	if export == nil {
		return
	}

	NormaliseStringOrSubstitutions(export.Description)
	export.SourceMeta = nil
}

// NormaliseMappingNode strips the source meta information from a mapping node
// to make it comparable in a way that ignores line and column numbers.
func NormaliseMappingNode(mappingNode *core.MappingNode) {
	if mappingNode == nil {
		return
	}

	if mappingNode.Literal != nil {
		NormaliseScalarValue(mappingNode.Literal)
	}

	if mappingNode.Fields != nil {
		for _, field := range mappingNode.Fields {
			NormaliseMappingNode(field)
		}
	}

	if mappingNode.Items != nil {
		for _, item := range mappingNode.Items {
			NormaliseMappingNode(item)
		}
	}

	if mappingNode.StringWithSubstitutions != nil {
		NormaliseStringOrSubstitutions(mappingNode.StringWithSubstitutions)
	}

	mappingNode.SourceMeta = nil
}

// NormaliseStringOrSubstitutions strips the source meta information from a string or substitutions
// to make it comparable in a way that ignores line and column numbers.
func NormaliseStringOrSubstitutions(stringOrSubstitutions *substitutions.StringOrSubstitutions) {
	if stringOrSubstitutions == nil {
		return
	}

	for _, value := range stringOrSubstitutions.Values {
		NormaliseStringOrSubstitution(value)
	}
	stringOrSubstitutions.SourceMeta = nil
}

// NormaliseStringOrSubstitution strips the source meta information from a string or substitution
// to make it comparable in a way that ignores line and column numbers.
func NormaliseStringOrSubstitution(stringOrSubstitution *substitutions.StringOrSubstitution) {
	if stringOrSubstitution == nil {
		return
	}

	if stringOrSubstitution.SubstitutionValue != nil {
		NormaliseSubstitution(stringOrSubstitution.SubstitutionValue)
	}
	stringOrSubstitution.SourceMeta = nil
}

// NormaliseSubstitution strips the source meta information from a substitution
// to make it comparable in a way that ignores line and column numbers.
func NormaliseSubstitution(substitution *substitutions.Substitution) {
	if substitution == nil {
		return
	}

	if substitution.Function != nil {
		NormaliseSubstitutionFunction(substitution.Function)
	}

	if substitution.Variable != nil {
		NormaliseSubstitutionVariable(substitution.Variable)
	}

	if substitution.DataSourceProperty != nil {
		NormaliseSubstitutionDataSourceProperty(
			substitution.DataSourceProperty,
		)
	}

	if substitution.ResourceProperty != nil {
		NormaliseSubstitutionResourceProperty(
			substitution.ResourceProperty,
		)
	}

	if substitution.Child != nil {
		NormaliseSubstitutionChild(substitution.Child)
	}

	substitution.SourceMeta = nil
}

// NormaliseSubstitutionFunction strips the source meta information from a substitution function
// to make it comparable in a way that ignores line and column numbers.
func NormaliseSubstitutionFunction(substitutionFunction *substitutions.SubstitutionFunction) {
	if substitutionFunction == nil {
		return
	}

	for _, argument := range substitutionFunction.Arguments {
		NormaliseSubstitution(argument)
	}
	substitutionFunction.SourceMeta = nil
}

// NormaliseSubstitutionVariable strips the source meta information from a substitution variable
// to make it comparable in a way that ignores line and column numbers.
func NormaliseSubstitutionVariable(substitutionVariable *substitutions.SubstitutionVariable) {
	if substitutionVariable == nil {
		return
	}

	substitutionVariable.SourceMeta = nil
}

// NormaliseSubstitutionDataSourceProperty strips the source meta information
// from a substitution data source property to make it comparable in a way that
// ignores line and column numbers.
func NormaliseSubstitutionDataSourceProperty(
	substitutionDataSourceProperty *substitutions.SubstitutionDataSourceProperty,
) {
	if substitutionDataSourceProperty == nil {
		return
	}

	substitutionDataSourceProperty.SourceMeta = nil
}

// NormaliseSubstitutionResourceProperty strips the source meta information from a resource property
// to make it comparable in a way that ignores line and column numbers.
func NormaliseSubstitutionResourceProperty(
	resourceProperty *substitutions.SubstitutionResourceProperty,
) {
	if resourceProperty == nil {
		return
	}

	resourceProperty.SourceMeta = nil
}

// NormaliseSubstitutionChild strips the source meta information from a substitution child
// to make it comparable in a way that ignores line and column numbers.
func NormaliseSubstitutionChild(substitutionChild *substitutions.SubstitutionChild) {
	if substitutionChild == nil {
		return
	}

	substitutionChild.SourceMeta = nil
}

// NormaliseScalarValue strips the source meta information from a scalar value
// to make it comparable in a way that ignores line and column numbers.
func NormaliseScalarValue(scalarValue *core.ScalarValue) {
	if scalarValue == nil {
		return
	}

	scalarValue.SourceMeta = nil
}

// NormaliseScalarValues strips the source meta information from a slice of scalar values
// to make it comparable in a way that ignores line and column numbers.
func NormaliseScalarValues(scalarValues []*core.ScalarValue) {
	if scalarValues == nil {
		return
	}

	for _, scalarValue := range scalarValues {
		NormaliseScalarValue(scalarValue)
	}
}
