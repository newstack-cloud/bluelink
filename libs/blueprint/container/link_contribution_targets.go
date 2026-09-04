package container

import (
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// BuildLinkContributionTargets reports the resources each link declared it will contribute
// to, keyed by the logical name of the link.
//
// A target's merged update states that resource's full desired spec, so it cannot be built
// until every link contributing to the resource contributes. Knowing which links those
// are before any of them runs is what makes the wait finite, and a link declares it at
// change staging through the resource data mappings it expects to produce.
//
// A link that declares nothing contributes nothing and appears here with no targets, which
// is every link that has not moved to contributions.
func BuildLinkContributionTargets(
	blueprintChanges *changes.BlueprintChanges,
) map[string][]string {
	if blueprintChanges == nil {
		return map[string][]string{}
	}

	targets := map[string][]string{}
	collectLinkContributionTargets(blueprintChanges.NewResources, targets)
	collectLinkContributionTargets(blueprintChanges.ResourceChanges, targets)

	return targets
}

func collectLinkContributionTargets(
	resourceChanges map[string]provider.Changes,
	targets map[string][]string,
) {
	for resourceName, changeSet := range resourceChanges {
		for targetName, linkChanges := range changeSet.NewOutboundLinks {
			addLinkContributionTargets(
				core.LogicalLinkName(resourceName, targetName),
				linkChanges.ResourceDataMappings,
				targets,
			)
		}
		for targetName, linkChanges := range changeSet.OutboundLinkChanges {
			addLinkContributionTargets(
				core.LogicalLinkName(resourceName, targetName),
				linkChanges.ResourceDataMappings,
				targets,
			)
		}
	}
}

// The declared mappings are keyed by "{resourceName}::{fieldPath}", so the targets are the
// distinct resources named across them. A link contributing several fields of one resource
// is waited on once for that resource rather than once per field.
func addLinkContributionTargets(
	linkName string,
	resourceDataMappings map[string]string,
	targets map[string][]string,
) {
	if _, alreadyCollected := targets[linkName]; alreadyCollected {
		return
	}

	linkTargets := []string{}
	for resourceFieldPath := range resourceDataMappings {
		resourceName, named := contributionTargetName(resourceFieldPath)
		if !named {
			continue
		}

		if !slices.Contains(linkTargets, resourceName) {
			linkTargets = append(linkTargets, resourceName)
		}
	}

	// Sorted so the links a target is waiting on are reported in the same order on every
	// run, which a deadlock reported against a stalled deployment has to be.
	slices.Sort(linkTargets)
	targets[linkName] = linkTargets
}

func contributionTargetName(resourceFieldPath string) (string, bool) {
	resourceName, _, found := strings.Cut(resourceFieldPath, "::")
	if !found || resourceName == "" {
		return "", false
	}

	return resourceName, true
}

func contributionLayerElementID(layer ContributionLayer) string {
	return fmt.Sprintf("contributions(%s,%d)", layer.ResourceName, layer.Depth)
}
