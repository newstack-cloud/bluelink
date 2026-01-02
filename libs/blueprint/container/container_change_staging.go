package container

import (
	"context"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

func (c *defaultBlueprintContainer) StageChanges(
	ctx context.Context,
	input *StageChangesInput,
	channels *ChangeStagingChannels,
	paramOverrides core.BlueprintParams,
) error {
	resolvedInstanceID, err := c.getInstanceID(ctx, input.InstanceID, input.InstanceName)
	if err != nil {
		return err
	}

	ctxWithInstanceID := context.WithValue(ctx, core.BlueprintInstanceIDKey, resolvedInstanceID)
	changeStagingLogger := c.logger.Named("stageChanges").WithFields(
		core.StringLogField("instanceID", resolvedInstanceID),
		core.StringLogField("instanceName", input.InstanceName),
	)
	instanceTreePath := getInstanceTreePath(paramOverrides, resolvedInstanceID)
	if exceedsMaxDepth(instanceTreePath, MaxBlueprintDepth) {
		changeStagingLogger.Debug("max nested blueprint depth exceeded")
		return errMaxBlueprintDepthExceeded(
			instanceTreePath,
			MaxBlueprintDepth,
		)
	}

	if input.Destroy {
		// For destroy operations, we must have a valid instance to destroy.
		// If no instance was resolved, return an error with the identifier
		// the user provided for a more helpful error message.
		if resolvedInstanceID == "" {
			identifier := input.InstanceID
			if identifier == "" {
				identifier = input.InstanceName
			}
			return state.InstanceNotFoundError(identifier)
		}
		changeStagingLogger.Info("staging changes for destroying blueprint instance")
		go c.stageInstanceRemoval(ctxWithInstanceID, resolvedInstanceID, channels)
		return nil
	}

	changeStagingLogger.Info(
		"preparing blueprint (expanding templates, applying resource conditions etc.) for change staging",
	)
	prepareResult, err := c.blueprintPreparer.Prepare(
		ctxWithInstanceID,
		c.spec.Schema(),
		subengine.ResolveForChangeStaging,
		/* changes */ nil,
		c.linkInfo,
		paramOverrides,
	)
	if err != nil {
		return err
	}

	go c.stageChanges(
		ctxWithInstanceID,
		resolvedInstanceID,
		prepareResult.ParallelGroups,
		paramOverrides,
		prepareResult.ResourceProviderMap,
		prepareResult.BlueprintContainer.BlueprintSpec().Schema(),
		channels,
		changeStagingLogger,
	)

	return nil
}

func (c *defaultBlueprintContainer) stageChanges(
	ctx context.Context,
	instanceID string,
	parallelGroups [][]*DeploymentNode,
	paramOverrides core.BlueprintParams,
	resourceProviders map[string]provider.Provider,
	blueprint *schema.Blueprint,
	channels *ChangeStagingChannels,
	changeStagingLogger core.Logger,
) {
	state := c.createChangeStagingState()
	resourceChangesChan := make(chan ResourceChangesMessage)
	childChangesChan := make(chan ChildChangesMessage)
	linkChangesChan := make(chan LinkChangesMessage)
	errChan := make(chan error)

	internalChannels := &ChangeStagingChannels{
		ResourceChangesChan: resourceChangesChan,
		ChildChangesChan:    childChangesChan,
		LinkChangesChan:     linkChangesChan,
		ErrChan:             errChan,
	}

	// Check for all the removed resources, links and child blueprints.
	// All removals will be handled before the groups of new elements and element
	// updates are staged.
	// Staging state changes will be applied synchronously for all resources, links and child blueprints
	// that have been removed in the source blueprint being staged for deployment.
	// A message is dispatched to the external channels for each removal so that the caller
	// can gather and display removals in the same way as other changes.
	changeStagingLogger.Info("staging removals for resources, links and child blueprints")
	err := c.stageRemovals(ctx, instanceID, state, parallelGroups, channels)
	if err != nil {
		changeStagingLogger.Debug("error staging removals", core.ErrorLogField("error", err))
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}

	for _, group := range parallelGroups {
		c.stageGroupChanges(
			ctx,
			instanceID,
			state,
			group,
			paramOverrides,
			resourceProviders,
			internalChannels,
			changeStagingLogger,
		)

		err := c.listenToAndProcessGroupChanges(
			ctx,
			group,
			internalChannels,
			channels,
			state,
		)
		if err != nil {
			channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
			return
		}
	}

	err = c.resolveAndCollectExportChanges(ctx, instanceID, blueprint, state)
	if err != nil {
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}

	err = c.resolveAndCollectMetadataChanges(ctx, instanceID, blueprint, state)
	if err != nil {
		channels.ErrChan <- wrapErrorForChildContext(err, paramOverrides)
		return
	}

	channels.CompleteChan <- state.ExtractBlueprintChanges()
}

func (c *defaultBlueprintContainer) listenToAndProcessGroupChanges(
	ctx context.Context,
	group []*DeploymentNode,
	internalChannels *ChangeStagingChannels,
	externalChannels *ChangeStagingChannels,
	state ChangeStagingState,
) error {
	// The criteria to move on to the next group is the following:
	// - All resources in the group current have been processed.
	// - All child blueprints in the current group have been processed.
	// - All links that were previously pending completion and waiting on the
	//    resources in the current group have been processed.
	expectedLinkChangesCount := state.CountPendingLinksForGroup(group) +
		// We need to account for soft links where two resources that are linked can be deployed
		// at the same time.
		countPendingLinksContainedInGroup(group)

	linkChangesCount := 0
	collected := map[string]*changesWrapper{}
	var err error
	waitingForLinkChanges := expectedLinkChangesCount > 0
	for (len(collected) < len(group) || waitingForLinkChanges) &&
		err == nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case changes := <-internalChannels.ResourceChangesChan:
			elementName := core.ResourceElementID(changes.ResourceName)
			collected[elementName] = &changesWrapper{
				resourceChanges: &changes.Changes,
			}
			externalChannels.ResourceChangesChan <- changes
		case changes := <-internalChannels.LinkChangesChan:
			state.ApplyLinkChanges(changes)
			linkChangesCount += 1
			externalChannels.LinkChangesChan <- changes
		case changes := <-internalChannels.ChildChangesChan:
			elementName := core.ChildElementID(changes.ChildBlueprintName)
			collected[elementName] = &changesWrapper{
				childChanges: &changes.Changes,
			}
			state.ApplyChildChanges(changes)
			externalChannels.ChildChangesChan <- changes
		case err = <-internalChannels.ErrChan:
		}

		waitingForLinkChanges = expectedLinkChangesCount > 0 && linkChangesCount < expectedLinkChangesCount
	}

	return err
}

