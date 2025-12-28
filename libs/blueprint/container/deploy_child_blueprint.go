package container

import (
	"context"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

// ChildBlueprintDeployer provides an interface for a service that deploys a child
// blueprint as a part of the deployment process for a blueprint instance.
type ChildBlueprintDeployer interface {
	Deploy(
		ctx context.Context,
		parentInstanceID string,
		parentInstanceTreePath string,
		includeTreePath string,
		childNode *refgraph.ReferenceChainNode,
		changes *changes.BlueprintChanges,
		deployCtx *DeployContext,
	)
}

// IncludeSubstitutionResolver provides an interface for a service that
// is responsible for resolving substitutions in an include definition.
type IncludeSubstitutionResolver interface {
	// ResolveInResource resolves substitutions in an include.
	ResolveInInclude(
		ctx context.Context,
		includeName string,
		include *schema.Include,
		resolveTargetInfo *subengine.ResolveIncludeTargetInfo,
	) (*subengine.ResolveInIncludeResult, error)
}

// NewDefaultChildBlueprintDeployer creates a new instance of the default implementation
// of the service that deploys a child blueprint as a part of the deployment process
// for a blueprint instance.
func NewDefaultChildBlueprintDeployer(
	substitutionResolver IncludeSubstitutionResolver,
	childResolver includes.ChildResolver,
	createChildBlueprintLoader ChildBlueprintLoaderFactory,
	stateContainer state.Container,
) ChildBlueprintDeployer {
	return &defaultChildBlueprintDeployer{
		substitutionResolver:       substitutionResolver,
		childResolver:              childResolver,
		createChildBlueprintLoader: createChildBlueprintLoader,
		stateContainer:             stateContainer,
	}
}

type defaultChildBlueprintDeployer struct {
	substitutionResolver       IncludeSubstitutionResolver
	childResolver              includes.ChildResolver
	createChildBlueprintLoader ChildBlueprintLoaderFactory
	stateContainer             state.Container
}

func (d *defaultChildBlueprintDeployer) Deploy(
	ctx context.Context,
	parentInstanceID string,
	parentInstanceTreePath string,
	includeTreePath string,
	childNode *refgraph.ReferenceChainNode,
	changes *changes.BlueprintChanges,
	deployCtx *DeployContext,
) {
	loadResult, err := loadChildBlueprint(
		ctx,
		&childBlueprintLoadInput{
			parentInstanceID:       parentInstanceID,
			parentInstanceTreePath: parentInstanceTreePath,
			instanceTreePath:       childNode.ElementName,
			includeTreePath:        includeTreePath,
			node:                   childNode,
			resolveFor:             subengine.ResolveForDeployment,
			logger:                 deployCtx.Logger,
		},
		d.substitutionResolver,
		d.childResolver,
		d.createChildBlueprintLoader,
		d.stateContainer,
		deployCtx.ParamOverrides,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	childInstanceName, err := d.resolveChildInstanceName(
		ctx,
		loadResult,
		parentInstanceID,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	childChannels := &DeployChannels{
		ResourceUpdateChan:   make(chan ResourceDeployUpdateMessage),
		LinkUpdateChan:       make(chan LinkDeployUpdateMessage),
		ChildUpdateChan:      make(chan ChildDeployUpdateMessage),
		DeploymentUpdateChan: make(chan DeploymentUpdateMessage),
		FinishChan:           make(chan DeploymentFinishedMessage),
		ErrChan:              make(chan error),
	}
	err = loadResult.childContainer.Deploy(
		ctx,
		&DeployInput{
			InstanceID:   loadResult.childState.InstanceID,
			InstanceName: childInstanceName,
			Changes:      changes,
			Rollback:     deployCtx.Rollback,
		},
		childChannels,
		loadResult.childParams,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	d.waitForChildDeployment(
		ctx,
		parentInstanceID,
		loadResult.includeName,
		loadResult.childState,
		childChannels,
		deployCtx,
	)
}

func (d *defaultChildBlueprintDeployer) resolveChildInstanceName(
	ctx context.Context,
	loadResult *childBlueprintLoadResult,
	parentInstanceID string,
) (string, error) {
	if loadResult.childState.InstanceName != "" {
		return loadResult.childState.InstanceName, nil
	}

	// Load the parent instance to get its name to generate
	// a name for the child based on the parent name and the name
	// of the child include in the parent blueprint.
	parentInstanceState, err := d.stateContainer.
		Instances().
		Get(ctx, parentInstanceID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s-%s",
		parentInstanceState.InstanceName,
		loadResult.includeName,
	), nil
}

func (d *defaultChildBlueprintDeployer) waitForChildDeployment(
	ctx context.Context,
	parentInstanceID string,
	childName string,
	childState *state.InstanceState,
	childChannels *DeployChannels,
	deployCtx *DeployContext,

) {
	childBlueprintElement := &ChildBlueprintIDInfo{
		ChildInstanceID: childState.InstanceID,
		ChildName:       childName,
	}

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
			// Parent's drain deadline reached. Give child a brief grace period
			// to send its finish message so we can capture its actual status
			// (e.g., FAILED) rather than marking it as INTERRUPTED.
			select {
			case msg := <-childChannels.FinishChan:
				deployCtx.Channels.ChildUpdateChan <- finishedToChildUpdateMessage(
					&msg,
					parentInstanceID,
					childBlueprintElement,
					deployCtx.CurrentGroupIndex,
				)
			case <-time.After(100 * time.Millisecond):
				// Child didn't finish in grace period, parent will mark it as interrupted.
			}
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
