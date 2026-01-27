package helpinfo

import (
	"testing"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type HelpInfoSuite struct {
	suite.Suite
}

// Helper to create a string pointer
func strPtr(s string) *string {
	return &s
}

// Helper to create an int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}

func (s *HelpInfoSuite) TestRenderVariableInfo_WithTypeAndDescription() {
	variable := &schema.Variable{
		Type: &schema.VariableTypeWrapper{Value: schema.VariableTypeString},
		Description: &bpcore.ScalarValue{
			StringValue: strPtr("A test variable"),
		},
	}
	result := RenderVariableInfo("testVar", variable)
	s.Contains(result, "```variables.testVar```")
	s.Contains(result, "**type:** `string`")
	s.Contains(result, "A test variable")
}

func (s *HelpInfoSuite) TestRenderVariableInfo_NilType() {
	variable := &schema.Variable{
		Type: nil,
		Description: &bpcore.ScalarValue{
			StringValue: strPtr("A test variable"),
		},
	}
	result := RenderVariableInfo("testVar", variable)
	s.Contains(result, "**type:** `unknown`")
}

func (s *HelpInfoSuite) TestRenderVariableInfo_NilDescription() {
	variable := &schema.Variable{
		Type:        &schema.VariableTypeWrapper{Value: schema.VariableTypeInteger},
		Description: nil,
	}
	result := RenderVariableInfo("testVar", variable)
	s.Contains(result, "**type:** `integer`")
}

func (s *HelpInfoSuite) TestRenderValueInfo_WithTypeAndDescription() {
	value := &schema.Value{
		Type: &schema.ValueTypeWrapper{Value: schema.ValueTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("A computed value")},
			},
		},
	}
	result := RenderValueInfo("computedVal", value)
	s.Contains(result, "```values.computedVal```")
	s.Contains(result, "**type:** `string`")
}

func (s *HelpInfoSuite) TestRenderValueInfo_NilType() {
	value := &schema.Value{
		Type: nil,
	}
	result := RenderValueInfo("testVal", value)
	s.Contains(result, "**type:** `unknown`")
}

func (s *HelpInfoSuite) TestRenderChildInfo_WithPathAndDescription() {
	child := &schema.Include{
		Path: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("./child.blueprint.yaml")},
			},
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("Child blueprint")},
			},
		},
	}
	result := RenderChildInfo("childBlueprint", child)
	s.Contains(result, "```includes.childBlueprint```")
	s.Contains(result, "**path:**")
}

func (s *HelpInfoSuite) TestRenderChildInfo_NilPath() {
	child := &schema.Include{
		Path: nil,
	}
	result := RenderChildInfo("childBlueprint", child)
	s.Contains(result, "```includes.childBlueprint```")
}

func (s *HelpInfoSuite) TestRenderBasicResourceInfo_WithTypeAndDescription() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("A Lambda function")},
			},
		},
	}
	result := RenderBasicResourceInfo("myFunction", resource)
	s.Contains(result, "```resources.myFunction```")
	s.Contains(result, "**type:** `aws/lambda/function`")
	s.Contains(result, "A Lambda function")
}

func (s *HelpInfoSuite) TestRenderBasicResourceInfo_NilType() {
	resource := &schema.Resource{
		Type: nil,
	}
	result := RenderBasicResourceInfo("myResource", resource)
	s.Contains(result, "**type:** `unknown`")
}

func (s *HelpInfoSuite) TestRenderResourceDefinitionFieldInfo_NilSchema() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myFunction",
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "spec"},
		},
	}
	result := RenderResourceDefinitionFieldInfo("myFunction", resource, resRef, nil)
	// When schema is nil, it should just return basic resource info
	s.Contains(result, "```resources.myFunction```")
	s.NotContains(result, "### Resource information")
}

func (s *HelpInfoSuite) TestRenderResourceDefinitionFieldInfo_WithSchema() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myFunction",
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "runtime"},
		},
	}
	specSchema := &provider.ResourceDefinitionsSchema{
		Type:        "string",
		Description: "The runtime for the Lambda function",
	}
	result := RenderResourceDefinitionFieldInfo("myFunction", resource, resRef, specSchema)
	s.Contains(result, "`runtime`")
	s.Contains(result, "**field type:** `string`")
	s.Contains(result, "The runtime for the Lambda function")
	s.Contains(result, "### Resource information")
}

