package container

import (
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourceContributionSources holds everything a resource's merged update is built from,
// separated by where each link's contributions come from.
//
// A link that ran in this deployment has produced its contributions and they are held
// against it. A link that did not run has not been asked, so what it recorded at its last
// deployment is used. A link being removed is gone from the blueprint and keeps only what it
// marked as outliving it.
type ResourceContributionSources struct {
	// Produced holds the contributions of the links that ran, paired with the link that
	// produced each one.
	Produced []specmerge.LinkResourceContribution
	// Stored holds the links whose contributions are read back from state, which is every
	// link in the instance that this deployment did not run.
	Stored []state.LinkState
	// RemovedLinkNames holds the links this deployment removes, whose contributions go
	// with them apart from any marked to outlive them.
	RemovedLinkNames []string
	// SupersededLinkNames holds the links whose stored contributions are not to be read
	// because what they produced in this deployment is all of what they contribute.
	SupersededLinkNames []string
}

// CollectResourceContributionSources gathers what a resource's merged update is built
// from, across every link that targets the resource rather than only the links this
// deployment is touching.
//
// A merged update states the resource's whole desired spec, so a link left out of this is
// a link whose contributions are withdrawn from the resource when the update lands.
func CollectResourceContributionSources(
	deployCtx *DeployContext,
	contributingLinkNames []string,
) *ResourceContributionSources {
	sources := &ResourceContributionSources{
		Produced: []specmerge.LinkResourceContribution{},
		Stored:   []state.LinkState{},
		// Removals are not carried here. A link this deployment removes is destroyed
		// before any resource is deployed, so by the time a resource's contributions are
		// composed the link is out of state and never reaches the composition at all.
		RemovedLinkNames: nil,
	}

	for _, linkName := range contributingLinkNames {
		result := deployCtx.State.GetLinkDeployResult(linkName)
		if result == nil {
			// The link has not produced anything, whatever the join says about it, so
			// what it last recorded is still the best account of what it contributes.
			continue
		}

		sources.Produced = append(
			sources.Produced,
			LinkContributionsFor(linkName, result.Contributions)...,
		)
	}

	sources.Stored = storedLinkStates(deployCtx)
	sources.SupersededLinkNames = supersededLinkNames(deployCtx, contributingLinkNames)

	return sources
}

// ComposeMergedResourceSpec builds the spec a layer's update carries, from the resource's
// own declared spec and the contributions of every link that targets it.
//
// The links deeper in the capability ordering than the layer have not run, so what they
// recorded at their last deployment is read back rather than dropped. That is what keeps an
// earlier layer from stripping the fields a later one will restate, for example, the environment
// variables an access link contributed last time stay on the function while the layer
// carrying its network attachment lands.
func ComposeMergedResourceSpec(
	deployCtx *DeployContext,
	layer ContributionLayer,
	declaredSpec *core.MappingNode,
	contributingLinkNames []string,
) (*specmerge.ContributionMergeResult, error) {
	sources := CollectResourceContributionSources(deployCtx, contributingLinkNames)

	return specmerge.ComposeResourceContributions(
		declaredSpec,
		layer.ResourceName,
		&specmerge.ContributionInputs{
			Produced:            sources.Produced,
			StoredLinks:         sources.Stored,
			RemovedLinkNames:    sources.RemovedLinkNames,
			SupersededLinkNames: sources.SupersededLinkNames,
		},
	)
}

// LinkContributorsFor reports which links contributed to each field of a resource, for
// the update that carries those contributions to say why each field holds what it does.
//
// Every link that contributed to a field is named, not just one of them. A shared
// execution role holds a statement from each of the links that needs one, all appended to
// the same field, and naming a single contributor would leave the rest unaccounted for.
func LinkContributorsFor(
	resourceName string,
	sources *ResourceContributionSources,
) map[string][]string {
	contributors := map[string][]string{}

	addContributor := func(linkName string, contribution *provider.ResourceContribution) {
		if contribution == nil || contribution.ResourceName != resourceName {
			return
		}

		if !slices.Contains(contributors[contribution.FieldPath], linkName) {
			contributors[contribution.FieldPath] = append(
				contributors[contribution.FieldPath],
				linkName,
			)
		}
	}

	for _, produced := range sources.Produced {
		addContributor(produced.LinkName, produced.Contribution)
	}

	for _, linkState := range sources.Stored {
		if slices.Contains(sources.SupersededLinkNames, linkState.Name) {
			continue
		}

		retainedOnly := slices.Contains(sources.RemovedLinkNames, linkState.Name)
		for _, stored := range specmerge.StoredResourceContributions(
			resourceName,
			linkState,
			retainedOnly,
		) {
			addContributor(stored.LinkName, stored.Contribution)
		}
	}

	if len(contributors) == 0 {
		return nil
	}

	// Sorted so a field contributed to by several links names them in the same order on
	// every run, which anything rendering the reason a field holds what it does has to be.
	for fieldPath := range contributors {
		slices.Sort(contributors[fieldPath])
	}

	return contributors
}

func storedLinkStates(deployCtx *DeployContext) []state.LinkState {
	if deployCtx.InstanceStateSnapshot == nil {
		return nil
	}

	linkStates := []state.LinkState{}
	for _, linkState := range deployCtx.InstanceStateSnapshot.Links {
		if linkState != nil {
			linkStates = append(linkStates, *linkState)
		}
	}

	// Sorted so a resource's merged update is composed from its links in the same order on
	// every run, since the order contributions are applied in decides where an appended
	// one ends up.
	slices.SortFunc(linkStates, func(a, b state.LinkState) int {
		return strings.Compare(a.Name, b.Name)
	})

	return linkStates
}

// The links whose stored contributions will be replaced by the current deployment.
func supersededLinkNames(
	deployCtx *DeployContext,
	contributingLinkNames []string,
) []string {
	superseded := []string{}
	for _, linkName := range contributingLinkNames {
		if deployCtx.State.GetLinkDeployResult(linkName) != nil {
			superseded = append(superseded, linkName)
		}
	}

	return superseded
}
