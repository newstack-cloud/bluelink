package container

import (
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// The field in a link's data that holds the contributions the framework keeps on the
// link's behalf, kept apart from the data a link writes for itself so the two cannot
// collide over a name.
const linkContributionsDataField = "contributions"

// ContributedLinkData holds what a link's produced contributions become once the framework
// takes them over which includes the values in the link's own data,
// the mappings saying which resource field each value feeds,
// and the records saying how each one is applied.
//
// A link that writes a resource imperatively records all of this itself. A link that
// contributes hands over the values and the framework records them, which is what lets a
// deployment that does not run the link rebuild what it contributed.
type ContributedLinkData struct {
	Data                 map[string]*core.MappingNode
	ResourceDataMappings map[string]string
	ContributionRecords  map[string]state.ContributionRecord
}

// ContributionsToLinkData records a link's contributions in the form its state holds.
//
// The values are stored as a list under a single field of the link's data rather than
// against paths derived from the resource fields they feed. A resource field path is not
// usable as a link data path, it holds the dots and brackets that a path is written in, so
// a contribution to "spec.policies[0].statements" would be read back as a traversal rather
// than as one field's name.
//
// The list is ordered by the resource and field each contribution targets, so a link
// producing the same contributions twice records them in the same order and the mappings
// keep pointing at the same entries.
func ContributionsToLinkData(
	contributions []*provider.ResourceContribution,
) *ContributedLinkData {
	ordered := orderedContributions(contributions)

	if len(ordered) == 0 {
		// A link that doesn't contribute anything, doesn't need to record anything.
		return &ContributedLinkData{}
	}

	contributed := &ContributedLinkData{
		Data:                 map[string]*core.MappingNode{},
		ResourceDataMappings: map[string]string{},
		ContributionRecords:  map[string]state.ContributionRecord{},
	}

	items := make([]*core.MappingNode, 0, len(ordered))
	for index, contribution := range ordered {
		items = append(items, contribution.Value)

		mappingKey := contributionMappingKey(contribution)
		contributed.ResourceDataMappings[mappingKey] = fmt.Sprintf(
			"%s[%d]",
			linkContributionsDataField,
			index,
		)
		contributed.ContributionRecords[mappingKey] = state.ContributionRecord{
			Action:          int(contribution.Action),
			RetainOnRemoval: contribution.RetainOnRemoval,
		}
	}

	contributed.Data[linkContributionsDataField] = &core.MappingNode{Items: items}

	return contributed
}

func orderedContributions(
	contributions []*provider.ResourceContribution,
) []*provider.ResourceContribution {
	ordered := []*provider.ResourceContribution{}
	for _, contribution := range contributions {
		if contribution != nil && contribution.Value != nil {
			ordered = append(ordered, contribution)
		}
	}

	slices.SortStableFunc(ordered, func(a, b *provider.ResourceContribution) int {
		if resourceOrder := strings.Compare(a.ResourceName, b.ResourceName); resourceOrder != 0 {
			return resourceOrder
		}

		return strings.Compare(a.FieldPath, b.FieldPath)
	})

	return ordered
}

func contributionMappingKey(contribution *provider.ResourceContribution) string {
	return fmt.Sprintf("%s::%s", contribution.ResourceName, contribution.FieldPath)
}

// LinkContributionsFor pairs a link's produced contributions with the link that made them,
// which is the form the merge takes them in.
func LinkContributionsFor(
	linkName string,
	contributions []*provider.ResourceContribution,
) []specmerge.LinkResourceContribution {
	paired := make([]specmerge.LinkResourceContribution, 0, len(contributions))
	for _, contribution := range contributions {
		if contribution == nil {
			continue
		}

		paired = append(paired, specmerge.LinkResourceContribution{
			LinkName:     linkName,
			Contribution: contribution,
		})
	}

	return paired
}
