package changes

import (
	"slices"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func collectMappingNodeChanges(
	changes *provider.Changes,
	newValue *bpcore.MappingNode,
	currentValue *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	if fieldChangeCtx.depth > bpcore.MappingNodeMaxTraverseDepth {
		return
	}

	if slices.Contains(fieldChangeCtx.fieldsToResolveOnDeploy, fieldChangeCtx.currentPath) {
		changes.FieldChangesKnownOnDeploy = append(
			changes.FieldChangesKnownOnDeploy,
			fieldChangeCtx.currentPath,
		)
		// Don't return so a change can be collected from current value -> nil
		// to avoid having to traverse through the current resource state again
		// to find the current value when displaying <currentValue> -> <knownOnDeploy> diffs.
		// `FieldChangesKnownOnDeploy` is used to look up whether the new value will be known
		// at deploy time and won't be removed or set to nil.
	}

	if isMapOrNil(newValue) && isMapOrNil(currentValue) {
		collectMappingNodeMapChanges(
			changes,
			newValue,
			currentValue,
			fieldChangeCtx,
		)
		return
	}

	if isArrayOrNil(newValue) && isArrayOrNil(currentValue) {
		collectMappingNodeArrayChanges(
			changes,
			newValue,
			currentValue,
			fieldChangeCtx,
		)
		return
	}

	if isScalarOrNil(newValue) && isScalarOrNil(currentValue) {
		collectMappingNodeScalarChanges(
			changes,
			newValue,
			currentValue,
			fieldChangeCtx,
		)
		return
	}

	// Value types do not match, so collect changes based on shallow comparisons.

	knownOnDeploy := slices.Contains(
		fieldChangeCtx.fieldsToResolveOnDeploy,
		fieldChangeCtx.currentPath,
	)
	if bpcore.IsNilMappingNode(newValue) && !bpcore.IsNilMappingNode(currentValue) && !knownOnDeploy {
		changes.RemovedFields = append(changes.RemovedFields, fieldChangeCtx.currentPath)
		return
	}

	if !bpcore.IsNilMappingNode(newValue) && bpcore.IsNilMappingNode(currentValue) {
		changes.NewFields = append(changes.NewFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    nil,
			NewValue:     newValue,
			MustRecreate: false,
			Sensitive:    carriesRecordedSensitiveValue(fieldChangeCtx),
		})
		return
	}

	if !bpcore.IsNilMappingNode(newValue) && !bpcore.IsNilMappingNode(currentValue) {
		changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    currentValue,
			NewValue:     newValue,
			MustRecreate: false,
			Sensitive:    carriesRecordedSensitiveValue(fieldChangeCtx),
		})
	}
}

func collectMappingNodeMapChanges(
	changes *provider.Changes,
	newMap *bpcore.MappingNode,
	currentMap *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	newFields := getFields(newMap)
	for fieldName, newValue := range newFields {
		currentValue := getField(currentMap, fieldName)
		collectMappingNodeChanges(
			changes,
			newValue,
			currentValue,
			mappingNodeChildContext(fieldChangeCtx, substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName)),
		)
	}

	currentFields := getFields(currentMap)
	for fieldName := range currentFields {
		if _, ok := newFields[fieldName]; !ok {
			collectMappingNodeChanges(
				changes,
				nil,
				currentFields[fieldName],
				mappingNodeChildContext(fieldChangeCtx, substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName)),
			)
		}
	}
}

// Reports whether the whole value at this path holds a secret.
func carriesRecordedSensitiveValue(fieldChangeCtx *fieldChangeContext) bool {
	return bpcore.IsPathCoveredBy(
		fieldChangeCtx.currentPath,
		fieldChangeCtx.recordedSensitivePaths,
	)
}

// The mapping node walk has no schema, so it cannot use childContext: that derives
// MustRecreate and sensitivity from one, and would panic on nil.
func mappingNodeChildContext(
	parent *fieldChangeContext,
	currentPath string,
) *fieldChangeContext {
	return &fieldChangeContext{
		fieldsToResolveOnDeploy: parent.fieldsToResolveOnDeploy,
		recordedSensitivePaths:  parent.recordedSensitivePaths,
		currentPath:             currentPath,
		depth:                   parent.depth + 1,
	}
}

func collectMappingNodeArrayChanges(
	changes *provider.Changes,
	newArray *bpcore.MappingNode,
	currentArray *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	newItems := getItems(newArray)
	currentItems := getItems(currentArray)

	for i, newValue := range newItems {
		currentValue := getArrayItem(currentItems, i)
		collectMappingNodeChanges(
			changes,
			newValue,
			currentValue,
			mappingNodeChildContext(fieldChangeCtx, renderFieldArrayPath(fieldChangeCtx.currentPath, i)),
		)
	}

	if len(newItems) < len(currentItems) {
		for i := len(newItems); i < len(currentItems); i++ {
			collectMappingNodeChanges(
				changes,
				nil,
				getArrayItem(currentItems, i),
				mappingNodeChildContext(fieldChangeCtx, renderFieldArrayPath(fieldChangeCtx.currentPath, i)),
			)
		}
	}
}

func collectMappingNodeScalarChanges(
	changes *provider.Changes,
	newScalar *bpcore.MappingNode,
	currentScalar *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	knownOnDeploy := slices.Contains(
		fieldChangeCtx.fieldsToResolveOnDeploy,
		fieldChangeCtx.currentPath,
	)

	if !bpcore.IsNilMappingNode(currentScalar) &&
		bpcore.IsNilMappingNode(newScalar) &&
		!knownOnDeploy {
		changes.RemovedFields = append(changes.RemovedFields, fieldChangeCtx.currentPath)
	}

	if !bpcore.IsNilMappingNode(currentScalar) &&
		!bpcore.ScalarMappingNodeEqual(newScalar, currentScalar) {
		changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    currentScalar,
			NewValue:     newScalar,
			MustRecreate: false,
			Sensitive:    hasRecordedSensitivePath(fieldChangeCtx),
		})
	}

	if bpcore.IsNilMappingNode(currentScalar) &&
		!bpcore.ScalarMappingNodeEqual(newScalar, currentScalar) {
		changes.NewFields = append(changes.NewFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    nil,
			NewValue:     newScalar,
			MustRecreate: false,
			Sensitive:    hasRecordedSensitivePath(fieldChangeCtx),
		})
	}
}
