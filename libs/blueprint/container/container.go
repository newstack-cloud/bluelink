package container

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/drift"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// BlueprintContainer provides the interface for a service that manages
// staging and deploying an instance of a blueprint.
type BlueprintContainer interface {
	// StageChanges deals with staging changes to be deployed, comparing the effect of applying
	// the loaded blueprint to the existing instance stored in state for the instance ID it was
	// loaded with.
	// This will stream changes to the provided channels for each resource, child blueprint and link
	// that will be affected by the changes, a final message will be sent to the complete channel
	// containing the full set of changes that will be made to the blueprint instance when deploying
	// the changes.
	// Parameter overrides can be provided to add extra instance-specific variables
	// that were not defined when the container was loaded or to provide all variables
	// when the container was loaded with an empty set.
	StageChanges(
		ctx context.Context,
		input *StageChangesInput,
		channels *ChangeStagingChannels,
		paramOverrides core.BlueprintParams,
	) error
	// Deploy deals with deploying the blueprint for the given instance ID.
	// When an instance ID is omitted, the container will treat the deployment
	// as a new instance of the blueprint where the provided change set only includes
	// new resources, children and links.
	// Deploying a blueprint involves creating, updating and destroying resources
	// based on the staged changes.
	// This will stream updates to the provided channels for each resource, child blueprint and link
	// that has either been updated, created or removed.
	// Deploy should also be used as the mechanism to rollback a blueprint to a previous
	// revision managed in version control or a data store for blueprint source documents.
	//
	// There is synchronous and asynchronous error handling, synchronous errors will be returned
	// during the initial setup phase of the deployment process.
	// Most errors should be handled by the container and sent to the appropriate channel
	// as a deployment update message.
	Deploy(
		ctx context.Context,
		input *DeployInput,
		channels *DeployChannels,
		paramOverrides core.BlueprintParams,
	) error
	// Destroy deals with destroying all the resources, child blueprints and links
	// for a blueprint instance.
	// Like Deploy, Destroy requires changes to be staged and passed in to ensure that
	// the user gets a chance to review everything that will be destroyed before
	// starting the process; this should go hand and hand with a confirmation step and prompts
	// to allow the user to dig deeper in the tools built on top of the framework.
	// This will stream updates to the provided channels for each resource, child blueprint and link
	// that has been removed.
	//
	// There is no synchronous error handling, all errors will be sent to the provided error
	// channel. Most errors should be handled by the container and sent to the appropriate channel
	// as an update message.
	Destroy(
		ctx context.Context,
		input *DestroyInput,
		channels *DeployChannels,
		paramOverrides core.BlueprintParams,
	)
	// SpecLinkInfo provides the chain link and warnings for potential issues
	// with links provided in the given specification.
	SpecLinkInfo() links.SpecLinkInfo
	// BlueprintSpec returns the specification for the loaded blueprint
	// including the parsed schema and a convenience method to get resource
	// schemas by name.
	BlueprintSpec() speccore.BlueprintSpec
	// RefChainCollector returns the reference chain collector used to collect
	// reference chains for resources and child blueprints.
	// This is useful for allowing parent contexts to get access to the collected
	// references for a blueprint.
	// For example, extracting the references from an expanded version of a blueprint
	// that contains resource templates.
	RefChainCollector() refgraph.RefChainCollector
	// ResourceTemplates holds a mapping of resource names to the name of the resource
	// template it was derived from.
	// This allows retention of information about the original resource template
	// that a resource was derived from in a source blueprint document.
	ResourceTemplates() map[string]string
	// Diagnostics returns warning and informational diagnostics for the loaded blueprint
	// that point out potential issues that may occur when executing
	// a blueprint.
	// These diagnostics can contain errors, however, the error returned on failure to load a blueprint
	// should also be unpacked to get the precise location and information about the reason loading the
	// blueprint failed.
	Diagnostics() []*core.Diagnostic
	// CheckReconciliation checks for resources and links that need reconciliation.
	// This is a read-only operation that fetches external state and compares it to
	// persisted state to identify elements needing attention.
	// Use this when:
	// - Elements are in an interrupted state from a previous deployment
	// - Drift detection found discrepancies between cloud and persisted state
	// - A manual state refresh is needed
	//
	// The result contains detailed information about each element including the
	// external state, persisted state, and a diff showing the changes.
	// Parameter overrides can be provided to resolve any substitutions needed
	// when fetching external state.
	CheckReconciliation(
		ctx context.Context,
		input *CheckReconciliationInput,
		paramOverrides core.BlueprintParams,
	) (*ReconciliationCheckResult, error)
	// ApplyReconciliation applies the specified reconciliation actions.
	// This updates persisted state based on the actions specified for each element.
	// Typically called after CheckReconciliation with user-approved actions.
	//
	// Actions include:
	// - AcceptExternal: Update persisted state to match external cloud state
	// - UpdateStatus: Only update the element's status
	// - ManualCleanupRequired: Mark the element as requiring manual cleanup
	//
	// Parameter overrides can be provided for any substitutions needed during
	// the reconciliation process.
	ApplyReconciliation(
		ctx context.Context,
		input *ApplyReconciliationInput,
		paramOverrides core.BlueprintParams,
	) (*ApplyReconciliationResult, error)
}

