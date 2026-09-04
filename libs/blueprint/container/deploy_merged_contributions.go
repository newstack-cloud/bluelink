package container

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// MergedContributionsDeployer updates a resource with what the links contributing to it
// need its spec to say, once every one of those links has settled.
//
// This is not the deployment of a change the user made, and does not go through the path
// that deploys one. A resource reached this way may not be in the change set at all, an
// execution role that a dozen links write to is untouched by a deployment that changes one of
// those links, and still has to be updated with what they contribute. The path that
// deploys a change starts from a change set entry, refuses without one, and reports what
// it does as a change to the blueprint, none of which holds here.
type MergedContributionsDeployer interface {
	// Deploy updates the named resource with the contributions of the links that target
	// it, reporting the update as one carrying link contributions rather than one the
	// blueprint asked for.
	Deploy(
		ctx context.Context,
		instanceID string,
		resourceName string,
		contributingLinkNames []string,
		deployCtx *DeployContext,
	) error
}

type defaultMergedContributionsDeployer struct {
	stateContainer state.Container
	clock          core.Clock
}

// NewDefaultMergedContributionsDeployer creates the default implementation of the service
// that updates a resource with the contributions made to it.
func NewDefaultMergedContributionsDeployer(
	stateContainer state.Container,
	clock core.Clock,
) MergedContributionsDeployer {
	return &defaultMergedContributionsDeployer{
		stateContainer: stateContainer,
		clock:          clock,
	}
}

func (d *defaultMergedContributionsDeployer) Deploy(
	ctx context.Context,
	instanceID string,
	resourceName string,
	contributingLinkNames []string,
	deployCtx *DeployContext,
) error {
	// Read live rather than from the deployment's snapshot of instance state. A resource
	// links contribute to is commonly created by the same deployment that runs them, and
	// the snapshot was taken before it existed.
	resourceState, err := d.currentResourceState(ctx, instanceID, resourceName)
	if err != nil {
		return err
	}

	if resourceState == nil {
		// Every resource links contribute to is either created by this deployment, which
		// has saved it by the time its links settle, or already in state. One that is in
		// neither is a resource a link declared a contribution to that does not exist, and
		// the contribution has nowhere to go.
		return d.reportFailure(
			instanceID,
			resourceState,
			resourceName,
			deployCtx,
			[]string{
				fmt.Sprintf(
					"the resource %q that links contribute to is not deployed, so the "+
						"contributions made to it cannot be applied",
					resourceName,
				),
			},
		)
	}

	declaredSpec := declaredSpecForMergedUpdate(deployCtx, resourceName, resourceState)

	merged, err := ComposeMergedResourceSpec(
		deployCtx,
		resourceName,
		declaredSpec,
		contributingLinkNames,
	)
	if err != nil {
		return d.reportFailure(
			instanceID,
			resourceState,
			resourceName,
			deployCtx,
			[]string{err.Error()},
		)
	}

	if len(merged.Unresolved) > 0 {
		// Deploying a spec that is missing a contribution the framework has a record of
		// would remove that contribution from the resource, which is the opposite of what
		// this update is for.
		return d.reportFailure(
			instanceID,
			resourceState,
			resourceName,
			deployCtx,
			unresolvedContributionReasons(merged.Unresolved),
		)
	}

	resourceImpl, err := getProviderResourceImplementation(
		ctx,
		resourceName,
		resourceState.Type,
		deployCtx.ResourceProviders,
	)
	if err != nil {
		return d.reportFailure(instanceID, resourceState, resourceName, deployCtx, []string{err.Error()})
	}

	contributors := LinkContributorsFor(
		resourceName,
		CollectResourceContributionSources(deployCtx, contributingLinkNames),
	)

	deployCtx.Channels.ResourceUpdateChan <- d.updateMessage(
		instanceID,
		resourceState,
		resourceName,
		deployCtx,
		core.ResourceStatusUpdating,
		core.PreciseResourceStatusUpdatingLinkContributions,
		contributors,
		/* failureReasons */ nil,
	)

	providerNamespace := provider.ExtractProviderFromItemType(resourceState.Type)
	_, err = resourceImpl.Deploy(
		ctx,
		&provider.ResourceDeployInput{
			InstanceID:            instanceID,
			InstanceName:          deployCtx.InstanceStateSnapshot.InstanceName,
			ResourceID:            resourceState.ResourceID,
			Changes:               mergedContributionChanges(resourceState, merged.Spec),
			FromLinkContributions: true,
			ProviderContext: provider.NewProviderContextFromParamsWithOptions(
				providerNamespace,
				deployCtx.ParamOverrides,
				&provider.ProviderContextOptions{
					TaggingConfig: createResourceTaggingConfig(
						deployCtx.TaggingConfig,
						providerNamespace,
						deployCtx.ProviderMetadataLookup,
					),
				},
			),
		},
	)
	if err != nil {
		return d.reportFailure(instanceID, resourceState, resourceName, deployCtx, []string{err.Error()})
	}

	deployCtx.Channels.ResourceUpdateChan <- d.updateMessage(
		instanceID,
		resourceState,
		resourceName,
		deployCtx,
		core.ResourceStatusUpdated,
		core.PreciseResourceStatusLinkContributionsUpdated,
		contributors,
		/* failureReasons */ nil,
	)

	return nil
}

