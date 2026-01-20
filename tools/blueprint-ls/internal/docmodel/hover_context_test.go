package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type HoverContextSuite struct {
	suite.Suite
}

func (s *HoverContextSuite) TestDetermineHoverContext_FunctionCall() {
	funcExpr := &substitutions.SubstitutionFunctionExpr{
		FunctionName: "len",
	}
	collected := []*schema.TreeNode{
		{Path: "/resources", SchemaElement: &schema.ResourceMap{}},
		{Path: "/resources/myResource", SchemaElement: &schema.Resource{}},
		{Path: "/resources/myResource/spec/functionCall/len", SchemaElement: funcExpr},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementFunctionCall, ctx.ElementKind)
	s.Assert().Equal(funcExpr, ctx.SchemaElement)
	s.Assert().Equal(collected[2], ctx.TreeNode)
	s.Assert().Len(ctx.AncestorNodes, 3)
}

func (s *HoverContextSuite) TestDetermineHoverContext_VariableRef() {
	varRef := &substitutions.SubstitutionVariable{
		VariableName: "myVar",
	}
	collected := []*schema.TreeNode{
		{Path: "/resources", SchemaElement: &schema.ResourceMap{}},
		{Path: "/resources/myResource/spec/varRef/myVar", SchemaElement: varRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementVariableRef, ctx.ElementKind)
	s.Assert().Equal(varRef, ctx.SchemaElement)
}

func (s *HoverContextSuite) TestDetermineHoverContext_ValueRef() {
	valRef := &substitutions.SubstitutionValueReference{
		ValueName: "myValue",
	}
	collected := []*schema.TreeNode{
		{Path: "/values/myValue/valRef/myValue", SchemaElement: valRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementValueRef, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_ChildRef() {
	childRef := &substitutions.SubstitutionChild{
		ChildName: "myChild",
	}
	collected := []*schema.TreeNode{
		{Path: "/include/myChild/childRef/myChild", SchemaElement: childRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementChildRef, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_ResourceRef() {
	resRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: "myResource",
	}
	collected := []*schema.TreeNode{
		{Path: "/resources/other/spec/resourceRef/myResource", SchemaElement: resRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementResourceRef, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_DataSourceRef() {
	dsRef := &substitutions.SubstitutionDataSourceProperty{
		DataSourceName: "myDS",
	}
	collected := []*schema.TreeNode{
		{Path: "/resources/myResource/spec/datasourceRef/myDS", SchemaElement: dsRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementDataSourceRef, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_ElemRef() {
	elemRef := &substitutions.SubstitutionElemReference{}
	collected := []*schema.TreeNode{
		{Path: "/resources/myResource/each/elemRef", SchemaElement: elemRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementElemRef, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_ElemIndexRef() {
	elemIndexRef := &substitutions.SubstitutionElemIndexReference{}
	collected := []*schema.TreeNode{
		{Path: "/resources/myResource/each/elemIndexRef", SchemaElement: elemIndexRef},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementElemIndexRef, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_ResourceType() {
	resType := &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"}
	collected := []*schema.TreeNode{
		{Path: "/resources", SchemaElement: &schema.ResourceMap{}},
		{Path: "/resources/myTable", SchemaElement: &schema.Resource{}},
		{Path: "/resources/myTable/type", SchemaElement: resType},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementResourceType, ctx.ElementKind)
	s.Assert().Equal(resType, ctx.SchemaElement)
}

func (s *HoverContextSuite) TestDetermineHoverContext_DataSourceType() {
	dsType := &schema.DataSourceTypeWrapper{Value: "aws/ec2/vpc"}
	collected := []*schema.TreeNode{
		{Path: "/datasources/myDS/type", SchemaElement: dsType},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	s.Assert().Equal(SchemaElementDataSourceType, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_SkipsNonHoverElements() {
	// Resource itself doesn't support hover, but ResourceType does
	resType := &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"}
	collected := []*schema.TreeNode{
		{Path: "/resources", SchemaElement: &schema.ResourceMap{}},
		{Path: "/resources/myTable", SchemaElement: &schema.Resource{}},
		{Path: "/resources/myTable/type", SchemaElement: resType},
	}

	ctx := DetermineHoverContext(collected)

	s.Require().NotNil(ctx)
	// Should find the ResourceType, not the Resource
	s.Assert().Equal(SchemaElementResourceType, ctx.ElementKind)
}

func (s *HoverContextSuite) TestDetermineHoverContext_EmptyCollected() {
	ctx := DetermineHoverContext([]*schema.TreeNode{})
	s.Assert().Nil(ctx)
}

func (s *HoverContextSuite) TestDetermineHoverContext_NoHoverableElements() {
	collected := []*schema.TreeNode{
		{Path: "/resources", SchemaElement: &schema.ResourceMap{}},
		{Path: "/resources/myTable", SchemaElement: &schema.Resource{}},
	}

	ctx := DetermineHoverContext(collected)
	s.Assert().Nil(ctx)
}

func (s *HoverContextSuite) TestSchemaElementKind_SupportsHover() {
	hoverableKinds := []SchemaElementKind{
		SchemaElementFunctionCall,
		SchemaElementVariableRef,
		SchemaElementValueRef,
		SchemaElementChildRef,
		SchemaElementResourceRef,
		SchemaElementDataSourceRef,
		SchemaElementElemRef,
		SchemaElementElemIndexRef,
		SchemaElementResourceType,
		SchemaElementDataSourceType,
	}

	for _, kind := range hoverableKinds {
		s.Assert().True(kind.SupportsHover(), "expected %s to support hover", kind)
	}

	nonHoverableKinds := []SchemaElementKind{
		SchemaElementUnknown,
		SchemaElementResource,
		SchemaElementDataSource,
		SchemaElementVariable,
		SchemaElementValue,
		SchemaElementResources,
		SchemaElementScalar,
	}

	for _, kind := range nonHoverableKinds {
		s.Assert().False(kind.SupportsHover(), "expected %s to not support hover", kind)
	}
}

func TestHoverContextSuite(t *testing.T) {
	suite.Run(t, new(HoverContextSuite))
}
