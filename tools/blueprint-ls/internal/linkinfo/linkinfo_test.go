package linkinfo

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type LinkInfoSuite struct {
	suite.Suite
}

// --- ProviderSource -----------------------------------------------------

func (s *LinkInfoSuite) Test_ProviderSource_LookupLink_returns_not_found_for_nil_registry() {
	src := NewProviderSource(nil)
	info, ok, err := src.LookupLink(context.Background(), "aws/a", "aws/b")
	s.Require().NoError(err)
	s.False(ok)
	s.Nil(info)
}

func (s *LinkInfoSuite) Test_ProviderSource_LookupLink_returns_info_when_registered() {
	mockLink := &stubLink{
		annotationDefs: map[string]*provider.LinkAnnotationDefinition{
			"key": {Name: "key", Type: core.ScalarTypeString},
		},
		cardinalityA: provider.LinkCardinality{Min: 0, Max: 5},
		cardinalityB: provider.LinkCardinality{Min: 1, Max: 1},
	}
	registry := &stubLinkRegistry{
		links: map[string]provider.Link{
			"aws/a::aws/b": mockLink,
		},
	}

	src := NewProviderSource(registry)
	info, ok, err := src.LookupLink(context.Background(), "aws/a", "aws/b")
	s.Require().NoError(err)
	s.True(ok)
	s.Require().NotNil(info)
	s.Equal(LinkInfoOriginProvider, info.Origin)
	s.Equal("aws/a", info.ResourceTypeA)
	s.Equal("aws/b", info.ResourceTypeB)
	s.Equal(provider.LinkCardinality{Min: 0, Max: 5}, info.CardinalityA)
	s.Equal(provider.LinkCardinality{Min: 1, Max: 1}, info.CardinalityB)
	s.Contains(info.AnnotationDefinitions, "key")
}

func (s *LinkInfoSuite) Test_ProviderSource_LookupLink_caches_results() {
	mockLink := &stubLink{
		cardinalityA: provider.LinkCardinality{Min: 0, Max: 2},
	}
	registry := &stubLinkRegistry{
		links: map[string]provider.Link{
			"aws/a::aws/b": mockLink,
		},
	}
	src := NewProviderSource(registry)

	_, _, err := src.LookupLink(context.Background(), "aws/a", "aws/b")
	s.Require().NoError(err)
	_, _, err = src.LookupLink(context.Background(), "aws/a", "aws/b")
	s.Require().NoError(err)

	s.Equal(1, registry.linkCalls, "Link lookup should be cached")
	s.Equal(1, mockLink.cardinalityCalls, "GetCardinality should be cached")
	s.Equal(1, mockLink.annotationCalls, "GetAnnotationDefinitions should be cached")
}

func (s *LinkInfoSuite) Test_ProviderSource_LookupLink_returns_error_from_registry() {
	registry := &stubLinkRegistry{err: errors.New("registry unavailable")}
	src := NewProviderSource(registry)

	info, ok, err := src.LookupLink(context.Background(), "aws/a", "aws/b")
	s.Require().Error(err)
	s.False(ok)
	s.Nil(info)
}

// --- TransformerSource --------------------------------------------------

func (s *LinkInfoSuite) Test_TransformerSource_LookupLink_returns_not_found_for_empty_transformers() {
	src := NewTransformerSource(nil)
	info, ok, err := src.LookupLink(context.Background(), "celerity/handler", "celerity/api")
	s.Require().NoError(err)
	s.False(ok)
	s.Nil(info)
}

func (s *LinkInfoSuite) Test_TransformerSource_LookupLink_returns_info_from_matching_transformer() {
	abstractLink := &stubAbstractLink{
		cardinalityA: provider.LinkCardinality{Min: 1, Max: 1},
		cardinalityB: provider.LinkCardinality{Min: 0, Max: 0},
		annotationDefs: map[string]*provider.LinkAnnotationDefinition{
			"foo": {Name: "foo", Type: core.ScalarTypeString},
		},
	}
	transformer := &stubTransformer{
		linkTypes:     []string{"celerity/handler::celerity/api"},
		abstractLinks: map[string]transform.AbstractLink{"celerity/handler::celerity/api": abstractLink},
	}

	src := NewTransformerSource(map[string]transform.SpecTransformer{
		"celerity": transformer,
	})

	info, ok, err := src.LookupLink(context.Background(), "celerity/handler", "celerity/api")
	s.Require().NoError(err)
	s.True(ok)
	s.Require().NotNil(info)
	s.Equal(LinkInfoOriginTransformer, info.Origin)
	s.Equal(provider.LinkCardinality{Min: 1, Max: 1}, info.CardinalityA)
	s.Contains(info.AnnotationDefinitions, "foo")
}

func (s *LinkInfoSuite) Test_TransformerSource_LookupLink_returns_not_found_for_unregistered_link_type() {
	transformer := &stubTransformer{
		linkTypes:     []string{"celerity/handler::celerity/api"},
		abstractLinks: map[string]transform.AbstractLink{},
	}
	src := NewTransformerSource(map[string]transform.SpecTransformer{
		"celerity": transformer,
	})

	info, ok, err := src.LookupLink(context.Background(), "aws/a", "aws/b")
	s.Require().NoError(err)
	s.False(ok)
	s.Nil(info)
}

// --- CompositeSource ----------------------------------------------------

