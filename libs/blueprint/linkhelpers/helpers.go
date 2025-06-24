package linkhelpers

import (
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// GetLinkDataFromState returns the link data from the given state,
// returning nil if the provided state is nil.
// This wraps the link data in a MappingNode with the "Fields" property
// populated with the link data.
func GetLinkDataFromState(state *state.LinkState) *core.MappingNode {
	if state == nil {
		return nil
	}

	return &core.MappingNode{
		Fields: state.Data,
	}
}

// GetResourceNameFromChanges returns the resource name from the given changes,
// returning an empty string if the provided changes are nil.
func GetResourceNameFromChanges(changes *provider.Changes) string {
	if changes == nil {
		return ""
	}

	return changes.AppliedResourceInfo.ResourceName
}

// CollectChanges deals with collecting changes into the given *provider.Changes
// struct reference for a given field in a resource.
// This only supports collecting changes for scalar fields in the resource spec.
// scalar fields meaning strings, booleans, integers and floating point numbers.
func CollectChanges(
	resourceFieldPath string,
	linkFieldPath string,
	currentLinkData *core.MappingNode,
	resourceChanges *provider.Changes,
	collectIn *provider.LinkChanges,
) error {
	linkFieldPathWithoutRoot := removeRootFromFieldPath(linkFieldPath, "$")
	resourceFieldPathWithoutRoot := removeRootFromFieldPath(resourceFieldPath, "$")
	if IsFieldKnownOnDeploy(resourceChanges, resourceFieldPathWithoutRoot) ||
		resourcehelpers.IsComputedField(resourceChanges, resourceFieldPathWithoutRoot) {
		collectIn.FieldChangesKnownOnDeploy = append(collectIn.FieldChangesKnownOnDeploy, linkFieldPathWithoutRoot)
		return nil
	}

	currentLinkDataValue, err := core.GetPathValue(
		linkFieldPath,
		currentLinkData,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		return err
	}

	resolvedResource := GetResolvedResource(resourceChanges)
	if resolvedResource == nil {
		return ErrMissingResolvedResource
	}

	resourceFieldSearchPath := replaceRootInFieldPath(resourceFieldPathWithoutRoot, "spec", "$")
	resourceSpecValue, err := core.GetPathValue(
		resourceFieldSearchPath,
		resolvedResource.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		return err
	}

	if core.IsNilMappingNode(currentLinkDataValue) && !core.IsNilMappingNode(resourceSpecValue) {
		collectIn.NewFields = append(collectIn.NewFields, &provider.FieldChange{
			FieldPath: linkFieldPathWithoutRoot,
			NewValue:  resourceSpecValue,
		})
		return nil
	}

	if !core.IsNilMappingNode(currentLinkDataValue) && core.IsNilMappingNode(resourceSpecValue) {
		collectIn.RemovedFields = append(collectIn.RemovedFields, linkFieldPathWithoutRoot)
		return nil
	}

	if !core.IsNilMappingNode(currentLinkDataValue) &&
		!core.ScalarMappingNodeEqual(currentLinkDataValue, resourceSpecValue) {
		collectIn.ModifiedFields = append(collectIn.ModifiedFields, &provider.FieldChange{
			FieldPath: linkFieldPathWithoutRoot,
			PrevValue: currentLinkDataValue,
			NewValue:  resourceSpecValue,
		})
		return nil
	}

	if core.ScalarMappingNodeEqual(currentLinkDataValue, resourceSpecValue) {
		collectIn.UnchangedFields = append(collectIn.UnchangedFields, linkFieldPathWithoutRoot)
	}

	return nil
}

// CollectLinkDataChanges deals with collecting changes into the given *provider.Changes
// struct reference for a given link data field.
// This should be used for any link data that is not derived directly from a resource spec field.
// For example, this should be used for values that are derived from configuration
// in resource annotations or static logic in the link implementation.
// This only supports collecting changes for scalar fields in the link data.
// Scalar fields meaning strings, booleans, integers and floating point numbers.
func CollectLinkDataChanges(
	linkFieldPath string,
	currentLinkData *core.MappingNode,
	collectIn *provider.LinkChanges,
	newValue *core.MappingNode,
) error {
	currentLinkDataValue, err := core.GetPathValue(
		linkFieldPath,
		currentLinkData,
		core.MappingNodeMaxTraverseDepth,
	)
	if err != nil {
		return err
	}

	linkFieldPathWithoutRoot := removeRootFromFieldPath(linkFieldPath, "$")
	if core.IsNilMappingNode(currentLinkDataValue) && !core.IsNilMappingNode(newValue) {
		collectIn.NewFields = append(collectIn.NewFields, &provider.FieldChange{
			FieldPath: linkFieldPathWithoutRoot,
			NewValue:  newValue,
		})
		return nil
	}

	if !core.IsNilMappingNode(currentLinkDataValue) && core.IsNilMappingNode(newValue) {
		collectIn.RemovedFields = append(collectIn.RemovedFields, linkFieldPathWithoutRoot)
		return nil
	}

	if !core.IsNilMappingNode(currentLinkDataValue) &&
		!core.ScalarMappingNodeEqual(currentLinkDataValue, newValue) {
		collectIn.ModifiedFields = append(collectIn.ModifiedFields, &provider.FieldChange{
			FieldPath: linkFieldPathWithoutRoot,
			PrevValue: currentLinkDataValue,
			NewValue:  newValue,
		})
		return nil
	}

	if core.ScalarMappingNodeEqual(currentLinkDataValue, newValue) {
		collectIn.UnchangedFields = append(collectIn.UnchangedFields, linkFieldPathWithoutRoot)
	}

	return nil
}

func removeRootFromFieldPath(fieldPath string, rootIdentifier string) string {
	if strings.HasPrefix(fieldPath, fmt.Sprintf("%s[", rootIdentifier)) {
		return fieldPath[len(rootIdentifier):]
	}

	if strings.HasPrefix(fieldPath, fmt.Sprintf("%s.", rootIdentifier)) {
		return fieldPath[len(rootIdentifier)+1:]
	}

	return fieldPath
}

func replaceRootInFieldPath(fieldPath string, currentRoot string, newRoot string) string {
	if strings.HasPrefix(fieldPath, currentRoot) {
		return strings.Replace(fieldPath, currentRoot, newRoot, 1)
	}

	return fieldPath
}

// GetAnnotation returns the annotation with the given name from the resolved resource
// contained in the given set of resource changes.
// If the annotation is not found, the provided default value is returned.
func GetAnnotation(
	resourceChanges *provider.Changes,
	annotationName string,
	defaultValue *core.MappingNode,
) *core.MappingNode {
	if resourceChanges == nil {
		return defaultValue
	}

	resolvedResource := resourceChanges.AppliedResourceInfo.ResourceWithResolvedSubs
	if resolvedResource == nil {
		return defaultValue
	}

	if resolvedResource.Metadata == nil ||
		resolvedResource.Metadata.Annotations == nil {
		return defaultValue
	}

	annotation, hasAnnotation := resolvedResource.Metadata.Annotations.Fields[annotationName]
	if !hasAnnotation {
		return defaultValue
	}

	return annotation
}

// GetResolvedResource attempts to extract the resolved resource from the given set of resource changes.
// If the provided resource changes are nil, this function returns nil.
func GetResolvedResource(resourceChanges *provider.Changes) *provider.ResolvedResource {
	if resourceChanges == nil {
		return nil
	}

	return resourceChanges.AppliedResourceInfo.ResourceWithResolvedSubs
}

// IsFieldKnownOnDeploy returns whether the given field path is known on deploy
// in the given set of resource changes.
func IsFieldKnownOnDeploy(changes *provider.Changes, fieldPath string) bool {
	if changes == nil {
		return false
	}

	return slices.Contains(changes.FieldChangesKnownOnDeploy, fieldPath)
}
