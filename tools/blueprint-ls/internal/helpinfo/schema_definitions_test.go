package helpinfo

import (
	"testing"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type SchemaDefinitionsSuite struct {
	suite.Suite
}

func (s *SchemaDefinitionsSuite) Test_RenderLabelsDefinition() {
	result := RenderLabelsDefinition()
	s.NotEmpty(result)
	s.Contains(result, "labels")
}

func (s *SchemaDefinitionsSuite) Test_RenderAnnotationsDefinition() {
	result := RenderAnnotationsDefinition()
	s.NotEmpty(result)
	s.Contains(result, "annotations")
}

func (s *SchemaDefinitionsSuite) Test_RenderLinkSelectorDefinition() {
	result := RenderLinkSelectorDefinition()
	s.NotEmpty(result)
	s.Contains(result, "linkSelector")
}

func (s *SchemaDefinitionsSuite) Test_RenderByLabelDefinition() {
	result := RenderByLabelDefinition()
	s.NotEmpty(result)
	s.Contains(result, "byLabel")
}

func (s *SchemaDefinitionsSuite) Test_RenderExcludeDefinition() {
	result := RenderExcludeDefinition()
	s.NotEmpty(result)
	s.Contains(result, "exclude")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterSearchDefinition() {
	result := RenderDataSourceFilterSearchDefinition()
	s.NotEmpty(result)
	s.Contains(result, "search")
}

func (s *SchemaDefinitionsSuite) Test_RenderSectionDefinition() {
	tests := []struct {
		name     string
		section  string
		contains string
		empty    bool
	}{
		{"resources section", "resources", "resources", false},
		{"variables section", "variables", "variables", false},
		{"values section", "values", "values", false},
		{"datasources section", "datasources", "datasources", false},
		{"includes section", "includes", "includes", false},
		{"exports section", "exports", "exports", false},
		{"version section", "version", "version", false},
		{"transform section", "transform", "transform", false},
		{"metadata section", "metadata", "metadata", false},
		{"unknown section", "unknown", "", true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := RenderSectionDefinition(tt.section)
			if tt.empty {
				s.Empty(result)
			} else {
				s.Contains(result, tt.contains)
			}
		})
	}
}

func (s *SchemaDefinitionsSuite) Test_RenderFieldDefinition() {
	tests := []struct {
		name    string
		field   string
		parent  string
		contains string
		empty   bool
	}{
		{"resource type field", "type", "resource", "type", false},
		{"resource metadata field", "metadata", "resource", "metadata", false},
		{"resource spec field", "spec", "resource", "spec", false},
		{"resource linkSelector field", "linkSelector", "resource", "linkSelector", false},
		{"resource condition field", "condition", "resource", "condition", false},
		{"resource each field", "each", "resource", "each", false},
		{"resource description field", "description", "resource", "description", false},
		{"resource dependsOn field", "dependsOn", "resource", "dependsOn", false},
		{"variable type field", "type", "variable", "type", false},
		{"variable description field", "description", "variable", "description", false},
		{"variable secret field", "secret", "variable", "secret", false},
		{"value type field", "type", "value", "type", false},
		{"value value field", "value", "value", "value", false},
		{"datasource type field", "type", "datasource", "type", false},
		{"datasource filter field", "filter", "datasource", "filter", false},
		{"datasource exports field", "exports", "datasource", "exports", false},
		{"include path field", "path", "include", "path", false},
		{"include variables field", "variables", "include", "variables", false},
		{"export type field", "type", "export", "type", false},
		{"export field field", "field", "export", "field", false},
		{"unknown parent", "type", "unknown", "", true},
		{"unknown field", "unknown", "resource", "", true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := RenderFieldDefinition(tt.field, tt.parent)
			if tt.empty {
				s.Empty(result)
			} else {
				s.Contains(result, tt.contains)
			}
		})
	}
}

func (s *SchemaDefinitionsSuite) Test_RenderMetadataFieldDefinition() {
	tests := []struct {
		name     string
		field    string
		contains string
		empty    bool
	}{
		{"displayName", "displayName", "displayName", false},
		{"annotations", "annotations", "annotations", false},
		{"labels", "labels", "labels", false},
		{"custom", "custom", "custom", false},
		{"unknown", "unknown", "", true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := RenderMetadataFieldDefinition(tt.field)
			if tt.empty {
				s.Empty(result)
			} else {
				s.Contains(result, tt.contains)
			}
		})
	}
}

