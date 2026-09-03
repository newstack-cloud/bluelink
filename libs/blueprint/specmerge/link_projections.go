package specmerge

import (
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// UnresolvedProjection identifies a link contribution that a resource's state claims
// exists and that could not be composed into the resource's spec.
type UnresolvedProjection struct {
	// LinkName is the logical name of the link that owns the field.
	LinkName string
	// ResourceFieldPath is the path in the resource spec the link owns, without the
	// resource name prefix.
	ResourceFieldPath string
	// LinkDataPath is the path in the link's data the value was expected at.
	LinkDataPath string
	// Reason says why the contribution could not be composed, so that a contribution
	// missing from the link's data can be told apart from one that was found but does
	// not fit the resource field it claims.
	Reason string
}

// LinkProjectionResult holds a spec with link contributions applied, along with any
// contributions that could not be resolved.
//
// Unresolved contributions are reported rather than raised so that each caller can decide
// what they mean. A deployment must not apply a spec that is missing a field a link owns,
// since applying it removes the field from the deployed resource. A drift check reporting
// a partial picture is less of a problem, as it simply reports the resource is not as it should be,
// and the caller can decide whether to treat that as a failure or a warning.
type LinkProjectionResult struct {
	Spec       *core.MappingNode
	Unresolved []UnresolvedProjection
}

// ApplyLinkProjections composes the contributions links have made to a resource into a
// copy of the given resource spec.
//
// A link writes to the resources it relates, and to intermediary resources such as a
// shared execution role. Those writes are not part of the blueprint's declared spec, so a
// spec used for a comparison or an apply is incomplete without them, comparing against one
// reports the link's fields as absent, and deploying one removes them from the live
// resource.
//
// The mappings recorded on each link say which resource field each contribution belongs to
// and where to read its value from in the link's data. They are the only record of that
// ownership, so this reads from them rather than inferring anything from the spec.
func ApplyLinkProjections(
	spec *core.MappingNode,
	resourceName string,
	links []state.LinkState,
) (*LinkProjectionResult, error) {
	result := &LinkProjectionResult{
		Spec:       core.CopyMappingNode(spec),
		Unresolved: []UnresolvedProjection{},
	}

	projections := orderedProjectionsFor(resourceName, links)
	for _, projection := range projections {
		err := applyLinkProjection(result, projection)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// The contributions that belong to a resource, in a stable order.
//
// Order is part of the result, not an implementation detail. A contribution to an array
// is appended when nothing matches it yet, so the order contributions are applied in is
// the order they end up in, and change staging compares arrays element by element unless
// the resource type opts into sorting them by a field. Composing the same contributions
// in a different order would report every element after the first difference as modified.
//
// Neither input is ordered, ResourceDataMappings is a map, and the order links come back
// in depends on the state store. Sorting by the resource field path gives an order
// derived from the contributions themselves, so it is the same in every process, on
// either side of a comparison, and on every run.
func orderedProjectionsFor(
	resourceName string,
	links []state.LinkState,
) []linkProjection {
	projections := []linkProjection{}
	for _, link := range links {
		for resourceFieldPath, linkDataPath := range link.ResourceDataMappings {
			fieldPath, ownedByResource := resourceFieldPathFor(resourceFieldPath, resourceName)
			if !ownedByResource {
				continue
			}

			projections = append(projections, linkProjection{
				linkName:     link.Name,
				linkData:     link.Data,
				fieldPath:    fieldPath,
				linkDataPath: linkDataPath,
			})
		}
	}

	slices.SortFunc(projections, func(a, b linkProjection) int {
		if fieldPathOrder := strings.Compare(a.fieldPath, b.fieldPath); fieldPathOrder != 0 {
			return fieldPathOrder
		}

		return strings.Compare(a.linkName, b.linkName)
	})

	return projections
}

// A single contribution a link makes to a resource.
type linkProjection struct {
	linkName     string
	linkData     map[string]*core.MappingNode
	fieldPath    string
	linkDataPath string
}

func applyLinkProjection(
	result *LinkProjectionResult,
	projection linkProjection,
) error {
	linkDataValue, _ := core.GetPathValue(
		core.AddRootToPath(projection.linkDataPath),
		&core.MappingNode{Fields: projection.linkData},
		core.MappingNodeMaxTraverseDepth,
	)
	if linkDataValue == nil {
		result.Unresolved = append(result.Unresolved, UnresolvedProjection{
			LinkName:          projection.linkName,
			ResourceFieldPath: projection.fieldPath,
			LinkDataPath:      projection.linkDataPath,
			Reason:            "the link's data holds no value at this path",
		})
		return nil
	}

	err := core.InjectPathValueReplaceFields(
		ResourceSpecPath(projection.fieldPath),
		linkDataValue,
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		// A contribution that does not fit the field it claims is reported in the same
		// way as one that is missing, rather than raised. Raising it stops the caller on
		// data it cannot do anything about, the mapping and the value are both already
		// recorded, so nothing the user changes in the blueprint repairs them, and a
		// caller that cannot proceed refuses on its own terms further up.
		result.Unresolved = append(result.Unresolved, UnresolvedProjection{
			LinkName:          projection.linkName,
			ResourceFieldPath: projection.fieldPath,
			LinkDataPath:      projection.linkDataPath,
			Reason:            err.Error(),
		})
	}

	return nil
}

// ResourceSpecPath roots a mapping's resource field path so that it can be applied to, or
// read from, a resource spec.
//
// Mappings are written against the resource, so a path usually starts at "spec", which is
// the spec itself. Providers also write paths that start at the spec's own fields, and
// rooting those as though "spec" had been trimmed produces a malformed path rather than an
// error, so the two forms are distinguished here.
func ResourceSpecPath(fieldPath string) string {
	if strings.HasPrefix(fieldPath, "spec.") || strings.HasPrefix(fieldPath, "spec[") {
		return core.ReplaceSpecWithRoot(fieldPath)
	}

	return core.AddRootToPath(fieldPath)
}

// Reports the resource field path a mapping refers to, and whether it belongs to the
// named resource at all.
//
// Mappings are keyed as "{resourceName}::{fieldPath}", and a single link holds mappings
// for every resource it contributes to. A link that writes a function's VPC configuration
// and a statement to that function's execution role carries mappings for both, so a
// mapping has to be matched to the resource being composed rather than applied to
// whichever resource brought the link into range.
func resourceFieldPathFor(resourceFieldPath string, resourceName string) (string, bool) {
	parts := strings.SplitN(resourceFieldPath, "::", 2)
	if len(parts) != 2 {
		return "", false
	}

	if parts[0] != resourceName {
		return "", false
	}

	return parts[1], true
}

// RemoveLinkProjections takes the contributions links have made to a resource back out of
// a copy of the given spec, along with any structure left holding nothing once they are
// gone.
//
// It is used where a spec comes from outside the framework and describes the resource as
// it actually is, which includes everything links have written to it. What a link
// contributed belongs to the link, which records it along with the resource field it
// wrote, so keeping it in the resource's spec as well would leave two records of the same
// thing that are free to disagree.
//
// A mapping that matches nothing in the spec is not an error. The spec describes a real
// resource, and a link's contribution being absent from it is a fact about that resource
// rather than a problem with the removal.
func RemoveLinkProjections(
	spec *core.MappingNode,
	resourceName string,
	links []state.LinkState,
) (*core.MappingNode, error) {
	stripped := core.CopyMappingNode(spec)
	if stripped == nil {
		return nil, nil
	}

	for _, projection := range orderedProjectionsFor(resourceName, links) {
		_, err := core.RemovePathValue(
			ResourceSpecPath(projection.fieldPath),
			stripped,
			core.MappingNodeMaxTraverseDepth,
		)
		if err != nil {
			return nil, errRemoveLinkProjection(projection.linkName, projection.fieldPath, err)
		}
	}

	return stripped, nil
}

// LinkFieldSource identifies where the value of a link-contributed resource field comes
// from, which is the link that owns it and the path in that link's data holding the value.
type LinkFieldSource struct {
	// LinkName is the logical name of the link that contributes the field.
	LinkName string
	// LinkDataPath is the path in the link's data the value is read from. It is the same
	// vocabulary a link reports its own field changes in, so the two can be compared.
	LinkDataPath string
}

// LinkFieldSources reports, for each field of a resource that links contribute, the link
// that owns it and the path in that link's data the value comes from.
//
// LinkOwnedFields answers who owns a field, which is what a change set reports. This
// answers where the value comes from as well, which is what a caller needs to line a
// resource's fields up against the changes a link has staged for its own data.
func LinkFieldSources(
	resourceName string,
	links []state.LinkState,
) map[string]LinkFieldSource {
	sources := map[string]LinkFieldSource{}
	for _, projection := range orderedProjectionsFor(resourceName, links) {
		sources[projection.fieldPath] = LinkFieldSource{
			LinkName:     projection.linkName,
			LinkDataPath: projection.linkDataPath,
		}
	}

	if len(sources) == 0 {
		return nil
	}

	return sources
}

// DeclaredLinkFieldSources reports the same correspondence as LinkFieldSources, read from
// what links declare at change staging rather than from what they recorded in state.
//
// Declarations are keyed by the logical name of the link that made them, each holding the
// "{resourceName}::{fieldPath}" to link data path mappings that link expects to produce.
//
// A link that has never run has nothing in state, so this is the only account of the
// fields it contributes while changes are being staged.
func DeclaredLinkFieldSources(
	resourceName string,
	declarations map[string]map[string]string,
) map[string]LinkFieldSource {
	linkNames := make([]string, 0, len(declarations))
	for linkName := range declarations {
		linkNames = append(linkNames, linkName)
	}
	// Sorted so a field two links both declare resolves to the same one of them on every
	// run, matching how LinkFieldSources orders the projections it reads from state.
	slices.Sort(linkNames)

	sources := map[string]LinkFieldSource{}
	for _, linkName := range linkNames {
		for resourceFieldPath, linkDataPath := range declarations[linkName] {
			fieldPath, ownedByResource := resourceFieldPathFor(resourceFieldPath, resourceName)
			if !ownedByResource {
				continue
			}

			sources[fieldPath] = LinkFieldSource{
				LinkName:     linkName,
				LinkDataPath: linkDataPath,
			}
		}
	}

	if len(sources) == 0 {
		return nil
	}

	return sources
}

// LinkOwnedFields reports the field paths of a resource that links contribute, mapped to
// the logical name of the link that contributes each one.
//
// The paths are the ones a change set reports, so a caller comparing two versions of a
// resource can say which link a changed field belongs to. Every link that maps into the
// resource is included, whether or not it still exists in the blueprint, since a link
// being removed is exactly when saying who owned a field matters most.
func LinkOwnedFields(
	resourceName string,
	links []state.LinkState,
) map[string]string {
	owners := map[string]string{}
	for _, projection := range orderedProjectionsFor(resourceName, links) {
		owners[projection.fieldPath] = projection.linkName
	}

	if len(owners) == 0 {
		return nil
	}

	return owners
}
