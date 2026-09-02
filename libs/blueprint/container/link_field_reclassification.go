package container

import (
	"context"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Reports a resource field a link contributes as changing when the link says the value
// behind it will change, having compared it against nothing of the sort.
//
// A resource's two sides are composed from link data stored at the last deploy, so a
// contribution whose value is about to change is compared against its own old value and
// reports no difference. The link knows better: it is staged with both endpoints' change
// sets in hand, and says so in its own changes, against the path in its link data rather
// than against the resource field that path feeds. The two sit in one change set, one
// saying the value will change on deploy and the other saying the field is unchanged.
//
// ResourceDataMappings spans them, so this reads the link's answer and applies it to the
// resource's fields. It runs over the assembled change set because a link's changes and
// the changes of the resource it contributes to are staged separately, and both have to be
// in hand.
func (c *defaultBlueprintContainer) reclassifyLinkOwnedFields(
	ctx context.Context,
	instanceID string,
	stagingState ChangeStagingState,
	parallelGroups [][]*DeploymentNode,
) error {
	// A blueprint that has never been deployed has no links holding contributions, and
	// every field of every resource in it is new.
	if instanceID == "" {
		return nil
	}

	instance, err := c.stateContainer.Instances().Get(ctx, instanceID)
	if err != nil {
		if state.IsInstanceNotFound(err) {
			return nil
		}

		return err
	}

	linkStates := linkStatesFrom(instance.Links)
	if len(linkStates) == 0 {
		return nil
	}

	resourceNames := resourceNamesInGroups(parallelGroups)
	stagedLinkChanges := collectStagedLinkChanges(resourceNames, stagingState)

	for _, resourceName := range resourceNames {
		resourceChanges := stagingState.GetResourceChanges(resourceName)
		if resourceChanges == nil {
			continue
		}

		reclassifyResourceLinkFields(
			resourceChanges,
			resourceName,
			linkStates,
			stagedLinkChanges,
		)
	}

	return nil
}

func resourceNamesInGroups(parallelGroups [][]*DeploymentNode) []string {
	resourceNames := []string{}
	for _, group := range parallelGroups {
		for _, node := range group {
			if node.Type() == DeploymentNodeTypeResource {
				resourceNames = append(resourceNames, node.ChainLinkNode.ResourceName)
			}
		}
	}

	return resourceNames
}

func linkStatesFrom(links map[string]*state.LinkState) []state.LinkState {
	linkStates := []state.LinkState{}
	for _, linkState := range links {
		if linkState != nil {
			linkStates = append(linkStates, *linkState)
		}
	}

	return linkStates
}

// The changes staged for every link in the blueprint, by logical link name.
//
// Link changes hang off the resource the link is from, keyed by the resource it links to,
// so the logical name has to be rebuilt from the pair to be looked up by a resource on the
// other end of it, or by one that is neither.
func collectStagedLinkChanges(
	resourceNames []string,
	stagingState ChangeStagingState,
) map[string]*provider.LinkChanges {
	staged := map[string]*provider.LinkChanges{}

	for _, resourceName := range resourceNames {
		resourceChanges := stagingState.GetResourceChanges(resourceName)
		if resourceChanges == nil {
			continue
		}

		for targetName, linkChanges := range resourceChanges.NewOutboundLinks {
			staged[core.LogicalLinkName(resourceName, targetName)] = &linkChanges
		}
		for targetName, linkChanges := range resourceChanges.OutboundLinkChanges {
			staged[core.LogicalLinkName(resourceName, targetName)] = &linkChanges
		}
	}

	return staged
}

// Applied in place, before resources with no changes are filtered out of the change set. A
// resource whose only change is a contribution the link says will move would otherwise be
// dropped before it could be reported.
func reclassifyResourceLinkFields(
	resourceChanges *provider.Changes,
	resourceName string,
	linkStates []state.LinkState,
	stagedLinkChanges map[string]*provider.LinkChanges,
) {
	sources := specmerge.LinkFieldSources(resourceName, linkStates)
	if len(sources) == 0 {
		return
	}

	stillUnchanged := []string{}
	for _, fieldPath := range resourceChanges.UnchangedFields {
		if linkWillChangeField(fieldPath, sources, stagedLinkChanges) {
			resourceChanges.FieldChangesKnownOnDeploy = append(
				resourceChanges.FieldChangesKnownOnDeploy,
				fieldPath,
			)
			continue
		}

		stillUnchanged = append(stillUnchanged, fieldPath)
	}
	resourceChanges.UnchangedFields = stillUnchanged
}

// Only fields currently reported as unchanged are considered by the caller. A field
// already in ModifiedFields or NewFields is being surfaced as changing, which is the
// outcome wanted, and moving it would say less rather than more.
func linkWillChangeField(
	fieldPath string,
	sources map[string]specmerge.LinkFieldSource,
	stagedLinkChanges map[string]*provider.LinkChanges,
) bool {
	source, isLinkOwned := sources[fieldPath]
	if !isLinkOwned {
		return false
	}

	linkChanges, staged := stagedLinkChanges[source.LinkName]
	if !staged {
		return false
	}

	return linkChangesTouchPath(linkChanges, source.LinkDataPath)
}

// Whether a link's staged changes say anything about the given path in its data.
//
// Every category counts, including fields the link reports as known on deploy, since the
// question being answered is whether the value behind a resource field is settled rather
// than what it will become.
func linkChangesTouchPath(linkChanges *provider.LinkChanges, linkDataPath string) bool {
	path := normaliseLinkDataPath(linkDataPath)

	for _, fieldChange := range linkChanges.ModifiedFields {
		if fieldChange != nil && normaliseLinkDataPath(fieldChange.FieldPath) == path {
			return true
		}
	}
	for _, fieldChange := range linkChanges.NewFields {
		if fieldChange != nil && normaliseLinkDataPath(fieldChange.FieldPath) == path {
			return true
		}
	}

	return slices.ContainsFunc(linkChanges.RemovedFields, func(removed string) bool {
		return normaliseLinkDataPath(removed) == path
	}) || slices.ContainsFunc(
		linkChanges.FieldChangesKnownOnDeploy,
		func(knownOnDeploy string) bool {
			return normaliseLinkDataPath(knownOnDeploy) == path
		},
	)
}

// Link data paths are written both rooted and bare. CollectChanges strips the root before
// recording a change, while a link appending a path to its own changes usually does not
// add one, and a mapping holds the bare form. Comparing them without normalising would
// fail to match while looking entirely correct.
func normaliseLinkDataPath(path string) string {
	return strings.TrimPrefix(path, "$.")
}
