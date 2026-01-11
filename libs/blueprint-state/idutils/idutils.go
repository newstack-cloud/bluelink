package idutils

import (
	"fmt"

	"github.com/google/uuid"
)

// ReourceInBlueprintID returns a unique identifier for a resource in the
// context of a blueprint instance.
// e.g. instance:123:resource:saveOrderFunction
func ReourceInBlueprintID(instanceID string, resourceName string) string {
	return fmt.Sprintf("instance:%s:resource:%s", instanceID, resourceName)
}

// ChildInBlueprintID returns a unique identifier for a child blueprint
// in the context of a parent blueprint.
// e.g. instance:123:child:coreInfra
func ChildInBlueprintID(instanceID string, childName string) string {
	return fmt.Sprintf("instance:%s:child:%s", instanceID, childName)
}

// LinkInBlueprintID returns a unique identifier for a link in the
// context of a blueprint instance.
// e.g. instance:123:link:saveOrderFunction::ordersTable_0
func LinkInBlueprintID(instanceID string, linkName string) string {
	return fmt.Sprintf("instance:%s:link:%s", instanceID, linkName)
}

// IsValidUUID returns true if the given string is a valid UUID.
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// SeparateIDsAndNames separates a slice of strings into valid UUIDs and names.
func SeparateIDsAndNames(idsOrNames []string) (ids []string, names []string) {
	for _, s := range idsOrNames {
		if IsValidUUID(s) {
			ids = append(ids, s)
		} else {
			names = append(names, s)
		}
	}
	return ids, names
}
