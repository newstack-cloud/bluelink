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
	// Deploy updates a layer's resource with the contributions of the links it carries,
	// reporting the update as one carrying link contributions rather than one the
	// blueprint asked for.
	Deploy(
		ctx context.Context,
		instanceID string,
		layer ContributionLayer,
		contributingLinkNames []string,
		deployCtx *DeployContext,
	) error
}

type defaultMergedContributionsDeployer struct {
	stateContainer state.Container
	clock          core.Clock
	// Holds each resource as it was resolved for its own deployment in this run, which is
	// the only place a reference to a field computed by another resource is resolved.
	resourceCache *core.Cache[*provider.ResolvedResource]
}

// NewDefaultMergedContributionsDeployer creates the default implementation of the service
// that updates a resource with the contributions made to it.
func NewDefaultMergedContributionsDeployer(
	stateContainer state.Container,
	clock core.Clock,
	resourceCache *core.Cache[*provider.ResolvedResource],
) MergedContributionsDeployer {
	return &defaultMergedContributionsDeployer{
		stateContainer: stateContainer,
		clock:          clock,
		resourceCache:  resourceCache,
	}
}

func (d *defaultMergedContributionsDeployer) Deploy(
	ctx context.Context,
	instanceID string,
	layer ContributionLayer,
	contributingLinkNames []string,
	deployCtx *DeployContext,
) error {
	resourceName := layer.ResourceName

	// Read the live state as a link records what it contributes
	// when it deploys, so a link that ran in this deployment has mappings the snapshot
	// taken before it started does not, and composing from the snapshot would withdraw
	// them from the resource.
	storedLinks, err := d.stateContainer.Links().ListWithResourceDataMappings(
		ctx,
		instanceID,
		resourceName,
	)
	if err != nil {
		return err
	}

	// Resolved before anything can fail. Every outcome of this update, successful or not,
	// is about what a set of links needed the resource's spec to include, and a failure that
	// names none of them leaves the resource reported as broken with nothing pointing at
	// the contribution responsible.
	contributors := LinkContributorsFor(
		resourceName,
		CollectResourceContributionSources(
			deployCtx,
			resourceName,
			contributingLinkNames,
			storedLinks,
		),
	)

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
			layer,
			deployCtx,
			contributors,
			layerFailure(fmt.Sprintf(
				"the resource %q that links contribute to is not deployed, so the "+
					"contributions made to it cannot be applied",
				resourceName,
			)),
		)
	}

	declaredSpec := d.declaredSpecForMergedUpdate(deployCtx, resourceName, resourceState)

	merged, err := ComposeMergedResourceSpec(
		deployCtx,
		layer,
		declaredSpec,
		contributingLinkNames,
		storedLinks,
	)
	if err != nil {
		return d.reportFailure(
			instanceID,
			resourceState,
			layer,
			deployCtx,
			contributors,
			layerFailure(err.Error()),
		)
	}

	if len(merged.Unresolved) > 0 {
		// Deploying a spec that is missing a contribution the framework has a record of
		// would remove that contribution from the resource, which is the opposite of what
		// this update is for.
		return d.reportFailure(
			instanceID,
			resourceState,
			layer,
			deployCtx,
			contributors,
			unresolvedContributionFailure(merged.Unresolved),
		)
	}

	resourceImpl, err := getProviderResourceImplementation(
		ctx,
		resourceName,
		resourceState.Type,
		deployCtx.ResourceProviders,
	)
	if err != nil {
		return d.reportFailure(
			instanceID,
			resourceState,
			layer,
			deployCtx,
			contributors,
			layerFailure(err.Error()),
		)
	}

	deployCtx.Channels.ResourceUpdateChan <- d.updateMessage(
		instanceID,
		resourceState,
		layer,
		deployCtx,
		core.ResourceStatusUpdating,
		core.PreciseResourceStatusUpdatingLinkContributions,
		contributors,
		contributionFailureDetail{},
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
		return d.reportFailure(
			instanceID,
			resourceState,
			layer,
			deployCtx,
			contributors,
			layerFailure(err.Error()),
		)
	}

	// Recorded before the message is sent. A link held for a capability this layer carries
	// is released by the scheduler on its own goroutine, which is the one that got here,
	// so the layer has to be applied by the time this returns rather than by the time the
	// message is read.
	deployCtx.State.MarkContributionLayerUpdated(layer)

	deployCtx.Channels.ResourceUpdateChan <- d.updateMessage(
		instanceID,
		resourceState,
		layer,
		deployCtx,
		core.ResourceStatusUpdated,
		core.PreciseResourceStatusLinkContributionsUpdated,
		contributors,
		contributionFailureDetail{},
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
	layer ContributionLayer,
	deployCtx *DeployContext,
	contributors map[string][]string,
	detail contributionFailureDetail,
) error {
	// Recorded before the message is sent, for the same reason applying one is, a link
	// waiting on this layer is abandoned by the scheduler on the goroutine that got here.
	deployCtx.State.MarkContributionLayerFailed(layer)

	deployCtx.Channels.ResourceUpdateChan <- d.updateMessage(
		instanceID,
		resourceState,
		layer,
		deployCtx,
		core.ResourceStatusUpdateFailed,
		core.PreciseResourceStatusLinkContributionsUpdateFailed,
		contributors,
		detail,
	)

	return nil
}

