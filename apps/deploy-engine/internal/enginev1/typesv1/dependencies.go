package typesv1

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginconfig"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/pluginmeta"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tagging"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// Dependencies holds all the dependency services
// that are required by the controllers that provide HTTP handlers
// for v1 of the Deploy Engine API.
type Dependencies struct {
	EventStore                 manage.Events
	ValidationStore            manage.Validation
	ChangesetStore             manage.Changesets
	ReconciliationResultsStore manage.ReconciliationResults
	CleanupOperationsStore     manage.CleanupOperations
	Instances                  state.InstancesContainer
	Exports                    state.ExportsContainer
	IDGenerator                core.IDGenerator
	EventIDGenerator           core.IDGenerator
	ValidationLoader           container.Loader
	DeploymentLoader           container.Loader
	BlueprintResolver          includes.ChildResolver
	ParamsProvider             params.Provider
	PluginConfigPreparer       pluginconfig.Preparer
	TaggingConfigProvider      tagging.ConfigProvider
	ProviderMetadataLookup     pluginmeta.Lookup
	Clock                      commoncore.Clock
	Logger                     core.Logger
}
