package container

import (
	"context"

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
	destroyChildChanges := createDestroyChangesFromChildState(childState)

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
			InstanceID: childBlueprintElement.ID(),
			Changes:    destroyChildChanges,
			Rollback:   deployCtx.Rollback,
		},
		childChannels,
		childParams,
	)

	finished := false
	var err error
	for !finished && err == nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
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
		case err = <-childChannels.ErrChan:
			deployCtx.Channels.ErrChan <- err
		}
	}
}
