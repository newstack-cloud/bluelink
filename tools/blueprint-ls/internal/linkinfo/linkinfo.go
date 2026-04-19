package linkinfo

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Source exposes cardinality information and annotation definitions
// for a pair of linked resource types that supports both concrete provider
// and abstract transformer links.
type Source interface {
	// LookupLink finds link info for (typeA, typeB). Returns ok=false if no
	// link exists in this source in either direction.
	LookupLink(
		ctx context.Context,
		typeA string,
		typeB string,
	) (*LinkInfo, bool, error)
}

type LinkInfo struct {
	ResourceTypeA         string
	ResourceTypeB         string
	AnnotationDefinitions map[string]*provider.LinkAnnotationDefinition
	CardinalityA          provider.LinkCardinality
	CardinalityB          provider.LinkCardinality
	Origin                LinkInfoOrigin
}

// LinkInfoOrigin indicates where the link info was sourced from, which can be used
// for telemetry and debugging purposes.
type LinkInfoOrigin string

const (
	// LinkInfoOriginProvider indicates that the link info was sourced from a concrete provider plugin.
	LinkInfoOriginProvider LinkInfoOrigin = "provider"
	// LinkInfoOriginTransformer indicates that the link info was sourced from an abstract transformer plugin.
	LinkInfoOriginTransformer LinkInfoOrigin = "transformer"
)

// ProviderSource is a Source that retrieves link info from provider plugins.
type ProviderSource struct {
	linkRegistry  provider.LinkRegistry
	linkInfoCache *core.Cache[*LinkInfo]
}

// NewProviderSource creates a new ProviderSource with the given provider.LinkRegistry.
// This source will cache link info in memory to avoid redundant calls to the provider plugin for the same link.
func NewProviderSource(linkRegistry provider.LinkRegistry) *ProviderSource {
	return &ProviderSource{
		linkRegistry:  linkRegistry,
		linkInfoCache: core.NewCache[*LinkInfo](),
	}
}

func (s *ProviderSource) LookupLink(
	ctx context.Context,
	typeA string,
	typeB string,
) (*LinkInfo, bool, error) {
	if s.linkRegistry == nil {
		return nil, false, nil
	}

	linkType := core.LinkType(typeA, typeB)
	if linkInfo, ok := s.linkInfoCache.Get(linkType); ok {
		return linkInfo, true, nil
	}

	link, err := s.linkRegistry.Link(ctx, typeA, typeB)
	if err != nil {
		return nil, false, err
	}
	if link == nil {
		return nil, false, nil
	}

	emptyParams := core.NewDefaultParams(nil, nil, nil, nil)
	linkCtx := provider.NewLinkContextFromParams(emptyParams)

	annotationDefsOutput, err := link.GetAnnotationDefinitions(
		ctx,
		&provider.LinkGetAnnotationDefinitionsInput{
			LinkContext: linkCtx,
		},
	)
	if err != nil {
		return nil, false, err
	}

	cardinalityOutput, err := link.GetCardinality(
		ctx,
		&provider.LinkGetCardinalityInput{
			LinkContext: linkCtx,
		},
	)
	if err != nil {
		return nil, false, err
	}

	linkInfo := &LinkInfo{
		ResourceTypeA:         typeA,
		ResourceTypeB:         typeB,
		AnnotationDefinitions: annotationDefsOutput.AnnotationDefinitions,
		CardinalityA:          cardinalityOutput.CardinalityA,
		CardinalityB:          cardinalityOutput.CardinalityB,
		Origin:                LinkInfoOriginProvider,
	}
	s.linkInfoCache.Set(linkType, linkInfo)

	return linkInfo, true, nil
}

// TransformerSource is a Source that retrieves link info from transformer plugins.
type TransformerSource struct {
	transformers          map[string]transform.SpecTransformer
	linkTransformersCache *core.Cache[*transformerInfo]
	linkInfoCache         *core.Cache[*LinkInfo]
}

