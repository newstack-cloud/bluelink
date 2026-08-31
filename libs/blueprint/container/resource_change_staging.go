package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

// ResourceChangeStager provides an interface for a service that stages changes for
// a resource in a blueprint.
type ResourceChangeStager interface {
	StageChanges(
		ctx context.Context,
		instanceID string,
		stagingState ChangeStagingState,
		node *links.ChainLinkNode,
		channels *ChangeStagingChannels,
		resourceProviders map[string]provider.Provider,
		params core.BlueprintParams,
		logger core.Logger,
	)
}

// NewDefaultResourceChangeStager creates a new instance of the
// default resource change stager.
func NewDefaultResourceChangeStager(
	substitutionResolver subengine.SubstitutionResolver,
	resourceCache *core.Cache[*provider.ResolvedResource],
	stateContainer state.Container,
	changeGenerator changes.ResourceChangeGenerator,
	linkChangeStager LinkChangeStager,
) ResourceChangeStager {
	return &defaultResourceChangeStager{
		substitutionResolver: substitutionResolver,
		resourceCache:        resourceCache,
		stateContainer:       stateContainer,
		changeGenerator:      changeGenerator,
		linkChangeStager:     linkChangeStager,
	}
}

type defaultResourceChangeStager struct {
	substitutionResolver subengine.SubstitutionResolver
	resourceCache        *core.Cache[*provider.ResolvedResource]
	stateContainer       state.Container
	changeGenerator      changes.ResourceChangeGenerator
	linkChangeStager     LinkChangeStager
}

func (s *defaultResourceChangeStager) StageChanges(
	ctx context.Context,
	instanceID string,
	stagingState ChangeStagingState,
	node *links.ChainLinkNode,
	channels *ChangeStagingChannels,
	resourceProviders map[string]provider.Provider,
	params core.BlueprintParams,
	logger core.Logger,
) {
	resourceTypeLogField := core.StringLogField("resourceType", node.Resource.Type.Value)
	logger.Debug(
		"loading resource plugin implementation",
		resourceTypeLogField,
	)
	resourceImplementation, err := getProviderResourceImplementation(
		ctx,
		node.ResourceName,
		node.Resource.Type.Value,
		resourceProviders,
	)
	if err != nil {
		logger.Debug(
			"failed to load resource plugin implementation",
			core.ErrorLogField("error", err),
			resourceTypeLogField,
		)
		channels.ErrChan <- err
		return
	}

	err = s.stageChanges(
		ctx,
		&stageResourceChangeInfo{
			node:       node,
			instanceID: instanceID,
		},
		resourceImplementation,
		channels.ResourceChangesChan,
		channels.LinkChangesChan,
		stagingState,
		params,
		logger,
	)
	if err != nil {
		channels.ErrChan <- err
		return
	}
}

// Composes the contributions links have made to a resource into both sides of the
// comparison that produces its change set.
//
// A link writes fields that the blueprint does not declare, and both sides have to account
// for them or the comparison is between different things. Composing neither side reports
// nothing when a link's contribution changes, and composing only the side that is being
// deployed reports every field a link owns as new on every update.
//
// The two sides are composed against different sets of links, which is what makes a
// removed link visible. The current side takes every link that contributes to the resource
// today. The side being deployed takes only the links that will still exist, so a link
// that the blueprint no longer has is present in what the resource is and absent from what
// it is going to be, and its contribution is reported as a removal rather than
// disappearing when the resource is deployed.
func (s *defaultResourceChangeStager) composeLinkContributions(
	ctx context.Context,
	stageResourceInfo *stageResourceChangeInfo,
	resourceInfo *provider.ResourceInfo,
) error {
	// A blueprint that has never been deployed has no instance to hold links, and a
	// resource that is being created has nothing contributed to it yet.
	if stageResourceInfo.instanceID == "" {
		return nil
	}

	resourceName := stageResourceInfo.node.ResourceName
	linksWithMappings, err := s.stateContainer.Links().ListWithResourceDataMappings(
		ctx,
		stageResourceInfo.instanceID,
		resourceName,
	)
	if err != nil {
		if state.IsInstanceNotFound(err) {
			return nil
		}

		return err
	}

	if len(linksWithMappings) == 0 {
		return nil
	}

	// Composed specs are swapped onto this ResourceInfo rather than written into what it
	// points at: the resolved resource is cached and read again to deploy the resource,
	// where the declared spec is the one that gets persisted. The current state is already
	// a copy, and is treated the same way so neither side reads as safe to compose in
	// place.
	if resourceInfo.CurrentResourceState != nil {
		current, err := specmerge.ApplyLinkProjections(
			resourceInfo.CurrentResourceState.SpecData,
			resourceName,
			linksWithMappings,
		)
		if err != nil {
			return err
		}

		currentState := *resourceInfo.CurrentResourceState
		currentState.SpecData = current.Spec
		resourceInfo.CurrentResourceState = &currentState
	}

	if resourceInfo.ResourceWithResolvedSubs != nil {
		desired, err := specmerge.ApplyLinkProjections(
			resourceInfo.ResourceWithResolvedSubs.Spec,
			resourceName,
			survivingLinks(linksWithMappings, stageResourceInfo.node),
		)
		if err != nil {
			return err
		}

		composedResource := *resourceInfo.ResourceWithResolvedSubs
		composedResource.Spec = desired.Spec
		resourceInfo.ResourceWithResolvedSubs = &composedResource
	}

	return nil
}

