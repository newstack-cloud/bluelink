package links

import (
	"context"
	"fmt"
	"strings"

	bpcore "github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/schema"
	"github.com/two-hundred/celerity/libs/blueprint/speccore"
	"github.com/two-hundred/celerity/libs/common/core"
)

// SpecLinkInfo provides the interface for a service that provides
// information about the links in a blueprint.
// This is mostly useful for validating and loading a blueprint specification.
// This also provides information for the blueprint container to source
// the provider link implementations by resource types.
type SpecLinkInfo interface {
	// Links deals with determining the links for all link selectors
	// and metadata labels defined in the spec.
	// This produces a slice of tree structures that represents link chains in the spec
	// where each link in the chain contains the from and to resource names,
	// the labels in the spec that link them together and the provider.Link implementation.
	// This will return an error when a link defined in the spec
	// is not supported.
	Links(ctx context.Context) ([]*ChainLink, error)
	// Warnings provides a list of warnings for potential issues
	// with the links in a provided specification.
	Warnings(ctx context.Context) ([]string, error)
}

// ChainLink provides a node in a chain of links that contains the name
// of the current resource in the chain, selectors used to link it with
// other resources, the implementation for each outward link and the chain link
// nodes that the current resource links out to.
type ChainLink struct {
	// ResourceName is the unique name in the spec for the current
	// resource in the chain.
	ResourceName string
	// Resource holds the information about a resource at the blueprint spec schema-level,
	// most importantly the resource type that allows us to efficiently get a resource type
	// provider implementation for a link in a chain.
	Resource *schema.Resource
	// Selectors provides a mapping of the selector attribute to the resources
	// the current resource links to.
	// (e.g. "label::app:orderApi" -> ["createOrderFunction", "removeOrderFunction"])
	Selectors map[string][]string
	// LinkImplementations holds the link provider implementations keyed by resource name
	// for all the resources the current resource in the chain links
	// to.
	LinkImplementations map[string]provider.Link
	// LinksTo holds the chain link nodes for the resources
	// that the curent resource links to.
	LinksTo []*ChainLink
	// LinkedFrom holds the chain link nodes that link to the current resource.
	// This information is important to allow backtracking when the blueprint container
	// is deciding the order in which resources should be deployed.
	LinkedFrom []*ChainLink
	// Paths holds all the different "routes" to get to the current link in a set of chains.
	// These are known as materialised paths in the context of tree data structures.
	// Having this information here allows us to efficiently find out if
	// there is a relationship between two links at any depth in the chain.
	Paths []string
}

func (l *ChainLink) Equal(otherLink *ChainLink) bool {
	return l.ResourceName == otherLink.ResourceName
}

type defaultSpecLinkInfo struct {
	resourceProviders        map[string]provider.Provider
	spec                     speccore.BlueprintSpec
	chains                   []*ChainLink
	linkMap                  map[string]*ChainLink
	linksToCleanFromTopLevel []*ChainLink
	blueprintParams          bpcore.BlueprintParams
}

// NewDefaultLinkInfoProvider creates a new instance of
// the default implementation of a link info provider.
// This prepares all the information as a part of initialisation
// and validates the linking in the spec.
// The map of resource providers must be a map of provider resource name
// to a provider.
func NewDefaultLinkInfoProvider(
	resourceProviders map[string]provider.Provider,
	spec speccore.BlueprintSpec,
	blueprintParams bpcore.BlueprintParams,
) (SpecLinkInfo, error) {
	return &defaultSpecLinkInfo{
		resourceProviders:        resourceProviders,
		spec:                     spec,
		chains:                   []*ChainLink{},
		linkMap:                  make(map[string]*ChainLink),
		linksToCleanFromTopLevel: []*ChainLink{},
		blueprintParams:          blueprintParams,
	}, nil
}

