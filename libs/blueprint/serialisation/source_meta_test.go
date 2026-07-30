package serialisation

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

// SourceMetaSerialiserTestSuite covers preservation of source positions across
// the protobuf round trip.
//
// Positions are what let a plugin report a diagnostic against the location a
// user wrote, rather than the start of the document. Dropping them leaves
// consumers with nothing to anchor to, so the round trip is asserted directly.
type SourceMetaSerialiserTestSuite struct {
	blueprint *schema.Blueprint
}

var _ = Suite(&SourceMetaSerialiserTestSuite{})

func (s *SourceMetaSerialiserTestSuite) SetUpSuite(c *C) {
	specBytes, err := os.ReadFile("__testdata/blueprint-full.yml")
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	s.blueprint = &schema.Blueprint{}
	if err := yaml.Unmarshal(specBytes, s.blueprint); err != nil {
		c.Error(err)
		c.FailNow()
	}
}

func (s *SourceMetaSerialiserTestSuite) roundTrip(c *C) *schema.Blueprint {
	blueprintPB, err := ToSchemaPB(s.blueprint)
	c.Assert(err, IsNil)

	roundTripped, err := FromSchemaPB(blueprintPB)
	c.Assert(err, IsNil)

	return roundTripped
}

func (s *SourceMetaSerialiserTestSuite) Test_preserves_resource_source_meta(c *C) {
	roundTripped := s.roundTrip(c)

	c.Assert(s.blueprint.Resources, NotNil)
	for name, original := range s.blueprint.Resources.Values {
		if original.SourceMeta == nil {
			continue
		}

		result := roundTripped.Resources.Values[name]
		c.Assert(result, NotNil)
		c.Assert(result.SourceMeta, NotNil)
		c.Assert(result.SourceMeta.Line, Equals, original.SourceMeta.Line)
		c.Assert(result.SourceMeta.Column, Equals, original.SourceMeta.Column)
	}
}

func (s *SourceMetaSerialiserTestSuite) Test_preserves_resource_spec_field_source_meta(c *C) {
	roundTripped := s.roundTrip(c)

	checked := 0
	for name, original := range s.blueprint.Resources.Values {
		if original.Spec == nil {
			continue
		}

		result := roundTripped.Resources.Values[name]
		c.Assert(result, NotNil)
		checked += assertMappingNodeMetaPreserved(c, original.Spec, result.Spec)
	}

	c.Assert(checked > 0, Equals, true)
}

func (s *SourceMetaSerialiserTestSuite) Test_preserves_scalar_source_meta(c *C) {
	roundTripped := s.roundTrip(c)

	c.Assert(s.blueprint.Version, NotNil)
	c.Assert(s.blueprint.Version.SourceMeta, NotNil)
	c.Assert(roundTripped.Version.SourceMeta, NotNil)
	c.Assert(roundTripped.Version.SourceMeta.Line, Equals, s.blueprint.Version.SourceMeta.Line)
	c.Assert(roundTripped.Version.SourceMeta.Column, Equals, s.blueprint.Version.SourceMeta.Column)
}

// A blueprint built in memory carries no positions, which must round trip as
// absent rather than as a zero position that would point at the wrong place.
func (s *SourceMetaSerialiserTestSuite) Test_absent_source_meta_stays_absent(c *C) {
	constructed := &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: strPtr("2025-11-02")},
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"generated": {
					Type: &schema.ResourceTypeWrapper{Value: "test/resource"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"name": {Scalar: &core.ScalarValue{StringValue: strPtr("value")}},
						},
					},
				},
			},
		},
	}

	blueprintPB, err := ToSchemaPB(constructed)
	c.Assert(err, IsNil)
	roundTripped, err := FromSchemaPB(blueprintPB)
	c.Assert(err, IsNil)

	resource := roundTripped.Resources.Values["generated"]
	c.Assert(resource, NotNil)
	c.Assert(resource.SourceMeta, IsNil)
	c.Assert(resource.FieldsSourceMeta, IsNil)
	c.Assert(resource.Spec.SourceMeta, IsNil)
	c.Assert(resource.Spec.Fields["name"].Scalar.SourceMeta, IsNil)
}

func (s *SourceMetaSerialiserTestSuite) Test_preserves_end_position_and_column_accuracy(c *C) {
	accuracy := source.ColumnAccuracyApproximate
	meta := &source.Meta{
		Position:       source.Position{Line: 12, Column: 3},
		EndPosition:    &source.Position{Line: 14, Column: 9},
		ColumnAccuracy: &accuracy,
	}

	roundTripped := FromSourceMetaPB(ToSourceMetaPB(meta))

	c.Assert(roundTripped, NotNil)
	c.Assert(roundTripped.Line, Equals, 12)
	c.Assert(roundTripped.Column, Equals, 3)
	c.Assert(roundTripped.EndPosition, NotNil)
	c.Assert(roundTripped.EndPosition.Line, Equals, 14)
	c.Assert(roundTripped.EndPosition.Column, Equals, 9)
	c.Assert(roundTripped.ColumnAccuracy, NotNil)
	c.Assert(*roundTripped.ColumnAccuracy, Equals, source.ColumnAccuracyApproximate)
}

// Walks a mapping node pair, asserting positions
// match. It returns the number of positioned nodes compared so callers can
// confirm the walk yielded at least one comparison.
func assertMappingNodeMetaPreserved(c *C, original, result *core.MappingNode) int {
	if original == nil || result == nil {
		return 0
	}

	compared := 0
	if original.SourceMeta != nil {
		c.Assert(result.SourceMeta, NotNil)
		c.Assert(result.SourceMeta.Line, Equals, original.SourceMeta.Line)
		c.Assert(result.SourceMeta.Column, Equals, original.SourceMeta.Column)
		compared++
	}

	for key, originalField := range original.Fields {
		compared += assertMappingNodeMetaPreserved(c, originalField, result.Fields[key])
	}

	for index, originalItem := range original.Items {
		if index < len(result.Items) {
			compared += assertMappingNodeMetaPreserved(c, originalItem, result.Items[index])
		}
	}

	return compared
}

func strPtr(value string) *string {
	return &value
}
