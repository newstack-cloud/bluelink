package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// SchemaElementKind provides type-safe schema element classification.
type SchemaElementKind int

const (
	SchemaElementUnknown SchemaElementKind = iota

	// Top-level sections
	SchemaElementVariables
	SchemaElementValues
	SchemaElementResources
	SchemaElementDataSources
	SchemaElementIncludes
	SchemaElementExports

	// Named elements
	SchemaElementVariable
	SchemaElementValue
	SchemaElementResource
	SchemaElementDataSource
	SchemaElementInclude
	SchemaElementExport

	// Field types
	SchemaElementResourceType
	SchemaElementDataSourceType
	SchemaElementVariableType
	SchemaElementValueType
	SchemaElementExportType
	SchemaElementDataSourceFieldType
	SchemaElementDataSourceFilterField
	SchemaElementDataSourceFilterOperator

	// Substitution elements
	SchemaElementSubstitution
	SchemaElementFunctionCall
	SchemaElementVariableRef
	SchemaElementValueRef
	SchemaElementResourceRef
	SchemaElementDataSourceRef
	SchemaElementChildRef
	SchemaElementElemRef
	SchemaElementElemIndexRef

	// Value nodes
	SchemaElementScalar
	SchemaElementMapping
	SchemaElementSequence
)

// KindFromSchemaElement determines the kind from a schema element interface.
func KindFromSchemaElement(elem any) SchemaElementKind {
	if elem == nil {
		return SchemaElementUnknown
	}

	switch elem.(type) {
	// Type wrappers
	case *schema.ResourceTypeWrapper:
		return SchemaElementResourceType
	case *schema.DataSourceTypeWrapper:
		return SchemaElementDataSourceType

	// Substitution types
	case *substitutions.SubstitutionFunctionExpr:
		return SchemaElementFunctionCall
	case *substitutions.SubstitutionVariable:
		return SchemaElementVariableRef
	case *substitutions.SubstitutionValueReference:
		return SchemaElementValueRef
	case *substitutions.SubstitutionResourceProperty:
		return SchemaElementResourceRef
	case *substitutions.SubstitutionDataSourceProperty:
		return SchemaElementDataSourceRef
	case *substitutions.SubstitutionChild:
		return SchemaElementChildRef
	case *substitutions.SubstitutionElemReference:
		return SchemaElementElemRef
	case *substitutions.SubstitutionElemIndexReference:
		return SchemaElementElemIndexRef

	// Schema structures
	case *schema.Blueprint:
		return SchemaElementUnknown
	case *schema.Resource:
		return SchemaElementResource
	case *schema.DataSource:
		return SchemaElementDataSource
	case *schema.Variable:
		return SchemaElementVariable
	case *schema.Value:
		return SchemaElementValue
	case *schema.Include:
		return SchemaElementInclude
	case *schema.Export:
		return SchemaElementExport

	// Maps of elements
	case *schema.ResourceMap:
		return SchemaElementResources
	case *schema.DataSourceMap:
		return SchemaElementDataSources
	case *schema.VariableMap:
		return SchemaElementVariables
	case *schema.ValueMap:
		return SchemaElementValues
	case *schema.IncludeMap:
		return SchemaElementIncludes
	case *schema.ExportMap:
		return SchemaElementExports

	default:
		return SchemaElementUnknown
	}
}

// String returns a string representation of SchemaElementKind.
func (k SchemaElementKind) String() string {
	switch k {
	case SchemaElementUnknown:
		return "unknown"
	case SchemaElementVariables:
		return "variables"
	case SchemaElementValues:
		return "values"
	case SchemaElementResources:
		return "resources"
	case SchemaElementDataSources:
		return "datasources"
	case SchemaElementIncludes:
		return "includes"
	case SchemaElementExports:
		return "exports"
	case SchemaElementVariable:
		return "variable"
	case SchemaElementValue:
		return "value"
	case SchemaElementResource:
		return "resource"
	case SchemaElementDataSource:
		return "datasource"
	case SchemaElementInclude:
		return "include"
	case SchemaElementExport:
		return "export"
	case SchemaElementResourceType:
		return "resource_type"
	case SchemaElementDataSourceType:
		return "datasource_type"
	case SchemaElementVariableType:
		return "variable_type"
	case SchemaElementValueType:
		return "value_type"
	case SchemaElementExportType:
		return "export_type"
	case SchemaElementDataSourceFieldType:
		return "datasource_field_type"
	case SchemaElementDataSourceFilterField:
		return "datasource_filter_field"
	case SchemaElementDataSourceFilterOperator:
		return "datasource_filter_operator"
	case SchemaElementSubstitution:
		return "substitution"
	case SchemaElementFunctionCall:
		return "function_call"
	case SchemaElementVariableRef:
		return "variable_ref"
	case SchemaElementValueRef:
		return "value_ref"
	case SchemaElementResourceRef:
		return "resource_ref"
	case SchemaElementDataSourceRef:
		return "datasource_ref"
	case SchemaElementChildRef:
		return "child_ref"
	case SchemaElementElemRef:
		return "elem_ref"
	case SchemaElementElemIndexRef:
		return "elem_index_ref"
	case SchemaElementScalar:
		return "scalar"
	case SchemaElementMapping:
		return "mapping"
	case SchemaElementSequence:
		return "sequence"
	default:
		return "unknown"
	}
}

// IsTypeField returns true if this is a type field kind.
func (k SchemaElementKind) IsTypeField() bool {
	switch k {
	case SchemaElementResourceType,
		SchemaElementDataSourceType,
		SchemaElementVariableType,
		SchemaElementValueType,
		SchemaElementExportType,
		SchemaElementDataSourceFieldType:
		return true
	}
	return false
}

// IsSubstitution returns true if this is a substitution kind.
func (k SchemaElementKind) IsSubstitution() bool {
	switch k {
	case SchemaElementSubstitution,
		SchemaElementFunctionCall,
		SchemaElementVariableRef,
		SchemaElementValueRef,
		SchemaElementResourceRef,
		SchemaElementDataSourceRef,
		SchemaElementChildRef,
		SchemaElementElemRef,
		SchemaElementElemIndexRef:
		return true
	}
	return false
}

// IsReference returns true if this is a reference kind.
func (k SchemaElementKind) IsReference() bool {
	switch k {
	case SchemaElementVariableRef,
		SchemaElementValueRef,
		SchemaElementResourceRef,
		SchemaElementDataSourceRef,
		SchemaElementChildRef,
		SchemaElementElemRef,
		SchemaElementElemIndexRef:
		return true
	}
	return false
}
