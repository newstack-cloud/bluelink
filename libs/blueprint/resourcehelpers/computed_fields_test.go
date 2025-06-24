package resourcehelpers

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ComputedFieldHelpersTestSuite struct {
	suite.Suite
}

func (s *ComputedFieldHelpersTestSuite) Test_check_matches_computed_field_for_array_item() {
	isComputed := IsComputedField(
		&provider.Changes{
			// [0] is a placeholder for any item in the `spec.configuration` array.
			ComputedFields: []string{"spec.configuration[0]"},
		},
		"spec.configuration[32]",
	)
	s.Assert().True(isComputed)
}

func (s *ComputedFieldHelpersTestSuite) Test_check_matches_computed_field_for_map_key_value_pair_1() {
	isComputed := IsComputedField(
		&provider.Changes{
			// ["<key>"] is a placeholder for any key-value pair in the `spec.configuration` map.
			ComputedFields: []string{"spec.configuration[\"<key>\"]"},
		},
		"spec.configuration.key1",
	)
	s.Assert().True(isComputed)
}

func (s *ComputedFieldHelpersTestSuite) Test_check_matches_computed_field_for_map_key_value_pair_2() {
	isComputed := IsComputedField(
		&provider.Changes{
			// ["<key>"] is a placeholder for any key-value pair in the `spec.configuration` map.
			ComputedFields: []string{"spec.configuration[\"<key>\"]"},
		},
		"spec.configuration[\"key5032\"]",
	)
	s.Assert().True(isComputed)
}

func (s *ComputedFieldHelpersTestSuite) Test_check_matches_computed_field_for_complex_path() {
	isComputed := IsComputedField(
		&provider.Changes{
			ComputedFields: []string{"spec.configuration[0].metadata[\"<key>\"].values[0][\"<key>\"]"},
		},
		"spec.configuration[35].metadata.key503.values[0][\"key8032\"]",
	)
	s.Assert().True(isComputed)
}

func (s *ComputedFieldHelpersTestSuite) Test_check_does_not_match_computed_field_for_complex_path() {
	isComputed := IsComputedField(
		&provider.Changes{
			ComputedFields: []string{"spec.configuration[0].metadata[\"<key>\"].values[0][\"<key>\"]"},
		},
		// the configuration property is expected to be an array, not a map.
		// the metadata property is expected to be a map, not an array.
		"spec.configuration[\"key4029\"].metadata[430].values[0][\"key8032\"]",
	)
	s.Assert().False(isComputed)
}

func TestComputedFieldHelpersTestSuite(t *testing.T) {
	suite.Run(t, new(ComputedFieldHelpersTestSuite))
}
