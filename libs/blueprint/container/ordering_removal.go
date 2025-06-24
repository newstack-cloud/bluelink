package container

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// OrderElementsForRemoval orders resources, children and links for removal based on dependencies
// in the current state of a blueprint instance.
//
// For the following dependency tree for a set of elements to be removed:
// (dependencies are the child nodes)
//
// ├── ResourceA1
// │	 ├── ResourceA2
// │	 │   ├── ChildA1
// │	 │   └── ResourceA5
// │	 └── ResourceA3
// │	 	 └── ResourceA6
// ├── LinkA1(ResourceA1, ResourceA2)
// │    ├── ResourceA1
// │    └── ResourceA2
//
// A valid order for removal would be:
// 1. LinkA1
// 2. ResourceA1
// 3. ResourceA2
// 4. ResourceA3
// 5. ResourceA5
// 6. ResourceA6
// 7. ChildA1
func OrderElementsForRemoval(
	elements *CollectedElements,
	currentState *state.InstanceState,
) ([]*ElementWithAllDeps, error) {
	combinedList := []state.Element{}
	for _, resourceInfo := range elements.Resources {
		combinedList = append(combinedList, resourceInfo)
	}

	for _, childInfo := range elements.Children {
		combinedList = append(combinedList, childInfo)
	}

	for _, linkInfo := range elements.Links {
		combinedList = append(combinedList, linkInfo)
	}

	combinedListWithDeps := collectElementsWithDependencies(combinedList, currentState)

	refChainCollector := refgraph.NewRefChainCollector()
	collectReferencesFromElements(combinedListWithDeps, refChainCollector)
	refChains := refChainCollector.ChainsByLeafDependants()
	refChainsDeepCopy := refgraph.DeepCopyReferenceChains(refChains)

	return refgraph.TopologicalSortReferences(
		refChainsDeepCopy,
		combinedListWithDeps,
		refgraph.ReferenceSortDirectionReferences,
		/* empty */ nil,
	)
}

func hasDependency(
	a, b *ElementWithAllDeps,
) bool {
	hasDep := slices.ContainsFunc(a.AllDependencies, func(current state.Element) bool {
		return current.ID() == b.Element.ID()
	})
	return hasDep
}

func collectElementsWithDependencies(
	elementList []state.Element,
	currentState *state.InstanceState,
) []*ElementWithAllDeps {
	elementsWithDeps := make([]*ElementWithAllDeps, len(elementList))
	for i, element := range elementList {
		elementsWithDeps[i] = &ElementWithAllDeps{
			Element:         element,
			AllDependencies: []state.Element{},
		}
	}

	for _, elementWithDeps := range elementsWithDeps {
		allDependencies, directDependencies := collectElementDependencies(
			elementWithDeps.Element,
			elementsWithDeps,
			currentState,
		)
		elementWithDeps.AllDependencies = allDependencies
		elementWithDeps.DirectDependencies = directDependencies
	}

	return elementsWithDeps
}

func collectElementDependencies(
	element state.Element,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
) ([]state.Element, []state.Element) {
	allDeps := []state.Element{}
	directDeps := []state.Element{}

	if element.Kind() == state.ResourceElement {
		resourceState := getResourceStateByName(currentState, element.LogicalName())
		if resourceState != nil {
			childDependencies := collectElementChildDependenciesForResource(
				resourceState,
				elementsWithDeps,
				currentState,
			)
			allDeps = append(allDeps, childDependencies...)
			directChildDeps := filterElementsByLogicalName(childDependencies, resourceState.DependsOnChildren)
			directDeps = append(directDeps, directChildDeps...)

			resourceDependencies := collectElementResourceDependenciesForResource(
				resourceState,
				elementsWithDeps,
				currentState,
			)
			allDeps = append(allDeps, resourceDependencies...)
			directResourceDeps := filterElementsByLogicalName(resourceDependencies, resourceState.DependsOnResources)
			directDeps = append(directDeps, directResourceDeps...)
		}
	}

	if element.Kind() == state.ChildElement {
		childDependencyInfo := getChildDependencies(currentState, element.LogicalName())
		if childDependencyInfo != nil {
			childDependencies := collectElementChildDependenciesForChild(
				childDependencyInfo,
				elementsWithDeps,
				currentState,
			)
			allDeps = append(allDeps, childDependencies...)
			directChildDeps := filterElementsByLogicalName(
				childDependencies,
				childDependencyInfo.DependsOnChildren,
			)
			directDeps = append(directDeps, directChildDeps...)

			resourceDependencies := collectElementResourceDependenciesForChild(
				childDependencyInfo,
				elementsWithDeps,
				currentState,
			)
			allDeps = append(allDeps, resourceDependencies...)
			directResourceDeps := filterElementsByLogicalName(
				resourceDependencies,
				childDependencyInfo.DependsOnResources,
			)
			directDeps = append(directDeps, directResourceDeps...)
		}
	}

	if element.Kind() == state.LinkElement {
		linkDependencyInfo := extractLinkDirectDependencies(element.LogicalName())
		if linkDependencyInfo != nil {
			resourceDependencies := collectElementResourceDependenciesForLink(
				linkDependencyInfo,
				elementsWithDeps,
				currentState,
			)
			allDeps = append(allDeps, resourceDependencies...)
			directResourceDeps := filterElementsByLogicalName(
				resourceDependencies,
				[]string{linkDependencyInfo.resourceAName, linkDependencyInfo.resourceBName},
			)
			directDeps = append(directDeps, directResourceDeps...)
		}
	}

	return allDeps, directDeps
}

