package validation

import (
	"fmt"
	"sort"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// A guarantee about one resource instance in the blueprint, which is the granularity
// capabilities are matched at.
type capabilityInstance struct {
	name     string
	resource string
}

// ValidateLinkCapabilities checks that every link declaring a capability it cannot
// function without has something in the blueprint to establish it.
//
// Presence is judged over the whole blueprint rather than over a change set. A provider
// link that exists but is left out of a deployment still counts, because it was left out
// on the grounds that nothing about it changed, so the guarantee it established is
// already in place.
//
// Requirements without MustExist are deliberately not checked. An access link requiring
// a network attachment must still be valid for a function that was never placed in a
// VPC; the requirement means "after any provider, if one exists".
func ValidateLinkCapabilities(
	linkCapabilities []*LinkInstanceCapabilities,
) []*core.Diagnostic {
	provided := map[capabilityInstance]bool{}
	for _, link := range linkCapabilities {
		for _, capability := range link.Provides {
			if instance, ok := link.resolve(capability); ok {
				provided[instance] = true
			}
		}
	}

	diagnostics := []*core.Diagnostic{}
	for _, link := range linkCapabilities {
		diagnostics = append(
			diagnostics,
			validateRequiredCapabilities(link, provided)...,
		)
	}

	return diagnostics
}

// LinkInstanceCapabilities holds the capabilities declared by one link between two named
// resources in a blueprint.
type LinkInstanceCapabilities struct {
	// LinkType is the type of the link, e.g. "aws/lambda/function::aws/sqs/queue".
	LinkType string
	// ResourceAName is the logical name of the resource on the A side of the link.
	ResourceAName string
	// ResourceBName is the logical name of the resource on the B side of the link.
	ResourceBName string
	// Provides holds the guarantees this link establishes once deployed.
	Provides []provider.LinkCapability
	// Requires holds the guarantees this link needs established before it runs.
	Requires []provider.LinkCapability
}

func (l *LinkInstanceCapabilities) resolve(
	capability provider.LinkCapability,
) (capabilityInstance, bool) {
	if capability.Name == "" {
		return capabilityInstance{}, false
	}

	switch capability.Resource {
	case provider.LinkPriorityResourceA:
		return capabilityInstance{
			name:     capability.Name,
			resource: l.ResourceAName,
		}, true
	case provider.LinkPriorityResourceB:
		return capabilityInstance{
			name:     capability.Name,
			resource: l.ResourceBName,
		}, true
	default:
		return capabilityInstance{}, false
	}
}

func validateRequiredCapabilities(
	link *LinkInstanceCapabilities,
	provided map[capabilityInstance]bool,
) []*core.Diagnostic {
	missing := []capabilityInstance{}
	for _, capability := range link.Requires {
		if !capability.MustExist {
			continue
		}

		instance, ok := link.resolve(capability)
		if ok && !provided[instance] {
			missing = append(missing, instance)
		}
	}

	sort.Slice(missing, func(i, j int) bool {
		return missing[i].name < missing[j].name
	})

	diagnostics := make([]*core.Diagnostic, 0, len(missing))
	for _, instance := range missing {
		diagnostics = append(diagnostics, &core.Diagnostic{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"The link between %q and %q requires the capability %q on resource %q,"+
					" which no link in this blueprint provides.",
				link.ResourceAName,
				link.ResourceBName,
				instance.name,
				instance.resource,
			),
		})
	}

	return diagnostics
}