func (s *SchemaDefinitionsSuite) Test_RenderMetadataDefinition() {
	tests := []struct {
		name    string
		parent  string
		contains string
	}{
		{"resource context", "resource", "resource"},
		{"datasource context", "datasource", "data source"},
		{"unknown context", "unknown", "metadata"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := RenderMetadataDefinition(tt.parent)
			s.Contains(result, tt.contains)
		})
	}
}

func (s *SchemaDefinitionsSuite) Test_RenderResourceHoverInfo_with_type_and_description() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("An orders table")},
			},
		},
	}
	result := RenderResourceHoverInfo("ordersTable", resource)
	s.Contains(result, "Resource")
	s.Contains(result, "ordersTable")
	s.Contains(result, "aws/dynamodb/table")
	s.Contains(result, "An orders table")
}

func (s *SchemaDefinitionsSuite) Test_RenderResourceHoverInfo_nil_type() {
	resource := &schema.Resource{}
	result := RenderResourceHoverInfo("ordersTable", resource)
	s.Contains(result, "Resource")
	s.Contains(result, "ordersTable")
	s.NotContains(result, "type:")
}

func (s *SchemaDefinitionsSuite) Test_RenderVariableHoverInfo_with_type_and_description() {
	variable := &schema.Variable{
		Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &bpcore.ScalarValue{
			StringValue: strPtr("The environment name"),
		},
	}
	result := RenderVariableHoverInfo("environment", variable)
	s.Contains(result, "Variable")
	s.Contains(result, "environment")
	s.Contains(result, "string")
	s.Contains(result, "The environment name")
}

func (s *SchemaDefinitionsSuite) Test_RenderVariableHoverInfo_minimal() {
	variable := &schema.Variable{}
	result := RenderVariableHoverInfo("env", variable)
	s.Contains(result, "Variable")
	s.Contains(result, "env")
	s.NotContains(result, "type:")
}

func (s *SchemaDefinitionsSuite) Test_RenderValueHoverInfo_with_type_and_description() {
	value := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("The table name")},
			},
		},
	}
	result := RenderValueHoverInfo("tableName", value)
	s.Contains(result, "Value")
	s.Contains(result, "tableName")
	s.Contains(result, "string")
	s.Contains(result, "The table name")
}

func (s *SchemaDefinitionsSuite) Test_RenderValueHoverInfo_minimal() {
	value := &schema.Value{}
	result := RenderValueHoverInfo("tableName", value)
	s.Contains(result, "Value")
	s.Contains(result, "tableName")
	s.NotContains(result, "type:")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceHoverInfo_with_type_and_description() {
	ds := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("The VPC data source")},
			},
		},
	}
	result := RenderDataSourceHoverInfo("network", ds)
	s.Contains(result, "Data Source")
	s.Contains(result, "network")
	s.Contains(result, "aws/vpc")
	s.Contains(result, "The VPC data source")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceHoverInfo_minimal() {
	ds := &schema.DataSource{}
	result := RenderDataSourceHoverInfo("network", ds)
	s.Contains(result, "Data Source")
	s.Contains(result, "network")
	s.NotContains(result, "type:")
}

func (s *SchemaDefinitionsSuite) Test_RenderIncludeHoverInfo_with_path_and_description() {
	include := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("./child.blueprint.yaml")},
			},
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("A child blueprint")},
			},
		},
	}
	result := RenderIncludeHoverInfo("paymentProcessor", include)
	s.Contains(result, "Include")
	s.Contains(result, "paymentProcessor")
	s.Contains(result, "./child.blueprint.yaml")
	s.Contains(result, "A child blueprint")
}

func (s *SchemaDefinitionsSuite) Test_RenderIncludeHoverInfo_minimal() {
	include := &schema.Include{}
	result := RenderIncludeHoverInfo("child", include)
	s.Contains(result, "Include")
	s.Contains(result, "child")
	s.NotContains(result, "path:")
}

