package container

import (
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

const (
	// How often a deployment is checked for having stopped making progress.
	stalledDeploymentCheckInterval = 15 * time.Second
	// How long the deployment has to look stalled before it is reported as one.
	//
	// This is generous on purpose.
	// A deployment that is briefly between elements looks identical
	// to one that will never move again, and the cost of the two mistakes is
	// not symmetric. Waiting another minute to report a genuine stall costs a minute,
	// while failing a healthy deployment because nothing happened to be in flight for a
	// moment destroys work that was going to succeed.
	stalledDeploymentGracePeriod = 60 * time.Second
)

// Reports a deployment that is waiting for elements it will never deploy.
//
// The event loop finishes when it has as many completions as the change set has elements.
// Nothing bounds that wait, every other stall in the deployment has a timeout, and an
// element that is never dispatched has none, so the deployment sits until the engine's own
// deployment timeout with no indication of what it is waiting for.
//
// The condition is decidable. If nothing is in flight, no completion can arrive, and
// without a completion nothing further is dispatched, because that is the only thing that
// releases the next elements. A deployment in that state with completions outstanding will
// stay there.
type deploymentStallDetector struct {
	elementsToDeploy int
	stalledSince     time.Time
	clock            core.Clock
}

func newDeploymentStallDetector(
	elementsToDeploy int,
	clock core.Clock,
) *deploymentStallDetector {
	return &deploymentStallDetector{
		elementsToDeploy: elementsToDeploy,
		clock:            clock,
	}
}

// Reports the names of the elements the deployment is waiting for once it has been stalled
// for longer than the grace period, and nil while it is making progress or has not been
// stalled long enough to be sure.
func (d *deploymentStallDetector) check(
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	draining bool,
) []string {
	if !d.looksStalled(deployCtx, finished, draining) {
		d.stalledSince = time.Time{}
		return nil
	}

	if d.stalledSince.IsZero() {
		d.stalledSince = d.clock.Now()
		return nil
	}

	if d.clock.Since(d.stalledSince) < stalledDeploymentGracePeriod {
		return nil
	}

	return outstandingElementNames(
		deployCtx.InputChanges,
		deployCtx.State.ContributionLayers(),
		finished,
	)
}

// A draining deployment is excluded because it is already ending and counts only the
// elements it started, so outstanding completions there are expected rather than a stall.
func (d *deploymentStallDetector) looksStalled(
	deployCtx *DeployContext,
	finished map[string]*deployUpdateMessageWrapper,
	draining bool,
) bool {
	if draining || len(finished) >= d.elementsToDeploy {
		return false
	}

	if len(deployCtx.State.GetInFlightElements()) > 0 {
		return false
	}

	// A link the scheduler is holding can still be released by a link completing, which
	// is work in progress even though the link is not in flight yet.
	return !deployCtx.LinkScheduler.HasPendingLinks()
}

// The elements in the change set that have not reported a completion, which is what the
// deployment is waiting for.
func outstandingElementNames(
	inputChanges *changes.BlueprintChanges,
	layers []ContributionLayer,
	finished map[string]*deployUpdateMessageWrapper,
) []string {
	if inputChanges == nil {
		return nil
	}

	outstanding := []string{}
	for resourceName := range inputChanges.NewResources {
		outstanding = appendIfUnfinished(outstanding, core.ResourceElementID(resourceName), finished)
	}
	for resourceName := range inputChanges.ResourceChanges {
		outstanding = appendIfUnfinished(outstanding, core.ResourceElementID(resourceName), finished)
	}
	for childName := range inputChanges.NewChildren {
		outstanding = appendIfUnfinished(outstanding, core.ChildElementID(childName), finished)
	}
	for childName := range inputChanges.ChildChanges {
		outstanding = appendIfUnfinished(outstanding, core.ChildElementID(childName), finished)
	}
	for _, childName := range inputChanges.RecreateChildren {
		outstanding = appendIfUnfinished(outstanding, core.ChildElementID(childName), finished)
	}

	// Links are awaited alongside resources and children, so a stall caused by one has to
	// name it. The sets here mirror those counted by countElementsToDeploy, or the report
	// would either miss a link the deployment is waiting for or name one it is not.
	for resourceName, resourceChanges := range inputChanges.NewResources {
		outstanding = appendOutstandingLinks(
			outstanding, resourceName, resourceChanges.NewOutboundLinks, finished,
		)
	}
	for resourceName, resourceChanges := range inputChanges.ResourceChanges {
		outstanding = appendOutstandingLinks(
			outstanding, resourceName, resourceChanges.NewOutboundLinks, finished,
		)
		outstanding = appendOutstandingLinks(
			outstanding, resourceName, resourceChanges.OutboundLinkChanges, finished,
		)
	}

	// Mirrors countElementsToDeploy, which waits for each update carrying contributions.
	for _, layer := range layers {
		outstanding = appendIfUnfinished(
			outstanding,
			contributionLayerElementID(layer),
			finished,
		)
	}

	return outstanding
}

// Both link change sets are keyed by the linked-to resource, with the linked-from resource
// implied by the change set they hang off, so the logical name is built from the pair.
func appendOutstandingLinks(
	outstanding []string,
	resourceName string,
	linkChanges map[string]provider.LinkChanges,
	finished map[string]*deployUpdateMessageWrapper,
) []string {
	for targetName := range linkChanges {
		outstanding = appendIfUnfinished(
			outstanding,
			linkElementID(core.LogicalLinkName(resourceName, targetName)),
			finished,
		)
	}

	return outstanding
}

func appendIfUnfinished(
	names []string,
	elementName string,
	finished map[string]*deployUpdateMessageWrapper,
) []string {
	if _, done := finished[elementName]; done {
		return names
	}

	return append(names, elementName)
}
