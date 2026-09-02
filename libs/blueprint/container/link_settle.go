package container

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinkSettlePollingConfig configures the wait for a resource to settle after a link has
// written it.
type LinkSettlePollingConfig struct {
	// PollingInterval is how long to wait between checks once the first one has reported
	// the resource is not yet stable.
	PollingInterval time.Duration
	// PollingTimeout bounds the wait for a single link phase.
	PollingTimeout time.Duration
}

// DefaultLinkSettlePollingConfig is a reasonable default for waiting on a resource a link
// has written.
//
// The timeout is deliberately far below DefaultResourceLockTimeout, which is fifteen
// minutes. This wait happens while the link holds the resource's lock, so a settle allowed
// to run to a fifteen minute ceiling would time out every other link waiting on that
// resource and report the failure against them rather than against the resource that
// caused it. Five minutes leaves room above the slowest legitimate settle, such as an AWS Lambda
// function attaching to a VPC for the first time while its network interfaces are
// provisioned, and well below the point where waiters start failing.
var DefaultLinkSettlePollingConfig = &LinkSettlePollingConfig{
	PollingInterval: 5 * time.Second,
	PollingTimeout:  5 * time.Minute,
}

// Waits for a resource a link has just written to reach a stable state, before the link
// releases the resource's lock.
//
// The wait belongs inside the lock. An asynchronous cloud API accepts a change and returns
// while the resource is still applying it, and rejects the next change until it has
// finished. Releasing the lock at the point the call returns hands the resource to the next
// link at exactly the moment it will be refused, which providers have had to work around
// individually with their own backoff.
//
// A provider that has nothing to settle returns true on the first check and pays a single
// plugin round trip for it.
func (d *defaultLinkDeployer) waitForResourceToSettle(
	ctx context.Context,
	resourceInfo *provider.ResourceInfo,
	deployCtx *DeployContext,
) error {
	resourceImpl, resourceData, err := d.resolveSettleTarget(ctx, resourceInfo, deployCtx)
	if err != nil || resourceImpl == nil {
		return err
	}

	settleCtx, cancel := context.WithTimeout(ctx, d.settlePollingConfig.PollingTimeout)
	defer cancel()

	input := &provider.ResourceHasStabilisedInput{
		InstanceID:   resourceInfo.InstanceID,
		ResourceID:   resourceInfo.ResourceID,
		ResourceSpec: resourceData.Spec,
		// The spec describes the resource as the framework deployed it, not as the link
		// has just written it, which the framework has no way to know. It identifies the
		// resource for the check rather than describing what to wait for, and an
		// implementation should not diff against it.
		ResourceMetadata: resourceData.Metadata,
		// This check follows a link's write rather than a deployment, which is a
		// different question and one some implementations can only answer if they are
		// told which is being asked.
		AfterLinkUpdate: true,
		ProviderContext: provider.NewProviderContextFromParams(
			provider.ExtractProviderFromItemType(resourceInfo.ResourceWithResolvedSubs.Type.Value),
			deployCtx.ParamOverrides,
		),
	}

	// Checked once before any wait. The polling loop for resources waits an interval
	// before its first check, which is reasonable for a resource that was just created
	// and cannot be ready yet. Applied to every link phase it would add that interval to
	// each one, inside the lock, for resources that are already stable.
	//
	// The timing is logged because "settled on the first check" is indistinguishable from
	// "this resource type cannot report a link's write" without it, and the two mean very
	// different things: the first is the wait working, the second is it doing nothing at
	// all. A resource whose HasStabilised is scoped to an operation the framework started,
	// for example, a Cloud Control backed one, reports stable immediately no matter what a
	// link just wrote through a service SDK.
	settleStart := time.Now()
	settled, err := d.hasResourceSettled(settleCtx, resourceImpl, input)
	if err != nil || settled {
		deployCtx.Logger.Debug(
			"link settle wait finished on the first check",
			core.StringLogField("resourceName", resourceInfo.ResourceName),
			core.StringLogField("resourceType", resourceInfo.ResourceWithResolvedSubs.Type.Value),
			core.StringLogField("settleDuration", time.Since(settleStart).String()),
			core.BoolLogField("settled", settled),
		)
		return err
	}

	err = d.pollUntilResourceSettles(ctx, settleCtx, resourceImpl, input, resourceInfo, deployCtx)
	deployCtx.Logger.Debug(
		"link settle wait polled until the resource stabilised",
		core.StringLogField("resourceName", resourceInfo.ResourceName),
		core.StringLogField("resourceType", resourceInfo.ResourceWithResolvedSubs.Type.Value),
		core.StringLogField("settleDuration", time.Since(settleStart).String()),
	)

	return err
}

