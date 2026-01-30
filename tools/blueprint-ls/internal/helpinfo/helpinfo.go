package helpinfo

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// RenderVariableInfo renders variable information for use in help info.
func RenderVariableInfo(varName string, variable *schema.Variable) string {
	varType := "unknown"
	if variable.Type != nil {
		varType = string(variable.Type.Value)
	}

	description := ""
	if variable.Description != nil && variable.Description.StringValue != nil {
		description = *variable.Description.StringValue
	}

	return fmt.Sprintf(
		"```variables.%s```\n\n"+
			"**type:** `%s`\n\n%s",
		varName,
		varType,
		description,
	)
}

// RenderValueInfo renders value information for use in help info.
func RenderValueInfo(valueName string, value *schema.Value) string {
	valueType := "unknown"
	if value.Type != nil {
		valueType = string(value.Type.Value)
	}

	description := ""
	if value.Description != nil {
		description, _ = substitutions.SubstitutionsToString("", value.Description)
	}

	return fmt.Sprintf(
		"```values.%s```\n\n"+
			"**type:** `%s`\n\n%s",
		valueName,
		valueType,
		description,
	)
}

// RenderChildInfo renders child blueprint information for use in help info.
func RenderChildInfo(childName string, child *schema.Include) string {
	path := ""
	if child.Path != nil {
		path, _ = substitutions.SubstitutionsToString("", child.Path)
	}

	description := ""
	if child.Description != nil {
		description, _ = substitutions.SubstitutionsToString("", child.Description)
	}

	return fmt.Sprintf(
		"```includes.%s```\n\n"+
			"**path:** `%s`\n\n%s",
		childName,
		path,
		description,
	)
}

// RenderBasicResourceInfo renders basic resource information
// for use in help info.
func RenderBasicResourceInfo(resourceName string, resource *schema.Resource) string {
	description := ""
	if resource.Description != nil {
		description, _ = substitutions.SubstitutionsToString("", resource.Description)
	}

	resourceType := "unknown"
	if resource.Type != nil {
		resourceType = resource.Type.Value
	}

	return fmt.Sprintf(
		"```resources.%s```\n\n"+
			"**type:** `%s`\n\n%s",
		resourceName,
		resourceType,
		description,
	)
}

// RenderResourceDefinitionFieldInfo renders resource definition field information
// for use in help info.
func RenderResourceDefinitionFieldInfo(
	resourceName string,
	resource *schema.Resource,
	resRef *substitutions.SubstitutionResourceProperty,
	specFieldSchema *provider.ResourceDefinitionsSchema,
) string {
	resourceInfo := RenderBasicResourceInfo(resourceName, resource)
	if specFieldSchema == nil {
		return resourceInfo
	}

	fieldPath := renderFieldPath(resRef.Path)

	description := ""
	if specFieldSchema.FormattedDescription != "" {
		description = specFieldSchema.FormattedDescription
	} else if specFieldSchema.Description != "" {
		description = specFieldSchema.Description
	}

	return fmt.Sprintf(
		"`%s`\n\n"+
			"**field type:** `%s`\n\n%s\n\n"+
			"### Resource information\n\n%s",
		fieldPath,
		specFieldSchema.Type,
		description,
		resourceInfo,
	)
}

// RenderDataSourceFieldInfo renders data source field information
// for use in help info.
func RenderDataSourceFieldInfo(
	dataSourceName string,
	dataSource *schema.DataSource,
	dataSourceRef *substitutions.SubstitutionDataSourceProperty,
	dataSourceField *schema.DataSourceFieldExport,
) string {
	dataSourceInfo := RenderBasicDataSourceInfo(dataSourceName, dataSource)

	dataSourceFieldType := "unknown"
	if dataSourceField.Type != nil {
		dataSourceFieldType = string(dataSourceField.Type.Value)
	}

	description := ""
	if dataSourceField.Description != nil {
		description, _ = substitutions.SubstitutionsToString("", dataSourceField.Description)
	}

	aliasForInfo := ""
	if dataSourceField.AliasFor != nil && dataSourceField.AliasFor.StringValue != nil {
		aliasForInfo = fmt.Sprintf(
			"**alias for:** `%s`\n\n",
			*dataSourceField.AliasFor.StringValue,
		)
	}

	return fmt.Sprintf(
		"`datasources.%s%s`\n%s\n\n"+
			"**field type:** `%s`\n\n%s\n\n"+
			"### Data source information\n\n%s",
		dataSourceName,
		dataSourceFieldNameOrIndexAccessor(dataSourceRef),
		aliasForInfo,
		dataSourceFieldType,
		description,
		dataSourceInfo,
	)
}