func (s *HelpInfoSuite) TestRenderResourceDefinitionFieldInfo_WithFormattedDescription() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myFunction",
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "runtime"},
		},
	}
	specSchema := &provider.ResourceDefinitionsSchema{
		Type:                 "string",
		Description:          "Basic description",
		FormattedDescription: "**Formatted** description",
	}
	result := RenderResourceDefinitionFieldInfo("myFunction", resource, resRef, specSchema)
	s.Contains(result, "**Formatted** description")
	s.NotContains(result, "Basic description")
}

func (s *HelpInfoSuite) TestRenderBasicDataSourceInfo_Complete() {
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("A VPC data source")},
			},
		},
	}
	result := RenderBasicDataSourceInfo("myVpc", dataSource)
	s.Contains(result, "```datasources.myVpc```")
	s.Contains(result, "**type:** `aws/vpc`")
	s.Contains(result, "A VPC data source")
}

func (s *HelpInfoSuite) TestRenderBasicDataSourceInfo_NilType() {
	dataSource := &schema.DataSource{
		Type: nil,
	}
	result := RenderBasicDataSourceInfo("myDataSource", dataSource)
	s.Contains(result, "**type:** `unknown`")
}

func (s *HelpInfoSuite) TestRenderDataSourceFieldInfo_WithAliasFor() {
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
	}
	dataSourceRef := &substitutions.SubstitutionDataSourceProperty{
		DataSourceName: "myVpc",
		FieldName:      "vpcId",
	}
	dataSourceField := &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{Value: schema.DataSourceFieldTypeString},
		AliasFor: &bpcore.ScalarValue{
			StringValue: strPtr("id"),
		},
	}
	result := RenderDataSourceFieldInfo("myVpc", dataSource, dataSourceRef, dataSourceField)
	s.Contains(result, "`datasources.myVpc.vpcId`")
	s.Contains(result, "**field type:** `string`")
}

func (s *HelpInfoSuite) TestRenderDataSourceFieldInfo_WithPrimitiveArrIndex() {
	dataSource := &schema.DataSource{
		Type: &schema.DataSourceTypeWrapper{Value: "aws/vpc"},
	}
	dataSourceRef := &substitutions.SubstitutionDataSourceProperty{
		DataSourceName:    "myVpc",
		FieldName:         "subnets",
		PrimitiveArrIndex: int64Ptr(0),
	}
	dataSourceField := &schema.DataSourceFieldExport{
		Type: &schema.DataSourceFieldTypeWrapper{Value: schema.DataSourceFieldTypeString},
	}
	result := RenderDataSourceFieldInfo("myVpc", dataSource, dataSourceRef, dataSourceField)
	s.Contains(result, "`datasources.myVpc.subnets[0]`")
}

func (s *HelpInfoSuite) TestRenderElemRefInfo_WithPath() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	elemRef := &substitutions.SubstitutionElemReference{
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "arn"},
		},
	}
	result := RenderElemRefInfo("myFunction", resource, elemRef)
	s.Contains(result, "`resources.myFunction[i].arn`")
	s.Contains(result, "nth element of resource template")
}

func (s *HelpInfoSuite) TestRenderElemIndexRefInfo() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	result := RenderElemIndexRefInfo("myFunction", resource)
	s.Contains(result, "index of nth element")
	s.Contains(result, "resources.myFunction")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_EmptyPath() {
	fieldType, description := getMetadataFieldInfo([]*substitutions.SubstitutionPathItem{})
	s.Equal("object", fieldType)
	s.Equal("Resource metadata container", description)
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_MetadataOnly() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "metadata"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("object", fieldType)
	s.Contains(description, "annotations and labels")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_AnnotationsPath() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "metadata"},
		{FieldName: "annotations"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("map[string]string", fieldType)
	s.Contains(description, "arbitrary metadata")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_SpecificAnnotationKey() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "metadata"},
		{FieldName: "annotations"},
		{FieldName: "myKey"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("string", fieldType)
	s.Contains(description, "myKey")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_LabelsPath() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "metadata"},
		{FieldName: "labels"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("map[string]string", fieldType)
	s.Contains(description, "organizing and categorizing")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_DisplayNamePath() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "metadata"},
		{FieldName: "displayName"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("string", fieldType)
	s.Contains(description, "Human-readable")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_CustomPath() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "metadata"},
		{FieldName: "custom"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("object", fieldType)
	s.Contains(description, "Custom metadata")
}

