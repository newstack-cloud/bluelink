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
}

func DeployContextWithChannels(
	deployCtx *DeployContext,
	channels *DeployChannels,
) *DeployContext {
	return &DeployContext{
		StartTime:             deployCtx.StartTime,
		State:                 deployCtx.State,
		Channels:              channels,
		Rollback:              deployCtx.Rollback,
		Destroying:            deployCtx.Destroying,
		InstanceStateSnapshot: deployCtx.InstanceStateSnapshot,
		ParamOverrides:        deployCtx.ParamOverrides,
		ResourceProviders:     deployCtx.ResourceProviders,
		CurrentGroupIndex:     deployCtx.CurrentGroupIndex,
		DeploymentGroups:      deployCtx.DeploymentGroups,
		InputChanges:          deployCtx.InputChanges,
		ResourceTemplates:     deployCtx.ResourceTemplates,
		PreparedContainer:     deployCtx.PreparedContainer,
		ResourceRegistry:      deployCtx.ResourceRegistry,
		Logger:                deployCtx.Logger,
		DrainDeadline:         deployCtx.DrainDeadline,
	}
}

func DeployContextWithGroup(
	deployCtx *DeployContext,
	groupIndex int,
) *DeployContext {
	return &DeployContext{
		StartTime:             deployCtx.StartTime,
		State:                 deployCtx.State,
		Channels:              deployCtx.Channels,
		Rollback:              deployCtx.Rollback,
		Destroying:            deployCtx.Destroying,
		InstanceStateSnapshot: deployCtx.InstanceStateSnapshot,
		ParamOverrides:        deployCtx.ParamOverrides,
		ResourceProviders:     deployCtx.ResourceProviders,
		CurrentGroupIndex:     groupIndex,
		DeploymentGroups:      deployCtx.DeploymentGroups,
		InputChanges:          deployCtx.InputChanges,
		ResourceTemplates:     deployCtx.ResourceTemplates,
		PreparedContainer:     deployCtx.PreparedContainer,
		ResourceRegistry:      deployCtx.ResourceRegistry,
		Logger:                deployCtx.Logger,
		DrainDeadline:         deployCtx.DrainDeadline,
	}
}

func DeployContextWithInstanceSnapshot(
	deployCtx *DeployContext,
	instanceSnapshot *state.InstanceState,
) *DeployContext {
	return &DeployContext{
		StartTime:             deployCtx.StartTime,
		State:                 deployCtx.State,
		Channels:              deployCtx.Channels,
		Rollback:              deployCtx.Rollback,
		Destroying:            deployCtx.Destroying,
		InstanceStateSnapshot: instanceSnapshot,
		ParamOverrides:        deployCtx.ParamOverrides,
		ResourceProviders:     deployCtx.ResourceProviders,
		CurrentGroupIndex:     deployCtx.CurrentGroupIndex,
		DeploymentGroups:      deployCtx.DeploymentGroups,
		InputChanges:          deployCtx.InputChanges,
		ResourceTemplates:     deployCtx.ResourceTemplates,
		PreparedContainer:     deployCtx.PreparedContainer,
		ResourceRegistry:      deployCtx.ResourceRegistry,
		Logger:                deployCtx.Logger,
		DrainDeadline:         deployCtx.DrainDeadline,
	}
}

func DeployContextWithLogger(
	deployCtx *DeployContext,
	logger core.Logger,
) *DeployContext {
	return &DeployContext{
		StartTime:             deployCtx.StartTime,
		State:                 deployCtx.State,
		Channels:              deployCtx.Channels,
		Rollback:              deployCtx.Rollback,
		Destroying:            deployCtx.Destroying,
		InstanceStateSnapshot: deployCtx.InstanceStateSnapshot,
		ParamOverrides:        deployCtx.ParamOverrides,
		ResourceProviders:     deployCtx.ResourceProviders,
		CurrentGroupIndex:     deployCtx.CurrentGroupIndex,
		DeploymentGroups:      deployCtx.DeploymentGroups,
		InputChanges:          deployCtx.InputChanges,
		ResourceTemplates:     deployCtx.ResourceTemplates,
		PreparedContainer:     deployCtx.PreparedContainer,
		ResourceRegistry:      deployCtx.ResourceRegistry,
		Logger:                logger,
		DrainDeadline:         deployCtx.DrainDeadline,
	}
}
