package drift

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SchemaForResourceFieldPath returns the schema for a field of a resource's spec,
// addressed the way resource data mappings address it, or nil when the path does not
// resolve against the given schema.
//
// The "spec" prefix a mapping carries is optional, so both "spec.tags[0]" and "tags[0]"
// address the same field. Each selector applied to a segment descends into the schema's
// items, so "statement[0]" resolves to the schema of a statement rather than of the list
// holding them.
//
// Nil is returned rather than an error. The schema only ever relaxes a comparison, so a
// path that cannot be resolved leaves the caller comparing the field exactly, which is the
// stricter reading.
func SchemaForResourceFieldPath(
	specSchema *provider.ResourceDefinitionsSchema,
	resourceFieldPath string,
) *provider.ResourceDefinitionsSchema {
	if specSchema == nil {
		return nil
	}

	trimmed := strings.TrimPrefix(resourceFieldPath, "spec")
	trimmed = strings.TrimPrefix(trimmed, ".")
	if trimmed == "" {
		return specSchema
	}

	current := specSchema
	for _, segment := range SplitFieldPathSegments(trimmed) {
		name, selectorCount := SplitSegmentSelectors(segment)
		if current == nil || name == "" {
			return nil
		}

		next, found := current.Attributes[name]
		if !found {
			return nil
		}

		current = next
		for range selectorCount {
			if current == nil {
				return nil
			}

			current = current.Items
		}
	}

	return current
}

// SplitFieldPathSegments splits a resource field path on the separators between its
// segments, leaving each segment's selectors attached to it.
//
// The split is done by hand rather than on every ".", because a selector holds a path of
// its own, for example, [@.policyName="x"] carries a dot that does not separate two fields
// of the resource.
func SplitFieldPathSegments(path string) []string {
	segments := []string{}
	depth := 0
	current := strings.Builder{}

	for _, char := range path {
		switch {
		case char == '[':
			depth++
			current.WriteRune(char)
		case char == ']':
			depth--
			current.WriteRune(char)
		case char == '.' && depth == 0:
			segments = append(segments, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}

	return append(segments, current.String())
}

// SplitSegmentSelectors returns a segment's field name and the number of selections
// applied to it, so that "statement" with one selector is told apart from a field of that
// name with none.
func SplitSegmentSelectors(segment string) (string, int) {
	openIndex := strings.Index(segment, "[")
	if openIndex < 0 {
		return segment, 0
	}

	// Only the brackets at the top level of the segment select anything from the field. A
	// selector holds a path of its own, so the inner brackets of
	// [@.resources[0]="x"] belong to that path rather than to the field being addressed.
	count := 0
	depth := 0
	for _, char := range segment[openIndex:] {
		switch char {
		case '[':
			if depth == 0 {
				count++
			}
			depth++
		case ']':
			depth--
		}
	}

	return segment[:openIndex], count
}
