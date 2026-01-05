package shared

// PathBuilder provides path building utilities for hierarchical item paths.
// It tracks the mapping between instance IDs and child names to construct
// full paths like "parentChild/childName/resourceName".
type PathBuilder struct {
	RootInstanceID       string
	InstanceIDToChildName map[string]string
	InstanceIDToParentID  map[string]string
}

// NewPathBuilder creates a new PathBuilder with the given root instance ID.
func NewPathBuilder(rootInstanceID string) *PathBuilder {
	return &PathBuilder{
		RootInstanceID:       rootInstanceID,
		InstanceIDToChildName: make(map[string]string),
		InstanceIDToParentID:  make(map[string]string),
	}
}

// BuildInstancePath builds a path from instance ID to the child name.
// For root instance children, returns just the name.
// For nested children, returns a path like "parentChild/childName".
func (p *PathBuilder) BuildInstancePath(parentInstanceID, childName string) string {
	if parentInstanceID == "" || parentInstanceID == p.RootInstanceID {
		return childName
	}

	pathParts := p.BuildParentChain(parentInstanceID)
	pathParts = append(pathParts, childName)
	return JoinPath(pathParts)
}

// BuildItemPath builds a path for an item (resource or link) based on its instance ID.
// For root instance items, returns just the item name.
// For nested items, returns a path like "parentChild/childName/itemName".
func (p *PathBuilder) BuildItemPath(instanceID, itemName string) string {
	if instanceID == "" || instanceID == p.RootInstanceID {
		return itemName
	}

	pathParts := p.BuildParentChain(instanceID)
	pathParts = append(pathParts, itemName)
	return JoinPath(pathParts)
}

// BuildParentChain constructs the path parts from the root to the given instance ID.
func (p *PathBuilder) BuildParentChain(startInstanceID string) []string {
	var pathParts []string
	currentID := startInstanceID
	for currentID != "" && currentID != p.RootInstanceID {
		if name, ok := p.InstanceIDToChildName[currentID]; ok {
			pathParts = append([]string{name}, pathParts...)
			currentID = p.InstanceIDToParentID[currentID]
		} else {
			break
		}
	}
	return pathParts
}

// TrackChildInstance records the mapping from a child instance ID to its name and parent.
func (p *PathBuilder) TrackChildInstance(childInstanceID, childName, parentInstanceID string) {
	if childInstanceID != "" && childName != "" {
		p.InstanceIDToChildName[childInstanceID] = childName
		p.InstanceIDToParentID[childInstanceID] = parentInstanceID
	}
}

// JoinPath joins path parts with "/" separator.
func JoinPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "/" + parts[i]
	}
	return result
}
