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
