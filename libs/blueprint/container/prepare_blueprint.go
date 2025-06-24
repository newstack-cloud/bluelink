package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

// BlueprintPreparer is an interface for a service that prepares
// a blueprint for deployment or change staging.
type BlueprintPreparer interface {
	Prepare(
		ctx context.Context,
		blueprint *schema.Blueprint,
		resolveFor subengine.ResolveForStage,
		changes *changes.BlueprintChanges,
		linkInfo links.SpecLinkInfo,
		paramOverrides core.BlueprintParams,
	) (*BlueprintPrepareResult, error)
}

// BlueprintPrepareResult contains the result of preparing a blueprint
// for deployment or change staging.
type BlueprintPrepareResult struct {
	ResourceProviderMap map[string]provider.Provider
	BlueprintContainer  BlueprintContainer
	ParallelGroups      [][]*DeploymentNode
}

// NewDefaultBlueprintPreparer creates a new instance of the default
// implementation of the service that prepares blueprints
// for deployment or change staging.
func NewDefaultBlueprintPreparer(
	providers map[string]provider.Provider,
	substitutionResolver subengine.SubstitutionResolver,
	resourceTemplateInputElemCache *core.Cache[[]*core.MappingNode],
	resourceRegistry resourcehelpers.Registry,
	resourceCache *core.Cache[*provider.ResolvedResource],
	childBlueprintLoaderFactory ChildBlueprintLoaderFactory,
) BlueprintPreparer {
	return &defaultBlueprintPreparer{
		providers:                      providers,
		substitutionResolver:           substitutionResolver,
		resourceTemplateInputElemCache: resourceTemplateInputElemCache,
		resourceRegistry:               resourceRegistry,
		resourceCache:                  resourceCache,
		createChildBlueprintLoader:     childBlueprintLoaderFactory,
	}
}

type defaultBlueprintPreparer struct {
	providers                      map[string]provider.Provider
	substitutionResolver           subengine.SubstitutionResolver
	resourceTemplateInputElemCache *core.Cache[[]*core.MappingNode]
	resourceRegistry               resourcehelpers.Registry
	resourceCache                  *core.Cache[*provider.ResolvedResource]
	createChildBlueprintLoader     ChildBlueprintLoaderFactory
}

func (p *defaultBlueprintPreparer) Prepare(
	ctx context.Context,
	blueprint *schema.Blueprint,
	resolveFor subengine.ResolveForStage,
	changes *changes.BlueprintChanges,
	linkInfo links.SpecLinkInfo,
	paramOverrides core.BlueprintParams,
) (*BlueprintPrepareResult, error) {
	expandedBlueprintContainer, err := p.expandResourceTemplates(
		ctx,
		blueprint,
		resolveFor,
		changes,
		linkInfo,
		paramOverrides,
	)
	if err != nil {
		return nil, wrapErrorForChildContext(err, paramOverrides)
	}

	chains, err := expandedBlueprintContainer.SpecLinkInfo().Links(ctx)
	if err != nil {
		return nil, wrapErrorForChildContext(err, paramOverrides)
	}

	// We must use the ref chain collector from the expanded blueprint to correctly
	// order and resolve references for resources expanded from templates
	// in the blueprint.
	refChainCollector := expandedBlueprintContainer.RefChainCollector()

	childrenRefNodes := extractChildRefNodes(
		expandedBlueprintContainer.BlueprintSpec().Schema(),
		refChainCollector,
	)
	orderedNodes, err := OrderItemsForDeployment(
		ctx,
		chains,
		childrenRefNodes,
		refChainCollector,
		paramOverrides,
	)
	if err != nil {
		return nil, wrapErrorForChildContext(err, paramOverrides)
	}
	parallelGroups, err := GroupOrderedNodes(orderedNodes, refChainCollector)
	if err != nil {
		return nil, wrapErrorForChildContext(err, paramOverrides)
	}

	expandedResourceProviderMap := createResourceProviderMap(
		expandedBlueprintContainer.BlueprintSpec(),
		p.providers,
	)

	return &BlueprintPrepareResult{
		BlueprintContainer:  expandedBlueprintContainer,
		ParallelGroups:      parallelGroups,
		ResourceProviderMap: expandedResourceProviderMap,
	}, nil
}