func filterElementsByLogicalName(
	elements []state.Element,
	includeNames []string,
) []state.Element {
	return core.Filter(elements, func(current state.Element, _ int) bool {
		return slices.Contains(includeNames, current.LogicalName())
	})
}

func collectElementChildDependenciesForResource(
	resourceState *state.ResourceState,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
) []state.Element {
	return collectElementDependenciesOfType(
		resourceState.DependsOnChildren,
		elementsWithDeps,
		currentState,
		state.ChildElement,
	)
}

func collectElementResourceDependenciesForResource(
	resourceState *state.ResourceState,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
) []state.Element {
	return collectElementDependenciesOfType(
		resourceState.DependsOnResources,
		elementsWithDeps,
		currentState,
		state.ResourceElement,
	)
}

func collectElementResourceDependenciesForChild(
	childDependencyInfo *state.DependencyInfo,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
) []state.Element {
	return collectElementDependenciesOfType(
		childDependencyInfo.DependsOnResources,
		elementsWithDeps,
		currentState,
		state.ChildElement,
	)
}

func collectElementChildDependenciesForChild(
	childDependencyInfo *state.DependencyInfo,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
) []state.Element {
	return collectElementDependenciesOfType(
		childDependencyInfo.DependsOnChildren,
		elementsWithDeps,
		currentState,
		state.ChildElement,
	)
}

func collectElementResourceDependenciesForLink(
	linkDependencyInfo *linkDependencyInfo,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
) []state.Element {
	return collectElementDependenciesOfType(
		[]string{
			linkDependencyInfo.resourceAName,
			linkDependencyInfo.resourceBName,
		},
		elementsWithDeps,
		currentState,
		state.ResourceElement,
	)
}

func collectElementDependenciesOfType(
	dependenciesOfType []string,
	elementsWithDeps []*ElementWithAllDeps,
	currentState *state.InstanceState,
	elementKind state.ElementKind,
) []state.Element {
	dependencies := []state.Element{}

	for _, dependency := range dependenciesOfType {
		elementWithDeps := findElement(elementsWithDeps, dependency, elementKind)
		if elementWithDeps != nil {
			if len(elementWithDeps.AllDependencies) == 0 {
				allDeps, directDeps := collectElementDependencies(
					elementWithDeps.Element,
					elementsWithDeps,
					currentState,
				)
				elementWithDeps.AllDependencies = allDeps
				elementWithDeps.DirectDependencies = directDeps
			}

			dependencies = append(dependencies, elementWithDeps.Element)
			dependencies = addUniqueElements(dependencies, elementWithDeps.AllDependencies)
		}

	}

	return dependencies
}

func findElement(
	elementsWithDeps []*ElementWithAllDeps,
	logicalName string,
	elementKind state.ElementKind,
) *ElementWithAllDeps {
	i := 0
	element := (*ElementWithAllDeps)(nil)
	for element == nil && i < len(elementsWithDeps) {
		elementWithDeps := elementsWithDeps[i]
		if elementWithDeps.Element.LogicalName() == logicalName &&
			elementWithDeps.Element.Kind() == elementKind {
			element = elementWithDeps
		}
		i += 1
	}

	return element
}

func addUniqueElements(
	elements []state.Element,
	elementsToAdd []state.Element,
) []state.Element {
	for _, element := range elementsToAdd {
		if !containsElement(elements, element) {
			elements = append(elements, element)
		}
	}

	return elements
}

func containsElement(
	elements []state.Element,
	element state.Element,
) bool {
	return slices.ContainsFunc(elements, func(current state.Element) bool {
		return current.LogicalName() == element.LogicalName() &&
			current.Kind() == element.Kind()
	})
}

func collectReferencesFromElements(
	elements []*ElementWithAllDeps,
	refChainCollector refgraph.RefChainCollector,
) {
	for _, element := range elements {
		// Collect element to make sure elements with no dependencies are included.
		refChainCollector.Collect(
			getNamespacedLogicalName(element.Element),
			element.Element,
			"",
			[]string{},
		)

		// Only collect direct dependencies that can be used in the
		// topological sort algorithm.
		for _, dependency := range element.DirectDependencies {
			refChainCollector.Collect(
				getNamespacedLogicalName(dependency),
				dependency,
				element.Name(),
				[]string{},
			)
		}
	}
}

// ElementsWithAllDeps stores a representation of an element in a blueprint
// with all of its dependencies, both direct and transitive.
// This is primarily used for collecting and ordering elements for removal.
type ElementWithAllDeps struct {
	Element state.Element
	// All collected dependencies for an element, both direct and transitive.
	// This is particularly useful for grouping a pre-sorted list of elements
	// to remove into sets of elements that can be processed in parallel.
	AllDependencies []state.Element
	// A list of direct dependencies for an element, particularly useful
	// for building a reference chain for sorting.
	DirectDependencies []state.Element
}

func (e *ElementWithAllDeps) Name() string {
	return getNamespacedLogicalName(e.Element)
}

type linkDependencyInfo struct {
	resourceAName string
	resourceBName string
}