// The links that contribute to a resource and that the blueprint being staged still has.
//
// The chain link node comes from the blueprint the change set is being produced for, so
// the links adjacent to it are the ones that will exist once it is deployed. A link that
// contributes to the resource and is not among them is being removed.
func survivingLinks(
	linksWithMappings []state.LinkState,
	node *links.ChainLinkNode,
) []state.LinkState {
	surviving := map[string]bool{}
	for _, linksToNode := range node.LinksTo {
		surviving[core.LogicalLinkName(node.ResourceName, linksToNode.ResourceName)] = true
	}
	for _, linkedFromNode := range node.LinkedFrom {
		surviving[core.LogicalLinkName(linkedFromNode.ResourceName, node.ResourceName)] = true
	}

	kept := []state.LinkState{}
	for _, link := range linksWithMappings {
		if surviving[link.Name] {
			kept = append(kept, link)
		}
	}

	return kept
}

func (s *defaultResourceChangeStager) stageChanges(
	ctx context.Context,
	stageResourceInfo *stageResourceChangeInfo,
	resourceImplementation provider.Resource,
	changesChan chan ResourceChangesMessage,
	linkChangesChan chan LinkChangesMessage,
	stagingState ChangeStagingState,
	params core.BlueprintParams,
	logger core.Logger,
) error {
	resourceIDLogger := logger.WithFields(
		core.StringLogField("resourceId", stageResourceInfo.resourceID),
	)
	resourceIDLogger.Debug(
		"resolving substitutions in resource definition and loading resource state",
	)
	resourceInfo, resolveResourceResult, err := getResourceInfo(
		ctx,
		stageResourceInfo,
		s.substitutionResolver,
		s.resourceCache,
		s.stateContainer,
	)
	if err != nil {
		resourceIDLogger.Debug(
			"failed to resolve substitutions in resource definition and load resource state",
			core.ErrorLogField("error", err),
		)
		return err
	}

	// What the resource is declared as, kept aside while the comparison is made against
	// specs that include what links have contributed.
	declaredResource := resourceInfo.ResourceWithResolvedSubs
	declaredResourceState := resourceInfo.CurrentResourceState

	err = s.composeLinkContributions(ctx, stageResourceInfo, resourceInfo)
	if err != nil {
		resourceIDLogger.Debug(
			"failed to compose the contributions links have made to the resource",
			core.ErrorLogField("error", err),
		)
		return err
	}

	resourceIDLogger.Info(
		"generating change set for resource",
	)
	changes, err := s.changeGenerator.GenerateChanges(
		ctx,
		resourceInfo,
		resourceImplementation,
		resolveResourceResult.ResolveOnDeploy,
		params,
	)
	if err != nil {
		resourceIDLogger.Debug(
			"failed to generate change set for resource",
			core.ErrorLogField("error", err),
		)
		return err
	}

	// The change set carries the resource for the deployment to apply, which is the
	// declared one. A deployment composes the contributions in itself, from the links as
	// they are when it runs rather than as they were when changes were staged, and what
	// it persists is the declared spec.
	changes.AppliedResourceInfo.ResourceWithResolvedSubs = declaredResource
	changes.AppliedResourceInfo.CurrentResourceState = declaredResourceState

	// The resource must be recreated if an element that it previously depended on
	// has been removed.
	if !changes.MustRecreate {
		changes.MustRecreate = stagingState.MustRecreateResourceOnRemovedDependencies(
			resourceInfo.ResourceName,
		)
	}

	changesMsg := ResourceChangesMessage{
		ResourceName:    stageResourceInfo.node.ResourceName,
		Changes:         *changes,
		Removed:         false,
		New:             isResourceNewForStaging(resourceInfo.CurrentResourceState),
		ResolveOnDeploy: resolveResourceResult.ResolveOnDeploy,
		ConditionKnownOnDeploy: isConditionKnownOnDeploy(
			stageResourceInfo.node.ResourceName,
			resolveResourceResult.ResolveOnDeploy,
		),
	}

	resourceIDLogger.Debug("applying resource changes to internal, ephemeral state")
	// We must make sure that resource changes are applied to the internal changing state
	// before we can stage links that are dependent on the resource changes.
	// Otherwise, we can end up with inconsistent state where links are staged before the
	// resource changes are applied, leading to incorrect link changes being reported.
	//
	// The ephemeral state must also be updated before broadcasting the change message
	// to ensure that the state is consistent to prevent bugs due to state updates
	// that have not settled.
	stagingState.ApplyResourceChanges(changesMsg)
	linksReadyToBeStaged := stagingState.UpdateLinkStagingState(stageResourceInfo.node)

	changesChan <- changesMsg

	resourceIDLogger.Info("preparing and staging link changes for resource")
	err = s.prepareAndStageLinkChanges(
		ctx,
		resourceInfo,
		linksReadyToBeStaged,
		linkChangesChan,
		stagingState,
		params,
		resourceIDLogger,
	)
	if err != nil {
		return err
	}

	return nil
}