func (p *defaultBlueprintPreparer) expandResourceTemplates(
	ctx context.Context,
	blueprint *schema.Blueprint,
	resolveFor subengine.ResolveForStage,
	changes *changes.BlueprintChanges,
	linkInfo links.SpecLinkInfo,
	params core.BlueprintParams,
) (BlueprintContainer, error) {

	chains, err := linkInfo.Links(ctx)
	if err != nil {
		return nil, err
	}

	expandResult, err := ExpandResourceTemplates(
		ctx,
		blueprint,
		p.substitutionResolver,
		chains,
		p.resourceTemplateInputElemCache,
	)
	if err != nil {
		return nil, err
	}

	populateDefaultsIn := blueprint
	if len(expandResult.ResourceTemplateMap) > 0 {
		populateDefaultsIn = expandResult.ExpandedBlueprint
	}

	// Populate defaults before applying conditions to ensure that the
	// resolved resources that are cached when applying conditions
	// are populated with the default values.
	applyConditionsTo, err := PopulateResourceSpecDefaults(
		ctx,
		populateDefaultsIn,
		params,
		p.resourceRegistry,
	)
	if err != nil {
		return nil, err
	}

	afterConditionsApplied, err := p.applyResourceConditions(
		ctx,
		applyConditionsTo,
		resolveFor,
		changes,
	)
	if err != nil {
		return nil, err
	}

	loader := p.createChildBlueprintLoader(
		flattenMapLists(expandResult.ResourceTemplateMap),
		invertMap(expandResult.ResourceTemplateMap),
	)
	return loader.LoadFromSchema(ctx, afterConditionsApplied, params)
}

func (p *defaultBlueprintPreparer) applyResourceConditions(
	ctx context.Context,
	blueprint *schema.Blueprint,
	resolveFor subengine.ResolveForStage,
	changes *changes.BlueprintChanges,
) (*schema.Blueprint, error) {

	if blueprint.Resources == nil {
		return blueprint, nil
	}

	resourcesAfterConditions := map[string]*schema.Resource{}
	for resourceName, resource := range blueprint.Resources.Values {
		if resource.Condition != nil {
			partiallyResolved := getPartiallyResolvedResourceFromChanges(changes, resourceName)
			resolveResourceResult, err := p.substitutionResolver.ResolveInResource(
				ctx,
				resourceName,
				resource,
				&subengine.ResolveResourceTargetInfo{
					ResolveFor:        resolveFor,
					PartiallyResolved: partiallyResolved,
				},
			)
			if err != nil {
				return nil, err
			}

			conditionKnownOnDeploy := isConditionKnownOnDeploy(
				resourceName,
				resolveResourceResult.ResolveOnDeploy,
			)
			conditionPasses := evaluateCondition(resolveResourceResult.ResolvedResource.Condition)
			// During change staging, if a condition can not be resolved, we include the resource
			// in the prepared blueprint with a nil condition so the resource is accounted for when
			// generating changes.
			// The `ConditionKnownOnDeploy` field of a change set for a resource informs the user
			// that whether or not the resource will be included in the final deployment
			// will be determined at deploy time.
			includeForChangeStaging := resolveFor == subengine.ResolveForChangeStaging &&
				(conditionKnownOnDeploy || conditionPasses)
			includeForDeployment := resolveFor == subengine.ResolveForDeployment && conditionPasses

			if includeForChangeStaging || includeForDeployment {
				p.resourceCache.Set(resourceName, resolveResourceResult.ResolvedResource)

				resourcesAfterConditions[resourceName] = resource
			}
		} else {
			resourcesAfterConditions[resourceName] = resource
		}
	}

	return &schema.Blueprint{
		Version:   blueprint.Version,
		Transform: blueprint.Transform,
		Variables: blueprint.Variables,
		Values:    blueprint.Values,
		Include:   blueprint.Include,
		Resources: &schema.ResourceMap{
			Values: resourcesAfterConditions,
		},
		DataSources: blueprint.DataSources,
		Exports:     blueprint.Exports,
		Metadata:    blueprint.Metadata,
	}, nil
}
