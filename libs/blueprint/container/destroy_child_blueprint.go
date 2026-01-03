package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ChildBlueprintDestroyer provides an interface for a service that destroys a child
// blueprint as a part of the deployment process for a blueprint instance.
type ChildBlueprintDestroyer interface {
	Destroy(
		ctx context.Context,
		childBlueprintElement state.Element,
		parentInstanceID string,
		parentInstanceTreePath string,
		includeTreePath string,
		blueprintDestroyer BlueprintDestroyer,
		deployCtx *DeployContext,
	)
}

// BlueprintDestroyer provides an interface for a service that will be used to destroy
// a blueprint instance.
// This is primarily useful for destroying child blueprints as part of the deployment
// process for a blueprint instance.
type BlueprintDestroyer interface {
	Destroy(
		ctx context.Context,
		input *DestroyInput,
		channels *DeployChannels,
		paramOverrides core.BlueprintParams,
	)
}

// NewDefaultChildBlueprintDestroyer creates a new instance of the default implementation
// of the service that destroys a child blueprint as a part of the deployment process
// for a blueprint instance.
func NewDefaultChildBlueprintDestroyer() ChildBlueprintDestroyer {
	return &defaultChildBlueprintDestroyer{}
}

type defaultChildBlueprintDestroyer struct{}

func (d *defaultChildBlueprintDestroyer) Destroy(
	ctx context.Context,
	childBlueprintElement state.Element,
	parentInstanceID string,
	parentInstanceTreePath string,
	includeTreePath string,
	blueprintDestroyer BlueprintDestroyer,
	deployCtx *DeployContext,
) {
	childState := getChildStateByName(deployCtx.InstanceStateSnapshot, childBlueprintElement.LogicalName())
	if childState == nil {
		deployCtx.Channels.ErrChan <- errChildNotFoundInState(
			childBlueprintElement.LogicalName(),
			parentInstanceID,
		)
		return
	}
	destroyChildChanges := getOrCreateChildDestroyChanges(
		deployCtx.InputChanges,
		childBlueprintElement.LogicalName(),
		childState,
	)

	childParams := deployCtx.ParamOverrides.
		WithContextVariables(
			createContextVarsForChildBlueprint(
				parentInstanceID,
				parentInstanceTreePath,
				includeTreePath,
				// Empty blueprint dir for destroy operations - child blueprints
				// don't need to be resolved during destruction.
				"",
			),
			/* keepExisting */ true,
		)

	// Create an intermediary set of channels so we can dispatch child blueprint-wide
	// events to the parent blueprint's channels.
	// Resource and link events will be passed through to be surfaced to the user,
	// trusting that they wil be handled within the Destroy call for the child blueprint.
	childChannels := CreateDeployChannels()
	// The blueprint destroyer is not expected to make use of the loaded blueprint spec directly.
	// For this reason, we don't need to load an entirely new container
	// for destroying a child blueprint instance.
	// Destroy is expected to rely purely on the provided blueprint changes and the current state
	// of the instance persisted in the state container.
	blueprintDestroyer.Destroy(
		ctx,
		&DestroyInput{
			InstanceID:             childBlueprintElement.ID(),
			Changes:                destroyChildChanges,
			Rollback:               deployCtx.Rollback,
			TaggingConfig:          deployCtx.TaggingConfig,
			ProviderMetadataLookup: deployCtx.ProviderMetadataLookup,
			DrainTimeout:           deployCtx.DrainTimeout,
		},
		childChannels,
		childParams,
	)

	finished := false
	ctxCancelled := false

	for !finished {
		// If context was cancelled and there's no drain deadline, exit immediately.
		// Otherwise, continue forwarding events until child finishes or drain deadline fires.
		if ctxCancelled && deployCtx.DrainDeadline == nil {
			return
		}

		select {
		case <-ctx.Done():
			// Mark context as cancelled but continue loop to check drain deadline
			// and forward any remaining events from the child.
			ctxCancelled = true

		case <-deployCtx.DrainDeadline:
			// Parent's drain deadline reached - exit and let the parent's
			// markInFlightElementsAsInterrupted handle sending INTERRUPTED for this child.
			return

		case msg := <-childChannels.DeploymentUpdateChan:
			deployCtx.Channels.ChildUpdateChan <- updateToChildUpdateMessage(
				&msg,
				parentInstanceID,
				childBlueprintElement,
				deployCtx.CurrentGroupIndex,
			)
		case msg := <-childChannels.FinishChan:
			deployCtx.Channels.ChildUpdateChan <- finishedToChildUpdateMessage(
				&msg,
				parentInstanceID,
				childBlueprintElement,
				deployCtx.CurrentGroupIndex,
			)
			finished = true
		case msg := <-childChannels.ResourceUpdateChan:
			deployCtx.Channels.ResourceUpdateChan <- msg
		case msg := <-childChannels.LinkUpdateChan:
			deployCtx.Channels.LinkUpdateChan <- msg
		case msg := <-childChannels.ChildUpdateChan:
			deployCtx.Channels.ChildUpdateChan <- msg
		case err := <-childChannels.ErrChan:
			deployCtx.Channels.ErrChan <- err
		}
	}
}

// getOrCreateChildDestroyChanges returns the child's changes from InputChanges if available,
// otherwise creates destroy changes from the child's state.
// This ensures that during rollback, filtered child changes (which exclude resources/links
// in failed states) are used instead of recreating changes that would include all items.
func getOrCreateChildDestroyChanges(
	inputChanges *changes.BlueprintChanges,
	childName string,
	childState *state.InstanceState,
) *changes.BlueprintChanges {
	if inputChanges != nil && inputChanges.ChildChanges != nil {
		if childChanges, ok := inputChanges.ChildChanges[childName]; ok {
			return &childChanges
		}
	}
	return createDestroyChangesFromChildState(childState)
}
