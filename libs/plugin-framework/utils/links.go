package utils

import "strings"

const linkTypeSeparator = "::"

// DeriveLinkableTypes extracts all resource types that the given resource can link to.
func DeriveLinkableTypes(resourceType string, allLinkTypes []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, linkType := range allLinkTypes {
		target := extractLinkTarget(resourceType, linkType)
		if target == "" {
			continue
		}
		if _, exists := seen[target]; exists {
			continue
		}
		seen[target] = struct{}{}
		result = append(result, target)
	}

	return result
}

// Returns the other resource type if resourceType participates in the link.
// Returns empty string if resourceType is not part of the link.
func extractLinkTarget(resourceType, linkType string) string {
	parts := strings.SplitN(linkType, linkTypeSeparator, 2)
	if len(parts) != 2 {
		return ""
	}

	if parts[0] == resourceType {
		return parts[1]
	}
	if parts[1] == resourceType {
		return parts[0]
	}
	return ""
}