// NewTransformerSource creates a new TransformerSource with the given map of transformers.
// This source will cache link info in memory to avoid redundant calls to the transformer plugin for the same link.
func NewTransformerSource(transformers map[string]transform.SpecTransformer) *TransformerSource {
	return &TransformerSource{
		transformers:          transformers,
		linkTransformersCache: core.NewCache[*transformerInfo](),
		linkInfoCache:         core.NewCache[*LinkInfo](),
	}
}

type transformerInfo struct {
	namespace   string
	transformer transform.SpecTransformer
}

func (s *TransformerSource) LookupLink(
	ctx context.Context,
	typeA string,
	typeB string,
) (*LinkInfo, bool, error) {
	if len(s.transformers) == 0 {
		return nil, false, nil
	}

	linkType := core.LinkType(typeA, typeB)

	if linkInfo, ok := s.linkInfoCache.Get(linkType); ok {
		return linkInfo, true, nil
	}

	transformerInfo, err := s.getLinkTransformer(ctx, typeA, typeB)
	if err != nil {
		return nil, false, err
	}

	if transformerInfo == nil || transformerInfo.transformer == nil {
		return nil, false, nil
	}

	emptyParams := core.NewDefaultParams(nil, nil, nil, nil)
	transformerCtx := transform.NewTransformerContextFromParams(
		transformerInfo.namespace,
		emptyParams,
	)

	abstractLink, err := transformerInfo.transformer.AbstractLink(
		ctx,
		linkType,
	)
	if err != nil {
		return nil, false, err
	}

	annotationDefsOutput, err := abstractLink.GetAnnotationDefinitions(
		ctx,
		&transform.AbstractLinkGetAnnotationDefinitionsInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return nil, false, err
	}

	cardinalityOutput, err := abstractLink.GetCardinality(
		ctx,
		&transform.AbstractLinkGetCardinalityInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return nil, false, err
	}

	linkInfo := &LinkInfo{
		ResourceTypeA:         typeA,
		ResourceTypeB:         typeB,
		AnnotationDefinitions: annotationDefsOutput.AnnotationDefinitions,
		CardinalityA:          cardinalityOutput.CardinalityA,
		CardinalityB:          cardinalityOutput.CardinalityB,
		Origin:                LinkInfoOriginTransformer,
	}
	s.linkInfoCache.Set(linkType, linkInfo)

	return linkInfo, true, nil
}

func (s *TransformerSource) getLinkTransformer(
	ctx context.Context,
	typeA string,
	typeB string,
) (*transformerInfo, error) {
	linkType := core.LinkType(typeA, typeB)
	if transformer, ok := s.linkTransformersCache.Get(linkType); ok {
		return transformer, nil
	}

	for namespace, transformer := range s.transformers {
		err := s.loadTransformerLinkTypes(ctx, transformer, namespace)
		if err != nil {
			return nil, err
		}
	}

	if transformer, ok := s.linkTransformersCache.Get(linkType); ok {
		return transformer, nil
	}

	return nil, nil
}

func (s *TransformerSource) loadTransformerLinkTypes(
	ctx context.Context,
	transformer transform.SpecTransformer,
	namespace string,
) error {
	linkTypes, err := transformer.ListAbstractLinkTypes(ctx)
	if err != nil {
		return err
	}

	for _, lt := range linkTypes {
		s.linkTransformersCache.Set(lt, &transformerInfo{
			namespace:   namespace,
			transformer: transformer,
		})
	}

	return nil
}

// CompositeSource is a Source that aggregates multiple underlying Sources.
// It queries each Source in order and returns the first successful result.
type CompositeSource struct {
	sources []Source
}

// NewCompositeSource creates a new composite source
// for link information with the given underlying Sources.
func NewCompositeSource(sources ...Source) *CompositeSource {
	return &CompositeSource{sources: sources}
}

func (s *CompositeSource) LookupLink(
	ctx context.Context,
	typeA string,
	typeB string,
) (*LinkInfo, bool, error) {
	for _, source := range s.sources {
		linkInfo, ok, err := source.LookupLink(ctx, typeA, typeB)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return linkInfo, true, nil
		}
	}

	return nil, false, nil
}