// What a failed layer has to say about itself, in both the form a client renders and the
// form that survives into state.
//
// Carried together because every failure has the first and only some have the second, a
// provider rejecting the update says nothing about which field was at fault, while a
// contribution that could not be composed names the link and the field it belongs to.
type contributionFailureDetail struct {
	reasons   []string
	unapplied []state.UnappliedLinkContribution
}

func layerFailure(reasons ...string) contributionFailureDetail {
	return contributionFailureDetail{reasons: reasons}
}

// A failure attributed to the individual contributions that could not be composed.
//
// The text is built from the same projections rather than alongside them, so what a client
// reads and what state holds cannot describe different failures.
func unresolvedContributionFailure(
	unresolved []specmerge.UnresolvedProjection,
) contributionFailureDetail {
	unapplied := make([]state.UnappliedLinkContribution, 0, len(unresolved))
	for _, projection := range unresolved {
		unapplied = append(unapplied, state.UnappliedLinkContribution{
			LinkName:  projection.LinkName,
			FieldPath: projection.ResourceFieldPath,
			Reason:    projection.Reason,
		})
	}

	return contributionFailureDetail{
		reasons:   unresolvedContributionReasons(unresolved),
		unapplied: unapplied,
	}
}

func (d *defaultMergedContributionsDeployer) updateMessage(
	instanceID string,
	resourceState *state.ResourceState,
	layer ContributionLayer,
	deployCtx *DeployContext,
	status core.ResourceStatus,
	preciseStatus core.PreciseResourceStatus,
	contributors map[string][]string,
	detail contributionFailureDetail,
) ResourceDeployUpdateMessage {
	return ResourceDeployUpdateMessage{
		InstanceID:             instanceID,
		ResourceID:             resourceIDOrEmpty(resourceState),
		ResourceName:           layer.ResourceName,
		Group:                  deployCtx.CurrentGroupIndex,
		ContributionLayerDepth: layer.Depth,
		Status:                 status,
		PreciseStatus:          preciseStatus,
		FromLinkContributions:  true,
		LinkContributors:       contributors,
		FailureReasons:         detail.reasons,

		UnappliedLinkContributions: detail.unapplied,
		UpdateTimestamp:            d.clock.Now().Unix(),
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

func (d *defaultMergedContributionsDeployer) declaredSpecForMergedUpdate(
	deployCtx *DeployContext,
	resourceName string,
	resourceState *state.ResourceState,
) *core.MappingNode {
	if d.resourceCache != nil {
		if deployed, cached := d.resourceCache.Get(resourceName); cached &&
			deployed != nil && deployed.Spec != nil {
			return deployed.Spec
		}
	}

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