func (s *HelpInfoSuite) TestGetMetadataFieldInfo_NonMetadataPath() {
	path := []*substitutions.SubstitutionPathItem{
		{FieldName: "spec"},
	}
	fieldType, description := getMetadataFieldInfo(path)
	s.Equal("any", fieldType)
	s.Empty(description)
}

func (s *HelpInfoSuite) TestRenderPathItemFieldInfo_NilSchema() {
	result := RenderPathItemFieldInfo("fieldName", nil)
	s.Equal("`fieldName`", result)
}

func (s *HelpInfoSuite) TestRenderPathItemFieldInfo_WithSchema() {
	specSchema := &provider.ResourceDefinitionsSchema{
		Type:        "string",
		Description: "Field description",
	}
	result := RenderPathItemFieldInfo("fieldName", specSchema)
	s.Contains(result, "`fieldName`")
	s.Contains(result, "**type:** `string`")
	s.Contains(result, "Field description")
}

func (s *HelpInfoSuite) TestRenderResourceMetadataFieldInfo_EmptyPath() {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
	}
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myFunction",
		Path:         []*substitutions.SubstitutionPathItem{},
	}
	result := RenderResourceMetadataFieldInfo("myFunction", resource, resRef)
	// When path is empty, returns basic resource info
	s.Contains(result, "```resources.myFunction```")
	s.NotContains(result, "### Resource information")
}

func (s *HelpInfoSuite) TestRenderResourceMetadataPathItemInfo_ValidPath() {
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myFunction",
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "metadata"},
			{FieldName: "annotations"},
		},
	}
	result := RenderResourceMetadataPathItemInfo("annotations", resRef, 1)
	s.Contains(result, "`resources.myFunction.metadata.annotations`")
	s.Contains(result, "**field type:** `map[string]string`")
}

func (s *HelpInfoSuite) TestRenderResourceMetadataPathItemInfo_IndexOutOfBounds() {
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myFunction",
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "metadata"},
		},
	}
	result := RenderResourceMetadataPathItemInfo("fieldName", resRef, 5)
	s.Equal("`fieldName`", result)
}

func (s *HelpInfoSuite) TestRenderValuePathItemInfo_WithFieldName() {
	valRef := &substitutions.SubstitutionValueReference{
		ValueName: "myValue",
	}
	pathItem := &substitutions.SubstitutionPathItem{
		FieldName: "nested",
	}
	result := RenderValuePathItemInfo("nested", valRef, pathItem)
	s.Contains(result, "`nested`")
	s.Contains(result, "Field access on `values.myValue`")
}

func (s *HelpInfoSuite) TestRenderValuePathItemInfo_WithArrayIndex() {
	valRef := &substitutions.SubstitutionValueReference{
		ValueName: "myValue",
	}
	pathItem := &substitutions.SubstitutionPathItem{
		ArrayIndex: int64Ptr(2),
	}
	result := RenderValuePathItemInfo("", valRef, pathItem)
	s.Contains(result, "`[2]`")
	s.Contains(result, "Array index access on `values.myValue`")
}

func (s *HelpInfoSuite) TestRenderChildPathItemInfo_WithFieldName() {
	childRef := &substitutions.SubstitutionChild{
		ChildName: "myChild",
	}
	pathItem := &substitutions.SubstitutionPathItem{
		FieldName: "output",
	}
	result := RenderChildPathItemInfo("output", childRef, pathItem)
	s.Contains(result, "`output`")
	s.Contains(result, "Field access on exported value from child blueprint `children.myChild`")
}

func (s *HelpInfoSuite) TestRenderChildPathItemInfo_WithArrayIndex() {
	childRef := &substitutions.SubstitutionChild{
		ChildName: "myChild",
	}
	pathItem := &substitutions.SubstitutionPathItem{
		ArrayIndex: int64Ptr(0),
	}
	result := RenderChildPathItemInfo("", childRef, pathItem)
	s.Contains(result, "`[0]`")
	s.Contains(result, "Array index access on `children.myChild`")
}

func TestHelpInfoSuite(t *testing.T) {
	suite.Run(t, new(HelpInfoSuite))
}