func countPendingLinksContainedInGroup(group []*DeploymentNode) int {
	count := 0
	for _, node := range group {
		if node.Type() == DeploymentNodeTypeResource {
			for _, otherNode := range node.ChainLinkNode.LinksTo {
				if groupContainsResourceLinkNode(group, otherNode) {
					count += 1
				}
			}
		}
	}

	return count
}

func groupContainsResourceLinkNode(group []*DeploymentNode, resourceLinkNode *links.ChainLinkNode) bool {
	return slices.ContainsFunc(group, func(compareWith *DeploymentNode) bool {
		return compareWith.Type() == DeploymentNodeTypeResource &&
			compareWith.ChainLinkNode.ResourceName == resourceLinkNode.ResourceName
	})
}

func (c *defaultBlueprintContainer) stageGroupChanges(
	ctx context.Context,
	instanceID string,
	stagingState ChangeStagingState,
	group []*DeploymentNode,
	paramOverrides core.BlueprintParams,
	resourceProviders map[string]provider.Provider,
	channels *ChangeStagingChannels,
	changeStagingLogger core.Logger,
) {
	instanceTreePath := getInstanceTreePath(paramOverrides, instanceID)

	for _, node := range group {
		changeStagingLogger.Info(
			"staging changes for element",
			core.StringLogField("element", node.Name()),
		)
		nodeLogger := changeStagingLogger.Named("element").WithFields(
			core.StringLogField("elementName", node.Name()),
		)
		if node.Type() == DeploymentNodeTypeResource {
			go c.changeStager.StageChanges(
				ctx,
				instanceID,
				stagingState,
				node.ChainLinkNode,
				channels,
				resourceProviders,
				paramOverrides,
				nodeLogger,
			)
		} else if node.Type() == DeploymentNodeTypeChild {
			includeTreePath := getIncludeTreePath(paramOverrides, node.Name())
			go c.childChangeStager.StageChanges(
				ctx,
				&ChildInstanceInfo{
					ParentInstanceID:       instanceID,
					ParentInstanceTreePath: instanceTreePath,
					IncludeTreePath:        includeTreePath,
				},
				node.ChildNode,
				paramOverrides,
				channels,
				nodeLogger,
			)
		}
	}
}

