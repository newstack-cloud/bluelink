package diagutils

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type ConcreteAction struct {
	// One or more CLI commands that can be executed as a suggested action
	// for a diagnostic.
	Commands []string
	// One or more links to a web URL that can be used as a suggested action
	// for a diagnostic.
	Links []*Link
}

type Link struct {
	// The title of the link.
	Title string
	// The URL of the link.
	URL string
}

// GetConcreteAction returns a concrete, context-specific action
// for a suggested action.
// When using this function, it is assumed that the current context is a machine
// that uses the bluelink CLI that can run the suggested commands.
//
// This will return nil if the suggested action is not supported.
func GetConcreteAction(
	suggestedAction errors.SuggestedAction,
	diagMetadata map[string]any,
) *ConcreteAction {
	// There are no concrete actions for installing or updating transformers
	// as the identifiers made available to diagnostics contain the
	// the transform string used in blueprints and not the plugin ID so there is no way to
	// create a useful concrete action such as a command to install the transformer.
	switch errors.ActionType(suggestedAction.Type) {
	case errors.ActionTypeInstallProvider:
		return installProviderAction(diagMetadata)
	case errors.ActionTypeUpdateProvider:
		return updateProviderAction(diagMetadata)
	case errors.ActionTypeCheckFunctionName:
		return checkFunctionNameAction()
	case errors.ActionTypeCheckResourceType:
		return checkResourceTypeAction(diagMetadata)
	case errors.ActionTypeCheckDataSourceType:
		return checkDataSourceTypeAction(diagMetadata)
	case errors.ActionTypeCheckVariableType:
		return checkVariableTypeAction(diagMetadata)
	case errors.ActionTypeCheckCustomVariableOptions:
		return checkCustomVariableOptionsAction(diagMetadata)
	case errors.ActionTypeCheckAbstractResourceType:
		return checkAbstractResourceTypeAction(diagMetadata)
	case errors.ActionTypeCheckTransformers:
		return exploreTransformersAction()
	case errors.ActionTypeCheckResourceTypeSchema:
		return checkResourceTypeSchemaAction(diagMetadata)
	}
	return nil
}

func installProviderAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["providerNamespace"].(string)
	category, _ := diagMetadata["category"].(string)

	// If no namespace is available, provide links to explore plugins in the registry.
	// For resources, both providers and transformers can provide resource types.
	if !hasProviderNamespace {
		links := []*Link{
			{
				Title: "Explore providers in the official registry",
				URL:   "https://registry.bluelink.dev/providers",
			},
		}
		if category == "resource" {
			links = append(links, &Link{
				Title: "Explore transformers in the official registry",
				URL:   "https://registry.bluelink.dev/transformers",
			})
		}
		return &ConcreteAction{
			Links: links,
		}
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Commands: []string{fmt.Sprintf("bluelink plugins install %s/%s", org, providerNamespace)},
	}
}

func exploreTransformersAction() *ConcreteAction {
	return &ConcreteAction{
		Links: []*Link{
			{
				Title: "Explore transformers in the official registry",
				URL:   "https://registry.bluelink.dev/transformers",
			},
		},
	}
}

func updateProviderAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["providerNamespace"].(string)
	if !hasProviderNamespace {
		return &ConcreteAction{
			Links: []*Link{
				{
					Title: "Check for new versions of the provider in the official registry",
					URL:   "https://registry.bluelink.dev/providers",
				},
			},
		}
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Commands: []string{fmt.Sprintf("bluelink plugins update %s/%s", org, providerNamespace)},
	}
}

func checkFunctionNameAction() *ConcreteAction {
	return &ConcreteAction{
		Links: []*Link{
			{
				Title: "Explore provider functions in the official registry",
				URL:   "https://registry.bluelink.dev/providers",
			},
		},
	}
}

func checkResourceTypeAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["providerNamespace"].(string)
	if !hasProviderNamespace {
		return defaultProvidersAction()
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Links: []*Link{
			{
				Title: fmt.Sprintf(
					"Explore resource types for the %s/%s provider in the official registry",
					org,
					providerNamespace,
				),
				URL: fmt.Sprintf(
					"https://registry.bluelink.dev/providers/%s/%s/latest",
					org,
					providerNamespace,
				),
			},
		},
	}
}