func (s *SchemaDefinitionsSuite) Test_RenderExportHoverInfo() {
	export := &schema.Export{}
	result := RenderExportHoverInfo("apiEndpoint", export)
	s.Contains(result, "Export")
	s.Contains(result, "apiEndpoint")
}

func (s *SchemaDefinitionsSuite) Test_RenderAnnotationKeyInfo_with_value() {
	result := RenderAnnotationKeyInfo("aws.lambda.timeout", "30")
	s.Contains(result, "Annotation")
	s.Contains(result, "aws.lambda.timeout")
	s.Contains(result, "30")
}

func (s *SchemaDefinitionsSuite) Test_RenderAnnotationKeyInfo_empty_value() {
	result := RenderAnnotationKeyInfo("aws.lambda.timeout", "")
	s.Contains(result, "Annotation")
	s.Contains(result, "aws.lambda.timeout")
	s.NotContains(result, "value:")
}

func (s *SchemaDefinitionsSuite) Test_RenderLinkAnnotationDefinition_full() {
	def := &provider.LinkAnnotationDefinition{
		Type:        bpcore.ScalarTypeString,
		Description: "Timeout in seconds",
		Required:    true,
		DefaultValue: &bpcore.ScalarValue{
			StringValue: strPtr("30"),
		},
		AllowedValues: []*bpcore.ScalarValue{
			{StringValue: strPtr("30")},
			{StringValue: strPtr("60")},
			{StringValue: strPtr("90")},
		},
	}
	result := RenderLinkAnnotationDefinition("aws.lambda.timeout", def)
	s.Contains(result, "Link Annotation")
	s.Contains(result, "aws.lambda.timeout")
	s.Contains(result, "string")
	s.Contains(result, "required")
	s.Contains(result, "Timeout in seconds")
	s.Contains(result, "default:")
	s.Contains(result, "allowed values:")
}

func (s *SchemaDefinitionsSuite) Test_RenderLinkAnnotationDefinition_minimal() {
	def := &provider.LinkAnnotationDefinition{
		Type: bpcore.ScalarTypeInteger,
	}
	result := RenderLinkAnnotationDefinition("myAnnotation", def)
	s.Contains(result, "Link Annotation")
	s.Contains(result, "myAnnotation")
	s.Contains(result, "integer")
	s.NotContains(result, "required")
	s.NotContains(result, "default:")
	s.NotContains(result, "allowed values:")
}

func (s *SchemaDefinitionsSuite) Test_RenderSpecFieldDefinition_with_schema() {
	fieldSchema := &provider.ResourceDefinitionsSchema{
		Type:                 "string",
		Description:          "The table name",
		FormattedDescription: "The **table** name",
	}
	result := RenderSpecFieldDefinition("tableName", fieldSchema)
	s.Contains(result, "tableName")
	s.Contains(result, "string")
	s.Contains(result, "The **table** name")
}