func (s *LinkInfoSuite) Test_CompositeSource_LookupLink_returns_first_match() {
	primary := &stubSource{info: &LinkInfo{ResourceTypeA: "p", Origin: LinkInfoOriginProvider}, ok: true}
	secondary := &stubSource{info: &LinkInfo{ResourceTypeA: "s", Origin: LinkInfoOriginTransformer}, ok: true}

	composite := NewCompositeSource(primary, secondary)
	info, ok, err := composite.LookupLink(context.Background(), "a", "b")
	s.Require().NoError(err)
	s.True(ok)
	s.Equal(LinkInfoOriginProvider, info.Origin)
	s.Equal(1, primary.calls)
	s.Equal(0, secondary.calls, "secondary source should not be queried after primary match")
}

func (s *LinkInfoSuite) Test_CompositeSource_LookupLink_falls_through_on_miss() {
	primary := &stubSource{ok: false}
	secondary := &stubSource{info: &LinkInfo{Origin: LinkInfoOriginTransformer}, ok: true}

	composite := NewCompositeSource(primary, secondary)
	info, ok, err := composite.LookupLink(context.Background(), "a", "b")
	s.Require().NoError(err)
	s.True(ok)
	s.Equal(LinkInfoOriginTransformer, info.Origin)
}

func (s *LinkInfoSuite) Test_CompositeSource_LookupLink_propagates_error() {
	primary := &stubSource{err: errors.New("boom")}
	secondary := &stubSource{info: &LinkInfo{Origin: LinkInfoOriginTransformer}, ok: true}

	composite := NewCompositeSource(primary, secondary)
	_, _, err := composite.LookupLink(context.Background(), "a", "b")
	s.Require().Error(err)
	s.Equal(0, secondary.calls, "secondary source should not run after primary error")
}

func TestLinkInfoSuite(t *testing.T) {
	suite.Run(t, new(LinkInfoSuite))
}

// --- test doubles -------------------------------------------------------

type stubLinkRegistry struct {
	links     map[string]provider.Link
	err       error
	linkCalls int
}

func (r *stubLinkRegistry) Link(
	ctx context.Context,
	typeA string,
	typeB string,
) (provider.Link, error) {
	r.linkCalls++
	if r.err != nil {
		return nil, r.err
	}
	key := typeA + "::" + typeB
	if link, ok := r.links[key]; ok {
		return link, nil
	}
	return nil, nil
}

func (r *stubLinkRegistry) Provider(
	typeA string,
	typeB string,
) (provider.Provider, error) {
	return nil, nil
}

type stubLink struct {
	annotationDefs   map[string]*provider.LinkAnnotationDefinition
	cardinalityA     provider.LinkCardinality
	cardinalityB     provider.LinkCardinality
	annotationCalls  int
	cardinalityCalls int
}

func (l *stubLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *stubLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *stubLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *stubLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *stubLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{}, nil
}

func (l *stubLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{}, nil
}

func (l *stubLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *stubLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	l.annotationCalls++
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: l.annotationDefs,
	}, nil
}

func (l *stubLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{}, nil
}

func (l *stubLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *stubLink) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	l.cardinalityCalls++
	return &provider.LinkGetCardinalityOutput{
		CardinalityA: l.cardinalityA,
		CardinalityB: l.cardinalityB,
	}, nil
}

func (l *stubLink) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}

type stubTransformer struct {
	linkTypes     []string
	abstractLinks map[string]transform.AbstractLink
}

func (t *stubTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "stub", nil
}

func (t *stubTransformer) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{}, nil
}

func (t *stubTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: &schema.Blueprint{},
	}, nil
}

func (t *stubTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{}, nil
}

func (t *stubTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	return nil, nil
}

func (t *stubTransformer) ListAbstractResourceTypes(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (t *stubTransformer) ListAbstractLinkTypes(ctx context.Context) ([]string, error) {
	return t.linkTypes, nil
}

func (t *stubTransformer) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	if link, ok := t.abstractLinks[linkType]; ok {
		return link, nil
	}
	return nil, errors.New("abstract link not found")
}

type stubAbstractLink struct {
	annotationDefs map[string]*provider.LinkAnnotationDefinition
	cardinalityA   provider.LinkCardinality
	cardinalityB   provider.LinkCardinality
}

func (l *stubAbstractLink) GetType(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeInput,
) (*transform.AbstractLinkGetTypeOutput, error) {
	return &transform.AbstractLinkGetTypeOutput{}, nil
}

func (l *stubAbstractLink) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeDescriptionInput,
) (*transform.AbstractLinkGetTypeDescriptionOutput, error) {
	return &transform.AbstractLinkGetTypeDescriptionOutput{}, nil
}

func (l *stubAbstractLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *transform.AbstractLinkGetAnnotationDefinitionsInput,
) (*transform.AbstractLinkGetAnnotationDefinitionsOutput, error) {
	return &transform.AbstractLinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: l.annotationDefs,
	}, nil
}

func (l *stubAbstractLink) GetCardinality(
	ctx context.Context,
	input *transform.AbstractLinkGetCardinalityInput,
) (*transform.AbstractLinkGetCardinalityOutput, error) {
	return &transform.AbstractLinkGetCardinalityOutput{
		CardinalityA: l.cardinalityA,
		CardinalityB: l.cardinalityB,
	}, nil
}

type stubSource struct {
	info  *LinkInfo
	ok    bool
	err   error
	calls int
}

func (s *stubSource) LookupLink(
	ctx context.Context,
	typeA string,
	typeB string,
) (*LinkInfo, bool, error) {
	s.calls++
	return s.info, s.ok, s.err
}