func checkResourceTypeSchemaAction(diagMetadata map[string]any) *ConcreteAction {
	resourceType, hasResourceType := diagMetadata["resourceType"].(string)
	if !hasResourceType {
		return defaultProvidersAction()
	}

	pluginNamespace := provider.ExtractProviderFromItemType(resourceType)
	org := getOrgForPluginNamespace(pluginNamespace)
	resourceTypeSlug := urlEncodeEntityType(resourceType)

	return &ConcreteAction{
		Links: []*Link{
			{
				Title: fmt.Sprintf(
					"See the resource type schema for the %s resource type in the %s/%s provider in the official registry",
					resourceType,
					org,
					pluginNamespace,
				),
				URL: fmt.Sprintf(
					"https://registry.bluelink.dev/providers/%s/%s/latest/resources/%s",
					org,
					pluginNamespace,
					resourceTypeSlug,
				),
			},
		},
	}
}

func checkAbstractResourceTypeAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["transformerNamespace"].(string)
	if !hasProviderNamespace {
		return defaultTransformersAction()
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Links: []*Link{
			{
				Title: fmt.Sprintf(
					"Explore abstract resource types for the %s/%s transformer in the official registry",
					org,
					providerNamespace,
				),
				URL: fmt.Sprintf(
					"https://registry.bluelink.dev/transformers/%s/%s/latest",
					org,
					providerNamespace,
				),
			},
		},
	}
}

func checkDataSourceTypeAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["providerNamespace"].(string)
	if !hasProviderNamespace {
		return defaultProvidersAction()
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Links: []*Link{
			{
				Title: fmt.Sprintf(
					"Explore data source types for the %s/%s provider in the official registry",
					org,
					providerNamespace,
				),
				URL: fmt.Sprintf(
					"https://registry.bluelink.dev/providers/%s/%s/latest",
					org,
					providerNamespace,
				),
			},
		},
	}
}

func checkVariableTypeAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["providerNamespace"].(string)
	if !hasProviderNamespace {
		return defaultProvidersAction()
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Links: []*Link{
			{
				Title: fmt.Sprintf(
					"Explore custom variable types for the %s/%s provider in the official registry",
					org,
					providerNamespace,
				),
				URL: fmt.Sprintf(
					"https://registry.bluelink.dev/providers/%s/%s/latest",
					org,
					providerNamespace,
				),
			},
		},
	}
}

func checkCustomVariableOptionsAction(diagMetadata map[string]any) *ConcreteAction {
	providerNamespace, hasProviderNamespace := diagMetadata["providerNamespace"].(string)
	if !hasProviderNamespace {
		return defaultProvidersAction()
	}

	org := getOrgForPluginNamespace(providerNamespace)

	return &ConcreteAction{
		Links: []*Link{
			{
				Title: fmt.Sprintf("Explore custom variable options for the %s/%s provider in the official registry", org, providerNamespace),
				URL:   fmt.Sprintf("https://registry.bluelink.dev/providers/%s/%s/latest", org, providerNamespace),
			},
		},
	}
}

func defaultProvidersAction() *ConcreteAction {
	return &ConcreteAction{
		Links: []*Link{
			{
				Title: "Explore providers in the official registry",
				URL:   "https://registry.bluelink.dev/providers",
			},
		},
	}
}

func defaultTransformersAction() *ConcreteAction {
	return &ConcreteAction{
		Links: []*Link{
			{
				Title: "Explore transformers in the official registry",
				URL:   "https://registry.bluelink.dev/transformers",
			},
		},
	}
}

// Matches the URL encoding of entity types for URLs in the official registry.
func urlEncodeEntityType(entityType string) string {
	slashesEncoded := strings.ReplaceAll(entityType, "/", "--")
	return strings.ReplaceAll(slashesEncoded, "::", "--")
}
