package container

import (
	"context"
	"slices"
	"time"

	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/resourcehelpers"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	"github.com/two-hundred/celerity/libs/blueprint/subengine"
	commoncore "github.com/two-hundred/celerity/libs/common/core"
)

const (
	prepareFailureMessage = "failed to load instance state while preparing to deploy"
)

func (c *defaultBlueprintContainer) Deploy(
	ctx context.Context,
	input *DeployInput,
	channels *DeployChannels,
	paramOverrides core.BlueprintParams,
) error {
	instanceID, err := c.getInstanceID(input)
	if err != nil {
		return err
	}

	ctxWithInstanceID := context.WithValue(ctx, core.BlueprintInstanceIDKey, instanceID)
	state := c.createDeploymentState()

	isNewInstance, err := checkDeploymentForNewInstance(input)
	if err != nil {
		return err
	}

	interceptDeploymentUpdateChan := make(chan DeploymentUpdateMessage)
	interceptDeploymentFinishChan := make(chan DeploymentFinishedMessage)
	rewiredChannels := &DeployChannels{
		ResourceUpdateChan:   channels.ResourceUpdateChan,
		ChildUpdateChan:      channels.ChildUpdateChan,
		LinkUpdateChan:       channels.LinkUpdateChan,
		ErrChan:              channels.ErrChan,
		DeploymentUpdateChan: interceptDeploymentUpdateChan,
		FinishChan:           interceptDeploymentFinishChan,
	}

	go c.deploy(
		ctxWithInstanceID,
		&DeployInput{
			InstanceID: instanceID,
			Changes:    input.Changes,
			Rollback:   input.Rollback,
		},
		rewiredChannels,
		state,
		isNewInstance,
		paramOverrides,
	)

	// Intercept the top-level instance deployment events
	// to ensure that the instance state is updated with status information
	// for failures.
	// Instead of making a call to persist the instance status updates
	// at every point a blueprint instance level update is made, before calling deploy
	// the channels are re-wired to intercept the top-level instance
	// deployment events, persist the status updates and then pass
	// the events to the caller-provided channels.
	//
	// This will ensure that the status will be persisted before the message reaches
	// the caller-provided channels, so even though this is called asynchronously,
	// it will ensure that no top-level status updates received by the caller go out of sync
	// with the status information in the persisted state.
	go c.persistInstanceDeploymentStatusUpdates(
		ctxWithInstanceID,
		instanceID,
		isNewInstance,
		input.Rollback,
		rewiredChannels,
		channels,
	)

	return nil
}

func (c *defaultBlueprintContainer) deploy(
	ctx context.Context,
	input *DeployInput,
	channels *DeployChannels,
	deployState DeploymentState,
	isNewInstance bool,
	paramOverrides core.BlueprintParams,
) {
	instanceTreePath := getInstanceTreePath(paramOverrides, input.InstanceID)
	if exceedsMaxDepth(instanceTreePath, MaxBlueprintDepth) {
		channels.ErrChan <- errMaxBlueprintDepthExceeded(
			instanceTreePath,
			MaxBlueprintDepth,
		)
		return
	}

	if input.Changes == nil {
		channels.FinishChan <- c.createDeploymentFinishedMessage(
			input.InstanceID,
			determineInstanceDeployFailedStatus(input.Rollback, isNewInstance),
			[]string{emptyChangesDeployFailedMessage(input.Rollback)},
			/* elapsedTime */ 0,
			/* prepareElapsedTime */ nil,
		)
		return
	}

	startTime := c.clock.Now()
	channels.DeploymentUpdateChan <- DeploymentUpdateMessage{
		InstanceID:      input.InstanceID,
		Status:          core.InstanceStatusPreparing,
		UpdateTimestamp: startTime.Unix(),
	}

	instances := c.stateContainer.Instances()
	currentInstanceState, err := instances.Get(ctx, input.InstanceID)
	if err != nil {
		if !state.IsInstanceNotFound(err) {
			channels.FinishChan <- c.createDeploymentFinishedMessage(
				input.InstanceID,
				determineInstanceDeployFailedStatus(input.Rollback, isNewInstance),
				[]string{prepareFailureMessage},
				c.clock.Since(startTime),
				/* prepareElapsedTime */ nil,
			)
			return
		}
	}

	// Use the same behaviour as change staging to extract the nodes
	// that need to be deployed or updated where they are grouped for concurrent deployment
	// and in order based on links, references and use of the `dependsOn` property.
	prepareResult, err := c.blueprintPreparer.Prepare(
		ctx,
		c.spec.Schema(),
		subengine.ResolveForDeployment,
		input.Changes,
		c.linkInfo,
		paramOverrides,
	)
	if err != nil {
		channels.FinishChan <- c.createDeploymentFinishedMessage(
			input.InstanceID,
			determineInstanceDeployFailedStatus(input.Rollback, isNewInstance),
			[]string{prepareFailureMessage},
			c.clock.Since(startTime),
			/* prepareElapsedTime */ nil,
		)
		return
	}

	deployCtx := &DeployContext{
		StartTime:             startTime,
		State:                 deployState,
		Rollback:              input.Rollback,
		Destroying:            false,
		Channels:              channels,
		ParamOverrides:        paramOverrides,
		InstanceStateSnapshot: &currentInstanceState,
		ResourceProviders:     prepareResult.ResourceProviderMap,
		DeploymentGroups:      prepareResult.ParallelGroups,
		InputChanges:          input.Changes,
		ResourceTemplates:     prepareResult.BlueprintContainer.ResourceTemplates(),
		ResourceRegistry:      c.resourceRegistry.WithParams(paramOverrides),
	}

	flattenedNodes := core.Flatten(prepareResult.ParallelGroups)
	// Ensure all direct dependencies are populated between nodes
	// in the deployment groups, this provides the information needed
	// to determine which elements can be deployed next upon completion
	// of others.
	err = PopulateDirectDependencies(
		ctx,
		flattenedNodes,
		c.refChainCollector,
		paramOverrides,
	)
	if err != nil {
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}

	sentFinishedMessage, err := c.removeElements(
		ctx,
		input,
		deployCtx,
		flattenedNodes,
	)
	if err != nil {
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}
	if sentFinishedMessage {
		return
	}

	err = c.saveNewInstance(
		ctx,
		input.InstanceID,
		isNewInstance,
		determineInstanceDeployingStatus(input.Rollback, isNewInstance),
	)
	if err != nil {
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}

	sentFinishedMessage, err = c.deployElements(
		ctx,
		input,
		deployCtx,
		isNewInstance,
	)
	if err != nil {
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}
	if sentFinishedMessage {
		return
	}

	channels.FinishChan <- c.createDeploymentFinishedMessage(
		input.InstanceID,
		determineInstanceDeployedStatus(input.Rollback, isNewInstance),
		[]string{},
		c.clock.Since(startTime),
		/* prepareElapsedTime */ nil,
	)
}

