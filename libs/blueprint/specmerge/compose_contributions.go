package specmerge

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ContributionInputs holds the links a resource's spec is composed from, grouped by what
// this deployment has done with each of them.
type ContributionInputs struct {
	// Produced holds the contributions of the links that ran, paired with the link that
	// produced each one.
	Produced []LinkResourceContribution
	// StoredLinks holds every link recorded against the instance, whose contributions are
	// read back from what each recorded at its last deploy.
	StoredLinks []state.LinkState
	// RemovedLinkNames holds the links this deployment removes, which keep only the
	// contributions they marked as outliving them.
	RemovedLinkNames []string
	// SupersededLinkNames holds the links whose stored contributions are not to be read,
	// because what they produced in this deployment is the whole of what they contribute.
	//
	// This is not the same as the links that appear in Produced. A link that contributes
	// and has withdrawn everything it used to contribute produces nothing, and reading its
	// stored contributions would reinstate exactly what it withdrew. A link that never
	// contributed produces nothing either, and its stored mappings are all there is of it,
	// so the two cannot be told apart by what they produced and have to be told apart by
	// whether they contribute at all.
	SupersededLinkNames []string
}

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
//   - a link that contributes and ran supplies what it just produced, and nothing it
//     recorded before. A field it contributed last time and did not contribute now has
//     been withdrawn, and reading its stored contributions alongside the new ones would
//     put the old value back. This holds even where it produced nothing at all, which is
//     a link that has withdrawn every contribution it used to make.
//   - a link that did not run supplies what it recorded at its last deploy, since it still
//     targets the resource and has not been asked to say so again.
//   - a link being removed supplies only the contributions it marked as outliving it.
//     Everything else it contributed goes, which is what removing a link means.
func ComposeResourceContributions(
	spec *core.MappingNode,
	resourceName string,
	inputs *ContributionInputs,
) (*ContributionMergeResult, error) {
	contributions := slices.Clone(inputs.Produced)
	contributions = append(
		contributions,
		carriedOverContributions(resourceName, inputs)...,
	)

	// Merged in one pass rather than stored first and produced second, so that the order
	// contributions land in depends on the links themselves and not on which of them a
	// deployment happened to run. Appending in two passes would move an element every
	// time its link went from running to not running.
	return MergeResourceContributions(spec, resourceName, contributions)
}

func carriedOverContributions(
	resourceName string,
	inputs *ContributionInputs,
) []LinkResourceContribution {
	carriedOver := []LinkResourceContribution{}
	for _, link := range inputs.StoredLinks {
		if slices.Contains(inputs.SupersededLinkNames, link.Name) {
			continue
		}

		retainedOnly := slices.Contains(inputs.RemovedLinkNames, link.Name)
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

		// A value the link's data no longer holds is carried through with the
		// contribution rather than passed over, so the merge reports it against the link
		// it belongs to. Dropping it here would leave the caller composing a spec that
		// silently lacks the field, which is the contribution being retracted by an
		// absence rather than by anyone deciding to retract it.
		value, _ := core.GetPathValue(
			core.AddRootToPath(linkDataPath),
			&core.MappingNode{Fields: link.Data},
			core.MappingNodeMaxTraverseDepth,
		)

		contributions = append(contributions, LinkResourceContribution{
			LinkName:     link.Name,
			LinkDataPath: linkDataPath,
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
