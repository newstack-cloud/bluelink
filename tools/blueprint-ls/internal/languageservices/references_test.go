package languageservices

import (
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type FindReferencesServiceSuite struct {
	suite.Suite
	service          *FindReferencesService
	state            *State
	logger           *zap.Logger
	blueprintContent string
	docCtx           *docmodel.DocumentContext
}

func (s *FindReferencesServiceSuite) SetupTest() {
	var err error
	s.logger, err = zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	s.state = NewState()
	s.service = NewFindReferencesService(s.state, s.logger)
	s.blueprintContent, err = loadTestBlueprintContent("blueprint-definitions.yaml")
	s.Require().NoError(err)

	blueprint, err := schema.LoadString(s.blueprintContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)

	tree := schema.SchemaToTree(blueprint)
	s.Require().NotNil(tree)

	s.docCtx = docmodel.NewDocumentContextFromSchema(
		"file:///blueprint.yaml",
		blueprint,
		tree,
	)
	s.docCtx.Content = s.blueprintContent
}

func (s *FindReferencesServiceSuite) Test_find_references_for_resource_from_definition() {
	// Cursor on getOrderHandler definition (line 126, 0-indexed: 125)
	refs := s.getReferences(125, 5, false)

	// Should find: substitution refs in metadata (lines 173, 179),
	// export field (line 189), dependsOn (line 161)
	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(173), "should find substitution ref on line 173")
	s.Assert().Contains(refLines, uint32(179), "should find substitution ref on line 179")
	s.Assert().Contains(refLines, uint32(161), "should find dependsOn ref on line 161")
	// Export field reference
	s.Assert().True(
		containsAnyLine(refLines, 189, 190),
		"should find export field ref around line 189-190",
	)
	s.Assert().NotContains(refLines, uint32(125), "should not include declaration")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_resource_from_definition_with_declaration() {
	refs := s.getReferences(125, 5, true)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(125), "should include declaration")
	s.Assert().Contains(refLines, uint32(173), "should find substitution ref on line 173")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_resource_from_substitution_ref() {
	// Cursor on ${resources.getOrderHandler...} (line 174, 0-indexed: 173)
	refs := s.getReferences(173, 25, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(173), "should find substitution ref on line 173")
	s.Assert().Contains(refLines, uint32(179), "should find substitution ref on line 179")
	s.Assert().Contains(refLines, uint32(161), "should find dependsOn ref on line 161")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_resource_from_depends_on() {
	// Cursor on getOrderHandler in dependsOn list (line 162, 0-indexed: 161)
	refs := s.getReferences(161, 10, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(173), "should find substitution ref on line 173")
	s.Assert().Contains(refLines, uint32(179), "should find substitution ref on line 179")
	s.Assert().Contains(refLines, uint32(161), "should find dependsOn ref on line 161")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_resource_from_export_field() {
	// Cursor on export field value resources.getOrderHandler... (line 190, 0-indexed: 189)
	refs := s.getReferences(189, 15, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(173), "should find substitution ref on line 173")
	s.Assert().Contains(refLines, uint32(161), "should find dependsOn ref on line 161")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_resource_from_exclude_item() {
	// Cursor on resource53 in exclude list (line 155, 0-indexed: 154)
	refs := s.getReferences(154, 10, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(154), "should find exclude ref on line 154")
	s.Assert().Contains(refLines, uint32(174), "should find substitution ref on line 174")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_variable_from_substitution_ref() {
	// Cursor on ${variables.certificateId} (line 108, 0-indexed: 107)
	refs := s.getReferences(107, 30, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(107), "should find substitution ref on line 107")
	// Export field reference for variables.certificateId
	s.Assert().True(
		containsAnyLine(refLines, 185, 186),
		"should find export field ref around line 185-186",
	)
	s.Assert().NotContains(refLines, uint32(8), "should not include declaration")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_variable_from_definition() {
	// Cursor on certificateId variable definition (line 9, 0-indexed: 8)
	refs := s.getReferences(8, 5, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(107), "should find substitution ref on line 107")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_variable_from_definition_with_declaration() {
	refs := s.getReferences(8, 5, true)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(8), "should include declaration")
	s.Assert().Contains(refLines, uint32(107), "should find substitution ref on line 107")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_value_from_substitution_ref() {
	// Cursor on ${values.derivitiveCertInfo} (line 172, 0-indexed: 171)
	refs := s.getReferences(171, 30, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(171), "should find substitution ref on line 171")
	// Export field reference for values.derivitiveCertInfo
	s.Assert().True(
		containsAnyLine(refLines, 201, 202),
		"should find export field ref around line 201-202",
	)
}

func (s *FindReferencesServiceSuite) Test_find_references_for_datasource_from_substitution_ref() {
	// Cursor on ${datasources.network.vpc} (line 177, 0-indexed: 176)
	refs := s.getReferences(176, 18, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(176), "should find substitution ref on line 176")
	s.Assert().Contains(refLines, uint32(177), "should find substitution ref on line 177")
	// Export field reference for datasources.network.vpc
	s.Assert().True(
		containsAnyLine(refLines, 197, 198),
		"should find export field ref around line 197-198",
	)
}

func (s *FindReferencesServiceSuite) Test_find_references_for_datasource_from_definition() {
	// Cursor on network datasource definition (line 50, 0-indexed: 49)
	refs := s.getReferences(49, 5, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(176), "should find substitution ref on line 176")
	s.Assert().Contains(refLines, uint32(177), "should find substitution ref on line 177")
}

func (s *FindReferencesServiceSuite) Test_find_references_for_child_from_substitution_ref() {
	// Cursor on ${children.networking.someChildInfo} (line 173, 0-indexed: 172)
	refs := s.getReferences(172, 25, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(172), "should find substitution ref on line 172")
	// Export field reference for children.networking.someExport
	s.Assert().True(
		containsAnyLine(refLines, 193, 194),
		"should find export field ref around line 193-194",
	)
}

func (s *FindReferencesServiceSuite) Test_find_references_for_child_from_definition() {
	// Cursor on networking include definition (line 167, 0-indexed: 166)
	refs := s.getReferences(166, 5, false)

	refLines := startLines(refs)
	s.Assert().Contains(refLines, uint32(172), "should find substitution ref on line 172")
}

func (s *FindReferencesServiceSuite) Test_find_references_returns_empty_for_non_ref_position() {
	refs := s.getReferences(0, 0, false)
	s.Assert().Empty(refs)
}

func (s *FindReferencesServiceSuite) Test_find_references_for_unreferenced_element() {
	// Cursor on logLevel2 variable definition (line 27, 0-indexed: 26)
	// logLevel2 is not referenced anywhere else in the blueprint
	refs := s.getReferences(26, 5, false)
	s.Assert().Empty(refs)
}

func (s *FindReferencesServiceSuite) Test_find_references_for_unreferenced_element_with_declaration() {
	refs := s.getReferences(26, 5, true)

	// Should only contain the declaration itself
	s.Assert().Len(refs, 1)
	s.Assert().Equal(uint32(26), refs[0].Range.Start.Line)
}

func (s *FindReferencesServiceSuite) getReferences(
	line uint32,
	character uint32,
	includeDeclaration bool,
) []lsp.Location {
	refs, err := s.service.GetReferencesFromContext(
		s.docCtx,
		&lsp.ReferencesParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: "file:///blueprint.yaml",
				},
				Position: lsp.Position{
					Line:      line,
					Character: character,
				},
			},
			Context: lsp.ReferenceContext{
				IncludeDeclaration: includeDeclaration,
			},
		},
	)
	s.Require().NoError(err)
	return refs
}

func startLines(locations []lsp.Location) []uint32 {
	lines := make([]uint32, len(locations))
	for i, loc := range locations {
		lines[i] = loc.Range.Start.Line
	}
	slices.Sort(lines)
	return lines
}

func containsAnyLine(lines []uint32, candidates ...uint32) bool {
	for _, line := range lines {
		for _, candidate := range candidates {
			if line == uint32(candidate) {
				return true
			}
		}
	}
	return false
}

func TestFindReferencesServiceSuite(t *testing.T) {
	suite.Run(t, new(FindReferencesServiceSuite))
}
