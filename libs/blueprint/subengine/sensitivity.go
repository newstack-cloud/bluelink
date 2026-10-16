package subengine

import (
	"slices"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Records the properties a secret reached while substitutions were being resolved, which is
// the only place a secret interpolated into a field that declares nothing is visible.
//
// A record is shared by every resolveContext derived from the one that created it, so that a
// substitution resolved several levels down records against the top-level call. That sharing
// is why it holds properties rather than a flag: a flag on a shared record would mark every
// property resolved after the first secret.
type sensitivityRecord struct {
	properties map[string]map[string]struct{}
}

func newSensitivityRecord() *sensitivityRecord {
	return &sensitivityRecord{
		properties: map[string]map[string]struct{}{},
	}
}

// Elements are named as they are in references ("resources.ordersApi", "values.config"), and
// the property is relative to the element, so a resource spec field is recorded as
// "spec.connectionString".
func (s *sensitivityRecord) record(elementName string, elementProperty string) {
	// A root context carries no property, and recording against one would produce a path
	// with nothing after the element name. Nothing resolves a secret there, so there is
	// nothing to lose by dropping it.
	if elementProperty == "" {
		return
	}

	if _, found := s.properties[elementName]; !found {
		s.properties[elementName] = map[string]struct{}{}
	}

	s.properties[elementName][elementProperty] = struct{}{}
}

// Sorted, so a caller does not inherit map iteration order.
func (s *sensitivityRecord) propertiesOf(elementName string) []string {
	recorded, found := s.properties[elementName]
	if !found {
		return nil
	}

	properties := make([]string, 0, len(recorded))
	for property := range recorded {
		properties = append(properties, property)
	}
	slices.Sort(properties)

	return properties
}

// Returns every recorded property as a full element property path, in a stable order, less
// the spec properties of the named element.
//
// Only the given element's spec paths are left out: they travel on the resolved resource itself,
// so repeating them here would be two answers to keep in step. Everything else it recorded,
// its description, condition and metadata among them, belongs in this list, as does anything
// recorded against the values and data sources resolved on the way.
func (s *sensitivityRecord) pathsExcludingSpecOf(elementName string) []string {
	excluded := bpcore.ElementPropertyPath(elementName, "spec")

	return slices.DeleteFunc(s.allPaths(), func(path string) bool {
		return bpcore.IsPathAtOrBelow(excluded, path)
	})
}

// Every recorded path. Unlike pathsExcludingSpecOf there is nothing to split out, because
// the elements this serves, includes and single substitutions, have no spec.
func (s *sensitivityRecord) allPaths() []string {
	var paths []string
	for element, recorded := range s.properties {
		for property := range recorded {
			paths = append(paths, bpcore.ElementPropertyPath(element, property))
		}
	}
	slices.Sort(paths)

	return paths
}

// Keeps the "spec" prefix as that is the form that field changes use.
func (s *sensitivityRecord) specPropertiesOf(elementName string) []string {
	return s.propertiesUnderRootOf(elementName, "spec")
}

// Properties under the given root, with the prefix kept. Separates an element's body from
// the fields beside it, its description among them.
func (s *sensitivityRecord) propertiesUnderRootOf(elementName string, root string) []string {
	var rooted []string
	for _, property := range s.propertiesOf(elementName) {
		if bpcore.IsPathAtOrBelow(root, property) {
			rooted = append(rooted, property)
		}
	}

	return rooted
}
