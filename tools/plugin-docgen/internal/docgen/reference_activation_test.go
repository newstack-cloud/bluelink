package docgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ruleResourceWithTargetArnSlot() *PluginDocsResource {
	return &PluginDocsResource{
		Type: "x/events/rule",
		Specification: &PluginDocResourceSpec{
			Schema: &PluginDocResourceSpecSchema{
				Type: "object",
				Attributes: map[string]*PluginDocResourceSpecSchema{
					"name": {Type: "string"},
					"targets": {
						Type: "array",
						Items: &PluginDocResourceSpecSchema{
							Type: "object",
							Attributes: map[string]*PluginDocResourceSpecSchema{
								"arn": {Type: "string", ActivatesLinkOnReference: true},
								"id":  {Type: "string"},
							},
						},
					},
				},
			},
		},
	}
}

func Test_collectActivatingSlots_collects_nested_array_slot_path(t *testing.T) {
	var slots []string
	collectActivatingSlots(ruleResourceWithTargetArnSlot().Specification.Schema, "", &slots)
	assert.Equal(t, []string{"targets[].arn"}, slots)
}

func Test_collectActivatingSlots_ignores_unflagged_fields(t *testing.T) {
	schema := &PluginDocResourceSpecSchema{
		Type: "object",
		Attributes: map[string]*PluginDocResourceSpecSchema{
			"endpoint": {Type: "string"},
		},
	}
	var slots []string
	collectActivatingSlots(schema, "", &slots)
	assert.Empty(t, slots)
}

func Test_correlateReferenceActivation_sets_activation_on_link_from_slotted_resource(t *testing.T) {
	resources := []*PluginDocsResource{ruleResourceWithTargetArnSlot()}
	links := []*PluginDocsLink{{Type: "x/events/rule::y/lambda/function"}}

	correlateReferenceActivation(resources, links)

	assert.NotNil(t, links[0].ReferenceActivation)
	assert.Equal(t, "x/events/rule", links[0].ReferenceActivation.ResourceType)
	assert.Equal(t, []string{"targets[].arn"}, links[0].ReferenceActivation.FieldPaths)
}

func Test_correlateReferenceActivation_leaves_links_without_a_slotted_source_unset(t *testing.T) {
	resources := []*PluginDocsResource{
		{
			Type:          "y/lambda/function",
			Specification: &PluginDocResourceSpec{Schema: &PluginDocResourceSpecSchema{Type: "object"}},
		},
	}
	// Source resource (y/lambda/function) has no wiring slot.
	links := []*PluginDocsLink{{Type: "y/lambda/function::z/dynamodb/table"}}

	correlateReferenceActivation(resources, links)

	assert.Nil(t, links[0].ReferenceActivation)
}

func Test_correlateReferenceActivation_skips_malformed_link_types(t *testing.T) {
	resources := []*PluginDocsResource{ruleResourceWithTargetArnSlot()}
	links := []*PluginDocsLink{{Type: "not-a-valid-link-type"}}

	assert.NotPanics(t, func() { correlateReferenceActivation(resources, links) })
	assert.Nil(t, links[0].ReferenceActivation)
}