func (c *defaultBlueprintContainer) persistInstanceDeploymentStatusUpdates(
	ctx context.Context,
	instanceID string,
	isNewInstance bool,
	rollingBack bool,
	listenToChannels *DeployChannels,
	forwardToChannels *DeployChannels,
) {
	finished := false
	for !finished {
		select {
		case msg := <-listenToChannels.DeploymentUpdateChan:
			updateTimestamp := int(msg.UpdateTimestamp)
			err := c.stateContainer.Instances().UpdateStatus(
				ctx,
				instanceID,
				state.InstanceStatusInfo{
					Status:                    msg.Status,
					LastStatusUpdateTimestamp: &updateTimestamp,
				},
			)
			if err != nil {
				forwardToChannels.ErrChan <- err
				return
			}
			forwardToChannels.DeploymentUpdateChan <- msg
		case msg := <-listenToChannels.FinishChan:
			statusInfo := createDeployFinishedInstanceStatusInfo(&msg, rollingBack, isNewInstance)
			err := c.stateContainer.Instances().UpdateStatus(
				ctx,
				instanceID,
				statusInfo,
			)
			if err != nil {
				forwardToChannels.ErrChan <- err
				return
			}
			forwardToChannels.FinishChan <- msg
			finished = true
		}
	}
}

func (c *defaultBlueprintContainer) getInstanceID(input *DeployInput) (string, error) {
	if input.InstanceID == "" {
		return c.idGenerator.GenerateID()
	}

	return input.InstanceID, nil
}

func (c *defaultBlueprintContainer) saveNewInstance(
	ctx context.Context,
	instanceID string,
	isNewInstance bool,
	currentStatus core.InstanceStatus,
) error {
	if !isNewInstance {
		return nil
	}

	return c.stateContainer.Instances().Save(
		ctx,
		state.InstanceState{
			InstanceID: instanceID,
			Status:     currentStatus,
		},
	)
}

func (c *defaultBlueprintContainer) deployElements(
	ctx context.Context,
	input *DeployInput,
	deployCtx *DeployContext,
	newInstance bool,
) (bool, error) {
	internalChannels := CreateDeployChannels()
	prepareElapsedTime := deployCtx.State.GetPrepareDuration()
	if len(deployCtx.DeploymentGroups) == 0 {
		deployCtx.Channels.FinishChan <- c.createDeploymentFinishedMessage(
			input.InstanceID,
			determineInstanceDeployedStatus(input.Rollback, newInstance),
			[]string{},
			c.clock.Since(deployCtx.StartTime),
			prepareElapsedTime,
		)
		return true, nil
	}

	c.startDeploymentFromFirstGroup(
		ctx,
		input.InstanceID,
		input.Changes,
		deployCtx,
	)

	return c.listenToAndProcessDeploymentEvents(
		ctx,
		input.InstanceID,
		deployCtx,
		input.Changes,
		internalChannels,
	)
}

func (c *defaultBlueprintContainer) startDeploymentFromFirstGroup(
	ctx context.Context,
	instanceID string,
	changes *BlueprintChanges,
	deployCtx *DeployContext,
) {
	instanceTreePath := getInstanceTreePath(deployCtx.ParamOverrides, instanceID)

	for _, node := range deployCtx.DeploymentGroups[0] {
		c.deployNode(
			ctx,
			node,
			instanceID,
			instanceTreePath,
			changes,
			deployCtx,
		)
	}
}

func (c *defaultBlueprintContainer) deployNode(
	ctx context.Context,
	node *DeploymentNode,
	instanceID string,
	instanceTreePath string,
	changes *BlueprintChanges,
	deployCtx *DeployContext,
) {
	if node.Type() == DeploymentNodeTypeResource {
		deployCtx.State.SetElementDependencies(
			&ResourceIDInfo{
				ResourceName: node.ChainLinkNode.ResourceName,
			},
			extractNodeDependencyInfo(node),
		)
		go c.resourceDeployer.Deploy(
			ctx,
			instanceID,
			node.ChainLinkNode,
			changes,
			deployCtx,
		)
	} else if node.Type() == DeploymentNodeTypeChild {
		includeTreePath := getIncludeTreePath(deployCtx.ParamOverrides, node.Name())
		childName := core.ToLogicalChildName(node.Name())
		deployCtx.State.SetElementDependencies(
			&ChildBlueprintIDInfo{
				ChildName: childName,
			},
			extractNodeDependencyInfo(node),
		)
		childChanges := getChildChanges(changes, childName)
		go c.childDeployer.Deploy(
			ctx,
			instanceID,
			instanceTreePath,
			includeTreePath,
			node.ChildNode,
			childChanges,
			deployCtx,
		)
	}
}

