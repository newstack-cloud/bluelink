package specmerge

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ComposeResourceContributions builds the spec a resource's merged update carries, from
// the contributions of every link that targets it rather than only the links a deployment
// happens to be touching.
//
// A merged update states the resource's whole desired spec, so a contribution missing from
// it is a contribution withdrawn. Building one from the links in the change set alone would
// deploy a single link and silently revoke what every other link contributed to the same
// resource.
//
// Contributions come from three places, and which one applies is decided per link:
//
//   - a link that ran supplies what it just produced, and nothing it recorded before. A
//     field it contributed last time and did not contribute now has been withdrawn, and
//     reading its stored contributions alongside the new ones would put the old value back.
//   - a link that did not run supplies what it recorded at its last deploy, since it still
//     targets the resource and has not been asked to say so again.
//   - a link being removed supplies only the contributions it marked as outliving it.
//     Everything else it contributed goes, which is what removing a link means.
func ComposeResourceContributions(
	spec *core.MappingNode,
	resourceName string,
	produced []LinkResourceContribution,
	storedLinks []state.LinkState,
	removedLinkNames []string,
) (*ContributionMergeResult, error) {
	contributions := slices.Clone(produced)
	contributions = append(
		contributions,
		carriedOverContributions(
			resourceName,
			produced,
			storedLinks,
			removedLinkNames,
		)...,
	)

	// Merged in one pass rather than stored first and produced second, so that the order
	// contributions land in depends on the links themselves and not on which of them a
	// deployment happened to run. Appending in two passes would move an element every
	// time its link went from running to not running.
	return MergeResourceContributions(spec, resourceName, contributions)
}

func carriedOverContributions(
	resourceName string,
	produced []LinkResourceContribution,
	storedLinks []state.LinkState,
	removedLinkNames []string,
) []LinkResourceContribution {
	ranThisDeployment := map[string]bool{}
	for _, contribution := range produced {
		ranThisDeployment[contribution.LinkName] = true
	}

	carriedOver := []LinkResourceContribution{}
	for _, link := range storedLinks {
		if ranThisDeployment[link.Name] {
			continue
		}

		retainedOnly := slices.Contains(removedLinkNames, link.Name)
		carriedOver = append(carriedOver, StoredResourceContributions(
			resourceName,
			link,
			retainedOnly,
		)...)
	}

	return carriedOver
}

// StoredResourceContributions reads the contributions a link recorded against a resource
// at its last deploy, so that a link which does not run in a deployment can still say what
// it needs the resource's spec to hold.
//
// The value comes from the link's data through the mapping that records where it lives,
// and how to apply it comes from the contribution record saved beside that mapping. A
// mapping with no record is applied as a replacement, which is what a contribution made
// before records were kept would have done.
//
// With retainedOnly set, only the contributions marked as outliving the link are returned,
// which is what a link being removed still contributes.
func StoredResourceContributions(
	resourceName string,
	link state.LinkState,
	retainedOnly bool,
) []LinkResourceContribution {
	contributions := []LinkResourceContribution{}

	for resourceFieldPath, linkDataPath := range link.ResourceDataMappings {
		fieldPath, ownedByResource := resourceFieldPathFor(resourceFieldPath, resourceName)
		if !ownedByResource {
			continue
		}

		record := link.ContributionRecords[resourceFieldPath]
		if retainedOnly && !record.RetainOnRemoval {
			continue
		}

		value, _ := core.GetPathValue(
			core.AddRootToPath(linkDataPath),
			&core.MappingNode{Fields: link.Data},
			core.MappingNodeMaxTraverseDepth,
		)
		if value == nil {
			// Left to the merge to report, so a contribution whose value has gone missing
			// is named against its link in the same way as one that will not apply.
			continue
		}

		contributions = append(contributions, LinkResourceContribution{
			LinkName: link.Name,
			Contribution: &provider.ResourceContribution{
				ResourceName:    resourceName,
				FieldPath:       fieldPath,
				Value:           value,
				Action:          provider.ContributionAction(record.Action),
				RetainOnRemoval: record.RetainOnRemoval,
			},
		})
	}

	return contributions
}