// The settle context carries the polling timeout on top of the deployment's own context,
// so it finishes for either reason. Reporting both as a settle timeout would tell the user
// a resource took longer than the timeout to stabilise whenever a deployment was cancelled,
// which is why the parent is checked first.
func (d *defaultLinkDeployer) pollUntilResourceSettles(
	ctx context.Context,
	settleCtx context.Context,
	resourceImpl provider.Resource,
	input *provider.ResourceHasStabilisedInput,
	resourceInfo *provider.ResourceInfo,
	deployCtx *DeployContext,
) error {
	for {
		select {
		case <-settleCtx.Done():
			if parentErr := ctx.Err(); parentErr != nil {
				return parentErr
			}

			return errLinkResourceSettleTimeout(
				resourceInfo.ResourceName,
				d.settlePollingConfig.PollingTimeout,
			)
		case <-time.After(d.settlePollingConfig.PollingInterval):
			settled, err := d.hasResourceSettled(settleCtx, resourceImpl, input)
			if err != nil {
				return err
			}
			if settled {
				deployCtx.Logger.Debug(
					"resource written by link has settled",
					core.StringLogField("resourceName", resourceInfo.ResourceName),
				)
				return nil
			}
		}
	}
}

func (d *defaultLinkDeployer) hasResourceSettled(
	ctx context.Context,
	resourceImpl provider.Resource,
	input *provider.ResourceHasStabilisedInput,
) (bool, error) {
	output, err := resourceImpl.HasStabilised(ctx, input)
	if err != nil {
		return false, err
	}

	return output.Stabilised, nil
}

// Finds the resource implementation and the spec to describe the resource with, or reports
// that there is nothing to wait on.
//
// A link may write a resource the change set left alone, which has no collected data from
// this deployment. The persisted state covers that. A resource with neither is not
// something this deployment knows enough about to poll, so the wait is skipped rather than
// guessed at.
func (d *defaultLinkDeployer) resolveSettleTarget(
	ctx context.Context,
	resourceInfo *provider.ResourceInfo,
	deployCtx *DeployContext,
) (provider.Resource, *CollectedResourceData, error) {
	if resourceInfo == nil ||
		resourceInfo.ResourceWithResolvedSubs == nil ||
		resourceInfo.ResourceWithResolvedSubs.Type == nil {
		return nil, nil, nil
	}

	resourceData := deployCtx.State.GetResourceData(resourceInfo.ResourceName)
	if resourceData == nil {
		resourceData = collectedDataFromResourceState(resourceInfo.CurrentResourceState)
	}
	if resourceData == nil {
		return nil, nil, nil
	}

	resourceImpl, err := getProviderResourceImplementation(
		ctx,
		resourceInfo.ResourceName,
		resourceInfo.ResourceWithResolvedSubs.Type.Value,
		deployCtx.ResourceProviders,
	)
	if err != nil {
		return nil, nil, err
	}

	return resourceImpl, resourceData, nil
}

func collectedDataFromResourceState(
	resourceState *state.ResourceState,
) *CollectedResourceData {
	if resourceState == nil {
		return nil
	}

	return &CollectedResourceData{
		Spec:     resourceState.SpecData,
		Metadata: resourceState.Metadata,
	}
}