func (c *defaultBlueprintContainer) listenToAndProcessDeploymentEvents(
	ctx context.Context,
	instanceID string,
	deployCtx *DeployContext,
	changes *BlueprintChanges,
	internalChannels *DeployChannels,
) (bool, error) {
	finished := map[string]*deployUpdateMessageWrapper{}
	// For this to work, the blueprint changes provided must match
	// the loaded blueprint.
	// The count must reflect the number of elements that will be deployed
	// taking resources, links and child blueprints into account.
	elementsToDeploy := countElementsToDeploy(changes)

	var err error
	for (len(finished) < elementsToDeploy) &&
		err == nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case msg := <-internalChannels.ResourceUpdateChan:
			err = c.handleResourceUpdateMessage(ctx, instanceID, msg, deployCtx, finished)
		case msg := <-internalChannels.ChildUpdateChan:
			err = c.handleChildUpdateMessage(ctx, instanceID, msg, deployCtx, finished)
		case msg := <-internalChannels.LinkUpdateChan:
			err = c.handleLinkUpdateMessage(ctx, instanceID, msg, deployCtx, finished)
		case err = <-internalChannels.ErrChan:
		}
	}

	if err != nil {
		return true, err
	}

	failed := getFailedElementDeploymentsAndUpdateState(finished, changes, deployCtx)
	if len(failed) > 0 {
		deployCtx.Channels.FinishChan <- c.createDeploymentFinishedMessage(
			instanceID,
			determineFinishedFailureStatus(
				/* destroyingInstance */ false,
				deployCtx.Rollback,
			),
			finishedFailureMessages(deployCtx, failed),
			c.clock.Since(deployCtx.StartTime),
			/* prepareElapsedTime */
			deployCtx.State.GetPrepareDuration(),
		)

		return true, nil
	}

	return false, nil
}

func (c *defaultBlueprintContainer) handleResourceUpdateMessage(
	ctx context.Context,
	instanceID string,
	msg ResourceDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
) error {
	if msg.InstanceID != instanceID {
		// If message is for a child blueprint, pass through to the client
		// to ensure updates within the child blueprint are surfaced.
		// This allows for the client to provide more detailed feedback to the user
		// for the progress within a child blueprint.
		deployCtx.Channels.ResourceUpdateChan <- msg
		return nil
	}

	elementName := core.ResourceElementID(msg.ResourceName)

	if isResourceDestroyEvent(msg.PreciseStatus, deployCtx.Rollback) {
		return c.handleResourceDestroyEvent(ctx, msg, deployCtx, finished, elementName)
	}

	if isResourceUpdateEvent(msg.PreciseStatus, deployCtx.Rollback) {
		return c.handleResourceUpdateEvent(ctx, msg, deployCtx, finished, elementName)
	}

	if isResourceCreationEvent(msg.PreciseStatus, deployCtx.Rollback) {
		return c.handleResourceCreationEvent(ctx, msg, deployCtx, finished, elementName)
	}

	return nil
}

func (c *defaultBlueprintContainer) handleResourceUpdateEvent(
	ctx context.Context,
	msg ResourceDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	elementName string,
) error {
	resources := c.stateContainer.Resources()
	element := &ResourceIDInfo{
		ResourceID:   msg.ResourceID,
		ResourceName: msg.ResourceName,
	}

	if startedUpdatingResource(msg.PreciseStatus, deployCtx.Rollback) {
		deployCtx.State.SetElementInProgress(element)
		updateTimestamp := int(msg.UpdateTimestamp)
		err := resources.UpdateStatus(
			ctx,
			msg.InstanceID,
			msg.ResourceID,
			state.ResourceStatusInfo{
				Status:                    msg.Status,
				PreciseStatus:             msg.PreciseStatus,
				LastStatusUpdateTimestamp: &updateTimestamp,
			},
		)
		if err != nil {
			return err
		}
	}

	if resourceUpdateConfigComplete(msg.PreciseStatus, deployCtx.Rollback) {
		err := c.handleResourceConfigComplete(
			ctx,
			msg,
			element,
			deployCtx,
			resources,
		)
		if err != nil {
			return err
		}
	}

	// This will not persist the current status update if the message
	// represents a failure that can be retried.
	// The initiator of the deployment process will receive failure messages
	// that can be retried so that the end user can be informed when
	// a resource update is taking longer due to a failure that can be retried.
	// For historical purposes, how many attempts have been made to deploy a resource
	// will be persisted under the durations section of the resource state.
	if finishedUpdatingResource(msg, deployCtx.Rollback) {
		msgWrapper := &deployUpdateMessageWrapper{
			resourceUpdateMessage: &msg,
		}
		finished[elementName] = msgWrapper

		if updateWasSuccessful(
			msgWrapper,
			deployCtx.Rollback,
		) {
			return c.handleSuccessfulResourceDeployment(
				ctx,
				msg,
				deployCtx,
				element,
				resources,
				deployCtx.State.SetUpdatedElement,
			)
		} else {
			updateTimestamp := int(msg.UpdateTimestamp)
			currentTimestamp := int(c.clock.Now().Unix())
			return resources.UpdateStatus(
				ctx,
				msg.InstanceID,
				msg.ResourceID,
				state.ResourceStatusInfo{
					Status:                     msg.Status,
					PreciseStatus:              msg.PreciseStatus,
					LastDeployAttemptTimestamp: &currentTimestamp,
					LastStatusUpdateTimestamp:  &updateTimestamp,
					Durations:                  msg.Durations,
					FailureReasons:             msg.FailureReasons,
				},
			)
		}
	}

	return nil
}

