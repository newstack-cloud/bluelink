package links

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// The validation phase tags a referenced element's reference-chain node with
// "subRefProp:<usedIn>:<usedInPropertyPath>" to record where each substitution
// reference was placed.
const subRefPropTagPrefix = "subRefProp:"

// Normalises concrete array indices in a property path ("targets[0].arn") to the
// schema-relative form ("targets[].arn").
var arrayIndexPattern = regexp.MustCompile(`\[\d+\]`)

// Activates a registered link from resource A to resource B whenever A places a
// reference to B at a field marked with ActivatesLinkOnReference, so a reference
// can do the same work as a link selector. Runs after the selector pass so connections
// de-duplicate against selector-derived links.
func (l *defaultSpecLinkInfo) activateReferenceImpliedLinks(ctx context.Context) error {
	if l.refChainCollector == nil {
		return nil
	}

	resources := map[string]*schema.Resource{}
	if l.spec.Schema().Resources != nil {
		resources = l.spec.Schema().Resources.Values
	}

	for _, fromName := range sortedResourceNames(resources) {
		fromResource := resources[fromName]
		if fromResource.Type == nil {
			continue
		}

		slots, err := l.activatingSlotsForType(ctx, fromResource.Type.Value)
		if err != nil {
			return err
		}
		if len(slots) == 0 {
			continue
		}

		for _, toName := range sortedResourceNames(resources) {
			if toName == fromName || !l.referencesAtActivatingSlot(fromName, toName, slots) {
				continue
			}
			if err := l.connectReferenceImpliedLink(
				ctx,
				fromName,
				fromResource,
				toName,
				resources[toName],
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *defaultSpecLinkInfo) connectReferenceImpliedLink(
	ctx context.Context,
	fromName string,
	fromResource *schema.Resource,
	toName string,
	toResource *schema.Resource,
) error {
	fromRes := &ResourceWithNameAndSelectors{Name: fromName, Resource: fromResource}
	toRes := &ResourceWithNameAndSelectors{Name: toName, Resource: toResource}

	checkInfo, err := l.checkCanLinkTo(ctx, fromRes, toRes)
	if err != nil {
		return err
	}
	if !checkInfo.canLinkTo {
		return nil
	}
	if checkInfo.linkImplementation == nil {
		return errMissingLinkImplementation(fromRes, toRes)
	}

	fromNode, exists := l.linkMap[fromName]
	if !exists {
		fromNode = &ChainLinkNode{
			ResourceName:        fromName,
			Resource:            fromResource,
			Selectors:           map[string][]string{},
			LinkImplementations: map[string]provider.Link{},
			LinksTo:             []*ChainLinkNode{},
			LinkedFrom:          []*ChainLinkNode{},
			Paths:               []string{},
		}
		l.linkMap[fromName] = fromNode
		// No inbound links yet, so the source starts a chain; this is corrected
		// later if another resource links to it.
		l.chains = append(l.chains, fromNode)
	}

	// Reuse the selector-link connection logic so de-duplication, path
	// materialisation and top-level chain correction behave identically.
	return l.connectCandidateIfMeetsConditions(
		fromNode,
		fromRes,
		toRes,
		checkInfo,
		fmt.Sprintf("ref::%s::%s", fromName, toName),
	)
}

func (l *defaultSpecLinkInfo) referencesAtActivatingSlot(
	fromName string,
	toName string,
	slots map[string]bool,
) bool {
	node := l.refChainCollector.Chain(bpcore.ResourceElementID(toName))
	if node == nil {
		return false
	}

	fromElementID := bpcore.ResourceElementID(fromName)
	specPrefix := fromElementID + ".spec."

	for _, tag := range node.Tags {
		if !strings.HasPrefix(tag, subRefPropTagPrefix) {
			continue
		}
		// Neither the element id nor the property path contain ':', so a 3-way
		// split of "subRefProp:<usedIn>:<path>" is not ambiguous.
		parts := strings.SplitN(tag, ":", 3)
		if len(parts) != 3 {
			continue
		}
		usedIn, propPath := parts[1], parts[2]
		if usedIn != fromElementID || !strings.HasPrefix(propPath, specPrefix) {
			continue
		}
		slotPath := arrayIndexPattern.ReplaceAllString(propPath[len(specPrefix):], "[]")
		if slots[slotPath] {
			return true
		}
	}

	return false
}

func (l *defaultSpecLinkInfo) activatingSlotsForType(
	ctx context.Context,
	resourceType string,
) (map[string]bool, error) {
	if cached, ok := l.activatingSlotCache[resourceType]; ok {
		return cached, nil
	}

	slots := map[string]bool{}
	resourceProvider, ok := l.resourceProviders[resourceType]
	if !ok {
		l.activatingSlotCache[resourceType] = slots
		return slots, nil
	}

	resourceImpl, err := resourceProvider.Resource(ctx, resourceType)
	if err != nil {
		return nil, err
	}
	if resourceImpl == nil {
		l.activatingSlotCache[resourceType] = slots
		return slots, nil
	}

	providerNamespace := provider.ExtractProviderFromItemType(resourceType)
	output, err := resourceImpl.GetSpecDefinition(ctx, &provider.ResourceGetSpecDefinitionInput{
		ProviderContext: provider.NewProviderContextFromParams(
			providerNamespace,
			l.blueprintParams,
		),
	})
	if err != nil {
		return nil, err
	}

	if output != nil && output.SpecDefinition != nil {
		visited := map[*provider.ResourceDefinitionsSchema]bool{}
		collectActivatingSlots(output.SpecDefinition.Schema, "", slots, visited)
	}

	l.activatingSlotCache[resourceType] = slots
	return slots, nil
}

// visited tracks the schemas on the current traversal path so that
// self-referential schema definitions (e.g. an "any JSON" schema that nests
// itself under Items or OneOf) terminate instead of recursing indefinitely,
// while still allowing a non-cyclic schema to be reused under multiple paths.
func collectActivatingSlots(
	schema *provider.ResourceDefinitionsSchema,
	path string,
	out map[string]bool,
	visited map[*provider.ResourceDefinitionsSchema]bool,
) {
	if schema == nil || visited[schema] {
		return
	}
	visited[schema] = true
	defer delete(visited, schema)

	if schema.ActivatesLinkOnReference && path != "" {
		out[path] = true
	}

	for attrName, attr := range schema.Attributes {
		attrPath := attrName
		if path != "" {
			attrPath = fmt.Sprintf("%s.%s", path, attrName)
		}
		collectActivatingSlots(attr, attrPath, out, visited)
	}

	if schema.Items != nil {
		collectActivatingSlots(schema.Items, fmt.Sprintf("%s[]", path), out, visited)
	}

	for _, oneOf := range schema.OneOf {
		collectActivatingSlots(oneOf, path, out, visited)
	}
}

func sortedResourceNames(resources map[string]*schema.Resource) []string {
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