func (d *defaultMergedContributionsDeployer) currentResourceState(
	ctx context.Context,
	instanceID string,
	resourceName string,
) (*state.ResourceState, error) {
	resourceState, err := d.stateContainer.Resources().GetByName(ctx, instanceID, resourceName)
	if err != nil {
		if state.IsResourceNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return &resourceState, nil
}

// Reported against the resource rather than raised, so the deployment drains rather than
// stopping where it stands. It still fails, the merged update is the only way a
// contribution reaches the resource, and a resource left without link contributions
// should be considered failed.
func (d *defaultMergedContributionsDeployer) reportFailure(
	instanceID string,
	resourceState *state.ResourceState,
	resourceName string,
	deployCtx *DeployContext,
	failureReasons []string,
) error {
	deployCtx.Channels.ResourceUpdateChan <- d.updateMessage(
		instanceID,
		resourceState,
		resourceName,
		deployCtx,
		core.ResourceStatusUpdateFailed,
		core.PreciseResourceStatusLinkContributionsUpdateFailed,
		/* contributors */ nil,
		failureReasons,
	)

	return nil
}

func (d *defaultMergedContributionsDeployer) updateMessage(
	instanceID string,
	resourceState *state.ResourceState,
	resourceName string,
	deployCtx *DeployContext,
	status core.ResourceStatus,
	preciseStatus core.PreciseResourceStatus,
	contributors map[string][]string,
	failureReasons []string,
) ResourceDeployUpdateMessage {
	return ResourceDeployUpdateMessage{
		InstanceID:            instanceID,
		ResourceID:            resourceIDOrEmpty(resourceState),
		ResourceName:          resourceName,
		Group:                 deployCtx.CurrentGroupIndex,
		Status:                status,
		PreciseStatus:         preciseStatus,
		FromLinkContributions: true,
		LinkContributors:      contributors,
		FailureReasons:        failureReasons,
		UpdateTimestamp:       d.clock.Now().Unix(),
	}
}

// A resource that is not deployed has no ID to report the failure against, and the
// resource is named either way.
func resourceIDOrEmpty(resourceState *state.ResourceState) string {
	if resourceState == nil {
		return ""
	}

	return resourceState.ResourceID
}

// The resource as the blueprint declares it, without the contributions links have made to
// it, which is what the merged spec is composed on top of.
//
// A resource in the change set has a resolved spec there. One that is not in the change
// set at all has only what state holds, which is the declared spec, since contributions
// are applied at deploy time and kept out of what is persisted.
func declaredSpecForMergedUpdate(
	deployCtx *DeployContext,
	resourceName string,
	resourceState *state.ResourceState,
) *core.MappingNode {
	resolved := getResolvedResourceFromInputChanges(deployCtx.InputChanges, resourceName)
	if resolved != nil && resolved.Spec != nil {
		return resolved.Spec
	}

	return resourceState.SpecData
}

// The provider is handed the composed spec as the resource to deploy. This is the input a
// provider takes rather than an entry in the change set, so a resource with no entry can
// still be given one.
func mergedContributionChanges(
	resourceState *state.ResourceState,
	mergedSpec *core.MappingNode,
) *provider.Changes {
	return &provider.Changes{
		AppliedResourceInfo: provider.ResourceInfo{
			ResourceID:           resourceState.ResourceID,
			ResourceName:         resourceState.Name,
			InstanceID:           resourceState.InstanceID,
			CurrentResourceState: resourceState,
			ResourceWithResolvedSubs: &provider.ResolvedResource{
				Type: &schema.ResourceTypeWrapper{Value: resourceState.Type},
				Spec: mergedSpec,
			},
		},
	}
}

// Named against the link each contribution belongs to, since a contribution that cannot
// be applied is the link's to explain rather than the resource's.
func unresolvedContributionReasons(
	unresolved []specmerge.UnresolvedProjection,
) []string {
	reasons := make([]string, 0, len(unresolved))
	for _, projection := range unresolved {
		reasons = append(reasons, fmt.Sprintf(
			"the contribution to %q from link %q could not be applied: %s",
			projection.ResourceFieldPath,
			projection.LinkName,
			projection.Reason,
		))
	}

	return reasons
}