func (c *defaultBlueprintContainer) stageRemovals(
	ctx context.Context,
	instanceID string,
	stagingState ChangeStagingState,
	// Use the grouped deployment nodes to compare with the current instance
	// state.
	// c.spec.Schema() must NOT be used at this stage as it does not contain
	// the expanded representation of blueprints that contain resource
	// templates.
	deploymentNodes [][]*DeploymentNode,
	channels *ChangeStagingChannels,
) error {
	instances := c.stateContainer.Instances()
	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		if !state.IsInstanceNotFound(err) {
			return err
		}
	}

	flattenedNodes := core.Flatten(deploymentNodes)

	c.stageResourceRemovals(&instanceState, stagingState, flattenedNodes, channels)
	c.stageLinkRemovals(&instanceState, stagingState, flattenedNodes, channels)
	c.stageChildRemovals(&instanceState, stagingState, flattenedNodes, channels)

	return nil
}

func (c *defaultBlueprintContainer) stageResourceRemovals(
	instanceState *state.InstanceState,
	stagingState ChangeStagingState,
	flattenedNodes []*DeploymentNode,
	channels *ChangeStagingChannels,
) {
	for _, resourceState := range instanceState.Resources {
		inDeployNodes := slices.ContainsFunc(flattenedNodes, func(node *DeploymentNode) bool {
			return node.ChainLinkNode != nil &&
				node.ChainLinkNode.ResourceName == resourceState.Name
		})
		if !inDeployNodes {
			dependents := findDependents(
				resourceState,
				flattenedNodes,
				instanceState,
			)
			stagingState.AddElementsThatMustBeRecreated(
				dependents,
			)
			changes := ResourceChangesMessage{
				ResourceName: resourceState.Name,
				Removed:      true,
			}
			stagingState.ApplyResourceChanges(changes)
			channels.ResourceChangesChan <- changes
		}
	}
}

func (c *defaultBlueprintContainer) stageLinkRemovals(
	instanceState *state.InstanceState,
	stagingState ChangeStagingState,
	flattenedNodes []*DeploymentNode,
	channels *ChangeStagingChannels,
) {
	for linkName := range instanceState.Links {
		// Links are stored as a map of "resourceAName::resourceBName"
		// so we need to split the link name to get the resource names.
		linkParts := strings.Split(linkName, "::")
		resourceAName := linkParts[0]
		resourceBName := linkParts[1]

		inDeployNodes := slices.ContainsFunc(flattenedNodes, func(node *DeploymentNode) bool {
			return node.ChainLinkNode != nil &&
				(node.ChainLinkNode.ResourceName == resourceAName ||
					node.ChainLinkNode.ResourceName == resourceBName)
		})
		if !inDeployNodes {
			changes := LinkChangesMessage{
				ResourceAName: resourceAName,
				ResourceBName: resourceBName,
				Removed:       true,
			}
			stagingState.ApplyLinkChanges(changes)
			channels.LinkChangesChan <- changes
		}
	}
}

