package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
	SchemaElementPathItem

	// Structural elements
	SchemaElementMappingNode
	SchemaElementDataSourceFieldExport
	SchemaElementDataSourceFieldExportMap
	SchemaElementDataSourceFilters
	SchemaElementDataSourceFilter
	SchemaElementDataSourceFilterSearch
	SchemaElementMetadata
	SchemaElementDataSourceMetadata
	SchemaElementLinkSelector
	SchemaElementStringMap
	SchemaElementStringOrSubstitutionsMap
	SchemaElementStringList

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
	case *substitutions.SubstitutionPathItem:
		return SchemaElementPathItem

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

	// Structural elements
	case *core.MappingNode:
		return SchemaElementMappingNode
	case *schema.DataSourceFieldExport:
		return SchemaElementDataSourceFieldExport
	case *schema.DataSourceFieldExportMap:
		return SchemaElementDataSourceFieldExportMap
	case *schema.DataSourceFieldTypeWrapper:
		return SchemaElementDataSourceFieldType
	case *schema.DataSourceFilters:
		return SchemaElementDataSourceFilters
	case *schema.DataSourceFilter:
		return SchemaElementDataSourceFilter
	case *schema.DataSourceFilterOperatorWrapper:
		return SchemaElementDataSourceFilterOperator
	case *schema.DataSourceFilterSearch:
		return SchemaElementDataSourceFilterSearch
	case *schema.Metadata:
		return SchemaElementMetadata
	case *schema.DataSourceMetadata:
		return SchemaElementDataSourceMetadata
	case *schema.LinkSelector:
		return SchemaElementLinkSelector
	case *schema.StringMap:
		return SchemaElementStringMap
	case *schema.StringOrSubstitutionsMap:
		return SchemaElementStringOrSubstitutionsMap
	case *schema.StringList:
		return SchemaElementStringList

	default:
		return SchemaElementUnknown
	}
}

var schemaElementKindNames = map[SchemaElementKind]string{
	SchemaElementUnknown:                  "unknown",
	SchemaElementVariables:                "variables",
	SchemaElementValues:                   "values",
	SchemaElementResources:                "resources",
	SchemaElementDataSources:              "datasources",
	SchemaElementIncludes:                 "includes",
	SchemaElementExports:                  "exports",
	SchemaElementVariable:                 "variable",
	SchemaElementValue:                    "value",
	SchemaElementResource:                 "resource",
	SchemaElementDataSource:               "datasource",
	SchemaElementInclude:                  "include",
	SchemaElementExport:                   "export",
	SchemaElementResourceType:             "resource_type",
	SchemaElementDataSourceType:           "datasource_type",
	SchemaElementVariableType:             "variable_type",
	SchemaElementValueType:                "value_type",
	SchemaElementExportType:               "export_type",
	SchemaElementDataSourceFieldType:      "datasource_field_type",
	SchemaElementDataSourceFilterField:    "datasource_filter_field",
	SchemaElementDataSourceFilterOperator: "datasource_filter_operator",
	SchemaElementSubstitution:             "substitution",
	SchemaElementFunctionCall:             "function_call",
	SchemaElementVariableRef:              "variable_ref",
	SchemaElementValueRef:                 "value_ref",
	SchemaElementResourceRef:              "resource_ref",
	SchemaElementDataSourceRef:            "datasource_ref",
	SchemaElementChildRef:                 "child_ref",
	SchemaElementElemRef:                  "elem_ref",
	SchemaElementElemIndexRef:             "elem_index_ref",
	SchemaElementPathItem:                 "path_item",
	SchemaElementMappingNode:              "mapping_node",
	SchemaElementDataSourceFieldExport:    "datasource_field_export",
	SchemaElementDataSourceFieldExportMap: "datasource_field_export_map",
	SchemaElementDataSourceFilters:        "datasource_filters",
	SchemaElementDataSourceFilter:         "datasource_filter",
	SchemaElementDataSourceFilterSearch:   "datasource_filter_search",
	SchemaElementMetadata:                 "metadata",
	SchemaElementDataSourceMetadata:       "datasource_metadata",
	SchemaElementLinkSelector:             "link_selector",
	SchemaElementStringMap:                "string_map",
	SchemaElementStringOrSubstitutionsMap: "string_or_substitutions_map",
	SchemaElementStringList:               "string_list",
	SchemaElementScalar:                   "scalar",
	SchemaElementMapping:                  "mapping",
	SchemaElementSequence:                 "sequence",
}

// String returns a string representation of SchemaElementKind.
func (k SchemaElementKind) String() string {
	if name, ok := schemaElementKindNames[k]; ok {
		return name
	}
	return "unknown"
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
		SchemaElementElemIndexRef,
		SchemaElementPathItem:
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
