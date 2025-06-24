package container

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

func createChangeStagingChannels() *ChangeStagingChannels {
	return &ChangeStagingChannels{
		ResourceChangesChan: make(chan ResourceChangesMessage),
		ChildChangesChan:    make(chan ChildChangesMessage),
		LinkChangesChan:     make(chan LinkChangesMessage),
		CompleteChan:        make(chan changes.BlueprintChanges),
		ErrChan:             make(chan error),
	}
}

func normaliseBlueprintChanges(bpChanges *changes.BlueprintChanges) *changes.BlueprintChanges {

	normalisedChanges := &changes.BlueprintChanges{
		NewResources:     normaliseResourceChangeMap(bpChanges.NewResources),
		ResourceChanges:  normaliseResourceChangeMap(bpChanges.ResourceChanges),
		RemovedResources: internal.OrderStringSlice(bpChanges.RemovedResources),
		RemovedLinks:     internal.OrderStringSlice(bpChanges.RemovedLinks),
		NewChildren:      normaliseNewChildMap(bpChanges.NewChildren),
		ChildChanges:     normaliseChildChangesMap(bpChanges.ChildChanges),
		RemovedChildren:  internal.OrderStringSlice(bpChanges.RemovedChildren),
		RecreateChildren: internal.OrderStringSlice(bpChanges.RecreateChildren),
		NewExports:       bpChanges.NewExports,
		ExportChanges:    bpChanges.ExportChanges,
		RemovedExports:   internal.OrderStringSlice(bpChanges.RemovedExports),
		UnchangedExports: internal.OrderStringSlice(bpChanges.UnchangedExports),
		ResolveOnDeploy:  internal.OrderStringSlice(bpChanges.ResolveOnDeploy),
		MetadataChanges:  normaliseMetadataChanges(bpChanges.MetadataChanges),
	}

	return normalisedChanges
}

func normaliseChildChangesMap(
	childChangesMap map[string]changes.BlueprintChanges,
) map[string]changes.BlueprintChanges {
	newMap := map[string]changes.BlueprintChanges{}
	for childName, child := range childChangesMap {
		newMap[childName] = *normaliseBlueprintChanges(&child)
	}
	return newMap
}

func normaliseResourceChangeMap(changeMap map[string]provider.Changes) map[string]provider.Changes {
	newChangeMap := map[string]provider.Changes{}
	for resourceName, resourceChange := range changeMap {
		newChangeMap[resourceName] = *internal.NormaliseResourceChanges(&resourceChange, false)
	}
	return newChangeMap
}

func normaliseNewChildMap(newChildMap map[string]changes.NewBlueprintDefinition) map[string]changes.NewBlueprintDefinition {
	newMap := map[string]changes.NewBlueprintDefinition{}
	for childName, child := range newChildMap {
		newMap[childName] = changes.NewBlueprintDefinition{
			NewResources: normaliseResourceChangeMap(child.NewResources),
			NewChildren:  normaliseNewChildMap(child.NewChildren),
			NewExports:   child.NewExports,
		}
	}
	return newMap
}

func normaliseMetadataChanges(metadataChanges changes.MetadataChanges) changes.MetadataChanges {
	return changes.MetadataChanges{
		NewFields:       internal.OrderFieldChanges(metadataChanges.NewFields),
		ModifiedFields:  internal.OrderFieldChanges(metadataChanges.ModifiedFields),
		UnchangedFields: internal.OrderStringSlice(metadataChanges.UnchangedFields),
		RemovedFields:   internal.OrderStringSlice(metadataChanges.RemovedFields),
	}
}
