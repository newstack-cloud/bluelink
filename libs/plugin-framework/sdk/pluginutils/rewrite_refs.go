package pluginutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subwalk"
)

// ResourcePropertyRewriter is called for each SubstitutionResourceProperty
// found during traversal. It receives the full SubstitutionResourceProperty
// (resource name, property path at any depth, template index) and returns:
// - A replacement *Substitution (any variant: resource ref, value ref, etc.)
// - nil to keep the original unchanged
type ResourcePropertyRewriter func(
	ref *substitutions.SubstitutionResourceProperty,
) *substitutions.Substitution

// RewriteResourcePropertyRefs builds a SubstitutionVisitor from a
// ResourcePropertyRewriter. Only SubstitutionResourceProperty substitutions
// are passed to the rewriter; all other substitution types pass through.
func RewriteResourcePropertyRefs(rewriter ResourcePropertyRewriter) subwalk.SubstitutionVisitor {
	return func(substitution *substitutions.Substitution) *substitutions.Substitution {
		if substitution != nil && substitution.ResourceProperty != nil {
			if newSub := rewriter(substitution.ResourceProperty); newSub != nil {
				return newSub
			}
		}
		return substitution
	}
}

// ChainResourcePropertyRewriters combines multiple rewriters into one.
// The first rewriter to return a non-nil substitution wins.
func ChainResourcePropertyRewriters(
	rewriters ...ResourcePropertyRewriter,
) ResourcePropertyRewriter {
	return func(
		ref *substitutions.SubstitutionResourceProperty,
	) *substitutions.Substitution {
		for _, rewriter := range rewriters {
			if newSub := rewriter(ref); newSub != nil {
				return newSub
			}
		}
		return nil
	}
}

// PathMatches returns true if the ref's path starts with the given field path segments,
// ignoring array indices. Useful for prefix matching on nested structures.
//
// For example:
//
//	ref: spec.vpc.securityGroups[0].id
//	fieldPathSegments: "spec", "vpc", "securityGroups"
//	returns: true (matches the prefix of the path, ignoring indices)
func PathMatches(
	ref *substitutions.SubstitutionResourceProperty,
	fieldPathSegments ...string,
) bool {
	return matchPathSegments(ref, fieldPathSegments) >= 0
}

// Walks ref.Path consuming one path item per segment
// (skipping array-index items, which match nothing on the segment side).
// On success it also consumes any trailing array-index items so the
// returned index lands on the final field-bearing position.
//
// Returns -1 if any segment fails to match.
func matchPathSegments(
	ref *substitutions.SubstitutionResourceProperty,
	fieldPathSegments []string,
) int {
	if ref == nil {
		if len(fieldPathSegments) == 0 {
			return 0
		}

		return -1
	}

	// Track the current index ignoring array index segments.
	i := 0
	for _, segment := range fieldPathSegments {
		for i < len(ref.Path) &&
			ref.Path[i].ArrayIndex != nil {
			i += 1
		}

		if i >= len(ref.Path) ||
			ref.Path[i] == nil ||
			ref.Path[i].FieldName != segment {
			return -1
		}

		i += 1
	}

	// Skip trailing array indices so exact match can
	// compare against len(ref.Path).
	for i < len(ref.Path) &&
		ref.Path[i] != nil &&
		ref.Path[i].ArrayIndex != nil {
		i += 1
	}

	return i
}

// PathExact returns true if the ref's path matches the given field names exactly
// (ignoring array indices between field names).
func PathExact(
	ref *substitutions.SubstitutionResourceProperty,
	fieldPathSegments ...string,
) bool {
	if ref == nil {
		return len(fieldPathSegments) == 0
	}

	end := matchPathSegments(ref, fieldPathSegments)
	return end >= 0 && end == len(ref.Path)
}

// RetargetRef returns a SubstitutionResourceProperty identical to ref but
// pointing at newResourceName. The path (including array indices) is preserved.
// Use when only the resource name changes.
func RetargetRef(
	ref *substitutions.SubstitutionResourceProperty,
	newResourceName string,
) *substitutions.Substitution {
	if ref == nil {
		return nil
	}

	return &substitutions.Substitution{
		ResourceProperty: &substitutions.SubstitutionResourceProperty{
			ResourceName:              newResourceName,
			ResourceEachTemplateIndex: ref.ResourceEachTemplateIndex,
			Path:                      ref.Path,
			SourceMeta:                ref.SourceMeta,
		},
	}
}