func dataSourceFieldNameOrIndexAccessor(
	dataSourceRef *substitutions.SubstitutionDataSourceProperty,
) string {
	var sb strings.Builder
	if dataSourceRef.FieldName != "" {
		sb.WriteString(".")
		sb.WriteString(dataSourceRef.FieldName)
	}

	if dataSourceRef.PrimitiveArrIndex != nil {
		sb.WriteString(fmt.Sprintf("[%d]", *dataSourceRef.PrimitiveArrIndex))
	}

	return sb.String()
}

// RenderBasicDataSourceInfo renders basic data source information
// for use in help info.
func RenderBasicDataSourceInfo(
	dataSourceName string,
	dataSource *schema.DataSource,
) string {
	dataSourceType := "unknown"
	if dataSource.Type != nil {
		dataSourceType = string(dataSource.Type.Value)
	}

	description := ""
	if dataSource.Description != nil {
		description, _ = substitutions.SubstitutionsToString("", dataSource.Description)
	}

	return fmt.Sprintf(
		"```datasources.%s```\n\n"+
			"**type:** `%s`\n\n%s",
		dataSourceName,
		dataSourceType,
		description,
	)
}

// RenderElemRefInfo renders element reference information
// for use in help info.
func RenderElemRefInfo(
	resourceName string,
	resource *schema.Resource,
	elemRef *substitutions.SubstitutionElemReference,
) string {

	resourceInfo := RenderBasicResourceInfo(resourceName, resource)
	fieldPath := fmt.Sprintf(".%s", renderFieldPath(elemRef.Path))
	return fmt.Sprintf(
		"`resources.%s[i]%s`\n\nnth element of resource template `resources.%s`\n\n"+
			"## Resource information\n\n%s",
		resourceName,
		fieldPath,
		resourceName,
		resourceInfo,
	)
}

// RenderElemIndexRefInfo renders element index reference information
// for use in help info.
func RenderElemIndexRefInfo(
	resourceName string,
	resource *schema.Resource,
) string {

	resourceInfo := RenderBasicResourceInfo(resourceName, resource)

	return fmt.Sprintf(
		"index of nth element in resource template `resources.%s`\n\n"+
			"## Resource information\n\n%s",
		resourceName,
		resourceInfo,
	)
}

func renderFieldPath(path []*substitutions.SubstitutionPathItem) string {
	var sb strings.Builder
	for i, item := range path {
		if item.FieldName != "" {
			if i > 0 {
				sb.WriteString(".")
			}
			sb.WriteString(item.FieldName)
		} else if item.ArrayIndex != nil {
			sb.WriteString(fmt.Sprintf("[%d]", *item.ArrayIndex))
		}
	}

	return sb.String()
}

// RenderResourceMetadataFieldInfo renders resource metadata field information
// for use in help info.
func RenderResourceMetadataFieldInfo(
	resourceName string,
	resource *schema.Resource,
	resRef *substitutions.SubstitutionResourceProperty,
) string {
	resourceInfo := RenderBasicResourceInfo(resourceName, resource)
	if len(resRef.Path) == 0 {
		return resourceInfo
	}

	fieldPath := renderFieldPath(resRef.Path)
	fieldType, description := getMetadataFieldInfo(resRef.Path)

	return fmt.Sprintf(
		"`resources.%s.%s`\n\n"+
			"**field type:** `%s`\n\n%s\n\n"+
			"### Resource information\n\n%s",
		resourceName,
		fieldPath,
		fieldType,
		description,
		resourceInfo,
	)
}