// isResourceNewForStaging determines if a resource should be treated as "new"
// (requiring creation) during change staging. A resource is considered new if:
//   - No persisted state exists, OR
//   - The persisted state indicates the resource was never successfully created
//     (e.g., previous creation attempt failed or was interrupted)
func isResourceNewForStaging(currentState *state.ResourceState) bool {
	if currentState == nil {
		return true
	}
	return core.ResourceStatusIsUnsuccessfulCreate(currentState.Status)
}

func (s *defaultResourceChangeStager) prepareAndStageLinkChanges(
	ctx context.Context,
	currentResourceInfo *provider.ResourceInfo,
	linksReadyToBeStaged []*LinkPendingCompletion,
	linkChangesChan chan LinkChangesMessage,
	stagingState ChangeStagingState,
	params core.BlueprintParams,
	logger core.Logger,
) error {
	for _, readyToStage := range linksReadyToBeStaged {
		resourceAName := getResourceNameFromLinkChainNode(readyToStage.resourceANode)
		resourceBName := getResourceNameFromLinkChainNode(readyToStage.resourceBNode)
		logicalLinkName := core.LogicalLinkName(
			resourceAName,
			resourceBName,
		)
		linkLogger := logger.Named("link").WithFields(
			core.StringLogField("resourceA", resourceAName),
			core.StringLogField("resourceB", resourceBName),
			core.StringLogField("linkName", logicalLinkName),
		)

		linkLogger.Info("loading link plugin implementation")
		linkImpl, _, err := getLinkImplementation(
			readyToStage.resourceANode,
			readyToStage.resourceBNode,
		)
		if err != nil {
			linkLogger.Debug(
				"failed to load link plugin implementation",
				core.ErrorLogField("error", err),
			)
			return err
		}

		// Links are staged in series. Staging only computes changes, so this is about
		// keeping the staged state consistent when several links modify the same
		// resource, and is unrelated to how they are deployed: deployment runs the
		// links in a batch concurrently and relies on resource locks instead.
		err = s.linkChangeStager.StageChanges(
			ctx,
			linkImpl,
			currentResourceInfo,
			readyToStage,
			stagingState,
			linkChangesChan,
			params,
			linkLogger,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
