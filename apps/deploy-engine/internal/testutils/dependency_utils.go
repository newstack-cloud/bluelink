package testutils

import "github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"

func CopyDependencies(deps *typesv1.Dependencies) *typesv1.Dependencies {
	return &typesv1.Dependencies{
		EventStore:                 deps.EventStore,
		ValidationStore:            deps.ValidationStore,
		ChangesetStore:             deps.ChangesetStore,
		ReconciliationResultsStore: deps.ReconciliationResultsStore,
		Instances:                  deps.Instances,
		Exports:                    deps.Exports,
		IDGenerator:                deps.IDGenerator,
		EventIDGenerator:           deps.EventIDGenerator,
		ValidationLoader:           deps.ValidationLoader,
		DeploymentLoader:           deps.DeploymentLoader,
		BlueprintResolver:          deps.BlueprintResolver,
		ParamsProvider:             deps.ParamsProvider,
		PluginConfigPreparer:       deps.PluginConfigPreparer,
		TaggingConfigProvider:      deps.TaggingConfigProvider,
		ProviderMetadataLookup:     deps.ProviderMetadataLookup,
		Clock:                      deps.Clock,
		Logger:                     deps.Logger,
	}
}