func (c *defaultBlueprintContainer) stageChildRemovals(
	instanceState *state.InstanceState,
	stagingState ChangeStagingState,
	flattenedNodes []*DeploymentNode,
	channels *ChangeStagingChannels,
) {
	for childName, childState := range instanceState.ChildBlueprints {
		childElementID := core.ChildElementID(childName)
		inDeployNodes := slices.ContainsFunc(flattenedNodes, func(node *DeploymentNode) bool {
			return node.ChildNode != nil &&
				node.ChildNode.ElementName == childElementID
		})
		if !inDeployNodes {
			dependents := findDependents(
				state.WrapChildBlueprintInstance(childName, childState),
				flattenedNodes,
				instanceState,
			)
			stagingState.AddElementsThatMustBeRecreated(
				dependents,
			)

			changes := ChildChangesMessage{
				ChildBlueprintName: childName,
				Removed:            true,
			}
			stagingState.ApplyChildChanges(changes)
			channels.ChildChangesChan <- changes
		}
	}
}

func (c *defaultBlueprintContainer) resolveAndCollectExportChanges(
	ctx context.Context,
	instanceID string,
	blueprint *schema.Blueprint,
	stagingState ChangeStagingState,
) error {

	if blueprint.Exports == nil {
		return nil
	}

	resolvedExports := map[string]*subengine.ResolveResult{}
	for exportName, export := range blueprint.Exports.Values {
		resolvedExport, err := c.resolveExport(
			ctx,
			exportName,
			export,
			subengine.ResolveForChangeStaging,
		)
		if err != nil {
			return err
		}

		if resolvedExport != nil {
			resolvedExports[exportName] = resolvedExport
		}
	}

	exports := c.stateContainer.Exports()
	blueprintExportsState, err := exports.GetAll(ctx, instanceID)
	if err != nil {
		if !state.IsInstanceNotFound(err) {
			return err
		}
	}
	// Collect export changes in a temporary structure to avoid locking the staging state
	// for the entire duration of the operation.
	collectedExportChanges := &changes.IntermediaryBlueprintChanges{
		NewExports:       map[string]*provider.FieldChange{},
		ExportChanges:    map[string]*provider.FieldChange{},
		RemovedExports:   []string{},
		UnchangedExports: []string{},
		ResolveOnDeploy:  []string{},
	}
	collectExportChanges(collectedExportChanges, blueprint.Exports.Values, resolvedExports, blueprintExportsState)
	stagingState.UpdateExportChanges(collectedExportChanges)

	return nil
}

func (c *defaultBlueprintContainer) resolveAndCollectMetadataChanges(
	ctx context.Context,
	instanceID string,
	blueprint *schema.Blueprint,
	stagingState ChangeStagingState,
) error {
	if blueprint.Metadata == nil {
		// Still need to set empty metadata changes to avoid nil dereference
		// in ExtractBlueprintChanges
		emptyChanges := changes.MetadataChanges{
			NewFields:       []provider.FieldChange{},
			ModifiedFields:  []provider.FieldChange{},
			RemovedFields:   []string{},
			UnchangedFields: []string{},
		}
		stagingState.UpdateMetadataChanges(&emptyChanges, nil)
		return nil
	}

	result, err := c.substitutionResolver.ResolveInMappingNode(
		ctx,
		"metadata",
		blueprint.Metadata,
		&subengine.ResolveMappingNodeTargetInfo{
			ResolveFor: subengine.ResolveForChangeStaging,
		},
	)
	if err != nil {
		return err
	}

	metadata := c.stateContainer.Metadata()
	blueprintMetadataState, err := metadata.Get(ctx, instanceID)
	if err != nil {
		if !state.IsInstanceNotFound(err) {
			return err
		}
	}
	metadataChanges := changes.MetadataChanges{
		NewFields:       []provider.FieldChange{},
		ModifiedFields:  []provider.FieldChange{},
		RemovedFields:   []string{},
		UnchangedFields: []string{},
	}
	resolveOnDeploy := commoncore.Map(
		result.ResolveOnDeploy,
		func(fieldPath string, _ int) string {
			return substitutions.RenderFieldPath("metadata", fieldPath)
		},
	)
	changes.CollectMetadataChanges(&metadataChanges, result, blueprintMetadataState)
	stagingState.UpdateMetadataChanges(&metadataChanges, resolveOnDeploy)

	return nil
}

