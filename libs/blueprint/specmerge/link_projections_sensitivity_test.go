package specmerge

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// A link assembles its contributions inside itself, so neither the target resource's schema
// nor the blueprint says anything about them. The mapping that says where a projected value
// came from is the only thing that can say where its sensitivity came from.
type LinkProjectionSensitivityTestSuite struct {
	suite.Suite
}

func (s *LinkProjectionSensitivityTestSuite) Test_marks_a_field_fed_from_sensitive_link_data() {
	result, err := ApplyLinkProjections(
		&core.MappingNode{Fields: map[string]*core.MappingNode{}},
		"ordersFunction",
		[]state.LinkState{sensitiveLinkState()},
	)
	s.Require().NoError(err)
	s.Require().Empty(result.Unresolved)

	s.Assert().Contains(result.SensitivePaths, "spec.environment.DB_PASSWORD")
	s.Assert().NotContains(
		result.SensitivePaths,
		"spec.environment.TABLE_NAME",
		"a field fed from a part of the link's data it did not mark was marked",
	)
}

// The path has to be in the form ResolvedResource.SensitiveSpecPaths uses, since that is
// what it is merged into. The root form ResourceSpecPath produces, for reading and writing
// a mapping node, would never match.
func (s *LinkProjectionSensitivityTestSuite) Test_reports_paths_in_the_spec_path_form() {
	for _, fieldPath := range []string{"spec.environment.DB_PASSWORD", "environment.DB_PASSWORD"} {
		s.Assert().Equal(
			"spec.environment.DB_PASSWORD",
			SensitiveSpecPath(fieldPath),
			"%s was not rendered in the spec path form",
			fieldPath,
		)
	}
}

func (s *LinkProjectionSensitivityTestSuite) Test_marks_nothing_when_the_link_marked_nothing() {
	link := sensitiveLinkState()
	link.SensitiveDataPaths = nil

	result, err := ApplyLinkProjections(
		&core.MappingNode{Fields: map[string]*core.MappingNode{}},
		"ordersFunction",
		[]state.LinkState{link},
	)
	s.Require().NoError(err)

	s.Assert().Empty(result.SensitivePaths)
}

func sensitiveLinkState() state.LinkState {
	return state.LinkState{
		Name: "ordersFunction::ordersTable",
		Data: map[string]*core.MappingNode{
			"contributions": {
				Items: []*core.MappingNode{
					core.MappingNodeFromString("orders-table"),
					core.MappingNodeFromString("super-secret-password"),
				},
			},
		},
		ResourceDataMappings: map[string]string{
			"ordersFunction::spec.environment.TABLE_NAME":  "contributions[0]",
			"ordersFunction::spec.environment.DB_PASSWORD": "contributions[1]",
		},
		SensitiveDataPaths: []string{"contributions[1]"},
	}
}

func TestLinkProjectionSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(LinkProjectionSensitivityTestSuite))
}
