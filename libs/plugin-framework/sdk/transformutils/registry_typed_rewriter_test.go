package transformutils

import (
	"reflect"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type TypedRewriterWithCapabilitiesTestSuite struct {
	suite.Suite
}

type capabilitiesTestResolved struct {
	name     string
	external bool
}

func (r *capabilitiesTestResolved) ResourceName() string {
	return r.name
}

func (r *capabilitiesTestResolved) ResourceType() string {
	return "test/resource"
}

func capabilitiesPropertyMap() *PropertyMap {
	return &PropertyMap{
		Renames: map[string][]string{
			"spec.id": {"spec", "arn"},
		},
	}
}

func (s *TypedRewriterWithCapabilitiesTestSuite) registry(
	registration RewriterRegistration,
) *TransformerRegistry {
	registry := NewTransformerRegistry()
	registration(registry, "aws")

	return registry
}

func (s *TypedRewriterWithCapabilitiesTestSuite) rewrite(
	registry *TransformerRegistry,
	resolved *capabilitiesTestResolved,
) *substitutions.Substitution {
	factory, found := registry.RewriteFactoryFor("aws", reflect.TypeFor[*capabilitiesTestResolved]())
	s.Require().True(found)

	rewriters := factory(resolved)
	s.Require().Len(rewriters, 1)

	return rewriters[0](&substitutions.SubstitutionResourceProperty{
		ResourceName: resolved.name,
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "spec"},
			{FieldName: "id"},
		},
	})
}

// One resolved primary can rewrite to a literal where another rewrites to a
// concrete resource property, which a PropertyMap alone cannot express.
func (s *TypedRewriterWithCapabilitiesTestSuite) Test_rewriter_is_chosen_per_resolved_primary() {
	propertyMap := capabilitiesPropertyMap()
	literal := "arn:aws:sns:eu-west-2:123456789012:external-topic"

	registry := s.registry(TypedRewriterWithCapabilities(
		propertyMap,
		func(r *capabilitiesTestResolved) []ResourcePropertyRewriter {
			if r.external {
				return []ResourcePropertyRewriter{
					func(*substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
						return &substitutions.Substitution{StringValue: &literal}
					},
				}
			}

			return []ResourcePropertyRewriter{
				propertyMap.Rewriter(r.ResourceName(), r.ResourceName()+"_sns_topic"),
			}
		},
	))

	externalResult := s.rewrite(registry, &capabilitiesTestResolved{name: "audit", external: true})
	s.Require().NotNil(externalResult.StringValue)
	s.Equal(literal, *externalResult.StringValue)

	managedResult := s.rewrite(registry, &capabilitiesTestResolved{name: "orders"})
	s.Require().NotNil(managedResult.ResourceProperty)
	s.Equal("orders_sns_topic", managedResult.ResourceProperty.ResourceName)
}

// Capabilities come from the PropertyMap rather than the rewriter, which is the
// whole reason this exists: a factory registered through TypedRewriter silently
// loses pre-emit reference validation for its type.
func (s *TypedRewriterWithCapabilitiesTestSuite) Test_capabilities_are_still_registered() {
	registry := s.registry(TypedRewriterWithCapabilities(
		capabilitiesPropertyMap(),
		func(*capabilitiesTestResolved) []ResourcePropertyRewriter {
			return []ResourcePropertyRewriter{
				func(*substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
					return nil
				},
			}
		},
	))

	capabilities, found := registry.CapabilitiesFor(
		"aws",
		reflect.TypeFor[*capabilitiesTestResolved](),
	)
	s.Require().True(found, "capabilities must be registered for the type")
	s.NotEmpty(capabilities)
}

// RewriterFromPropertyMap now delegates here, so it has to keep behaving
// exactly as it did.
func (s *TypedRewriterWithCapabilitiesTestSuite) Test_concrete_name_form_is_unchanged() {
	propertyMap := capabilitiesPropertyMap()
	registry := s.registry(RewriterFromPropertyMap(
		propertyMap,
		func(r *capabilitiesTestResolved) string {
			return r.ResourceName() + "_sns_topic"
		},
	))

	result := s.rewrite(registry, &capabilitiesTestResolved{name: "orders"})
	s.Require().NotNil(result.ResourceProperty)
	s.Equal("orders_sns_topic", result.ResourceProperty.ResourceName)

	_, found := registry.CapabilitiesFor("aws", reflect.TypeFor[*capabilitiesTestResolved]())
	s.True(found)
}

func TestTypedRewriterWithCapabilitiesTestSuite(t *testing.T) {
	suite.Run(t, new(TypedRewriterWithCapabilitiesTestSuite))
}
