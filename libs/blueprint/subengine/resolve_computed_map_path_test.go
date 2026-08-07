package subengine

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/require"
)

// A computed field's contents are as unknown as the field itself until it is deployed,
// so a reference that goes deeper than a computed field has to defer to deploy just as a
// reference to the field itself does.
//
// The schema for a map's values, or an array's items, is shared by every entry and
// carries no Computed flag of its own. Reading Computed off the schema the path lands on
// therefore reports false for anything inside a computed map, and change staging fails
// with a missing property instead of deferring. Referencing the computed field directly
// worked, which is what made this look like a problem with maps rather than with depth.
func computedMapSpecDefinition() *provider.ResourceSpecDefinition {
	return &provider.ResourceSpecDefinition{
		IDField: "id",
		Schema: &provider.ResourceDefinitionsSchema{
			Type: provider.ResourceDefinitionsSchemaTypeObject,
			Attributes: map[string]*provider.ResourceDefinitionsSchema{
				"id": {Type: provider.ResourceDefinitionsSchemaTypeString},
				"name": {
					Type: provider.ResourceDefinitionsSchemaTypeString,
				},
				"idsByName": {
					Type:     provider.ResourceDefinitionsSchemaTypeMap,
					Computed: true,
					MapValues: &provider.ResourceDefinitionsSchema{
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
				"computedIds": {
					Type:     provider.ResourceDefinitionsSchemaTypeArray,
					Computed: true,
					Items: &provider.ResourceDefinitionsSchema{
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}
}

func specPropertyPath(fields ...string) *substitutions.SubstitutionResourceProperty {
	path := []*substitutions.SubstitutionPathItem{{FieldName: "spec"}}
	for _, field := range fields {
		path = append(path, &substitutions.SubstitutionPathItem{FieldName: field})
	}

	return &substitutions.SubstitutionResourceProperty{
		ResourceName: "exampleResource",
		Path:         path,
	}
}

func TestGetResourceSpecPropertyDefinitionReportsComputedForAMapKey(t *testing.T) {
	schema, computedOnPath, err := getResourceSpecPropertyDefinition(
		computedMapSpecDefinition(),
		specPropertyPath("idsByName", "anyKeyTheAuthorChose"),
		"exampleResourceType",
		&resolveContext{currentElementName: "resources.other"},
	)

	require.NoError(t, err)
	require.Equal(t, provider.ResourceDefinitionsSchemaTypeString, schema.Type)
	require.True(
		t,
		computedOnPath,
		"a key of a computed map is not known until the resource is deployed",
	)
}

// The field itself has always reported computed; this is the case that already worked and
// must keep working.
func TestGetResourceSpecPropertyDefinitionReportsComputedForTheMapItself(t *testing.T) {
	_, computedOnPath, err := getResourceSpecPropertyDefinition(
		computedMapSpecDefinition(),
		specPropertyPath("idsByName"),
		"exampleResourceType",
		&resolveContext{currentElementName: "resources.other"},
	)

	require.NoError(t, err)
	require.True(t, computedOnPath)
}

// Arrays share the same shape of schema for their items, so an index into a computed
// array loses the flag the same way a map key does.
func TestGetResourceSpecPropertyDefinitionReportsComputedForAnArrayIndex(t *testing.T) {
	index := int64(0)
	property := specPropertyPath("computedIds")
	property.Path = append(property.Path, &substitutions.SubstitutionPathItem{ArrayIndex: &index})

	_, computedOnPath, err := getResourceSpecPropertyDefinition(
		computedMapSpecDefinition(),
		property,
		"exampleResourceType",
		&resolveContext{currentElementName: "resources.other"},
	)

	require.NoError(t, err)
	require.True(
		t,
		computedOnPath,
		"an entry of a computed array is not known until the resource is deployed",
	)
}

// A field the author supplies is resolvable at change staging and must not be deferred,
// or every reference would wait for a deploy that has nothing to resolve.
func TestGetResourceSpecPropertyDefinitionDoesNotReportComputedForAuthoredFields(t *testing.T) {
	_, computedOnPath, err := getResourceSpecPropertyDefinition(
		computedMapSpecDefinition(),
		specPropertyPath("name"),
		"exampleResourceType",
		&resolveContext{currentElementName: "resources.other"},
	)

	require.NoError(t, err)
	require.False(t, computedOnPath)
}