func (l *defaultSpecLinkInfo) Links(ctx context.Context) ([]*ChainLink, error) {
	resourcesGroupedBySelectors := GroupResourcesBySelector(l.spec)
	err := l.buildChainLinks(ctx, resourcesGroupedBySelectors)
	if err != nil {
		return l.chains, err
	}
	return l.chains, nil
}

func (l *defaultSpecLinkInfo) collectResourceNamesWithLinks() []string {
	resourceNamesWithLinks := []string{}
	for name := range l.linkMap {
		resourceNamesWithLinks = append(resourceNamesWithLinks, name)
	}
	return resourceNamesWithLinks
}

func (l *defaultSpecLinkInfo) buildChainLinks(
	ctx context.Context,
	groupedBySelector map[string]*SelectGroup,
) error {
	for selectorLabel, selectGroup := range groupedBySelector {
		err := l.addOrUpdateChainsForSelector(ctx, selectorLabel, selectGroup, groupedBySelector)
		if err != nil {
			return err
		}
	}

	standaloneResources := l.extractStandaloneResources(groupedBySelector)
	for _, standaloneResource := range standaloneResources {
		standaloneChainLink := &ChainLink{
			ResourceName: standaloneResource.Name,
			Resource:     standaloneResource.Resource,
			// Values will be filled in for link implementations, selectors and linksTo
			// when the candidate link appears as a selector in SelectGroup.SelectorResources.
			Selectors:           map[string][]string{},
			LinkImplementations: map[string]provider.Link{},
			LinksTo:             []*ChainLink{},
			LinkedFrom:          []*ChainLink{},
			Paths:               []string{},
		}
		l.chains = append(l.chains, standaloneChainLink)
	}

	// At this point we have built all the chains, only after we have built the chains
	// can we reliably detect both direct and indirect circular links.
	circularLinksInfo, err := l.findCircularLinks(ctx)
	if err != nil {
		return err
	}

	// Ensure we clean up the top-level links for chains
	// regardless of whether or not circular links
	// were detected.
	// Circular links should be left at the top-level as the entire chain
	// would be removed if all links are descendants of a given top-level
	// link that has been selected as a candidate for clean up.
	l.chains = core.Filter(
		l.chains,
		linkNotInList(l.linksToCleanFromTopLevel, circularLinksInfo),
	)

	circularLinkErrors := extractCircularLinkErrors(circularLinksInfo)
	if len(circularLinkErrors) > 0 {
		return errCircularLinks(circularLinkErrors)
	}

	// Correct the paths for sub-chains for links that have been cleaned up
	// from the top-level.
	// This MUST come after checking for circular links as it recursively
	// traverses through the cleaned up chains to correct paths.
	correctSubChainPaths(l.linksToCleanFromTopLevel)

	return nil
}

func (l *defaultSpecLinkInfo) extractStandaloneResources(
	resourcesGroupedBySelectors map[string]*SelectGroup,
) []*ResourceWithNameAndSelectors {
	standaloneResources := []*ResourceWithNameAndSelectors{}
	resourceNamesWithLinks := l.collectResourceNamesWithLinks()
	resources := map[string]*schema.Resource{}
	if l.spec.Schema().Resources != nil {
		resources = l.spec.Schema().Resources.Values
	}

	for resourceName, resource := range resources {
		if !core.SliceContainsComparable(resourceNamesWithLinks, resourceName) {
			standaloneResources = append(standaloneResources, &ResourceWithNameAndSelectors{
				Name:      resourceName,
				Resource:  resource,
				Selectors: []string{},
			})
		}
	}
	return standaloneResources
}

func (l *defaultSpecLinkInfo) addOrUpdateChainsForSelector(
	ctx context.Context,
	selectorLabel string,
	selectGroup *SelectGroup,
	selectGroupMappings map[string]*SelectGroup,
) (err error) {
	for _, selectorResource := range selectGroup.SelectorResources {
		chainLinkForResource := l.findChainLink(selectorResource.Name)
		_, err = l.addResourceChainToChains(
			ctx,
			selectGroup,
			selectGroupMappings,
			selectorResource,
			selectorLabel,
			chainLinkForResource,
		)
	}

	return err
}

