package specmerge

import (
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// LinkResourceContribution pairs a contribution with the link that produced it.
//
// The link is not carried on the contribution itself, and it is needed both to order
// contributions that target the same field and to name the link responsible when one of
// them cannot be applied.
type LinkResourceContribution struct {
	// LinkName is the logical name of the link that produced the contribution.
	LinkName string
	// Contribution is what the link needs the resource's spec to say.
	Contribution *provider.ResourceContribution
	// LinkDataPath is where in the link's data the value was read from, for a
	// contribution read back from state. It is empty for one a link has just produced,
	// which carries its value directly.
	//
	// Held for reporting as a contribution that cannot be applied names the path it came
	// from, which is what says whether the link's data is missing the value or the
	// resource will not take it.
	LinkDataPath string
}

// ContributionMergeResult holds a resource's spec with the contributions made to it
// applied, along with the ones that could not be.
type ContributionMergeResult struct {
	// Spec is a copy of the spec passed in, with every contribution that could be applied
	// applied to it.
	Spec *core.MappingNode
	// Unresolved holds the contributions that could not be applied, each naming the link
	// responsible and why. A contribution is reported rather than raised for the same
	// reason a projection is, the caller cannot repair it, and refusing to merge would
	// make one link's bad contribution an issue for the resource as a whole.
	Unresolved []UnresolvedProjection
}

// MergeResourceContributions applies to a resource's spec the contributions links have
// produced for it, which is what the resource's merged update carries.
//
// Contributions targeting a resource other than the named one are ignored, so a caller
// can pass everything the deployment produced rather than filtering per resource first.
//
// The order contributions are applied in is part of the result rather than an
// implementation detail. Two links appending to one list would otherwise land in whichever
// order they finished in, and change staging compares arrays element by element, so the
// same deployment run twice would report elements as modified that nothing changed.
// Ordering by field path and then by link name gives an order that depends on neither the
// scheduling of the links nor the iteration of a map.
func MergeResourceContributions(
	spec *core.MappingNode,
	resourceName string,
	contributions []LinkResourceContribution,
) (*ContributionMergeResult, error) {
	result := &ContributionMergeResult{
		Spec:       core.CopyMappingNode(spec),
		Unresolved: []UnresolvedProjection{},
	}

	ordered := orderedContributionsFor(resourceName, contributions)
	for _, contribution := range ordered {
		err := applyResourceContribution(result, contribution)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func orderedContributionsFor(
	resourceName string,
	contributions []LinkResourceContribution,
) []LinkResourceContribution {
	forResource := []LinkResourceContribution{}
	for _, contribution := range contributions {
		if contribution.Contribution == nil {
			continue
		}

		if contribution.Contribution.ResourceName == resourceName {
			forResource = append(forResource, contribution)
		}
	}

	slices.SortStableFunc(forResource, func(a, b LinkResourceContribution) int {
		pathOrder := strings.Compare(
			a.Contribution.FieldPath,
			b.Contribution.FieldPath,
		)
		if pathOrder != 0 {
			return pathOrder
		}

		return strings.Compare(a.LinkName, b.LinkName)
	})

	return forResource
}

func applyResourceContribution(
	result *ContributionMergeResult,
	contribution LinkResourceContribution,
) error {
	if contribution.Contribution.Value == nil {
		result.Unresolved = append(result.Unresolved, unresolvedContribution(
			contribution,
			"the link contributed no value for this field",
		))
		return nil
	}

	switch contribution.Contribution.Action {
	case provider.ContributionActionAppend:
		return appendResourceContribution(result, contribution)
	default:
		return setResourceContribution(result, contribution)
	}
}

func setResourceContribution(
	result *ContributionMergeResult,
	contribution LinkResourceContribution,
) error {
	err := core.InjectPathValueReplaceFields(
		ResourceSpecPath(contribution.Contribution.FieldPath),
		contribution.Contribution.Value,
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		result.Unresolved = append(result.Unresolved, unresolvedContribution(
			contribution,
			err.Error(),
		))
	}

	return nil
}

// Appending is a read of the list at the path followed by a write of the whole list back,
// since injecting at a path replaces what is there rather than adding to it. A path
// holding nothing yet becomes a list of one, which is what lets the first link to
// contribute to a list not have to know it is the first.
func appendResourceContribution(
	result *ContributionMergeResult,
	contribution LinkResourceContribution,
) error {
	path := ResourceSpecPath(contribution.Contribution.FieldPath)

	existing, _ := core.GetPathValue(
		path,
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)

	items := []*core.MappingNode{}
	if existing != nil {
		if existing.Items == nil {
			// The path holds something that is not a list, so appending to it would
			// discard whatever is there. Reported rather than overwritten: the resource
			// says one thing about this field and the link says another, and only the
			// author of either can settle it.
			result.Unresolved = append(result.Unresolved, unresolvedContribution(
				contribution,
				"the field already holds a value that is not a list, "+
					"so the contribution cannot be added to it",
			))
			return nil
		}

		items = append(items, existing.Items...)
	}

	items = append(items, contribution.Contribution.Value)

	err := core.InjectPathValueReplaceFields(
		path,
		&core.MappingNode{Items: items},
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		result.Unresolved = append(result.Unresolved, unresolvedContribution(
			contribution,
			err.Error(),
		))
	}

	return nil
}

func unresolvedContribution(
	contribution LinkResourceContribution,
	reason string,
) UnresolvedProjection {
	return UnresolvedProjection{
		LinkName:          contribution.LinkName,
		ResourceFieldPath: contribution.Contribution.FieldPath,
		LinkDataPath:      contribution.LinkDataPath,
		Reason:            reason,
	}
}

// ConflictingContributions reports the fields of a resource that more than one link claims
// the whole value of.
//
// Two links appending to one list is the case appending exists for. Two links setting one
// field is not. Each states a whole value, the one applied last wins by an ordering that
// says nothing about which is right, and the deployment reports success having silently
// discarded one of them.
func ConflictingContributions(
	resourceName string,
	contributions []LinkResourceContribution,
) []string {
	settingLinks := map[string][]string{}
	for _, contribution := range orderedContributionsFor(resourceName, contributions) {
		if contribution.Contribution.Action != provider.ContributionActionSet {
			continue
		}

		fieldPath := contribution.Contribution.FieldPath
		if !slices.Contains(settingLinks[fieldPath], contribution.LinkName) {
			settingLinks[fieldPath] = append(
				settingLinks[fieldPath],
				contribution.LinkName,
			)
		}
	}

	conflicts := []string{}
	for fieldPath, linkNames := range settingLinks {
		if len(linkNames) > 1 {
			conflicts = append(conflicts, fmt.Sprintf(
				"%s is set by %s",
				fieldPath,
				strings.Join(linkNames, " and "),
			))
		}
	}

	slices.Sort(conflicts)

	return conflicts
}