func (c *defaultBlueprintContainer) buildResourceState(
	msg ResourceDeployUpdateMessage,
	dependencyInfo *state.DependencyInfo,
	deployCtx *DeployContext,
) state.ResourceState {
	resourceTemplateName := deployCtx.ResourceTemplates[msg.ResourceName]
	resourceData := deployCtx.State.GetResourceData(msg.ResourceName)
	resourceState := state.ResourceState{
		ResourceID:                 msg.ResourceID,
		ResourceName:               msg.ResourceName,
		ResourceTemplateName:       resourceTemplateName,
		InstanceID:                 msg.InstanceID,
		Status:                     msg.Status,
		PreciseStatus:              msg.PreciseStatus,
		Durations:                  msg.Durations,
		FailureReasons:             msg.FailureReasons,
		DependsOnResources:         dependencyInfo.DependsOnResources,
		DependsOnChildren:          dependencyInfo.DependsOnChildren,
		LastStatusUpdateTimestamp:  int(msg.UpdateTimestamp),
		LastDeployAttemptTimestamp: int(c.clock.Now().Unix()),
	}

	if resourceData != nil {
		resourceState.Metadata = resourceData.Metadata
		resourceState.Description = resourceData.Description
	}

	wrappedMsg := &deployUpdateMessageWrapper{
		resourceUpdateMessage: &msg,
	}
	successfulUpdate := updateWasSuccessful(
		wrappedMsg,
		deployCtx.Rollback,
	)
	successfulCreation := creationWasSuccessful(
		wrappedMsg,
		deployCtx.Rollback,
	)
	if successfulUpdate || successfulCreation {
		if resourceData != nil {
			resourceState.ResourceSpecData = resourceData.Spec
		}

		resourceState.LastDeployedTimestamp = int(c.clock.Now().Unix())
	}

	return resourceState
}

