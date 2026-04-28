package plugintestsuites

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	"gopkg.in/yaml.v3"
)

func createTransformInput() (*transform.SpecTransformerTransformInput, error) {
	blueprint, err := loadTestBlueprint()
	if err != nil {
		return nil, err
	}

	return &transform.SpecTransformerTransformInput{
		InputBlueprint:     blueprint,
		TransformerContext: testutils.CreateTestTransformerContext("celerity"),
	}, nil
}

func loadTestBlueprint() (*schema.Blueprint, error) {
	blueprintBytes, err := os.ReadFile("__testdata/transform/blueprint.yml")
	if err != nil {
		return nil, err
	}

	blueprint := &schema.Blueprint{}
	err = yaml.Unmarshal(blueprintBytes, blueprint)
	if err != nil {
		return nil, err
	}

	return blueprint, nil
}

// testResourceEntry holds a resource schema and its classification
// for use in the testDeclaredLinkGraph.
type testResourceEntry struct {
	schema         *schema.Resource
	classification linktypes.ResourceClass
}

// testDeclaredLinkGraph is a test implementation of linktypes.DeclaredLinkGraph.
type testDeclaredLinkGraph struct {
	edges     []*linktypes.ResolvedLink
	outgoing  map[string][]*linktypes.ResolvedLink
	incoming  map[string][]*linktypes.ResolvedLink
	resources map[string]testResourceEntry
}

func newTestDeclaredLinkGraph(
	edges []*linktypes.ResolvedLink,
	resources map[string]testResourceEntry,
) *testDeclaredLinkGraph {
	outgoing := make(map[string][]*linktypes.ResolvedLink)
	incoming := make(map[string][]*linktypes.ResolvedLink)
	for _, edge := range edges {
		outgoing[edge.Source] = append(outgoing[edge.Source], edge)
		incoming[edge.Target] = append(incoming[edge.Target], edge)
	}
	return &testDeclaredLinkGraph{
		edges:     edges,
		outgoing:  outgoing,
		incoming:  incoming,
		resources: resources,
	}
}

func (g *testDeclaredLinkGraph) Edges() []*linktypes.ResolvedLink {
	return g.edges
}

func (g *testDeclaredLinkGraph) EdgesFrom(resourceName string) []*linktypes.ResolvedLink {
	if edges, ok := g.outgoing[resourceName]; ok {
		return edges
	}
	return []*linktypes.ResolvedLink{}
}

func (g *testDeclaredLinkGraph) EdgesTo(resourceName string) []*linktypes.ResolvedLink {
	if edges, ok := g.incoming[resourceName]; ok {
		return edges
	}
	return []*linktypes.ResolvedLink{}
}

func (g *testDeclaredLinkGraph) Resource(name string) (*schema.Resource, linktypes.ResourceClass, bool) {
	if entry, ok := g.resources[name]; ok {
		return entry.schema, entry.classification, true
	}
	return nil, "", false
}

func createAbstractHandlerResource(annotations map[string]string) *schema.Resource {
	annotationValues := map[string]*substitutions.StringOrSubstitutions{}
	for key, val := range annotations {
		v := val
		annotationValues[key] = &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &v},
			},
		}
	}

	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "celerity/handler"},
		Metadata: &schema.Metadata{
			Annotations: &schema.StringOrSubstitutionsMap{
				Values: annotationValues,
			},
		},
		Spec: core.MappingNodeFromString("stub"),
	}
}

func createAbstractAPIResource() *schema.Resource {
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "celerity/api"},
		Metadata: &schema.Metadata{
			Annotations: &schema.StringOrSubstitutionsMap{
				Values: map[string]*substitutions.StringOrSubstitutions{},
			},
		},
		Spec: core.MappingNodeFromString("stub"),
	}
}

func createValidateLinksInput(
	edges []*linktypes.ResolvedLink,
	resources map[string]testResourceEntry,
) (*transform.SpecTransformerValidateLinksInput, error) {
	// The plugin-side link graph reconstruction joins graph resource entries
	// against blueprint.Resources.Values to recover schemas, so the blueprint
	// must carry the same resources the graph references.
	resourceValues := make(map[string]*schema.Resource, len(resources))
	for name, entry := range resources {
		resourceValues[name] = entry.schema
	}

	blueprint := &schema.Blueprint{
		Version:   core.ScalarFromString("2025-02-01"),
		Resources: &schema.ResourceMap{Values: resourceValues},
	}

	return &transform.SpecTransformerValidateLinksInput{
		Blueprint:          blueprint,
		LinkGraph:          newTestDeclaredLinkGraph(edges, resources),
		TransformerContext: testutils.CreateTestTransformerContext("celerity"),
	}, nil
}
