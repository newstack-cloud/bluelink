package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinkDestroyer provides an interface for a service
// that destroys a link between two resources.
type LinkDestroyer interface {
	Destroy(
		ctx context.Context,
		element state.Element,
		instanceID string,
		instanceName string,
		deployCtx *DeployContext,
	)
}

// NewDefaultLinkDestroyer creates a new instance of the default
// implementation of the service that destroys a link between
// two resources.
func NewDefaultLinkDestroyer(
	linkDeployer LinkDeployer,
	linkRegistry provider.LinkRegistry,
	defaultRetryPolicy *provider.RetryPolicy,
) LinkDestroyer {
	return &defaultLinkDestroyer{
		linkDeployer:       linkDeployer,
		linkRegistry:       linkRegistry,
		defaultRetryPolicy: defaultRetryPolicy,
	}
}

type defaultLinkDestroyer struct {
	linkDeployer       LinkDeployer
	linkRegistry       provider.LinkRegistry
	defaultRetryPolicy *provider.RetryPolicy
}

func (d *defaultLinkDestroyer) Destroy(
	ctx context.Context,
	element state.Element,
	instanceID string,
	instanceName string,
	deployCtx *DeployContext,
) {
	linkState := getLinkStateByName(
		deployCtx.InstanceStateSnapshot,
		element.LogicalName(),
	)
	if linkState == nil {
		// A link in the removal set with no persisted state was never
		// deployed (e.g. a previous deploy failed before reaching it), so
		// there is nothing to destroy. It is reported as destroyed so the
		// removal process can run to completion for the elements that do
		// have persisted state.
		deployCtx.Logger.Info(
			"skipping destruction for a link with no persisted state",
		)
		d.reportLinkDestroyedWithoutState(element, instanceID, deployCtx)
		return
	}

	if linkResourceStateMissing(element.LogicalName(), deployCtx.InstanceStateSnapshot) {
		// Without the persisted state of both resources in the link, the
		// link plugin implementation can not be resolved and there is no
		// meaningful resource state left to clean up.
		deployCtx.Logger.Info(
			"skipping destruction for a link with a linked resource that has no persisted state",
		)
		d.reportLinkDestroyedWithoutState(element, instanceID, deployCtx)
		return
	}

	deployCtx.Logger.Info("loading link plugin implementation for destruction")
	linkImplementation, err := d.getProviderLinkImplementation(
		ctx,
		element.LogicalName(),
		deployCtx.InstanceStateSnapshot,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	deployCtx.Logger.Info("loading provider retry policy for link destruction")
	retryPolicy, err := getLinkRetryPolicy(
		ctx,
		element.LogicalName(),
		deployCtx.InstanceStateSnapshot,
		d.linkRegistry,
		d.defaultRetryPolicy,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
		return
	}

	err = d.linkDeployer.Deploy(
		ctx,
		element,
		instanceID,
		instanceName,
		provider.LinkUpdateTypeDestroy,
		linkImplementation,
		deployCtx,
		retryPolicy,
	)
	if err != nil {
		deployCtx.Channels.ErrChan <- err
	}
}

func (d *defaultLinkDestroyer) reportLinkDestroyedWithoutState(
	element state.Element,
	instanceID string,
	deployCtx *DeployContext,
) {
	deployCtx.Channels.LinkUpdateChan <- LinkDeployUpdateMessage{
		InstanceID: instanceID,
		LinkID:     element.ID(),
		LinkName:   element.LogicalName(),
		Status: determineLinkOperationSuccessfullyFinishedStatus(
			deployCtx.Rollback,
			provider.LinkUpdateTypeDestroy,
		),
		UpdateTimestamp:  core.SystemClock{}.Now().Unix(),
		MissingFromState: true,
	}
}

func linkResourceStateMissing(linkName string, currentState *state.InstanceState) bool {
	linkDependencyInfo := extractLinkDirectDependencies(linkName)
	if linkDependencyInfo == nil {
		return false
	}
	return getResourceStateByName(currentState, linkDependencyInfo.resourceAName) == nil ||
		getResourceStateByName(currentState, linkDependencyInfo.resourceBName) == nil
}

func (d *defaultLinkDestroyer) getProviderLinkImplementation(
	ctx context.Context,
	linkName string,
	currentState *state.InstanceState,
) (provider.Link, error) {

	resourceTypeA, resourceTypeB, err := getResourceTypesForLink(linkName, currentState)
	if err != nil {
		return nil, err
	}

	return d.linkRegistry.Link(ctx, resourceTypeA, resourceTypeB)
}