// getMetadataFieldInfo returns the type and description for a metadata path.
func getMetadataFieldInfo(path []*substitutions.SubstitutionPathItem) (string, string) {
	if len(path) == 0 {
		return "object", "Resource metadata container"
	}

	firstField := path[0].FieldName
	if firstField != "metadata" {
		return "any", ""
	}

	if len(path) == 1 {
		return "object", "Resource metadata containing annotations and labels"
	}

	secondField := path[1].FieldName
	switch secondField {
	case "annotations":
		if len(path) == 2 {
			return "map[string]string", "Key-value pairs for storing arbitrary metadata about the resource"
		}
		// Specific annotation key
		return "string", fmt.Sprintf("Annotation value for key `%s`", path[2].FieldName)
	case "labels":
		if len(path) == 2 {
			return "map[string]string", "Key-value pairs for organizing and categorizing resources"
		}
		// Specific label key
		return "string", fmt.Sprintf("Label value for key `%s`", path[2].FieldName)
	case "displayName":
		return "string", "Human-readable display name for the resource"
	case "custom":
		if len(path) == 2 {
			return "object", "Custom metadata fields defined by the resource provider"
		}
		return "any", "Custom metadata field"
	}

	return "any", ""
}

// RenderPathItemFieldInfo renders path item field information from a resource spec schema.
func RenderPathItemFieldInfo(
	fieldName string,
	specFieldSchema *provider.ResourceDefinitionsSchema,
) string {
	if specFieldSchema == nil {
		return fmt.Sprintf("`%s`", fieldName)
	}

	description := ""
	if specFieldSchema.FormattedDescription != "" {
		description = specFieldSchema.FormattedDescription
	} else if specFieldSchema.Description != "" {
		description = specFieldSchema.Description
	}

	return fmt.Sprintf(
		"`%s`\n\n**type:** `%s`\n\n%s",
		fieldName,
		specFieldSchema.Type,
		description,
	)
}

// RenderResourceMetadataPathItemInfo renders path item information for resource metadata fields.
func RenderResourceMetadataPathItemInfo(
	fieldName string,
	resRef *substitutions.SubstitutionResourceProperty,
	pathItemIndex int,
) string {
	if pathItemIndex >= len(resRef.Path) {
		return fmt.Sprintf("`%s`", fieldName)
	}

	pathToItem := resRef.Path[:pathItemIndex+1]
	fieldType, description := getMetadataFieldInfo(pathToItem)
	fullPath := renderFieldPath(pathToItem)

	return fmt.Sprintf(
		"`resources.%s.%s`\n\n**field type:** `%s`\n\n%s",
		resRef.ResourceName,
		fullPath,
		fieldType,
		description,
	)
}

// RenderValuePathItemInfo renders path item information for value references.
func RenderValuePathItemInfo(
	fieldName string,
	valRef *substitutions.SubstitutionValueReference,
	pathItem *substitutions.SubstitutionPathItem,
) string {
	if pathItem.ArrayIndex != nil {
		return fmt.Sprintf(
			"`[%d]`\n\nArray index access on `values.%s`",
			*pathItem.ArrayIndex,
			valRef.ValueName,
		)
	}

	return fmt.Sprintf(
		"`%s`\n\nField access on `values.%s`",
		fieldName,
		valRef.ValueName,
	)
}

// RenderChildExportFieldInfo renders enriched hover information for a child
// blueprint exported field, showing the export's type, field path, and description.
func RenderChildExportFieldInfo(
	fieldName string,
	childRef *substitutions.SubstitutionChild,
	exportType string,
	exportField string,
	exportDescription string,
) string {
	if exportType == "" {
		exportType = "unknown"
	}

	fieldInfo := ""
	if exportField != "" {
		fieldInfo = fmt.Sprintf("**field:** `%s`\n\n", exportField)
	}

	return fmt.Sprintf(
		"`children.%s.%s`\n\n"+
			"**type:** `%s`\n\n%s%s",
		childRef.ChildName,
		fieldName,
		exportType,
		fieldInfo,
		exportDescription,
	)
}

// RenderChildPathItemInfo renders path item information for child references.
func RenderChildPathItemInfo(
	fieldName string,
	childRef *substitutions.SubstitutionChild,
	pathItem *substitutions.SubstitutionPathItem,
) string {
	if pathItem.ArrayIndex != nil {
		return fmt.Sprintf(
			"`[%d]`\n\nArray index access on `children.%s`",
			*pathItem.ArrayIndex,
			childRef.ChildName,
		)
	}

	return fmt.Sprintf(
		"`%s`\n\nField access on exported value from child blueprint `children.%s`",
		fieldName,
		childRef.ChildName,
	)
}