// StageChangesInput contains the primary input needed to stage changes
// for a blueprint instance.
type StageChangesInput struct {
	// InstanceID is the ID of the blueprint instance that the changes will be applied to.
	// If this is set, `InstanceName` must be empty.
	InstanceID string
	// InstanceName is the user-defined name of the blueprint instance
	// that the changes will be applied to.
	// If this is set, `InstanceID` must be empty.
	InstanceName string
	// Destroy is used to indicate that the changes being staged should be for a destroy operation.
	// If this is set to true, the change set will be generated for removal all components
	// in the current state of the blueprint instance.
	Destroy bool
}

// DeployInput contains the primary input needed to deploy a blueprint instance.
type DeployInput struct {
	// InstanceID is the ID of the blueprint instance that the changes will be deployed for.
	// If this is set, `InstanceName` must be empty.
	InstanceID string
	// InstanceName is the user-defined name of the blueprint instance that
	// the changes will be deployed for.
	// If this is set, `InstanceID` must be empty.
	InstanceName string
	// Changes contains the changes that will be made to the blueprint instance.
	Changes *changes.BlueprintChanges
	// Rollback is used to indicate that the changes being deployed represent a rollback.
	// This is useful for ensuring the correct statuses are applied when changes within a child
	// blueprint need to be rolled back due to a failure in the parent blueprint.
	// The loaded blueprint is expected to be the version of the blueprint to roll back to
	// for a given blueprint instance.
	Rollback bool
	// Force bypasses state validation checks that prevent deployment when the instance
	// is already in an active state (e.g., Deploying, Updating).
	// This is an escape hatch for recovering from stuck states where the instance
	// is in an inconsistent state due to a crash or unexpected termination.
	Force bool
	// TaggingConfig holds the configuration for Bluelink resource tagging.
	// If nil, tagging will not be applied to resources.
	TaggingConfig *provider.TaggingConfig
	// ProviderMetadataLookup returns provider plugin metadata for a provider namespace.
	// This is used to populate ProviderPluginID and ProviderPluginVersion
	// in the TaggingConfig for each resource.
	ProviderMetadataLookup func(providerNamespace string) (pluginID, pluginVersion string)
	// DrainTimeout is the maximum time to wait for in-flight operations
	// to complete after a terminal failure before marking them as interrupted.
	// If zero, defaults to DefaultDrainTimeout (2 minutes).
	// Resources in CONFIG_COMPLETE (stabilization polling) benefit from
	// longer drain times to reach finalized states.
	DrainTimeout time.Duration
}

// DestroyInput contains the primary input needed to destroy a blueprint instance.
type DestroyInput struct {
	// InstanceID is the ID of the blueprint instance that will be destroyed.
	// If this is set, `InstanceName` must be empty.
	InstanceID string
	// InstanceName is the user-defined name of the blueprint instance
	// that will be destroyed.
	// If this is set, `InstanceID` must be empty.
	InstanceName string
	// Changes contains a description of all the elements that need to be
	// removed when destroying the blueprint instance.
	Changes *changes.BlueprintChanges
	// Rollback is used to indicate that the blueprint instance is being destroyed
	// as part of a rollback operation.
	// This is useful for ensuring the correct statuses are applied when changes within a child
	// blueprint need to be rolled back due to a failure in the parent blueprint.
	Rollback bool
	// Force continues the destroy operation even if individual resource/link/child
	// destruction fails, and removes the blueprint instance record from state
	// regardless of whether all resources were successfully destroyed.
	// This is useful for removing instances where underlying resources were manually
	// deleted or when a provider is unavailable.
	Force bool
	// TaggingConfig holds the configuration for Bluelink resource tagging.
	// If nil, tagging will not be applied to resources.
	TaggingConfig *provider.TaggingConfig
	// ProviderMetadataLookup returns provider plugin metadata for a provider namespace.
	// This is used to populate ProviderPluginID and ProviderPluginVersion
	// in the TaggingConfig for each resource.
	ProviderMetadataLookup func(providerNamespace string) (pluginID, pluginVersion string)
	// DrainTimeout is the maximum time to wait for in-flight operations
	// to complete after a terminal failure before marking them as interrupted.
	// If zero, defaults to DefaultDrainTimeout (2 minutes).
	// Resources in CONFIG_COMPLETE (stabilization polling) benefit from
	// longer drain times to reach finalized states.
	DrainTimeout time.Duration
}

