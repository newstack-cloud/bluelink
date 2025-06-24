package container

import (
	"context"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ChildChangeStager provides an interface for a service that stages changes for
// a child blueprint included in a parent blueprint.
type ChildChangeStager interface {
	StageChanges(
		ctx context.Context,
		childInstanceInfo *ChildInstanceInfo,
		node *refgraph.ReferenceChainNode,
		paramOverrides core.BlueprintParams,
		channels *ChangeStagingChannels,
		logger core.Logger,
	)
}

// NewDefaultLinkChangeStager creates a new instance of the default implementation
// of the service that stages changes for a child blueprint.
func NewDefaultChildChangeStager(
	childResolver includes.ChildResolver,
	createChildBlueprintLoader ChildBlueprintLoaderFactory,
	stateContainer state.Container,
	childExportFieldCache *core.Cache[*subengine.ChildExportFieldInfo],
	substitutionResolver subengine.SubstitutionResolver,
) ChildChangeStager {
	return &defaultChildChangeStager{
		childResolver:              childResolver,
		createChildBlueprintLoader: createChildBlueprintLoader,
		stateContainer:             stateContainer,
		childExportFieldCache:      childExportFieldCache,
		substitutionResolver:       substitutionResolver,
	}
}

type defaultChildChangeStager struct {
	childResolver              includes.ChildResolver
	substitutionResolver       subengine.SubstitutionResolver
	createChildBlueprintLoader ChildBlueprintLoaderFactory
	stateContainer             state.Container
	childExportFieldCache      *core.Cache[*subengine.ChildExportFieldInfo]
}

func (d *defaultChildChangeStager) StageChanges(
	ctx context.Context,
	childInstanceInfo *ChildInstanceInfo,
	node *refgraph.ReferenceChainNode,
	paramOverrides core.BlueprintParams,
	channels *ChangeStagingChannels,
	logger core.Logger,
) {
	loadResult, err := loadChildBlueprint(
		ctx,
		&childBlueprintLoadInput{
			parentInstanceID:       childInstanceInfo.ParentInstanceID,
			parentInstanceTreePath: childInstanceInfo.ParentInstanceTreePath,
			instanceTreePath:       node.ElementName,
			includeTreePath:        childInstanceInfo.IncludeTreePath,
			node:                   node,
			resolveFor:             subengine.ResolveForChangeStaging,
			logger:                 logger,
		},
		d.substitutionResolver,
		d.childResolver,
		d.createChildBlueprintLoader,
		d.stateContainer,
		paramOverrides,
	)
	if err != nil {
		channels.ErrChan <- err
		return
	}

	childChannels := &ChangeStagingChannels{
		ResourceChangesChan: make(chan ResourceChangesMessage),
		ChildChangesChan:    make(chan ChildChangesMessage),
		LinkChangesChan:     make(chan LinkChangesMessage),
		CompleteChan:        make(chan changes.BlueprintChanges),
		ErrChan:             make(chan error),
	}
	err = loadResult.childContainer.StageChanges(
		ctx,
		&StageChangesInput{
			InstanceID: loadResult.childState.InstanceID,
		},
		childChannels,
		loadResult.childParams,
	)
	if err != nil {
		channels.ErrChan <- err
		return
	}

	d.waitForChildChanges(
		ctx,
		loadResult.includeName,
		loadResult.childState,
		childChannels,
		channels,
	)
}

func (d *defaultChildChangeStager) waitForChildChanges(
	ctx context.Context,
	includeName string,
	childState *state.InstanceState,
	childChannels *ChangeStagingChannels,
	channels *ChangeStagingChannels,
) {
	// For now, when it comes to child blueprints,
	// wait for all changes to be staged before sending
	// an update message for the parent blueprint context.
	// In the future, we may want to stream changes
	// in child blueprints like with resources and links
	// in the parent blueprint.
	var changes changes.BlueprintChanges
	receivedFullChildChanges := false
	var stagingErr error
	for !receivedFullChildChanges && stagingErr == nil {
		select {
		case <-ctx.Done():
			stagingErr = ctx.Err()
		case <-childChannels.ResourceChangesChan:
		case <-childChannels.LinkChangesChan:
		case <-childChannels.ChildChangesChan:
		case changes = <-childChannels.CompleteChan:
			d.cacheChildExportFields(includeName, &changes)
			channels.ChildChangesChan <- ChildChangesMessage{
				ChildBlueprintName: includeName,
				Removed:            false,
				New:                childState.InstanceID == "",
				Changes:            changes,
			}
		case stagingErr = <-childChannels.ErrChan:
			channels.ErrChan <- stagingErr
		}
	}
}

func (d *defaultChildChangeStager) cacheChildExportFields(
	childName string,
	changes *changes.BlueprintChanges,
) {
	for exportName, fieldChange := range changes.ExportChanges {
		d.cacheChildExportField(
			childName,
			changes,
			exportName,
			fieldChange,
		)
	}

	for exportName, fieldChange := range changes.NewExports {
		d.cacheChildExportField(
			childName,
			changes,
			exportName,
			fieldChange,
		)
	}

	for _, exportName := range changes.RemovedExports {
		key := substitutions.RenderFieldPath(childName, exportName)
		d.childExportFieldCache.Set(
			key,
			&subengine.ChildExportFieldInfo{
				Value:           nil,
				Removed:         true,
				ResolveOnDeploy: false,
			},
		)
	}
}

func (d *defaultChildChangeStager) cacheChildExportField(
	childName string,
	changes *changes.BlueprintChanges,
	exportName string,
	fieldChange provider.FieldChange,
) {
	key := substitutions.RenderFieldPath(childName, exportName)
	exportFieldPath := substitutions.RenderFieldPath("exports", exportName)
	willResolveOnDeploy := slices.Contains(
		changes.ResolveOnDeploy,
		exportFieldPath,
	)

	d.childExportFieldCache.Set(
		key,
		&subengine.ChildExportFieldInfo{
			Value:           fieldChange.NewValue,
			Removed:         false,
			ResolveOnDeploy: willResolveOnDeploy,
		},
	)
}

// ChildInstanceInfo provides information about a child blueprint instance
// that is being deployed as part of a parent blueprint.
type ChildInstanceInfo struct {
	ParentInstanceID       string
	ParentInstanceTreePath string
	IncludeTreePath        string
}
