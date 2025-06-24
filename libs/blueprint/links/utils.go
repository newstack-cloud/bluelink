package links

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// SelectGroup provides a grouping of selector and selected resources
// that should be used under the banner of a selector attribute such as a label.
// A selector will have a "linkSelector" definition which could be "label::app:orderApi" for example
// and then the candidate resources for selection
// are all the resources in the spec that have the "orderApi" label.
type SelectGroup struct {
	SelectorResources              []*ResourceWithNameAndSelectors
	CandidateResourcesForSelection []*ResourceWithNameAndSelectors
}

// ResourceWithNameAndSelectors holds a resource in a blueprint spec schema
// along with a name in the context of a blueprint and the selectors used
// to link to other resources.
type ResourceWithNameAndSelectors struct {
	Name      string
	Resource  *schema.Resource
	Selectors []string
}

// GroupResourcesBySelector deals with collecting resources by selectors
// from the given blueprint spec.
// This produces a mapping of
// {selectorAttributeType}::{selectorAttributeKey}:{selectorAttributeValue} ->
//
//	{ SelectorResources, CandidateResourcesForSelection }.
//
// For example, "label::app:orderApi" -> { SelectorResources, CandidateResourcesForSelection }.
func GroupResourcesBySelector(spec speccore.BlueprintSpec) map[string]*SelectGroup {
	groupedResources := map[string]*SelectGroup{}
	intermediaryResources := map[string]*ResourceWithNameAndSelectors{}
	resources := map[string]*schema.Resource{}
	if spec.Schema().Resources != nil {
		resources = spec.Schema().Resources.Values
	}

	for name, resource := range resources {
		if resource.LinkSelector != nil && resource.LinkSelector.ByLabel != nil {
			// Labels used to select other resources that imply links.
			selectorLabels := extractSelectorLabelsForGrouping(resource.LinkSelector.ByLabel.Values)
			addResourceAsSelectors(groupedResources, selectorLabels, name, resource, intermediaryResources)
		}

		if resource.Metadata != nil && resource.Metadata.Labels != nil {
			// Labels that allow other resources to select this resource.
			labels := extractSelectorLabelsForGrouping(resource.Metadata.Labels.Values)
			addResourceAsSelectionCandidates(groupedResources, labels, name, resource, intermediaryResources)
		}
	}

	return groupedResources
}

func addResourceAsSelectors(
	selectGroupMap map[string]*SelectGroup,
	selectorLabels []string,
	resourceName string,
	resource *schema.Resource,
	intermediaryResources map[string]*ResourceWithNameAndSelectors,
) {
	intermediaryResource := addToIntermediaryResourcesIfNeeded(
		resourceName,
		resource,
		selectorLabels,
		intermediaryResources,
		true,
	)

	for _, selectorLabel := range selectorLabels {
		selectGroup, exists := selectGroupMap[selectorLabel]
		if exists {
			selectGroup.SelectorResources = append(selectGroup.SelectorResources, intermediaryResource)
		} else {
			selectGroupMap[selectorLabel] = &SelectGroup{
				SelectorResources: []*ResourceWithNameAndSelectors{
					intermediaryResource,
				},
			}
		}
	}
}

func addResourceAsSelectionCandidates(
	selectGroupMap map[string]*SelectGroup,
	selectorLabels []string,
	resourceName string,
	resource *schema.Resource,
	intermediaryResources map[string]*ResourceWithNameAndSelectors,
) {
	intermediaryResource := addToIntermediaryResourcesIfNeeded(
		resourceName,
		resource,
		[]string{},
		intermediaryResources,
		false,
	)

	for _, selectorLabel := range selectorLabels {
		selectGroup, exists := selectGroupMap[selectorLabel]
		if exists {
			selectGroup.CandidateResourcesForSelection = append(
				selectGroup.CandidateResourcesForSelection,
				intermediaryResource,
			)
		} else {
			selectGroupMap[selectorLabel] = &SelectGroup{
				CandidateResourcesForSelection: []*ResourceWithNameAndSelectors{
					intermediaryResource,
				},
			}
		}
	}
}

func addToIntermediaryResourcesIfNeeded(
	resourceName string,
	resource *schema.Resource,
	selectors []string,
	collectedIntermediaryResources map[string]*ResourceWithNameAndSelectors,
	appendMissingSelectors bool,
) *ResourceWithNameAndSelectors {
	intermediaryResource, irExists := collectedIntermediaryResources[resourceName]
	if !irExists {
		intermediaryResource = &ResourceWithNameAndSelectors{
			Name:      resourceName,
			Resource:  resource,
			Selectors: selectors,
		}
		collectedIntermediaryResources[resourceName] = intermediaryResource
	}
	if appendMissingSelectors && len(selectors) > 0 {
		missingSelectors := core.Filter(selectors, func(selector string, index int) bool {
			return !core.SliceContainsComparable(intermediaryResource.Selectors, selector)
		})
		intermediaryResource.Selectors = append(intermediaryResource.Selectors, missingSelectors...)
	}
	return intermediaryResource
}

func extractSelectorLabelsForGrouping(selectorLabelMap map[string]string) []string {
	selectorLabels := []string{}
	for label, value := range selectorLabelMap {
		labelKey := fmt.Sprintf("label::%s:%s", label, value)
		selectorLabels = append(selectorLabels, labelKey)
	}
	return selectorLabels
}