func (c *defaultBlueprintContainer) stageInstanceRemoval(
	ctx context.Context,
	instanceID string,
	channels *ChangeStagingChannels,
) {

	instances := c.stateContainer.Instances()
	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		channels.ErrChan <- err
		return
	}

	changes := getInstanceRemovalChanges(&instanceState)

	// For staging changes for destroying an instance, we don't need to individually
	// dispatch resource, link, and child changes. We can just send the complete
	// set of changes to the complete channel.
	channels.CompleteChan <- changes
}

// ChangeStagingChannels contains all the channels required to stream
// change staging events.
type ChangeStagingChannels struct {
	// ResourceChangesChan receives change sets for each resource in the blueprint.
	ResourceChangesChan chan ResourceChangesMessage
	// ChildChangesChan receives change sets for child blueprints once all
	// changes for the child blueprint have been staged.
	ChildChangesChan chan ChildChangesMessage
	// LinkChangesChan receives change sets for links between resources.
	LinkChangesChan chan LinkChangesMessage
	// CompleteChan is used to signal that all changes have been staged
	// containing the full set of changes that will be made to the blueprint instance
	// when deploying the changes.
	CompleteChan chan changes.BlueprintChanges
	// ErrChan is used to signal that an error occurred while staging changes.
	ErrChan chan error
}

// ResourceChangesMessage provides a message containing the changes
// that will be made to a resource in a blueprint instance.
type ResourceChangesMessage struct {
	ResourceName    string           `json:"resourceName"`
	Removed         bool             `json:"removed"`
	New             bool             `json:"new"`
	Changes         provider.Changes `json:"changes"`
	ResolveOnDeploy []string         `json:"resolveOnDeploy"`
	// ConditionKnownOnDeploy is used to indicate that the condition for the resource
	// can not be resolved until the blueprint is deployed.
	// This means the changes described in this message may not be applied
	// if the condition evaluates to false when the blueprint is deployed.
	ConditionKnownOnDeploy bool `json:"conditionKnownOnDeploy"`
}

// ChildChangesMessage provides a message containing the changes
// that will be made to a child blueprint in a blueprint instance.
type ChildChangesMessage struct {
	ChildBlueprintName string                   `json:"childBlueprintName"`
	Removed            bool                     `json:"removed"`
	New                bool                     `json:"new"`
	Changes            changes.BlueprintChanges `json:"changes"`
}

// LinkChangesMessage provides a message containing the changes
// that will be made to a link between resources in a blueprint instance.
type LinkChangesMessage struct {
	ResourceAName string               `json:"resourceAName"`
	ResourceBName string               `json:"resourceBName"`
	Removed       bool                 `json:"removed"`
	New           bool                 `json:"new"`
	Changes       provider.LinkChanges `json:"changes"`
}

type stageResourceChangeInfo struct {
	node       *links.ChainLinkNode
	instanceID string
	resourceID string
}

type changesWrapper struct {
	resourceChanges *provider.Changes
	childChanges    *changes.BlueprintChanges
}