// Adds a resource to chains if it has not already been created.
// Importantly, this needs to run regardless of whether or not the given resource
// has already been added to a chain, this is because we need to be able to fill in
// missing links from resources that have both inbound and outbound links.
func (l *defaultSpecLinkInfo) addResourceChainToChains(
	ctx context.Context,
	currentSelectGroup *SelectGroup,
	selectGroupMappings map[string]*SelectGroup,
	resource *ResourceWithNameAndSelectors,
	primarySelectorLabel string,
	existingChainLink *ChainLink,
) (*ChainLink, error) {
	chainLink := determineChainLink(existingChainLink, resource)
	l.linkMap[resource.Name] = chainLink
	resourceLinkCount := 0

	if resource.Resource.Metadata != nil && resource.Resource.Metadata.Labels != nil {
		for key, value := range resource.Resource.Metadata.Labels.Values {
			lookUpSelectorName := fmt.Sprintf("label::%s:%s", key, value)
			selectGroup, exists := selectGroupMappings[lookUpSelectorName]
			if exists {
				// When this resource is linked to by other resources using a selector
				// let's make sure we add it in the right places in the chains.
				selectorChainLinks, err := l.collectSelectorChainLinks(ctx, selectGroup, resource)
				if err != nil {
					return nil, err
				}
				if len(selectorChainLinks) > 0 {
					resourceLinkCount += 1
				}

				err = l.addLinkInChainsIfMissing(ctx, chainLink, selectorChainLinks, lookUpSelectorName)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if resourceLinkCount == 0 {
		// At this stage there is nothing that can link to the current resource,
		// so we'll make it the start link of a chain for now.
		// This can change later as we can't rely on ordering from iterating
		// over maps.
		l.chains = append(l.chains, chainLink)
	}

	// Now we know where to place the current resource in the chain, let's build out
	// the chain for the current resource.
	err := l.buildChainForLink(
		ctx,
		chainLink,
		resource,
		currentSelectGroup.CandidateResourcesForSelection,
		primarySelectorLabel,
	)

	return chainLink, err
}

func (l *defaultSpecLinkInfo) collectSelectorChainLinks(
	ctx context.Context,
	selectGroup *SelectGroup,
	targetResource *ResourceWithNameAndSelectors,
) ([]*ChainLink, error) {
	existingLinks := []*ChainLink{}
	for _, selectorResource := range selectGroup.SelectorResources {
		linkToTargetCheckInfo, err := l.checkCanLinkTo(ctx, selectorResource, targetResource)
		if err != nil {
			return nil, err
		}
		existingLinkForSelectorResource, linkExists := l.linkMap[selectorResource.Name]

		if linkToTargetCheckInfo.canLinkTo && linkToTargetCheckInfo.linkImplementation != nil && linkExists {
			existingLinks = append(existingLinks, existingLinkForSelectorResource)
		}
	}
	return existingLinks, nil
}

type linkCheckInfo struct {
	linkImplementation provider.Link
	canLinkTo          bool
}

func (l *defaultSpecLinkInfo) checkCanLinkTo(
	ctx context.Context,
	linkFromResource *ResourceWithNameAndSelectors,
	linkToResource *ResourceWithNameAndSelectors,
) (*linkCheckInfo, error) {
	linkFromResourceType := linkFromResource.Resource.Type.Value
	linkToResourceType := linkToResource.Resource.Type.Value
	resourceProvider, rpExists := l.resourceProviders[linkFromResourceType]
	if rpExists {
		linkImplementation, err := resourceProvider.Link(ctx, linkFromResourceType, linkToResourceType)
		if err != nil {
			return nil, err
		}
		linkFromResourceType, err := resourceProvider.Resource(ctx, linkFromResourceType)
		if err != nil {
			return nil, err
		}
		linkFromResourceOutput, err := linkFromResourceType.CanLinkTo(ctx, &provider.ResourceCanLinkToInput{
			Params: l.blueprintParams,
		})
		if err != nil {
			return nil, err
		}
		linkAllowed := core.SliceContainsComparable(linkFromResourceOutput.CanLinkTo, linkToResourceType)
		return &linkCheckInfo{
			linkImplementation: linkImplementation,
			canLinkTo:          linkAllowed,
		}, nil
	}
	return &linkCheckInfo{
		linkImplementation: nil,
		canLinkTo:          false,
	}, nil
}

func (l *defaultSpecLinkInfo) addLinkInChainsIfMissing(ctx context.Context, newLink *ChainLink, selectorChainLinks []*ChainLink, contextSelectorLabel string) error {
	for _, selectorChainLink := range selectorChainLinks {
		existingWithResourceName := core.Filter(
			selectorChainLink.LinksTo,
			checkLinkHasResourceName(newLink.ResourceName),
		)

		selectorAsIntermediaryResource := &ResourceWithNameAndSelectors{
			Name:     selectorChainLink.ResourceName,
			Resource: selectorChainLink.Resource,
		}
		candidateAsIntermediaryResource := &ResourceWithNameAndSelectors{
			Name:     newLink.ResourceName,
			Resource: newLink.Resource,
		}
		candidateLinkCheckInfo, err := l.checkCanLinkTo(
			ctx,
			selectorAsIntermediaryResource,
			candidateAsIntermediaryResource,
		)
		if err != nil {
			return err
		}

		selectorCanLinkToCandidate := candidateLinkCheckInfo.canLinkTo && candidateLinkCheckInfo.linkImplementation != nil

		if candidateLinkCheckInfo.canLinkTo && candidateLinkCheckInfo.linkImplementation == nil {
			return errMissingLinkImplementation(selectorAsIntermediaryResource, candidateAsIntermediaryResource)
		}

		if len(existingWithResourceName) == 0 && selectorCanLinkToCandidate {

			selectorChainLink.LinksTo = append(selectorChainLink.LinksTo, newLink)
			alreadyInLinkedFrom := len(core.Filter(
				newLink.LinkedFrom,
				checkLinkHasResourceName(selectorChainLink.ResourceName),
			)) > 0
			if !alreadyInLinkedFrom {
				newLink.LinkedFrom = append(newLink.LinkedFrom, selectorChainLink)
			}

			alreadyHasPathWithLinkedFrom := len(core.Filter(
				newLink.Paths,
				checkLinkHasParentInPaths(selectorChainLink.ResourceName),
			)) > 0
			if !alreadyHasPathWithLinkedFrom {
				newLink.Paths = append(newLink.Paths, buildLinkPaths(selectorChainLink)...)
			}

			selectorChainLink.LinkImplementations[newLink.ResourceName] = candidateLinkCheckInfo.linkImplementation
			resourceNamesBySelector := selectorChainLink.Selectors[contextSelectorLabel]
			resourceInSelectors := core.SliceContainsComparable(resourceNamesBySelector, newLink.ResourceName)
			if !resourceInSelectors {
				selectorChainLink.Selectors[contextSelectorLabel] = append(resourceNamesBySelector, newLink.ResourceName)
			}
		}
	}

	return nil
}

func (l *defaultSpecLinkInfo) findChainLink(resourceName string) *ChainLink {
	return l.linkMap[resourceName]
}

func (l *defaultSpecLinkInfo) buildChainForLink(
	ctx context.Context,
	startLink *ChainLink,
	startLinkResource *ResourceWithNameAndSelectors,
	candidateLinkedResources []*ResourceWithNameAndSelectors,
	contextSelectorLabel string,
) error {

	for _, candidateLinkedResource := range candidateLinkedResources {
		candidateLinkCheckInfo, err := l.checkCanLinkTo(
			ctx,
			startLinkResource,
			candidateLinkedResource,
		)
		if err != nil {
			return err
		}

		isSameResource := candidateLinkedResource.Name == startLinkResource.Name
		if !isSameResource {
			err := l.connectCandidateIfMeetsConditions(
				startLink,
				startLinkResource,
				candidateLinkedResource,
				candidateLinkCheckInfo,
				contextSelectorLabel,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *defaultSpecLinkInfo) connectCandidateIfMeetsConditions(
	startLink *ChainLink,
	startLinkResource *ResourceWithNameAndSelectors,
	candidateLinkedResource *ResourceWithNameAndSelectors,
	candidateLinkCheckInfo *linkCheckInfo,
	contextSelectorLabel string,
) error {
	// Instead of letting providers that are set up incorrectly through,
	// we'll stop if we dedect incosistencies between what resource providers say
	// resources can link to and the corresponding link implementations.
	if candidateLinkCheckInfo.canLinkTo && candidateLinkCheckInfo.linkImplementation == nil {
		return errMissingLinkImplementation(startLinkResource, candidateLinkedResource)
	}

	linkToChainLink, candidateLinkExists := l.linkMap[candidateLinkedResource.Name]

	if candidateLinkExists && candidateLinkCheckInfo.canLinkTo {
		// When the candidate already exists and can be linked to by the given start link,
		// it may have been assigned as a start of a chain
		// as it could've been added before a resource that links to it.
		// We need to make sure we correct this on the fly to prevent things like deploying
		// resources before those they are dependent on
		// because they are incorrectly assigned as the start of a chain.
		// For example, if an AWS Subnet is prepared before an AWS VPC resource, this blueprint container
		// will end up trying to deploy the subnet before the VPC
		// if we don't correct this as it is at the start of a chain.
		//
		// We need to collect all the links at the top-level so they can be cleaned out
		// after circular link detection. We need to retain top-level links that are referenced
		// further down in chains in order to detect circular dependencies.
		l.linksToCleanFromTopLevel = append(
			l.linksToCleanFromTopLevel,
			// There could be duplicates, but it does not matter
			// as we will be searching for the first occurrence of a link
			// in the clean up process.
			core.Filter(l.chains, checkLinkHasResourceName(candidateLinkedResource.Name))...,
		)
	} else if !candidateLinkExists {
		linkToChainLink = &ChainLink{
			ResourceName: candidateLinkedResource.Name,
			Resource:     candidateLinkedResource.Resource,
			// Values will be filled in for link implementations, selectors and linksTo
			// when the candidate link appears as a selector in SelectGroup.SelectorResources.
			Selectors:           core.SliceToMapKeys[string](candidateLinkedResource.Selectors),
			LinkImplementations: map[string]provider.Link{},
			LinksTo:             []*ChainLink{},
			LinkedFrom:          []*ChainLink{},
			Paths:               []string{},
		}
		l.linkMap[candidateLinkedResource.Name] = linkToChainLink
	}

	alreadyLinkedTo := len(core.Filter(
		startLink.LinksTo,
		checkLinkHasResourceName(candidateLinkedResource.Name),
	)) > 0

	if candidateLinkCheckInfo.canLinkTo && !alreadyLinkedTo {

		startLink.LinksTo = append(startLink.LinksTo, linkToChainLink)
		alreadyInLinkedFrom := len(core.Filter(
			linkToChainLink.LinkedFrom,
			checkLinkHasResourceName(startLink.ResourceName),
		)) > 0
		if !alreadyInLinkedFrom {
			linkToChainLink.LinkedFrom = append(linkToChainLink.LinkedFrom, startLink)
		}

		alreadyHasPathWithLinkedFrom := len(core.Filter(
			linkToChainLink.Paths,
			checkLinkHasParentInPaths(startLink.ResourceName),
		)) > 0
		if !alreadyHasPathWithLinkedFrom {
			linkToChainLink.Paths = append(linkToChainLink.Paths, buildLinkPaths(startLink)...)
		}

		startLink.LinkImplementations[candidateLinkedResource.Name] = candidateLinkCheckInfo.linkImplementation
		resourceNamesBySelector := startLink.Selectors[contextSelectorLabel]
		resourceInSelectors := core.SliceContainsComparable(resourceNamesBySelector, candidateLinkedResource.Name)
		if !resourceInSelectors {
			startLink.Selectors[contextSelectorLabel] = append(resourceNamesBySelector, candidateLinkedResource.Name)
		}
	}

	return nil
}

func (l *defaultSpecLinkInfo) findCircularLinks(ctx context.Context) ([]*circularLinkInfoItem, error) {
	circularLinkInfoItems := []*circularLinkInfoItem{}
	for _, chainLink := range l.chains {
		collected, err := collectCircularLinkInfo(
			ctx,
			chainLink,
			[]*ancestorLinkInfo{},
			l.blueprintParams,
		)
		if err != nil {
			return nil, err
		}

		circularLinkInfoItems = append(
			circularLinkInfoItems,
			collected...,
		)
	}

	return circularLinkInfoItems, nil
}

func (l *defaultSpecLinkInfo) Warnings(ctx context.Context) ([]string, error) {
	if len(l.chains) == 0 {
		// Build out chains if warnings are requested first.
		// This will always try to build the chain
		// if the blueprint spec has no resources.
		_, err := l.Links(ctx)
		if err != nil {
			return []string{}, nil
		}
	}
	// Ignore the second return value which is a slice to track whether
	// or not warnings have already been collected for a given resource name,
	// as the same resource can appear multiple times in a set of chains.
	warnings, _, err := l.collectWarnings(ctx, l.chains, []string{}, []string{})
	return warnings, err
}

func (l *defaultSpecLinkInfo) collectWarnings(
	ctx context.Context,
	chainLinks []*ChainLink,
	existingWarnings []string,
	alreadyCollectedResourceNames []string,
) ([]string, []string, error) {
	newWarnings := []string{}
	allCollectedResourceNamesSoFar := append([]string{}, alreadyCollectedResourceNames...)
	// Traverse through chain links to identify chain links that don't link out to any other
	// resources but perhaps should do so.
	// The critera for a warning is that if a resource represented
	// by a chain link can link to other resource
	// types and is not a "common terminal resource".
	for _, currentLink := range chainLinks {
		if len(currentLink.LinksTo) == 0 {
			currentLinkResourceType := currentLink.Resource.Type.Value
			resourceProvider, rpExists := l.resourceProviders[currentLinkResourceType]
			if rpExists {

				canLinkTo, isCommonTerminal, err := l.getResourceLinkInfo(ctx, resourceProvider, currentLinkResourceType)
				if err != nil {
					return []string{}, []string{}, err
				}

				shouldHaveOutboundLinks := len(canLinkTo) > 0 && !isCommonTerminal
				alreadyCollectedWarning := core.SliceContainsComparable(allCollectedResourceNamesSoFar, currentLink.ResourceName)
				if shouldHaveOutboundLinks && !alreadyCollectedWarning {
					newWarnings = append(newWarnings, fmt.Sprintf(
						"resource \"%s\" of type \"%s\" does not link out to any other"+
							" resources where in most use-cases a resource of type \"%s\" is expected to link to other resources",
						currentLink.ResourceName,
						currentLink.Resource.Type.Value,
						currentLink.Resource.Type.Value,
					))
					allCollectedResourceNamesSoFar = append(allCollectedResourceNamesSoFar, currentLink.ResourceName)
				}
			}
		} else {
			var err error
			newWarnings, allCollectedResourceNamesSoFar, err = l.collectWarnings(
				ctx,
				currentLink.LinksTo,
				newWarnings,
				allCollectedResourceNamesSoFar,
			)
			if err != nil {
				return []string{}, []string{}, err
			}
		}
	}
	return append(existingWarnings, newWarnings...), allCollectedResourceNamesSoFar, nil
}

func (l *defaultSpecLinkInfo) getResourceLinkInfo(
	ctx context.Context,
	resourceProvider provider.Provider,
	resourceType string,
) ([]string, bool, error) {
	resourceImplementation, err := resourceProvider.Resource(ctx, resourceType)
	if err != nil {
		return nil, false, err
	}
	canLinkToOutput, err := resourceImplementation.CanLinkTo(ctx, &provider.ResourceCanLinkToInput{
		Params: l.blueprintParams,
	})
	if err != nil {
		return nil, false, err
	}

	commonTerminalOutput, err := resourceImplementation.IsCommonTerminal(ctx, &provider.ResourceIsCommonTerminalInput{
		Params: l.blueprintParams,
	})
	if err != nil {
		return nil, false, err
	}

	return canLinkToOutput.CanLinkTo, commonTerminalOutput.IsCommonTerminal, nil
}

func checkLinkHasResourceName(searchForResourceName string) func(*ChainLink, int) bool {
	return func(link *ChainLink, index int) bool {
		return link.ResourceName == searchForResourceName
	}
}

func checkLinkHasParentInPaths(parentResourceName string) func(string, int) bool {
	return func(path string, index int) bool {
		return strings.Contains(path, fmt.Sprintf("/%s", parentResourceName))
	}
}

func checkLinkDoesNotHaveParentInPaths(parentResourceName string) func(string, int) bool {
	return func(path string, index int) bool {
		return !strings.Contains(path, fmt.Sprintf("/%s", parentResourceName))
	}
}

func determineChainLink(existingChainLink *ChainLink, resource *ResourceWithNameAndSelectors) *ChainLink {
	if existingChainLink != nil {
		return existingChainLink
	}

	return &ChainLink{
		ResourceName: resource.Name,
		Resource:     resource.Resource,
		// Add selected resources once they have been filtered
		// down to those that can be linked to by the current resource.
		// For now we'll just prepare an empty slice for each selector key.
		Selectors:           core.SliceToMapKeys[string](resource.Selectors),
		LinkImplementations: map[string]provider.Link{},
		LinksTo:             []*ChainLink{},
		LinkedFrom:          []*ChainLink{},
		Paths:               []string{},
	}
}

type ancestorLinkInfo struct {
	resourceName     string
	outboundLinkKind provider.LinkKind
}

type circularLinkInfoItem struct {
	resourceName string
	err          error
}

func collectCircularLinkInfo(
	ctx context.Context,
	chainLink *ChainLink,
	ancestors []*ancestorLinkInfo,
	blueprintParams bpcore.BlueprintParams,
) ([]*circularLinkInfoItem, error) {
	collected := []*circularLinkInfoItem{}
	for _, linkedTo := range chainLink.LinksTo {
		linkedToAncestorEntry := core.Find(ancestors, ancestorIsResource(linkedTo.ResourceName))

		if linkedToAncestorEntry != nil {
			hardCycle := isHardCycle(ancestors, linkedToAncestorEntry)

			if hardCycle {
				isIndirect := ancestors[len(ancestors)-1].resourceName != linkedTo.ResourceName
				collected = append(
					collected,
					&circularLinkInfoItem{
						err:          errCircularLink(chainLink, linkedTo, isIndirect),
						resourceName: chainLink.ResourceName,
					},
				)
			} else {
				collected = append(
					collected,
					&circularLinkInfoItem{
						err:          nil,
						resourceName: chainLink.ResourceName,
					},
				)
			}
		}

		// As soon as we reach any kind of circular link, we can't explore any further circular links,
		// otherwise we'll be infinitely going in circles searching for circular links!
		if len(linkedTo.LinksTo) > 0 && linkedToAncestorEntry == nil {
			linkImpl := chainLink.LinkImplementations[linkedTo.ResourceName]
			linkKindOutput, err := linkImpl.GetKind(ctx, &provider.LinkGetKindInput{
				Params: blueprintParams,
			})
			if err != nil {
				return nil, err
			}

			collectedFromLinkedTo, err := collectCircularLinkInfo(
				ctx,
				linkedTo,
				append(ancestors, &ancestorLinkInfo{
					resourceName:     linkedTo.ResourceName,
					outboundLinkKind: linkKindOutput.Kind,
				}),
				blueprintParams,
			)
			if err != nil {
				return nil, err
			}

			collected = append(
				collected,
				collectedFromLinkedTo...,
			)
		}
	}
	return collected, nil
}

func isHardCycle(ancestors []*ancestorLinkInfo, ancestorWithCycle *ancestorLinkInfo) bool {
	ancestorWithCycleIndex := core.FindIndex(ancestors, ancestorIsResource(ancestorWithCycle.resourceName))
	ancestorsInCycle := ancestors[ancestorWithCycleIndex:]
	hardLinksInCycle := core.Filter(
		ancestorsInCycle,
		func(currentAncestor *ancestorLinkInfo, index int) bool {
			return currentAncestor.outboundLinkKind == provider.LinkKindHard
		},
	)
	return len(hardLinksInCycle) == len(ancestorsInCycle)
}

func ancestorIsResource(resourceName string) func(*ancestorLinkInfo, int) bool {
	return func(ancestor *ancestorLinkInfo, index int) bool {
		return ancestor.resourceName == resourceName
	}
}

func linkNotInList(
	list []*ChainLink,
	circularLinkInfoItems []*circularLinkInfoItem,
) func(*ChainLink, int) bool {
	return func(link *ChainLink, index int) bool {
		circularEntryIndex := core.FindIndex(circularLinkInfoItems, isInCircularLinks(link.ResourceName))
		return !core.SliceContains(list, link) || circularEntryIndex > -1
	}
}

func isInCircularLinks(resourceName string) func(*circularLinkInfoItem, int) bool {
	return func(circularEntry *circularLinkInfoItem, index int) bool {
		return circularEntry.resourceName == resourceName
	}
}

func buildLinkPaths(link *ChainLink) []string {
	if len(link.Paths) == 0 {
		return []string{fmt.Sprintf("/%s", link.ResourceName)}
	}

	pathsFromCurrentLink := []string{}
	for _, pathToCurrentLink := range link.Paths {
		pathsFromCurrentLink = append(
			pathsFromCurrentLink,
			fmt.Sprintf("%s/%s", pathToCurrentLink, link.ResourceName),
		)
	}
	return pathsFromCurrentLink
}

func correctSubChainPaths(linksCleanedFromTopLevel []*ChainLink) {
	for _, subChain := range linksCleanedFromTopLevel {
		correctChainPaths(subChain.LinksTo, subChain, []string{subChain.ResourceName})
	}
}

func correctChainPaths(links []*ChainLink, parent *ChainLink, ancestors []string) {
	for _, link := range links {
		isCycle := core.SliceContainsComparable(ancestors, link.ResourceName)
		if !isCycle {
			pathsNotIncludingParent := core.Filter(
				link.Paths,
				checkLinkDoesNotHaveParentInPaths(parent.ResourceName),
			)
			link.Paths = append(pathsNotIncludingParent, buildLinkPaths(parent)...)

			correctChainPaths(link.LinksTo, link, append(ancestors, link.ResourceName))
		}
	}
}

func extractCircularLinkErrors(circularLinkInfoItems []*circularLinkInfoItem) []error {
	errors := []error{}
	for _, item := range circularLinkInfoItems {
		if item.err != nil {
			errors = append(errors, item.err)
		}
	}
	return errors
}
