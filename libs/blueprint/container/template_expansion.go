package container

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

type ExpandedResourceTemplateResult struct {
	// ResourceTemplateMap is a map of resource template names
	// to the expanded resource names derived from the template.
	// This allows for looking up a resource in a template for substitution references
	// such as `resources.orderBucket[0]` without having to update references
	// as a part of resource template expansion.
	ResourceTemplateMap map[string][]string
	ExpandedBlueprint   *schema.Blueprint
}

// ExpandResourceTemplates expands resource templates in a parsed blueprint.
// This function carries out the following work:
//   - Resolves the `each` input for resource templates.
//   - Converts a resource template into individual resources in the blueprint.
//   - Adjusts link selectors and labels for each resource derived from a template.
//   - Caches the resolved items for the `each` property of a resource template
//     so they can be used to resolve each resource derived from the template later.
//
// The resources expanded from the template will have names in the format:
//
//	{templateName}_{index} (e.g. orderBucket_0)
//
// The following link relationships are supported for resource templates:
//
//   - A regular resource links to a resource template.
//     The labels from the resource template are applied to each expanded resource.
//
//   - A resource template links to a regular resource.
//     The link selector from the resource template is applied to each expanded resource.
//
//   - A resource template links to another resource template where the resolved items
//     list is of the same length.
//     In the following definition, "RT" stands for resource template.
//     Link selectors in RT(a) that correspend to labels in RT(b) are updated to include
//     an index to match the resource at the same index in RT(b).
//     Labels in RT(b) that correspond to link selectors in RT(a) are updated to include
//     an index to allow the resource from RT(a) to select the resource from RT(b).
//
// Links between resource templates of different lengths are not supported,
// this will result in an error during an attempt to expand the resource templates.
// This error has to be determined at runtime and not at the validation stage
// because the length of the resolved items for a resource template is not known
// until the value of the `each` property is resolved.
func ExpandResourceTemplates(
	ctx context.Context,
	blueprint *schema.Blueprint,
	substitutionResolver subengine.SubstitutionResolver,
	linkChains []*links.ChainLinkNode,
	cache *core.Cache[[]*core.MappingNode],
) (*ExpandedResourceTemplateResult, error) {
	if blueprint.Resources == nil {
		return &ExpandedResourceTemplateResult{ExpandedBlueprint: blueprint}, nil
	}

	resourceTemplateMap := map[string][]string{}
	expandedBlueprint := &schema.Blueprint{
		Version:   blueprint.Version,
		Transform: blueprint.Transform,
		Variables: blueprint.Variables,
		Values:    blueprint.Values,
		Include:   blueprint.Include,
		Resources: &schema.ResourceMap{
			Values:     map[string]*schema.Resource{},
			SourceMeta: map[string]*source.Meta{},
		},
		DataSources: blueprint.DataSources,
		Exports:     blueprint.Exports,
		Metadata:    blueprint.Metadata,
	}

	for resourceName, resource := range blueprint.Resources.Values {
		if !substitutions.IsNilStringSubs(resource.Each) {
			items, err := substitutionResolver.ResolveResourceEach(
				ctx,
				resourceName,
				resource,
				// This is also called during deployment, however, for the `each` property of a resource template.
				// resources and child blueprints references can not be used so changing the resolve
				// mode will not make a difference.
				subengine.ResolveForChangeStaging,
			)
			if err != nil {
				return nil, err
			}
			// Cache to be used to resolve each resource derived from a template later.
			cache.Set(resourceName, items)
		} else {
			expandedBlueprint.Resources.Values[resourceName] = resource
		}
	}

	// All resource templates have been resolved, now expand them.
	// We need to expand the resources after all each properties have been resolved in order to compare
	// the lengths of the resolved items for templates where there are links between them.
	for resourceName, resource := range blueprint.Resources.Values {
		if !substitutions.IsNilStringSubs(resource.Each) {
			items, _ := cache.Get(resourceName)
			err := expandResourcesFromTemplate(
				&resourceTemplateInfo{
					resourceTemplateName: resourceName,
					resourceTemplate:     resource,
					items:                items,
				},
				expandedBlueprint,
				resourceTemplateMap,
				linkChains,
				cache,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ExpandedResourceTemplateResult{
		ResourceTemplateMap: resourceTemplateMap,
		ExpandedBlueprint:   expandedBlueprint,
	}, nil
}

type resourceTemplateInfo struct {
	resourceTemplateName string
	resourceTemplate     *schema.Resource
	items                []*core.MappingNode
}

func expandResourcesFromTemplate(
	templateInfo *resourceTemplateInfo,
	expandedBlueprint *schema.Blueprint,
	resourceTemplateMap map[string][]string,
	linkChains []*links.ChainLinkNode,
	cache *core.Cache[[]*core.MappingNode],
) error {
	linkNode := findLinkNode(linkChains, templateInfo.resourceTemplateName, map[string]bool{})

	labelInfo, err := collectLabelsToApply(
		linkNode,
		templateInfo.resourceTemplateName,
		cache,
		templateInfo.items,
	)
	if err != nil {
		return err
	}

	linkSelectorInfo, err := collectLinkSelectorsToApply(
		linkNode,
		templateInfo.resourceTemplateName,
		cache,
		templateInfo.items,
	)
	if err != nil {
		return err
	}

	resourceTemplate := templateInfo.resourceTemplate

	for index := range templateInfo.items {
		expandedResource := expandResource(resourceTemplate, labelInfo, linkSelectorInfo, index)
		resourceName := core.ExpandedResourceName(templateInfo.resourceTemplateName, index)
		expandedBlueprint.Resources.Values[resourceName] = expandedResource

		if resourceTemplateMap[templateInfo.resourceTemplateName] == nil {
			resourceTemplateMap[templateInfo.resourceTemplateName] = []string{}
		}

		resourceTemplateMap[templateInfo.resourceTemplateName] = append(
			resourceTemplateMap[templateInfo.resourceTemplateName],
			resourceName,
		)
	}

	return nil
}

func expandResource(
	resourceTemplate *schema.Resource,
	labelInfo *resourceTemplateLabelInfo,
	linkSelectorInfo *resourceTemplateLinkSelectorInfo,
	index int,
) *schema.Resource {
	metadata := createExpandedResourceMetadata(resourceTemplate.Metadata, labelInfo, index)
	linkSelector := createExpandedResourceLinkSelector(resourceTemplate.LinkSelector, linkSelectorInfo, index)
	return &schema.Resource{
		Type:         resourceTemplate.Type,
		Description:  resourceTemplate.Description,
		Metadata:     metadata,
		DependsOn:    resourceTemplate.DependsOn,
		Condition:    resourceTemplate.Condition,
		LinkSelector: linkSelector,
		Spec:         resourceTemplate.Spec,
		SourceMeta:   resourceTemplate.SourceMeta,
	}
}

func createExpandedResourceMetadata(
	metadata *schema.Metadata,
	labelInfo *resourceTemplateLabelInfo,
	index int,
) *schema.Metadata {
	if metadata == nil {
		return nil
	}

	return &schema.Metadata{
		DisplayName: metadata.DisplayName,
		Annotations: metadata.Annotations,
		Labels: &schema.StringMap{
			Values: combineSelectors(
				labelInfo.collectedLabels,
				labelInfo.labelsToBeMadeUnique,
				index,
			),
		},
		Custom: metadata.Custom,
	}
}

func createExpandedResourceLinkSelector(
	linkSelector *schema.LinkSelector,
	linkSelectorInfo *resourceTemplateLinkSelectorInfo,
	index int,
) *schema.LinkSelector {
	if linkSelector == nil || linkSelector.ByLabel == nil {
		return nil
	}

	return &schema.LinkSelector{
		ByLabel: &schema.StringMap{
			Values: combineSelectors(
				linkSelectorInfo.collectedLinkSelectors,
				linkSelectorInfo.linkSelectorsToBeMadeUnique,
				index,
			),
		},
	}
}

func combineSelectors(
	collected map[string]string,
	toMakeUnique map[string]string,
	index int,
) map[string]string {

	selectors := map[string]string{}
	for selector, value := range collected {
		selectors[selector] = value
	}

	for selector, value := range toMakeUnique {
		selectors[fmt.Sprintf("%s_%d", selector, index)] = value
	}

	return selectors
}

type resourceTemplateLabelInfo struct {
	collectedLabels      map[string]string
	labelsToBeMadeUnique map[string]string
}

func collectLabelsToApply(
	linkNode *links.ChainLinkNode,
	resourceTemplateName string,
	cache *core.Cache[[]*core.MappingNode],
	items []*core.MappingNode,
) (*resourceTemplateLabelInfo, error) {
	collectedLabels := map[string]string{}
	labelsToBeMadeUnique := map[string]string{}
	for _, linkedFrom := range linkNode.LinkedFrom {
		if !substitutions.IsNilStringSubs(linkedFrom.Resource.Each) {
			linkedFromItems, _ := cache.Get(linkedFrom.ResourceName)
			if len(items) != len(linkedFromItems) {
				return nil, errResourceTemplateLinkLengthMismatch(
					linkedFrom.ResourceName,
					resourceTemplateName,
				)
			} else {
				// When a resource template is linked from another resource template and the resolved
				// items are of the same length, labels are collected to be made unique with an index suffix
				// for each resource derived from the template.
				matchingLabels := findMatchingSelectors(linkedFrom, linkNode)
				for label, value := range matchingLabels {
					labelsToBeMadeUnique[label] = value
				}
			}
		} else {
			// When a regular resource links to a resource template,
			// the matching labels from the resource template are applied to the resource.
			matchingLabels := findMatchingSelectors(linkedFrom, linkNode)
			for label, value := range matchingLabels {
				collectedLabels[label] = value
			}
		}
	}

	return &resourceTemplateLabelInfo{
		collectedLabels:      collectedLabels,
		labelsToBeMadeUnique: labelsToBeMadeUnique,
	}, nil
}

type resourceTemplateLinkSelectorInfo struct {
	collectedLinkSelectors      map[string]string
	linkSelectorsToBeMadeUnique map[string]string
}

func collectLinkSelectorsToApply(
	linkNode *links.ChainLinkNode,
	resourceTemplateName string,
	cache *core.Cache[[]*core.MappingNode],
	items []*core.MappingNode,
) (*resourceTemplateLinkSelectorInfo, error) {
	collectedLinkSelectors := map[string]string{}
	linkSelectorsToBeMadeUnique := map[string]string{}
	for _, linksTo := range linkNode.LinksTo {
		if !substitutions.IsNilStringSubs(linksTo.Resource.Each) {
			linksToItems, _ := cache.Get(linksTo.ResourceName)
			if len(items) != len(linksToItems) {
				return nil, errResourceTemplateLinkLengthMismatch(
					resourceTemplateName,
					linksTo.ResourceName,
				)
			} else {
				// When a resource template links to from another resource template and the resolved
				// items are of the same length, link selectors are collected to be made unique with an index suffix
				// for each resource derived from the template.
				matchingLinkSelectors := findMatchingSelectors(linkNode, linksTo)
				for label, value := range matchingLinkSelectors {
					linkSelectorsToBeMadeUnique[label] = value
				}
			}
		} else {
			// When a resource template links to a regular resource,
			// the matching link selectors from the resource template are applied to the resource.
			matchingLinkSelectors := findMatchingSelectors(linkNode, linksTo)
			for label, value := range matchingLinkSelectors {
				collectedLinkSelectors[label] = value
			}
		}
	}

	return &resourceTemplateLinkSelectorInfo{
		collectedLinkSelectors:      collectedLinkSelectors,
		linkSelectorsToBeMadeUnique: linkSelectorsToBeMadeUnique,
	}, nil
}

func findMatchingSelectors(
	linkedFrom *links.ChainLinkNode,
	linkedTo *links.ChainLinkNode,
) map[string]string {
	matchingSelectors := map[string]string{}

	for selector, selectedNames := range linkedFrom.Selectors {
		if slices.Contains(selectedNames, linkedTo.ResourceName) {
			key, value := extractKeyValueFromSelectorString(selector, "label")
			if key != "" {
				matchingSelectors[key] = value
			}
		}
	}

	return matchingSelectors
}

func extractKeyValueFromSelectorString(selector string, selectorType string) (string, string) {
	// The selector string format is `{selectorType}::{key}:{value}`
	parts := strings.Split(selector, "::")
	if len(parts) != 2 {
		return "", ""
	}

	if parts[0] != selectorType {
		return "", ""
	}

	keyValue := strings.Split(parts[1], ":")
	if len(keyValue) != 2 {
		return "", ""
	}

	return keyValue[0], keyValue[1]
}

func findLinkNode(
	linkChains []*links.ChainLinkNode,
	resourceName string,
	visited map[string]bool,
) *links.ChainLinkNode {
	for _, linkNode := range linkChains {
		if !visited[linkNode.ResourceName] {
			visited[linkNode.ResourceName] = true
			if linkNode.ResourceName == resourceName {
				return linkNode
			} else if len(linkNode.LinksTo) > 0 {
				descendantNode := findLinkNode(linkNode.LinksTo, resourceName, visited)
				if descendantNode != nil {
					return descendantNode
				}
			}
		}
	}

	return nil
}
