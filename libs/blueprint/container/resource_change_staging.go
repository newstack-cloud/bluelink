package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
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
		New:             resourceInfo.CurrentResourceState == nil,
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

		// Links are staged in series to reflect what happens with deployment.
		// For deployment, multiple links could be modifying the same resource,
		// to ensure consistency in state, links involving the same resource will be
		// both staged and deployed synchronously.
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