const (
	// DefaultDrainTimeout is the default time to wait for in-flight operations
	// to complete after a terminal failure before marking them as interrupted.
	// This applies to both deploy and destroy operations.
	DefaultDrainTimeout = 2 * time.Minute
)

// getErrorChannelDrainTimeout returns a shorter timeout for errors from ErrChan
// since the erroring goroutine may have already terminated without
// sending a completion message. This is derived from the configured drain timeout
// to ensure it scales appropriately for both tests (short) and production (longer).
func getErrorChannelDrainTimeout(configuredDrainTimeout time.Duration) time.Duration {
	// Use 25% of the configured timeout, with a minimum of 100ms for tests
	// and a maximum of 30 seconds for production.
	timeout := configuredDrainTimeout / 4
	if timeout < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	if timeout > 30*time.Second {
		return 30 * time.Second
	}
	return timeout
}

type defaultBlueprintContainer struct {
	stateContainer state.Container
	// Should be a namespace to provider map.
	providers                map[string]provider.Provider
	resourceRegistry         resourcehelpers.Registry
	linkRegistry             provider.LinkRegistry
	spec                     speccore.BlueprintSpec
	linkInfo                 links.SpecLinkInfo
	resourceTemplates        map[string]string
	refChainCollector        refgraph.RefChainCollector
	substitutionResolver     subengine.SubstitutionResolver
	changeStager             ResourceChangeStager
	diagnostics              []*core.Diagnostic
	clock                    core.Clock
	idGenerator              core.IDGenerator
	createDeploymentState    func() DeploymentState
	createChangeStagingState func() ChangeStagingState
	blueprintPreparer        BlueprintPreparer
	linkChangeStager         LinkChangeStager
	childChangeStager        ChildChangeStager
	resourceDestroyer        ResourceDestroyer
	childBlueprintDestroyer  ChildBlueprintDestroyer
	linkDestroyer            LinkDestroyer
	linkDeployer             LinkDeployer
	driftChecker             drift.Checker
	resourceDeployer         ResourceDeployer
	childDeployer            ChildBlueprintDeployer
	defaultRetryPolicy       *provider.RetryPolicy
	logger                   core.Logger
}

// ChildBlueprintLoaderFactory provides a factory function for creating a new loader
// for child or derived blueprints.
type ChildBlueprintLoaderFactory func(
	derivedFromTemplate []string,
	resourceTemplates map[string]string,
) Loader

// DeploymentStateFactory provides a factory function for creating a new instance
// of a deployment state that is used as an ephemeral store for tracking the state
// of a deployment operation.
type DeploymentStateFactory func() DeploymentState

// ChangeStagingStateFactory provides a factory function for creating a new instance
// of a change staging state that is used as an ephemeral store for tracking the state
// of a change staging operation.
type ChangeStagingStateFactory func() ChangeStagingState

// BlueprintContainerDependencies provides the dependencies
// required to create a new instance of a blueprint container.
type BlueprintContainerDependencies struct {
	StateContainer            state.Container
	Providers                 map[string]provider.Provider
	ResourceRegistry          resourcehelpers.Registry
	LinkRegistry              provider.LinkRegistry
	LinkInfo                  links.SpecLinkInfo
	ResourceTemplates         map[string]string
	RefChainCollector         refgraph.RefChainCollector
	SubstitutionResolver      subengine.SubstitutionResolver
	ChangeStager              ResourceChangeStager
	Clock                     core.Clock
	IDGenerator               core.IDGenerator
	DeploymentStateFactory    DeploymentStateFactory
	ChangeStagingStateFactory ChangeStagingStateFactory
	BlueprintPreparer         BlueprintPreparer
	LinkChangeStager          LinkChangeStager
	ChildChangeStager         ChildChangeStager
	ResourceDestroyer         ResourceDestroyer
	ChildBlueprintDestroyer   ChildBlueprintDestroyer
	LinkDestroyer             LinkDestroyer
	LinkDeployer              LinkDeployer
	DriftChecker              drift.Checker
	ResourceDeployer          ResourceDeployer
	ChildBlueprintDeployer    ChildBlueprintDeployer
	DefaultRetryPolicy        *provider.RetryPolicy
	Logger                    core.Logger
}