func (s *SchemaDefinitionsSuite) Test_RenderSpecFieldDefinition_nil_schema() {
	result := RenderSpecFieldDefinition("tableName", nil)
	s.Contains(result, "tableName")
	s.NotContains(result, "type:")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceExportFieldDefinition_with_alias() {
	ds := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
	}
	export := &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{Value: schema.DataSourceFieldTypeString},
		AliasFor: &bpcore.ScalarValue{
			StringValue: strPtr("vpcId"),
		},
	}
	result := RenderDataSourceExportFieldDefinition("id", ds, export, nil)
	s.Contains(result, "Export Field")
	s.Contains(result, "id")
	s.Contains(result, "string")
	s.Contains(result, "alias for:")
	s.Contains(result, "vpcId")
	s.Contains(result, "aws/vpc")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceExportFieldDefinition_with_spec_schema() {
	ds := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
	}
	export := &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{Value: schema.DataSourceFieldTypeString},
	}
	specSchema := &provider.DataSourceSpecSchema{
		Description: "The VPC identifier",
	}
	result := RenderDataSourceExportFieldDefinition("vpcId", ds, export, specSchema)
	s.Contains(result, "Export Field")
	s.Contains(result, "vpcId")
	s.Contains(result, "The VPC identifier")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceExportFieldDefinition_nil_schema() {
	ds := &schema.DataSource{}
	export := &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{Value: schema.DataSourceFieldTypeInteger},
	}
	result := RenderDataSourceExportFieldDefinition("count", ds, export, nil)
	s.Contains(result, "Export Field")
	s.Contains(result, "count")
	s.Contains(result, "integer")
	s.Contains(result, "unknown") // data source type is unknown
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFieldTypeDefinition() {
	result := RenderDataSourceFieldTypeDefinition("string")
	s.Contains(result, "string")
	s.Contains(result, "type")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterDefinition_with_field_and_operator() {
	filter := &schema.DataSourceFilter{
		Field: &bpcore.ScalarValue{
			StringValue: strPtr("vpcId"),
		},
		Operator: &schema.DataSourceFilterOperatorWrapper{
			Value: "=",
		},
	}
	result := RenderDataSourceFilterDefinition(filter)
	s.Contains(result, "filter")
	s.Contains(result, "vpcId")
	s.Contains(result, "=")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterDefinition_minimal() {
	filter := &schema.DataSourceFilter{}
	result := RenderDataSourceFilterDefinition(filter)
	s.Contains(result, "filter")
	s.NotContains(result, "field:")
	s.NotContains(result, "operator:")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterOperatorDefinition() {
	result := RenderDataSourceFilterOperatorDefinition("=")
	s.Contains(result, "=")
	s.Contains(result, "operator")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterFieldKeyDefinition_with_schema() {
	filterSchema := &provider.DataSourceFilterSchema{
		Type:        "string",
		Description: "The VPC identifier field",
	}
	result := RenderDataSourceFilterFieldKeyDefinition("vpcId", filterSchema)
	s.Contains(result, "vpcId")
	s.Contains(result, "The VPC identifier field")
	s.Contains(result, "string")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterFieldKeyDefinition_nil_schema() {
	result := RenderDataSourceFilterFieldKeyDefinition("vpcId", nil)
	s.Contains(result, "vpcId")
	s.NotContains(result, "type:")
}

func (s *SchemaDefinitionsSuite) Test_RenderDataSourceFilterFieldKeyDefinition_empty_value() {
	result := RenderDataSourceFilterFieldKeyDefinition("", nil)
	s.Contains(result, "field")
	s.NotContains(result, "`\n")
}

func (s *SchemaDefinitionsSuite) Test_RenderByLabelHoverContent_no_matches_no_key() {
	result := RenderByLabelHoverContent("", "", nil)
	s.Contains(result, "byLabel")
	s.Contains(result, "No matching resources found")
	s.NotContains(result, "Matching resources:")
}

func (s *SchemaDefinitionsSuite) Test_RenderByLabelHoverContent_with_label_key_no_matches() {
	result := RenderByLabelHoverContent("application", "orders", nil)
	s.Contains(result, "byLabel")
	s.Contains(result, "**application:** `orders`")
	s.Contains(result, "No matching resources found")
}

func (s *SchemaDefinitionsSuite) Test_RenderByLabelHoverContent_with_matches() {
	matches := []MatchingResourceInfo{
		{Name: "handler", ResourceType: "aws/lambda/function"},
	}
	result := RenderByLabelHoverContent("", "", matches)
	s.Contains(result, "Matching resources:")
	s.Contains(result, "handler")
	s.Contains(result, "aws/lambda/function")
	s.NotContains(result, "linked")
}

func (s *SchemaDefinitionsSuite) Test_RenderByLabelHoverContent_with_linked_matches() {
	matches := []MatchingResourceInfo{
		{Name: "handler", ResourceType: "aws/lambda/function", Linked: true},
	}
	result := RenderByLabelHoverContent("application", "orders", matches)
	s.Contains(result, "**application:** `orders`")
	s.Contains(result, "Matching resources:")
	s.Contains(result, "handler")
	s.Contains(result, "aws/lambda/function")
	s.Contains(result, "linked")
}

func (s *SchemaDefinitionsSuite) Test_RenderByLabelHoverContent_match_without_type() {
	matches := []MatchingResourceInfo{
		{Name: "handler"},
	}
	result := RenderByLabelHoverContent("", "", matches)
	s.Contains(result, "handler")
	s.NotContains(result, "—")
}

func TestSchemaDefinitionsSuite(t *testing.T) {
	suite.Run(t, new(SchemaDefinitionsSuite))
}
