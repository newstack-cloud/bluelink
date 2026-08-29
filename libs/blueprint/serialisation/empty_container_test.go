package serialisation

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	. "gopkg.in/check.v1"
)

type EmptyContainerTestSuite struct{}

var _ = Suite(&EmptyContainerTestSuite{})

func blueprintWithSpec(spec *core.MappingNode) *schema.Blueprint {
	version := "2025-11-02"
	return &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: &version},
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"fanoutQueue": {
					Type: &schema.ResourceTypeWrapper{Value: "aws/sqs/queue"},
					Spec: spec,
				},
			},
		},
	}
}

func roundTripSpec(c *C, spec *core.MappingNode) *core.MappingNode {
	serialiser := NewProtobufSerialiser()

	data, err := serialiser.Marshal(blueprintWithSpec(spec))
	c.Assert(err, IsNil)

	back, err := serialiser.Unmarshal(data)
	c.Assert(err, IsNil)

	return back.Resources.Values["fanoutQueue"].Spec
}

// A resource spec holding an empty mapping is a value, and one holding nothing
// is an unresolved node that is rejected. proto3 puts neither an empty map nor
// an absent one on the wire, so without a marker the first arrives as the
// second and the blueprint cannot be read back.
func (s *EmptyContainerTestSuite) Test_an_empty_mapping_survives_the_wire(c *C) {
	spec := roundTripSpec(c, &core.MappingNode{Fields: map[string]*core.MappingNode{}})

	c.Assert(spec, NotNil)
	c.Assert(spec.Fields, NotNil)
	c.Assert(len(spec.Fields), Equals, 0)
}

func (s *EmptyContainerTestSuite) Test_an_empty_list_survives_the_wire(c *C) {
	spec := roundTripSpec(c, &core.MappingNode{Items: []*core.MappingNode{}})

	c.Assert(spec, NotNil)
	c.Assert(spec.Items, NotNil)
	c.Assert(len(spec.Items), Equals, 0)
}

// The marker must not make a populated container look empty.
func (s *EmptyContainerTestSuite) Test_a_populated_mapping_is_unaffected(c *C) {
	spec := roundTripSpec(c, &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"queueName": core.MappingNodeFromString("orders"),
		},
	})

	c.Assert(spec.Fields, NotNil)
	c.Assert(len(spec.Fields), Equals, 1)
	c.Assert(core.StringValue(spec.Fields["queueName"]), Equals, "orders")
}

// Nested containers are converted as optional, where an absent node is dropped
// to nil. An empty one must not be dropped with it.
func (s *EmptyContainerTestSuite) Test_a_nested_empty_mapping_survives_the_wire(c *C) {
	spec := roundTripSpec(c, &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"tags": {Fields: map[string]*core.MappingNode{}},
		},
	})

	nested := spec.Fields["tags"]
	c.Assert(nested, NotNil)
	c.Assert(nested.Fields, NotNil)
	c.Assert(len(nested.Fields), Equals, 0)
}