// NewDefaultBlueprintContainer creates a new instance of the default
// implementation of a blueprint container for a loaded spec.
// The map of resource providers must be a map of provider resource name
// to a provider.
func NewDefaultBlueprintContainer(
	spec speccore.BlueprintSpec,
	deps *BlueprintContainerDependencies,
	diagnostics []*core.Diagnostic,
) BlueprintContainer {
	return &defaultBlueprintContainer{
		deps.StateContainer,
		deps.Providers,
		deps.ResourceRegistry,
		deps.LinkRegistry,
		spec,
		deps.LinkInfo,
		deps.ResourceTemplates,
		deps.RefChainCollector,
		deps.SubstitutionResolver,
		deps.ChangeStager,
		diagnostics,
		deps.Clock,
		deps.IDGenerator,
		deps.DeploymentStateFactory,
		deps.ChangeStagingStateFactory,
		deps.BlueprintPreparer,
		deps.LinkChangeStager,
		deps.ChildChangeStager,
		deps.ResourceDestroyer,
		deps.ChildBlueprintDestroyer,
		deps.LinkDestroyer,
		deps.LinkDeployer,
		deps.DriftChecker,
		deps.ResourceDeployer,
		deps.ChildBlueprintDeployer,
		deps.DefaultRetryPolicy,
		deps.Logger,
	}
}

func (c *defaultBlueprintContainer) SpecLinkInfo() links.SpecLinkInfo {
	return c.linkInfo
}

func (c *defaultBlueprintContainer) BlueprintSpec() speccore.BlueprintSpec {
	return c.spec
}

func (c *defaultBlueprintContainer) Diagnostics() []*core.Diagnostic {
	return c.diagnostics
}

func (c *defaultBlueprintContainer) RefChainCollector() refgraph.RefChainCollector {
	return c.refChainCollector
}

func (c *defaultBlueprintContainer) ResourceTemplates() map[string]string {
	return c.resourceTemplates
}

func (c *defaultBlueprintContainer) resolveExport(
	ctx context.Context,
	exportName string,
	export *schema.Export,
	resolveFor subengine.ResolveForStage,
) (*subengine.ResolveResult, error) {
	if export.Field != nil && export.Field.StringValue != nil {
		exportFieldAsSub, err := substitutions.ParseSubstitution(
			"exports",
			*export.Field.StringValue,
			/* parentSourceStart */ &source.Meta{Position: source.Position{}},
			/* outputLineInfo */ false,
			/* ignoreParentColumn */ true,
		)
		if err != nil {
			return nil, err
		}

		return c.substitutionResolver.ResolveSubstitution(
			ctx,
			&substitutions.StringOrSubstitution{
				SubstitutionValue: exportFieldAsSub,
			},
			core.ExportElementID(exportName),
			"field",
			&subengine.ResolveTargetInfo{
				ResolveFor: resolveFor,
			},
		)
	}

	return nil, nil
}

// A method to get an instance ID, allowing for it to be derived from
// a name for the purpose of staging changes for or destroying
// a blueprint instance.
func (c *defaultBlueprintContainer) getInstanceID(
	ctx context.Context,
	instanceID string,
	instanceName string,
) (string, error) {
	if instanceID != "" && instanceName != "" {
		// For staging changes and destroying an instance,
		// the user can only provide one of instance ID or name.
		return "", errInstanceIDAndNameProvided()
	}

	if instanceID == "" && instanceName != "" {
		resolvedInstanceID, err := c.stateContainer.
			Instances().
			LookupIDByName(ctx, instanceName)
		if err != nil {
			if state.IsInstanceNotFound(err) {
				// If no instance exists for the given name,
				// then assume it's a new instance for the purpose of change staging.
				// Destroy will handle an empty instance ID at a later stage.
				return "", nil
			}
			return "", err
		}

		return resolvedInstanceID, nil
	}

	return instanceID, nil
}
