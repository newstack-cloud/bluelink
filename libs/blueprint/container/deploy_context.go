package container

import (
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// DeployContext holds information shared between components that handle different
// parts of the deployment process for a blueprint instance.
type DeployContext struct {
	StartTime  time.Time
	Rollback   bool
	Destroying bool
	State      DeploymentState
	Channels   *DeployChannels
	// A snapshot of the instance state taken before deployment.
	InstanceStateSnapshot *state.InstanceState
	ParamOverrides        core.BlueprintParams
	ResourceProviders     map[string]provider.Provider
	CurrentGroupIndex     int
	DeploymentGroups      [][]*DeploymentNode
	InputChanges          *changes.BlueprintChanges
	// A mapping of resource names to the name of the resource
	// templates they were derived from.
	ResourceTemplates map[string]string
	// Holds the container for the blueprint after preparation/pre-processing
	// including template expansion and applying resource conditions.
	PreparedContainer BlueprintContainer
	// Provides a deployment-scoped registry for resources that will be packed
	// with the parameter overrides supplied in a container "Deploy" or "Destroy"
	// method call.
	ResourceRegistry resourcehelpers.Registry
	Logger           core.Logger
	// DrainDeadline is set by the parent when it enters drain mode.
	// Child deployers use this to continue forwarding events until drain completes.
	DrainDeadline <-chan time.Time
	// TaggingConfig holds the base tagging configuration for the deployment.
	// ProviderPluginID and ProviderPluginVersion will be set per-resource
	// using ProviderMetadataLookup.
	TaggingConfig *provider.TaggingConfig
	// ProviderMetadataLookup returns provider plugin metadata for a provider namespace.
	// This is used to populate ProviderPluginID and ProviderPluginVersion
	// in the TaggingConfig for each resource.
	ProviderMetadataLookup func(providerNamespace string) (pluginID, pluginVersion string)
	// DrainTimeout is the configured timeout for draining in-flight operations
	// after a terminal failure. This should be set from configuration when creating
	// the context, and defaults to DefaultDrainTimeout if not set.
	DrainTimeout time.Duration
	// LinkSlots bounds how many links are deployed at the same time across the whole
	// deployment, shared by every blueprint instance in it rather than held per
	// instance, which would multiply the bound by the size of the instance tree.
	LinkSlots *LinkSlots
	// LinkScheduler deploys this instance's links for the lifetime of the deployment.
	LinkScheduler *linkScheduler
}

func DeployContextWithChannels(
	deployCtx *DeployContext,
	channels *DeployChannels,
) *DeployContext {
	return &DeployContext{
		StartTime:              deployCtx.StartTime,
		State:                  deployCtx.State,
		Channels:               channels,
		Rollback:               deployCtx.Rollback,
		Destroying:             deployCtx.Destroying,
		InstanceStateSnapshot:  deployCtx.InstanceStateSnapshot,
		ParamOverrides:         deployCtx.ParamOverrides,
		ResourceProviders:      deployCtx.ResourceProviders,
		CurrentGroupIndex:      deployCtx.CurrentGroupIndex,
		DeploymentGroups:       deployCtx.DeploymentGroups,
		InputChanges:           deployCtx.InputChanges,
		ResourceTemplates:      deployCtx.ResourceTemplates,
		PreparedContainer:      deployCtx.PreparedContainer,
		ResourceRegistry:       deployCtx.ResourceRegistry,
		Logger:                 deployCtx.Logger,
		DrainDeadline:          deployCtx.DrainDeadline,
		TaggingConfig:          deployCtx.TaggingConfig,
		ProviderMetadataLookup: deployCtx.ProviderMetadataLookup,
		DrainTimeout:           deployCtx.DrainTimeout,
		LinkSlots:              deployCtx.LinkSlots,
		LinkScheduler:          deployCtx.LinkScheduler,
	}
}

func DeployContextWithGroup(
	deployCtx *DeployContext,
	groupIndex int,
) *DeployContext {
	return &DeployContext{
		StartTime:              deployCtx.StartTime,
		State:                  deployCtx.State,
		Channels:               deployCtx.Channels,
		Rollback:               deployCtx.Rollback,
		Destroying:             deployCtx.Destroying,
		InstanceStateSnapshot:  deployCtx.InstanceStateSnapshot,
		ParamOverrides:         deployCtx.ParamOverrides,
		ResourceProviders:      deployCtx.ResourceProviders,
		CurrentGroupIndex:      groupIndex,
		DeploymentGroups:       deployCtx.DeploymentGroups,
		InputChanges:           deployCtx.InputChanges,
		ResourceTemplates:      deployCtx.ResourceTemplates,
		PreparedContainer:      deployCtx.PreparedContainer,
		ResourceRegistry:       deployCtx.ResourceRegistry,
		Logger:                 deployCtx.Logger,
		DrainDeadline:          deployCtx.DrainDeadline,
		TaggingConfig:          deployCtx.TaggingConfig,
		ProviderMetadataLookup: deployCtx.ProviderMetadataLookup,
		DrainTimeout:           deployCtx.DrainTimeout,
		LinkSlots:              deployCtx.LinkSlots,
		LinkScheduler:          deployCtx.LinkScheduler,
	}
}

func DeployContextWithInstanceSnapshot(
	deployCtx *DeployContext,
	instanceSnapshot *state.InstanceState,
) *DeployContext {
	return &DeployContext{
		StartTime:              deployCtx.StartTime,
		State:                  deployCtx.State,
		Channels:               deployCtx.Channels,
		Rollback:               deployCtx.Rollback,
		Destroying:             deployCtx.Destroying,
		InstanceStateSnapshot:  instanceSnapshot,
		ParamOverrides:         deployCtx.ParamOverrides,
		ResourceProviders:      deployCtx.ResourceProviders,
		CurrentGroupIndex:      deployCtx.CurrentGroupIndex,
		DeploymentGroups:       deployCtx.DeploymentGroups,
		InputChanges:           deployCtx.InputChanges,
		ResourceTemplates:      deployCtx.ResourceTemplates,
		PreparedContainer:      deployCtx.PreparedContainer,
		ResourceRegistry:       deployCtx.ResourceRegistry,
		Logger:                 deployCtx.Logger,
		DrainDeadline:          deployCtx.DrainDeadline,
		TaggingConfig:          deployCtx.TaggingConfig,
		ProviderMetadataLookup: deployCtx.ProviderMetadataLookup,
		DrainTimeout:           deployCtx.DrainTimeout,
		LinkSlots:              deployCtx.LinkSlots,
		LinkScheduler:          deployCtx.LinkScheduler,
	}
}

func DeployContextWithLogger(
	deployCtx *DeployContext,
	logger core.Logger,
) *DeployContext {
	return &DeployContext{
		StartTime:              deployCtx.StartTime,
		State:                  deployCtx.State,
		Channels:               deployCtx.Channels,
		Rollback:               deployCtx.Rollback,
		Destroying:             deployCtx.Destroying,
		InstanceStateSnapshot:  deployCtx.InstanceStateSnapshot,
		ParamOverrides:         deployCtx.ParamOverrides,
		ResourceProviders:      deployCtx.ResourceProviders,
		CurrentGroupIndex:      deployCtx.CurrentGroupIndex,
		DeploymentGroups:       deployCtx.DeploymentGroups,
		InputChanges:           deployCtx.InputChanges,
		ResourceTemplates:      deployCtx.ResourceTemplates,
		PreparedContainer:      deployCtx.PreparedContainer,
		ResourceRegistry:       deployCtx.ResourceRegistry,
		Logger:                 logger,
		DrainDeadline:          deployCtx.DrainDeadline,
		TaggingConfig:          deployCtx.TaggingConfig,
		ProviderMetadataLookup: deployCtx.ProviderMetadataLookup,
		DrainTimeout:           deployCtx.DrainTimeout,
		LinkSlots:              deployCtx.LinkSlots,
		LinkScheduler:          deployCtx.LinkScheduler,
	}
}