func (c *defaultBlueprintContainer) prepareAndDeployLinks(
	ctx context.Context,
	instanceID string,
	linksReadyToBeDeployed []*LinkPendingCompletion,
	deployCtx *DeployContext,
) error {
	if len(linksReadyToBeDeployed) == 0 {
		// Make sure that the latest instance state is only loaded
		// if it is needed for links ready to be deployed.
		return nil
	}

	// Get the latest instance state that will be fully updated with the current
	// state of the resources that the links depend on.
	instances := c.stateContainer.Instances()
	latestInstanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		return err
	}

	// Links are staged in series to reflect what happens with deployment.
	// For deployment, multiple links could be modifying the same resource,
	// to ensure consistency in state, links involving the same resource will be
	// both staged and deployed synchronously.
	for _, readyToDeploy := range linksReadyToBeDeployed {
		linkImpl, err := getLinkImplementation(
			readyToDeploy.resourceANode,
			readyToDeploy.resourceBNode,
		)
		if err != nil {
			return err
		}

		err = c.deployLink(
			ctx,
			linkImpl,
			readyToDeploy,
			&latestInstanceState,
			deployCtx,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *defaultBlueprintContainer) deployLink(
	ctx context.Context,
	linkImpl provider.Link,
	readyToDeploy *LinkPendingCompletion,
	latestInstanceState *state.InstanceState,
	deployCtx *DeployContext,
) error {
	links := c.stateContainer.Links()
	linkName := createLogicalLinkName(
		readyToDeploy.resourceANode.ResourceName,
		readyToDeploy.resourceBNode.ResourceName,
	)
	linkState, err := links.GetByName(ctx, latestInstanceState.InstanceID, linkName)
	if err != nil && !state.IsLinkNotFound(err) {
		return err
	}
	linkID, err := c.getLinkID(linkState)
	if err != nil {
		return err
	}

	linkUpdateType := getLinkUpdateTypeFromState(linkState)

	retryPolicy, err := getLinkRetryPolicy(
		ctx,
		linkName,
		// We must use a fresh snapshot of the state that includes
		// the resources that the link depends on.
		// When a new blueprint instance is being deployed or
		// new resources are being added, those
		// that the link is for will not be in the instance snapshot taken
		// before deployment.
		latestInstanceState,
		c.linkRegistry,
		c.defaultRetryPolicy,
	)
	if err != nil {
		return err
	}

	result, err := c.linkDeployer.Deploy(
		ctx,
		&LinkIDInfo{
			LinkID:   linkID,
			LinkName: linkName,
		},
		latestInstanceState.InstanceID,
		linkUpdateType,
		linkImpl,
		deployCtx,
		retryPolicy,
	)
	if err != nil {
		return err
	}

	deployCtx.State.SetLinkDeployResult(linkName, result)

	return nil
}

func (c *defaultBlueprintContainer) getLinkID(linkState state.LinkState) (string, error) {
	if linkState.LinkID != "" {
		return linkState.LinkID, nil
	}

	return c.idGenerator.GenerateID()
}

func (c *defaultBlueprintContainer) deployNextElementsAfterResource(
	ctx context.Context,
	instanceID string,
	deployCtx *DeployContext,
	deployedResource *ResourceIDInfo,
	configComplete bool,
) {
	if deployCtx.CurrentGroupIndex == len(deployCtx.DeploymentGroups)-1 {
		// No more groups to deploy.
		return
	}

	elementName := core.ResourceElementID(deployedResource.ResourceName)
	nextGroup := deployCtx.DeploymentGroups[deployCtx.CurrentGroupIndex+1]
	for _, node := range nextGroup {
		dependencyNode := commoncore.Find(
			node.DirectDependencies,
			func(dep *DeploymentNode, _ int) bool {
				return dep.Name() == elementName
			},
		)

		stabilisedDependencies, err := c.getStabilisedDependencies(
			ctx,
			node,
			deployCtx.ResourceRegistry,
			deployCtx.ParamOverrides,
		)
		if err != nil {
			deployCtx.Channels.ErrChan <- err
			return
		}

		otherDependenciesInProgress := c.checkDependenciesInProgress(
			node,
			stabilisedDependencies,
			[]string{elementName},
			deployCtx.State,
		)

		if dependencyNode != nil &&
			!otherDependenciesInProgress &&
			readyToDeployAfterDependency(
				node,
				dependencyNode,
				stabilisedDependencies,
				configComplete,
			) {
			instanceTreePath := getInstanceTreePath(deployCtx.ParamOverrides, instanceID)
			c.deployNode(
				ctx,
				node,
				instanceID,
				instanceTreePath,
				deployCtx.InputChanges,
				deployCtx,
			)
		}
	}
}

func (c *defaultBlueprintContainer) getStabilisedDependencies(
	ctx context.Context,
	node *DeploymentNode,
	resourceRegistry resourcehelpers.Registry,
	paramOverrides core.BlueprintParams,
) ([]string, error) {
	if node.Type() == DeploymentNodeTypeResource {
		dependentResource := node.ChainLinkNode.Resource
		dependentResourceType := getResourceType(dependentResource)

		providerNamespace := provider.ExtractProviderFromItemType(dependentResourceType)
		stabilisedDepsOutput, err := resourceRegistry.StabilisedDependencies(
			ctx,
			dependentResourceType,
			&provider.ResourceStabilisedDependenciesInput{
				ProviderContext: provider.NewProviderContextFromParams(
					providerNamespace,
					paramOverrides,
				),
			},
		)
		if err != nil {
			return nil, err
		}

		return stabilisedDepsOutput.StabilisedDependencies, nil
	}

	return []string{}, nil
}

func (c *defaultBlueprintContainer) checkDependenciesInProgress(
	dependant *DeploymentNode,
	dependantStabilisedDeps []string,
	ignoreElements []string,
	state DeploymentState,
) bool {
	atLeastOneInProgress := false
	i := 0
	for !atLeastOneInProgress && i < len(dependant.DirectDependencies) {
		dependency := dependant.DirectDependencies[i]
		if !slices.Contains(ignoreElements, dependency.Name()) {
			dependencyElement := createElementFromDeploymentNode(dependency)
			inProgress := state.IsElementInProgress(dependencyElement)
			if inProgress {
				atLeastOneInProgress = true
			} else {
				// The dependency is considered in progress if it has a "config complete"
				// status and the dependant is a resource that requires the dependency to be stable
				// before it can be deployed.
				atLeastOneInProgress = c.configCompleteDependencyMustStabilise(
					state,
					dependant,
					dependantStabilisedDeps,
					dependencyElement,
					dependency,
				)
			}
		}
		i += 1
	}

	return atLeastOneInProgress
}

func (c *defaultBlueprintContainer) configCompleteDependencyMustStabilise(
	state DeploymentState,
	dependant *DeploymentNode,
	dependantStabilisedDeps []string,
	dependencyElement state.Element,
	dependency *DeploymentNode,
) bool {
	configComplete := state.IsElementConfigComplete(dependencyElement)
	if configComplete {
		return dependencyMustStabilise(dependant, dependency, dependantStabilisedDeps)
	}

	return false
}

func (c *defaultBlueprintContainer) handleResourceCreationEvent(
	ctx context.Context,
	msg ResourceDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	elementName string,
) error {
	resources := c.stateContainer.Resources()
	element := &ResourceIDInfo{
		ResourceID:   msg.ResourceID,
		ResourceName: msg.ResourceName,
	}

	if startedCreatingResource(msg.PreciseStatus, deployCtx.Rollback) {
		deployCtx.State.SetElementInProgress(element)
		updateTimestamp := int(msg.UpdateTimestamp)
		err := resources.UpdateStatus(
			ctx,
			msg.InstanceID,
			msg.ResourceID,
			state.ResourceStatusInfo{
				Status:                    msg.Status,
				PreciseStatus:             msg.PreciseStatus,
				LastStatusUpdateTimestamp: &updateTimestamp,
			},
		)
		if err != nil {
			return err
		}
	}

	if resourceCreationConfigComplete(msg.PreciseStatus, deployCtx.Rollback) {
		err := c.handleResourceConfigComplete(
			ctx,
			msg,
			element,
			deployCtx,
			resources,
		)
		if err != nil {
			return err
		}
	}

	// This will not persist the current status update if the message
	// represents a failure that can be retried.
	// The initiator of the deployment process will receive failure messages
	// that can be retried so that the end user can be informed when
	// a resource deployment is taking longer due to a failure that can be retried.
	// For historical purposes, how many attempts have been made to deploy a resource
	// will be persisted under the durations section of the resource state.
	if finishedCreatingResource(msg, deployCtx.Rollback) {
		msgWrapper := &deployUpdateMessageWrapper{
			resourceUpdateMessage: &msg,
		}
		finished[elementName] = msgWrapper

		resourceCreationSuccessful := creationWasSuccessful(
			msgWrapper,
			deployCtx.Rollback,
		)

		if resourceCreationSuccessful {
			return c.handleSuccessfulResourceDeployment(
				ctx,
				msg,
				deployCtx,
				element,
				resources,
				deployCtx.State.SetCreatedElement,
			)
		} else {
			updateTimestamp := int(msg.UpdateTimestamp)
			currentTimestamp := int(c.clock.Now().Unix())
			return resources.UpdateStatus(
				ctx,
				msg.InstanceID,
				msg.ResourceID,
				state.ResourceStatusInfo{
					Status:                     msg.Status,
					PreciseStatus:              msg.PreciseStatus,
					LastDeployAttemptTimestamp: &currentTimestamp,
					LastStatusUpdateTimestamp:  &updateTimestamp,
					Durations:                  msg.Durations,
					FailureReasons:             msg.FailureReasons,
				},
			)
		}
	}

	return nil
}

func (c *defaultBlueprintContainer) handleSuccessfulResourceDeployment(
	ctx context.Context,
	msg ResourceDeployUpdateMessage,
	deployCtx *DeployContext,
	element *ResourceIDInfo,
	resources state.ResourcesContainer,
	saveElementInEphemeralState func(state.Element),
) error {
	// Update the ephemeral deploy state before persisting
	// the status update with the state container
	// to make sure deployment state is consistent
	// as the deploy state will be used across multiple goroutines
	// to determine the next elements to deploy.
	saveElementInEphemeralState(element)

	resourceDeps := deployCtx.State.GetElementDependencies(element)
	err := resources.Save(
		ctx,
		msg.InstanceID,
		c.buildResourceState(msg, resourceDeps, deployCtx),
	)
	if err != nil {
		return err
	}

	node := getDeploymentNode(
		element,
		deployCtx.DeploymentGroups,
		deployCtx.CurrentGroupIndex,
	)
	linksReadyToBeDeployed := deployCtx.State.UpdateLinkDeploymentState(
		node.ChainLinkNode,
	)

	go c.prepareAndDeployLinks(
		ctx,
		msg.InstanceID,
		linksReadyToBeDeployed,
		deployCtx,
	)

	c.deployNextElementsAfterResource(
		ctx,
		msg.InstanceID,
		deployCtx,
		element,
		/* configComplete */ false,
	)

	return nil
}

func (c *defaultBlueprintContainer) handleResourceConfigComplete(
	ctx context.Context,
	msg ResourceDeployUpdateMessage,
	element *ResourceIDInfo,
	deployCtx *DeployContext,
	resources state.ResourcesContainer,
) error {
	deployCtx.State.SetElementConfigComplete(element)
	updateTimestamp := int(msg.UpdateTimestamp)
	err := resources.UpdateStatus(
		ctx,
		msg.InstanceID,
		msg.ResourceID,
		state.ResourceStatusInfo{
			Status:                    msg.Status,
			PreciseStatus:             msg.PreciseStatus,
			Durations:                 msg.Durations,
			LastStatusUpdateTimestamp: &updateTimestamp,
		},
	)
	if err != nil {
		return err
	}

	c.deployNextElementsAfterResource(
		ctx,
		msg.InstanceID,
		deployCtx,
		element,
		/* configComplete */ true,
	)

	return nil
}

func (c *defaultBlueprintContainer) handleChildUpdateMessage(
	ctx context.Context,
	instanceID string,
	msg ChildDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
) error {
	if msg.ParentInstanceID != instanceID {
		// If message is for a child blueprint, pass through to the client
		// to ensure updates within the child blueprint are surfaced.
		// This allows for the client to provide more detailed feedback to the user
		// for the progress within a child blueprint.
		deployCtx.Channels.ChildUpdateChan <- msg
		return nil
	}

	elementName := core.ChildElementID(msg.ChildName)

	if isChildDestroyEvent(msg.Status, deployCtx.Rollback) {
		return c.handleChildDestroyEvent(ctx, msg, deployCtx, finished, elementName)
	}

	if isChildUpdateEvent(msg.Status, deployCtx.Rollback) {
		return c.handleChildUpdateEvent(ctx, msg, deployCtx, finished, elementName)
	}

	if isChildDeployEvent(msg.Status, deployCtx.Rollback) {
		return c.handleChildDeployEvent(ctx, msg, deployCtx, finished, elementName)
	}

	return nil
}

func (c *defaultBlueprintContainer) handleChildUpdateEvent(
	ctx context.Context,
	msg ChildDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	elementName string,
) error {
	children := c.stateContainer.Children()
	element := &ChildBlueprintIDInfo{
		ChildInstanceID: msg.ChildInstanceID,
		ChildName:       msg.ChildName,
	}

	if startedUpdatingChild(msg.Status, deployCtx.Rollback) {
		deployCtx.State.SetElementInProgress(element)
	}

	if finishedUpdatingChild(msg.Status, deployCtx.Rollback) {
		msgWrapper := &deployUpdateMessageWrapper{
			childUpdateMessage: &msg,
		}
		finished[elementName] = msgWrapper

		childUpdateSuccessful := updateWasSuccessful(
			msgWrapper,
			deployCtx.Rollback,
		)
		if childUpdateSuccessful {
			// Update the ephemeral deploy state before persisting
			// the status update with the state container
			// to make sure deployment state is consistent
			// as the deploy state will be used across multiple goroutines
			// to determine the next elements to deploy.
			deployCtx.State.SetUpdatedElement(element)
		}

		err := children.Attach(
			ctx,
			msg.ParentInstanceID,
			msg.ChildInstanceID,
			msg.ChildName,
		)
		if err != nil {
			return err
		}

		dependencies := deployCtx.State.GetElementDependencies(element)
		err = children.SaveDependencies(
			ctx,
			msg.ParentInstanceID,
			msg.ChildName,
			dependencies,
		)
		if err != nil {
			return err
		}

		if childUpdateSuccessful {
			c.deployNextElementsAfterChild(
				ctx,
				msg.ParentInstanceID,
				deployCtx,
				element,
			)
		}
	}

	return nil
}

func (c *defaultBlueprintContainer) deployNextElementsAfterChild(
	ctx context.Context,
	instanceID string,
	deployCtx *DeployContext,
	deployedChild *ChildBlueprintIDInfo,
) {
	if deployCtx.CurrentGroupIndex == len(deployCtx.DeploymentGroups)-1 {
		// No more groups to deploy.
		return
	}

	elementName := core.ChildElementID(deployedChild.ChildName)
	nextGroup := deployCtx.DeploymentGroups[deployCtx.CurrentGroupIndex+1]
	for _, node := range nextGroup {
		dependencyNode := commoncore.Find(
			node.DirectDependencies,
			func(dep *DeploymentNode, _ int) bool {
				return dep.Name() == elementName
			},
		)

		// The next element may be a resource that depends on another resource
		// that is expected to be stable before the resource in question can be deployed.
		// For this reason, even when we are choosing elements to deploy after a child blueprint,
		// other dependencies must be considered and stabilised dependencies must be checked.
		stabilisedDependencies, err := c.getStabilisedDependencies(
			ctx,
			node,
			deployCtx.ResourceRegistry,
			deployCtx.ParamOverrides,
		)
		if err != nil {
			deployCtx.Channels.ErrChan <- err
			return
		}

		otherDependenciesInProgress := c.checkDependenciesInProgress(
			node,
			stabilisedDependencies,
			[]string{elementName},
			deployCtx.State,
		)

		if dependencyNode != nil &&
			!otherDependenciesInProgress &&
			readyToDeployAfterDependency(
				node,
				dependencyNode,
				stabilisedDependencies,
				/* configComplete */ false,
			) {
			go c.resourceDeployer.Deploy(
				ctx,
				instanceID,
				node.ChainLinkNode,
				deployCtx.InputChanges,
				deployCtx,
			)
		}
	}
}

func (c *defaultBlueprintContainer) handleChildDeployEvent(
	ctx context.Context,
	msg ChildDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	elementName string,
) error {
	children := c.stateContainer.Children()
	element := &ChildBlueprintIDInfo{
		ChildInstanceID: msg.ChildInstanceID,
		ChildName:       msg.ChildName,
	}

	if startedDeployingChild(msg.Status, deployCtx.Rollback) {
		deployCtx.State.SetElementInProgress(element)
	}

	if finishedDeployingChild(msg.Status, deployCtx.Rollback) {
		msgWrapper := &deployUpdateMessageWrapper{
			childUpdateMessage: &msg,
		}
		finished[elementName] = msgWrapper

		childDeploySuccessful := creationWasSuccessful(
			msgWrapper,
			deployCtx.Rollback,
		)
		if childDeploySuccessful {
			// Update the ephemeral deploy state before persisting
			// the status update with the state container
			// to make sure deployment state is consistent
			// as the deploy state will be used across multiple goroutines
			// to determine the next elements to deploy.
			deployCtx.State.SetCreatedElement(element)
		}

		err := children.Attach(
			ctx,
			msg.ParentInstanceID,
			msg.ChildInstanceID,
			msg.ChildName,
		)
		if err != nil {
			return err
		}

		dependencies := deployCtx.State.GetElementDependencies(element)
		err = children.SaveDependencies(
			ctx,
			msg.ParentInstanceID,
			msg.ChildName,
			dependencies,
		)
		if err != nil {
			return err
		}

		if childDeploySuccessful {
			c.deployNextElementsAfterChild(
				ctx,
				msg.ParentInstanceID,
				deployCtx,
				element,
			)
		}
	}

	return nil
}

func (c *defaultBlueprintContainer) handleLinkUpdateMessage(
	ctx context.Context,
	instanceID string,
	msg LinkDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
) error {
	if msg.InstanceID != instanceID {
		// If message is for a child blueprint, pass through to the client
		// to ensure updates within the child blueprint are surfaced.
		// This allows for the client to provide more detailed feedback to the user
		// for the progress within a child blueprint.
		deployCtx.Channels.LinkUpdateChan <- msg
		return nil
	}

	elementName := linkElementID(msg.LinkName)

	if isLinkDestroyEvent(msg.Status, deployCtx.Rollback) {
		return c.handleLinkDestroyEvent(ctx, msg, deployCtx, finished, elementName)
	}

	if isLinkUpdateEvent(msg.Status, deployCtx.Rollback) {
		return c.handleLinkUpdateEvent(ctx, msg, deployCtx, finished, elementName)
	}

	if isLinkCreationEvent(msg.Status, deployCtx.Rollback) {
		return c.handleLinkCreationEvent(ctx, msg, deployCtx, finished, elementName)
	}

	return nil
}

func (c *defaultBlueprintContainer) handleLinkUpdateEvent(
	ctx context.Context,
	msg LinkDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	elementName string,
) error {
	links := c.stateContainer.Links()
	element := &LinkIDInfo{
		LinkID:   msg.LinkID,
		LinkName: msg.LinkName,
	}

	if startedUpdatingLink(msg.Status, deployCtx.Rollback) {
		deployCtx.State.SetElementInProgress(element)
		updateTimestamp := int(msg.UpdateTimestamp)
		err := links.UpdateStatus(
			ctx,
			msg.InstanceID,
			msg.LinkID,
			state.LinkStatusInfo{
				Status:                    msg.Status,
				PreciseStatus:             msg.PreciseStatus,
				LastStatusUpdateTimestamp: &updateTimestamp,
			},
		)
		if err != nil {
			return err
		}
	}

	// This will not persist the current status update if the message
	// represents a failure that can be retried.
	// The initiator of the deployment process will receive failure messages
	// that can be retried so that the end user can be informed when
	// a link update is taking longer due to a failure that can be retried.
	// For historical purposes, how many attempts have been made to update a link
	// will be persisted under the durations section of the link state.
	if finishedUpdatingLink(msg, deployCtx.Rollback) {
		msgWrapper := &deployUpdateMessageWrapper{
			linkUpdateMessage: &msg,
		}
		finished[elementName] = msgWrapper

		linkUpdateSuccessful := updateWasSuccessful(
			msgWrapper,
			deployCtx.Rollback,
		)
		if linkUpdateSuccessful {
			// Update the ephemeral deploy state before persisting
			// the status update with the state container
			// to make sure deployment state is consistent
			// as the deploy state will be used across multiple goroutines
			// to determine the next elements to deploy.
			deployCtx.State.SetUpdatedElement(element)
		}

		// Instead of just updating the status, ensure that the link data
		// and intermediary resource states are also persisted.
		return links.Save(
			ctx,
			msg.InstanceID,
			c.buildLinkState(msg, deployCtx),
		)
	}

	return nil
}

func (c *defaultBlueprintContainer) buildLinkState(
	msg LinkDeployUpdateMessage,
	deployCtx *DeployContext,
) state.LinkState {
	linkDeployResult := deployCtx.State.GetLinkDeployResult(msg.LinkName)
	linkState := state.LinkState{
		LinkID:                     msg.LinkID,
		LinkName:                   msg.LinkName,
		InstanceID:                 msg.InstanceID,
		Status:                     msg.Status,
		PreciseStatus:              msg.PreciseStatus,
		Durations:                  msg.Durations,
		FailureReasons:             msg.FailureReasons,
		LastStatusUpdateTimestamp:  int(msg.UpdateTimestamp),
		LastDeployAttemptTimestamp: int(c.clock.Now().Unix()),
	}

	wrappedMsg := &deployUpdateMessageWrapper{
		linkUpdateMessage: &msg,
	}
	successfulUpdate := updateWasSuccessful(
		wrappedMsg,
		deployCtx.Rollback,
	)
	successfulCreation := creationWasSuccessful(
		wrappedMsg,
		deployCtx.Rollback,
	)

	if successfulUpdate || successfulCreation {
		if linkDeployResult != nil {
			if linkDeployResult.LinkData != nil {
				linkState.LinkData = linkDeployResult.LinkData.Fields
			}
			linkState.IntermediaryResourceStates = linkDeployResult.IntermediaryResourceStates
		}

		linkState.LastDeployedTimestamp = int(c.clock.Now().Unix())
	}

	return linkState
}

func (c *defaultBlueprintContainer) handleLinkCreationEvent(
	ctx context.Context,
	msg LinkDeployUpdateMessage,
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	elementName string,
) error {
	links := c.stateContainer.Links()
	element := &LinkIDInfo{
		LinkID:   msg.LinkID,
		LinkName: msg.LinkName,
	}

	if startedCreatingLink(msg.Status, deployCtx.Rollback) {
		deployCtx.State.SetElementInProgress(element)
		updateTimestamp := int(msg.UpdateTimestamp)
		err := links.UpdateStatus(
			ctx,
			msg.InstanceID,
			msg.LinkID,
			state.LinkStatusInfo{
				Status:                    msg.Status,
				PreciseStatus:             msg.PreciseStatus,
				LastStatusUpdateTimestamp: &updateTimestamp,
			},
		)
		if err != nil {
			return err
		}
	}

	// This will not persist the current status update if the message
	// represents a failure that can be retried.
	// The initiator of the deployment process will receive failure messages
	// that can be retried so that the end user can be informed when
	// a link creation is taking longer due to a failure that can be retried.
	// For historical purposes, how many attempts have been made to create a link
	// will be persisted under the durations section of the link state.
	if finishedCreatingLink(msg, deployCtx.Rollback) {
		msgWrapper := &deployUpdateMessageWrapper{
			linkUpdateMessage: &msg,
		}
		finished[elementName] = msgWrapper

		linkCreationSuccessful := creationWasSuccessful(
			msgWrapper,
			deployCtx.Rollback,
		)
		if linkCreationSuccessful {
			// Update the ephemeral deploy state before persisting
			// the status update with the state container
			// to make sure deployment state is consistent
			// as the deploy state will be used across multiple goroutines
			// to determine the next elements to deploy.
			deployCtx.State.SetCreatedElement(element)
		}

		// Instead of just updating the status, ensure that the link data
		// and intermediary resource states are also persisted.
		return links.Save(
			ctx,
			msg.InstanceID,
			c.buildLinkState(msg, deployCtx),
		)
	}

	return nil
}

func (c *defaultBlueprintContainer) createDeploymentFinishedMessage(
	instanceID string,
	status core.InstanceStatus,
	failureReasons []string,
	elapsedTime time.Duration,
	prepareElapsedTime *time.Duration,
) DeploymentFinishedMessage {
	elapsedMilliseconds := core.FractionalMilliseconds(elapsedTime)
	currentTimestamp := c.clock.Now().Unix()
	msg := DeploymentFinishedMessage{
		InstanceID:      instanceID,
		Status:          status,
		FailureReasons:  failureReasons,
		FinishTimestamp: currentTimestamp,
		UpdateTimestamp: currentTimestamp,
		Durations: &state.InstanceCompletionDuration{
			TotalDuration: &elapsedMilliseconds,
		},
	}

	if prepareElapsedTime != nil {
		prepareEllapsedMilliseconds := core.FractionalMilliseconds(*prepareElapsedTime)
		msg.Durations.PrepareDuration = &prepareEllapsedMilliseconds
	}

	return msg
}

type deployUpdateMessageWrapper struct {
	resourceUpdateMessage *ResourceDeployUpdateMessage
	linkUpdateMessage     *LinkDeployUpdateMessage
	childUpdateMessage    *ChildDeployUpdateMessage
}

type retryInfo struct {
	attempt            int
	exceededMaxRetries bool
	policy             *provider.RetryPolicy
	attemptDurations   []float64
}

type linkUpdateResourceInfo struct {
	failureReasons []string
	input          *provider.LinkUpdateResourceInput
}

type linkUpdateIntermediaryResourcesInfo struct {
	failureReasons []string
	input          *provider.LinkUpdateIntermediaryResourcesInput
}

type deploymentElementInfo struct {
	element    state.Element
	instanceID string
}

type resourceDeployInfo struct {
	instanceID   string
	resourceID   string
	resourceName string
	resourceImpl provider.Resource
	changes      *provider.Changes
	isNew        bool
}

// DeployChannels contains all the channels required to stream
// deployment events.
type DeployChannels struct {
	// ResourceUpdateChan receives messages about the status of deployment for resources.
	ResourceUpdateChan chan ResourceDeployUpdateMessage
	// LinkUpdateChan receives messages about the status of deployment for links.
	LinkUpdateChan chan LinkDeployUpdateMessage
	// ChildUpdateChan receives messages about the status of deployment for child blueprints.
	ChildUpdateChan chan ChildDeployUpdateMessage
	// DeploymentUpdateChan receives messages about the status of the blueprint instance deployment.
	DeploymentUpdateChan chan DeploymentUpdateMessage
	// FinishChan is used to signal that the blueprint instance deployment has finished,
	// the message will contain the final status of the deployment.
	FinishChan chan DeploymentFinishedMessage
	// ErrChan is used to signal that an unexpected error occurred during deployment of changes.
	ErrChan chan error
}
