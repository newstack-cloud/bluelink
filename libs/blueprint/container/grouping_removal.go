package container

import "github.com/newstack-cloud/bluelink/libs/blueprint/state"

// GroupOrderedElements groups ordered elements for removal
// so that elements that are not related can be removed concurrently.
func GroupOrderedElementsForRemoval(orderedElements []*ElementWithAllDeps) [][]state.Element {
	if len(orderedElements) == 0 {
		return [][]state.Element{}
	}

	currentGroupIndex := 0
	groups := [][]state.Element{{}}
	elementGroupMap := map[string]groupElementInfo{}

	for _, element := range orderedElements {
		hasDependentInCurrentGroup := hasDependentInGroup(
			element,
			elementGroupMap,
			currentGroupIndex,
		)

		if hasDependentInCurrentGroup {
			currentGroupIndex += 1
			newGroup := []state.Element{element.Element}
			groups = append(groups, newGroup)
		} else {
			groups[currentGroupIndex] = append(groups[currentGroupIndex], element.Element)
		}

		elementGroupMap[element.Element.ID()] = groupElementInfo{
			groupIndex:         currentGroupIndex,
			elementWithAllDeps: element,
		}
	}

	return groups
}

func hasDependentInGroup(
	element *ElementWithAllDeps,
	elementGroupMap map[string]groupElementInfo,
	currentGroupIndex int,
) bool {

	for _, groupElementInfo := range elementGroupMap {
		isDependent := hasDependency(groupElementInfo.elementWithAllDeps, element)
		if isDependent && groupElementInfo.groupIndex == currentGroupIndex {
			return true
		}
	}

	return false
}

type groupElementInfo struct {
	groupIndex         int
	elementWithAllDeps *ElementWithAllDeps
}
