package deploymentsv1

import (
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginconfig"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

const (
	// An internal timeout used for the background goroutine
	// that performs change staging.
	// 30 minutes allows for provider or transformer plugins
	// that may take a while to respond if network requests are involved.
	// Examples of this could include fetching data sources that are referenced
	// by blueprint resources.
	changeStagingTimeout = 30 * time.Minute
	// An internal timeout used for the cleanup process
	// that cleans up old change sets.
	// 10 minutes is a reasonable time to wait for the cleanup process
	// to complete for instances of the deploy engine with a lot of use.
	changesetCleanupTimeout = 10 * time.Minute
)

const (
	// Shared event types.
	eventTypeError = "error"

	// Event types for change staging.
	eventTypeResourceChanges       = "resourceChanges"
	eventTypeChildChanges          = "childChanges"
	eventTypeLinkChanges           = "linkChanges"
	eventTypeChangeStagingComplete = "changeStagingComplete"

	// Event types for deployment.
	eventTypeResourceUpdate = "resource"
	eventTypeChildUpdate    = "child"
	eventTypeLinkUpdate     = "link"
	eventTypeInstanceUpdate = "instanceUpdate"
	eventTypeDeployFinished = "finish"
)

// Controller handles deployment-related HTTP requests
// including change staging and deployment events over Server-Sent Events (SSE).
type Controller struct {
	changesetRetentionPeriod time.Duration
	deploymentTimeout        time.Duration
	eventStore               manage.Events
	instances                state.InstancesContainer
	exports                  state.ExportsContainer
	changesetStore           manage.Changesets
	idGenerator              core.IDGenerator
	eventIDGenerator         core.IDGenerator
	blueprintLoader          container.Loader
	// Behaviour used to resolve child blueprints in the blueprint container
	// package is reused to load the "root" blueprints from multiple sources.
	blueprintResolver    includes.ChildResolver
	paramsProvider       params.Provider
	pluginConfigPreparer pluginconfig.Preparer
	clock                commoncore.Clock
	logger               core.Logger
}

// NewController creates a new deployments Controller
// instance with the provided dependencies.
func NewController(
	changesetRetentionPeriod time.Duration,
	deploymentTimeout time.Duration,
	deps *typesv1.Dependencies,
) *Controller {
	return &Controller{
		changesetRetentionPeriod: changesetRetentionPeriod,
		deploymentTimeout:        deploymentTimeout,
		eventStore:               deps.EventStore,
		instances:                deps.Instances,
		exports:                  deps.Exports,
		changesetStore:           deps.ChangesetStore,
		idGenerator:              deps.IDGenerator,
		eventIDGenerator:         deps.EventIDGenerator,
		blueprintLoader:          deps.DeploymentLoader,
		blueprintResolver:        deps.BlueprintResolver,
		paramsProvider:           deps.ParamsProvider,
		pluginConfigPreparer:     deps.PluginConfigPreparer,
		clock:                    deps.Clock,
		logger:                   deps.Logger,
	}
}