// RewriteFields is a declarative high-level helper to be used
// by transformers when rewriting references to resource properties.
//
// It can rename fields in ref.Path one-to-one, with N-dimensional array indices auto-preserved at
// their original relative positions. The i-th field-name item in ref.Path
// becomes newFields[i]; index items sandwiched between fields are kept at
// the same relative slot (between the renamed fields they followed). Source
// items beyond len(newFields) field positions are appended unchanged.
//
// Examples (paths shown logically; "[i]" stands for any source array index):
//
//	.spec.memory                  -> .spec.memorySize
//	    newFields = "spec", "memorySize"
//	.spec.routes[i].method        -> .spec.paths[i].httpMethod
//	    newFields = "spec", "paths", "httpMethod"     (one-dimensional)
//	.spec.rules[i].targets[j].arn -> .spec.rules[i].destinations[j].arn
//	    newFields = "spec", "rules", "destinations", "arn"   (two-dimensional)
//
// Extra field names beyond the original path are appended.
// Use MakeRef when the rewrite needs to insert or remove fields, restructure
// nesting depth (e.g. environmentVariables -> environment.variables), or
// introduce literal array indices that don't exist in the source path.
func RewriteFields(
	ref *substitutions.SubstitutionResourceProperty,
	newResourceName string,
	newFields ...string,
) *substitutions.Substitution {
	if ref == nil {
		return nil
	}

	newResProp := &substitutions.SubstitutionResourceProperty{
		ResourceName:              newResourceName,
		ResourceEachTemplateIndex: ref.ResourceEachTemplateIndex,
		SourceMeta:                ref.SourceMeta,
	}

	newPath := make([]*substitutions.SubstitutionPathItem, 0, len(ref.Path))
	fieldIndex := 0
	for _, item := range ref.Path {
		if item.FieldName != "" {
			if fieldIndex < len(newFields) {
				newPath = append(newPath, &substitutions.SubstitutionPathItem{
					FieldName: newFields[fieldIndex],
				})
				fieldIndex += 1
			} else {
				newPath = append(newPath, item)
			}
		} else if item.ArrayIndex != nil {
			newPath = append(newPath, item)
		}
	}

	// Ensuring extra field names beyond the original path are appended.
	for fieldIndex < len(newFields) {
		newPath = append(newPath, &substitutions.SubstitutionPathItem{
			FieldName: newFields[fieldIndex],
		})
		fieldIndex += 1
	}

	newResProp.Path = newPath

	return &substitutions.Substitution{
		ResourceProperty: newResProp,
		// Approximate a location for the new substitution wrapper
		// based on the original ref's source metadata.
		SourceMeta: ref.SourceMeta,
	}
}

// MakeRef is a low-level constructor that builds a SubstitutionResourceProperty
// pointing at newResourceName with the given path. The caller assembles the
// path explicitly from Field / Index items and (where useful) slices of
// ref.Path. Used for cases that don't fit RewriteFields' 1:1 model.
func MakeRef(
	newResourceName string,
	path []*substitutions.SubstitutionPathItem,
) *substitutions.Substitution {
	return &substitutions.Substitution{
		ResourceProperty: &substitutions.SubstitutionResourceProperty{
			ResourceName: newResourceName,
			Path:         path,
		},
	}
}

// Field is sugar for a literal field-name path item:
//
//	&SubstitutionPathItem{FieldName: name}.
func Field(name string) *substitutions.SubstitutionPathItem {
	return &substitutions.SubstitutionPathItem{
		FieldName: name,
	}
}

// Index is sugar for a literal array-index path item:
//
//	&SubstitutionPathItem{ArrayIndex: &index}.
func Index(index int) *substitutions.SubstitutionPathItem {
	indexI64 := int64(index)
	return &substitutions.SubstitutionPathItem{
		ArrayIndex: &indexI64,
	}
}

// ValueRef returns a SubstitutionValueReference. With no path items it is
// the flat form ${values.<name>}; trailing path items target a nested field
// or array element when the transformer-generated value is a complex object
// or list. Path items are built with Field / Index, identical to resource refs.
//
// Examples:
//
//	ValueRef("ordersHandler_lambda_arn")
//	    -> ${values.ordersHandler_lambda_arn}
//	ValueRef("ordersDb_connection", Field("host"))
//	    -> ${values.ordersDb_connection.host}
//	ValueRef("api_endpoints", Index(0), Field("url"))
//	    -> ${values.api_endpoints[0].url}
func ValueRef(
	valueName string,
	path ...*substitutions.SubstitutionPathItem,
) *substitutions.Substitution {
	return &substitutions.Substitution{
		ValueReference: &substitutions.SubstitutionValueReference{
			ValueName: valueName,
			Path:      path,
		},
	}
}
