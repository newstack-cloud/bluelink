package changes

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ResourceChangeGenerator is an interface for a service that handles
// generating changes for a resource based on the current state of the
// resource, the resolved replacement resource spec and the spec definition
// provided by the resource plugin implementation.
type ResourceChangeGenerator interface {
	GenerateChanges(
		ctx context.Context,
		resourceInfo *provider.ResourceInfo,
		resourceImplementation provider.Resource,
		resolveOnDeploy []string,
		params bpcore.BlueprintParams,
	) (*provider.Changes, error)
}

type defaultResourceChangeGenerator struct{}

// NewDefaultResourceChangeGenerator returns a new instance of the default
// implementation of a resource change generator.
func NewDefaultResourceChangeGenerator() ResourceChangeGenerator {
	return &defaultResourceChangeGenerator{}
}

func (s *defaultResourceChangeGenerator) GenerateChanges(
	ctx context.Context,
	resourceInfo *provider.ResourceInfo,
	resourceImplementation provider.Resource,
	resolveOnDeploy []string,
	params bpcore.BlueprintParams,
) (*provider.Changes, error) {
	resolvedResource := resourceInfo.ResourceWithResolvedSubs
	resourceType := GetResourceTypeFromResolved(resolvedResource)
	providerNamespace := provider.ExtractProviderFromItemType(resourceType)
	specDefinitionOutput, err := resourceImplementation.GetSpecDefinition(
		ctx,
		&provider.ResourceGetSpecDefinitionInput{
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				params,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	// there is no need to gather all the detailed changes as there is no real value
	// to comparing field changes between different resource types unlike an update
	// where the resource type remains the same.
	currentStateResourceType := getResourceTypeFromState(resourceInfo.CurrentResourceState)
	newResourceType := GetResourceTypeFromResolved(resourceInfo.ResourceWithResolvedSubs)
	if !AnyEmptyString(currentStateResourceType, newResourceType) &&
		newResourceType != currentStateResourceType {
		return &provider.Changes{
			AppliedResourceInfo: *resourceInfo,
			MustRecreate:        true,
		}, nil
	}

	changes := &provider.Changes{
		AppliedResourceInfo: *resourceInfo,
	}
	newSpec := resourceInfo.ResourceWithResolvedSubs.Spec
	currentResourceSpec := getResourceSpecFromState(resourceInfo.CurrentResourceState)
	resourceElementID := bpcore.ResourceElementID(resourceInfo.ResourceName)
	fieldsToResolveOnDeploy := core.Map(resolveOnDeploy, func(path string, _ int) string {
		return strings.TrimPrefix(path, fmt.Sprintf("%s.", resourceElementID))
	})
	collectSpecFieldChanges(
		changes,
		specDefinitionOutput.SpecDefinition.Schema,
		newSpec,
		currentResourceSpec,
		&fieldChangeContext{
			fieldsToResolveOnDeploy: fieldsToResolveOnDeploy,
			parentMustRecreate:      false,
			parentSensitive:         false,
			recordedSensitivePaths:  resourceInfo.ResourceWithResolvedSubs.SensitiveSpecPaths,
			currentPath:             "spec",
			depth:                   0,
		},
	)

	changes.MustRecreate = mustRecreateResource(changes.ModifiedFields, resourceInfo)
	currentResourceMetadata := getResourceMetadataFromState(resourceInfo.CurrentResourceState)
	collectMetadataFieldChanges(
		changes,
		resourceInfo.ResourceWithResolvedSubs.Metadata,
		currentResourceMetadata,
		&fieldChangeContext{
			fieldsToResolveOnDeploy: fieldsToResolveOnDeploy,
			recordedSensitivePaths:  resourceInfo.ResourceWithResolvedSubs.SensitiveMetadataPaths,
			currentPath:             "metadata",
			depth:                   0,
		},
	)
	changes.ComputedFields = append(
		changes.ComputedFields,
		specmerge.CollectComputedFields(specDefinitionOutput.SpecDefinition.Schema, "spec")...,
	)
	// Computed-when-omitted fields (e.g. auto-generated resource names) are only
	// computed for this change set when the blueprint provides no value for them.
	changes.ComputedFields = append(
		changes.ComputedFields,
		specmerge.CollectComputedWhenOmittedFields(
			specDefinitionOutput.SpecDefinition.Schema,
			newSpec,
			"spec",
		)...,
	)

	return changes, nil
}

func getResourceSpecFromState(resourceState *state.ResourceState) *bpcore.MappingNode {
	if resourceState == nil {
		return nil
	}

	return resourceState.SpecData
}

func getResourceTypeFromState(resourceState *state.ResourceState) string {
	if resourceState == nil {
		return ""
	}

	return resourceState.Type
}

// GetResourceTypeFromResolved extracts a resource type from a resolved resource
// if possible. An empty string is returned if the resolved resource is nil or
// the type is nil.
func GetResourceTypeFromResolved(resourceWithResolvedSubs *provider.ResolvedResource) string {
	if resourceWithResolvedSubs == nil ||
		resourceWithResolvedSubs.Type == nil {
		return ""
	}

	return resourceWithResolvedSubs.Type.Value
}

func getResourceMetadataFromState(
	resourceState *state.ResourceState,
) *state.ResourceMetadataState {
	if resourceState == nil {
		return nil
	}

	return resourceState.Metadata
}

func collectSpecFieldChanges(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	valueInNewSpec *bpcore.MappingNode,
	valueInCurrentState *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	if fieldChangeCtx.depth > bpcore.MappingNodeMaxTraverseDepth {
		return
	}

	if schema.Computed && !schema.TrackDrift {
		// Change staging is not supported for computed fields that are not explicitly
		// marked with `TrackDrift: true` in the schema.
		// Change staging is the process of comparing user-defined values in a new resource
		// spec with the current state of a deployed resource.
		// The user should be presented with information that computed field values will be known
		// after the resource is deployed.
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

	if schema.Type == provider.ResourceDefinitionsSchemaTypeObject &&
		schema.Attributes != nil &&
		isMapOrNil(valueInNewSpec) &&
		isMapOrNil(valueInCurrentState) {
		collectObjectFieldChanges(
			changes,
			schema,
			valueInNewSpec,
			valueInCurrentState,
			fieldChangeCtx,
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeMap &&
		schema.MapValues != nil &&
		isMapOrNil(valueInNewSpec) &&
		isMapOrNil(valueInCurrentState) {
		collectMapFieldChanges(
			changes,
			schema,
			valueInNewSpec,
			valueInCurrentState,
			fieldChangeCtx,
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeArray &&
		schema.Items != nil &&
		isArrayOrNil(valueInNewSpec) &&
		isArrayOrNil(valueInCurrentState) {
		collectArrayFieldChanges(
			changes,
			schema,
			valueInNewSpec,
			valueInCurrentState,
			fieldChangeCtx,
		)
	}

	if isScalarSchemaType(schema.Type) &&
		isScalarOrNil(valueInNewSpec) &&
		isScalarOrNil(valueInCurrentState) {
		collectScalarFieldChanges(
			changes,
			schema,
			valueInNewSpec,
			valueInCurrentState,
			fieldChangeCtx,
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeUnion &&
		schema.OneOf != nil {
		collectUnionFieldChanges(
			changes,
			schema,
			valueInNewSpec,
			valueInCurrentState,
			fieldChangeCtx,
		)
	}
}

func collectScalarFieldChanges(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	scalarInNewSpec *bpcore.MappingNode,
	scalarInCurrentState *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {

	knownOnDeploy := slices.Contains(
		fieldChangeCtx.fieldsToResolveOnDeploy,
		fieldChangeCtx.currentPath,
	)

	if !bpcore.IsNilMappingNode(scalarInCurrentState) &&
		bpcore.IsNilMappingNode(scalarInNewSpec) &&
		!knownOnDeploy {
		changes.RemovedFields = append(changes.RemovedFields, fieldChangeCtx.currentPath)
		return
	}

	if !bpcore.IsNilMappingNode(scalarInCurrentState) &&
		!bpcore.ScalarMappingNodeEqual(scalarInNewSpec, scalarInCurrentState) {
		changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    scalarInCurrentState,
			NewValue:     scalarInNewSpec,
			MustRecreate: fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
			Sensitive:    fieldChangeSensitive(fieldChangeCtx, schema),
		})
		return
	}

	if bpcore.IsNilMappingNode(scalarInCurrentState) &&
		!bpcore.ScalarMappingNodeEqual(scalarInNewSpec, scalarInCurrentState) {
		// For nullable fields with defaults: if the current state is nil (user didn't provide a value)
		// and the new value equals the schema default, treat it as unchanged.
		// This prevents false drift detection when GetExternalState returns the provider's default
		// value for fields that the user intentionally left unspecified.
		if schema.Nullable && schema.Default != nil &&
			bpcore.ScalarMappingNodeEqual(scalarInNewSpec, schema.Default) {
			changes.UnchangedFields = append(changes.UnchangedFields, fieldChangeCtx.currentPath)
			return
		}

		changes.NewFields = append(changes.NewFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    nil,
			NewValue:     scalarInNewSpec,
			MustRecreate: fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
			Sensitive:    fieldChangeSensitive(fieldChangeCtx, schema),
		})
		return
	}

	if bpcore.ScalarMappingNodeEqual(scalarInNewSpec, scalarInCurrentState) && !knownOnDeploy {
		changes.UnchangedFields = append(changes.UnchangedFields, fieldChangeCtx.currentPath)
	}
}

func collectArrayFieldChanges(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	arrayInNewSpec *bpcore.MappingNode,
	arrayInCurrentState *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	newSpecItems := getItems(arrayInNewSpec)
	currentStateItems := getItems(arrayInCurrentState)

	// Sort arrays by the specified field if configured for order-independent comparison.
	// This is useful for arrays like tags where the logical identity is a key field
	// but the order may vary between the blueprint spec and external state.
	if schema.SortArrayByField != "" || schema.IgnoreItemOrder {
		sortedNewSpecItems := SortArrayItemsForComparison(newSpecItems, schema)
		fieldChangeCtx = contextWithSortedArrayPaths(
			fieldChangeCtx,
			newSpecItems,
			sortedNewSpecItems,
		)
		newSpecItems = sortedNewSpecItems
		currentStateItems = SortArrayItemsForComparison(currentStateItems, schema)
	}

	for i, newValue := range newSpecItems {
		currentValue := getArrayItem(currentStateItems, i)
		collectSpecFieldChanges(
			changes,
			schema.Items,
			newValue,
			currentValue,
			childContext(fieldChangeCtx, schema, renderFieldArrayPath(fieldChangeCtx.currentPath, i)),
		)
	}

	if len(newSpecItems) < len(currentStateItems) {
		for i := len(newSpecItems); i < len(currentStateItems); i += 1 {
			collectSpecFieldChanges(
				changes,
				schema.Items,
				nil,
				getArrayItem(currentStateItems, i),
				childContext(fieldChangeCtx, schema, renderFieldArrayPath(fieldChangeCtx.currentPath, i)),
			)
		}
	}
}

func collectObjectFieldChanges(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	objectInNewSpec *bpcore.MappingNode,
	objectInCurrentState *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	for fieldName, fieldSchema := range schema.Attributes {
		collectSpecFieldChanges(
			changes,
			fieldSchema,
			getField(objectInNewSpec, fieldName),
			getField(objectInCurrentState, fieldName),
			childContext(fieldChangeCtx, schema, substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName)),
		)
	}
}

func collectMapFieldChanges(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	mapInNewSpec *bpcore.MappingNode,
	mapInCurrentState *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	newSpecFields := getFields(mapInNewSpec)
	for fieldName, newValue := range newSpecFields {
		currentValue := getField(mapInCurrentState, fieldName)
		collectSpecFieldChanges(
			changes,
			schema.MapValues,
			newValue,
			currentValue,
			childContext(fieldChangeCtx, schema, substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName)),
		)
	}

	// Collect changes for key/value pairs that have been removed in the new spec.
	currentSpecFields := getFields(mapInCurrentState)
	for fieldName, currentValue := range currentSpecFields {
		if _, ok := newSpecFields[fieldName]; !ok {
			collectSpecFieldChanges(
				changes,
				schema.MapValues,
				nil,
				currentValue,
				childContext(fieldChangeCtx, schema, substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName)),
			)
		}
	}
}

func collectUnionFieldChanges(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	unionValueInNewSpec *bpcore.MappingNode,
	unionValueInCurrentState *bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {

	mappingNodeTypeMatchInfo := checkMappingNodeTypes(unionValueInCurrentState, unionValueInNewSpec, schema)
	if unionValueInCurrentState != nil && !mappingNodeTypeMatchInfo.TypeMatches {
		// Carry out a shallow comparison when the types don't match for the two values.
		changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    unionValueInCurrentState,
			NewValue:     unionValueInNewSpec,
			MustRecreate: fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
			// unionHasSensitiveVariant also covers the union schema's own declaration,
			// so this adds only the variants to what fieldChangeSensitive already knows.
			Sensitive: fieldChangeSensitive(fieldChangeCtx, schema) ||
				unionHasSensitiveVariant(schema),
		})
		return
	}

	collectSpecFieldChanges(
		changes,
		// Use the schema of the matching type to compare the values.
		mappingNodeTypeMatchInfo.Schema,
		unionValueInNewSpec,
		unionValueInCurrentState,
		&fieldChangeContext{
			fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
			recordedSensitivePaths:  fieldChangeCtx.recordedSensitivePaths,
			// Ensure that if the field with the union type changes requires a resource to be recreated,
			// this information is passed down to all child values.
			parentMustRecreate: fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
			parentSensitive:    fieldChangeSensitive(fieldChangeCtx, schema),
			currentPath:        fieldChangeCtx.currentPath,
			depth:              fieldChangeCtx.depth,
		},
	)
}

func unionHasSensitiveVariant(
	unionSchema *provider.ResourceDefinitionsSchema,
) bool {
	if unionSchema.Sensitive {
		return true
	}

	for _, typeInUnion := range unionSchema.OneOf {
		if typeInUnion.Sensitive {
			return true
		}
	}

	return false
}

func collectMetadataFieldChanges(
	changes *provider.Changes,
	newResourceMetadata *provider.ResolvedResourceMetadata,
	currentResourceMetadata *state.ResourceMetadataState,
	fieldChangeCtx *fieldChangeContext,
) {
	displayNamePath := "metadata.displayName"
	newDisplayName := extractDisplayNameFromResolved(newResourceMetadata)
	currentDisplayName := extractDisplayNameFromState(currentResourceMetadata)
	if currentDisplayName != "" && newDisplayName != currentDisplayName {
		changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
			FieldPath: displayNamePath,
			PrevValue: &bpcore.MappingNode{
				Scalar: &bpcore.ScalarValue{StringValue: &currentDisplayName},
			},
			NewValue: &bpcore.MappingNode{
				Scalar: &bpcore.ScalarValue{StringValue: &newDisplayName},
			},
			MustRecreate: false,
			Sensitive:    isPathRecordedSensitive(fieldChangeCtx.recordedSensitivePaths, displayNamePath),
		})
	} else if newDisplayName != currentDisplayName {
		changes.NewFields = append(changes.NewFields, provider.FieldChange{
			FieldPath: displayNamePath,
			PrevValue: nil,
			NewValue: &bpcore.MappingNode{
				Scalar: &bpcore.ScalarValue{StringValue: &newDisplayName},
			},
			MustRecreate: false,
			Sensitive:    isPathRecordedSensitive(fieldChangeCtx.recordedSensitivePaths, displayNamePath),
		})
	}

	collectMetadataLabelChanges(
		changes,
		extractLabelsFromResolved(newResourceMetadata),
		extractLabelsFromState(currentResourceMetadata),
		fieldChangeCtx,
	)

	collectMetadataAnnotationChanges(
		changes,
		extractAnnotationsFromResolved(newResourceMetadata),
		extractAnnotationsFromState(currentResourceMetadata),
		fieldChangeCtx,
	)

	collectMappingNodeChanges(
		changes,
		extractCustomFromResolved(newResourceMetadata),
		extractCustomFromState(currentResourceMetadata),
		&fieldChangeContext{
			fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
			recordedSensitivePaths:  fieldChangeCtx.recordedSensitivePaths,
			currentPath:             "metadata.custom",
			depth:                   0,
		},
	)
}

func collectMetadataLabelChanges(
	changes *provider.Changes,
	newLabelsMapWrapper *schema.StringMap,
	currentLabels map[string]string,
	fieldChangeCtx *fieldChangeContext,
) {
	newLabels := map[string]string{}
	if newLabelsMapWrapper != nil {
		newLabels = newLabelsMapWrapper.Values
	}

	for key, newValue := range newLabels {
		currentPath := renderMetadataLabelFieldPath(key)

		if slices.Contains(fieldChangeCtx.fieldsToResolveOnDeploy, currentPath) {
			changes.FieldChangesKnownOnDeploy = append(
				changes.FieldChangesKnownOnDeploy,
				currentPath,
			)
		}

		currentValue, hasCurrentValue := currentLabels[key]
		if hasCurrentValue && newValue != currentValue {
			changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
				FieldPath:    currentPath,
				PrevValue:    &bpcore.MappingNode{Scalar: &bpcore.ScalarValue{StringValue: &currentValue}},
				NewValue:     &bpcore.MappingNode{Scalar: &bpcore.ScalarValue{StringValue: &newValue}},
				MustRecreate: false,
				Sensitive:    isPathRecordedSensitive(fieldChangeCtx.recordedSensitivePaths, currentPath),
			})
		}

		if !hasCurrentValue {
			changes.NewFields = append(changes.NewFields, provider.FieldChange{
				FieldPath:    currentPath,
				PrevValue:    nil,
				NewValue:     &bpcore.MappingNode{Scalar: &bpcore.ScalarValue{StringValue: &newValue}},
				MustRecreate: false,
				Sensitive:    isPathRecordedSensitive(fieldChangeCtx.recordedSensitivePaths, currentPath),
			})
		}
	}

	for key := range currentLabels {
		if _, ok := newLabels[key]; !ok {
			if !slices.Contains(
				fieldChangeCtx.fieldsToResolveOnDeploy,
				renderMetadataLabelFieldPath(key),
			) {
				changes.RemovedFields = append(
					changes.RemovedFields,
					renderMetadataLabelFieldPath(key),
				)
			}
		}
	}
}

func collectMetadataAnnotationChanges(
	changes *provider.Changes,
	newAnnotationsMap *bpcore.MappingNode,
	currentAnnotations map[string]*bpcore.MappingNode,
	fieldChangeCtx *fieldChangeContext,
) {
	newAnnotations := map[string]*bpcore.MappingNode{}
	if newAnnotationsMap != nil {
		newAnnotations = newAnnotationsMap.Fields
	}

	for key, newValue := range newAnnotations {
		currentPath := renderMetadataAnnotationFieldPath(key)

		if slices.Contains(fieldChangeCtx.fieldsToResolveOnDeploy, currentPath) {
			changes.FieldChangesKnownOnDeploy = append(
				changes.FieldChangesKnownOnDeploy,
				currentPath,
			)
		}

		currentValue, hasCurrentValue := currentAnnotations[key]
		if hasCurrentValue &&
			bpcore.StringValue(newValue) != bpcore.StringValue(currentValue) {
			changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
				FieldPath:    currentPath,
				PrevValue:    currentValue,
				NewValue:     newValue,
				MustRecreate: false,
				Sensitive:    isPathRecordedSensitive(fieldChangeCtx.recordedSensitivePaths, currentPath),
			})
		}

		if !hasCurrentValue {
			changes.NewFields = append(changes.NewFields, provider.FieldChange{
				FieldPath:    currentPath,
				PrevValue:    nil,
				NewValue:     newValue,
				MustRecreate: false,
				Sensitive:    isPathRecordedSensitive(fieldChangeCtx.recordedSensitivePaths, currentPath),
			})
		}
	}

	for key := range currentAnnotations {
		if _, ok := newAnnotations[key]; !ok {
			if !slices.Contains(
				fieldChangeCtx.fieldsToResolveOnDeploy,
				renderMetadataAnnotationFieldPath(key),
			) {
				changes.RemovedFields = append(
					changes.RemovedFields,
					renderMetadataAnnotationFieldPath(key),
				)
			}
		}
	}
}

type fieldChangeContext struct {
	fieldsToResolveOnDeploy []string
	parentMustRecreate      bool
	// Whether an ancestor of the field is sensitive, from the schema or a recorded path.
	//
	// Sensitivity is declared on the field a plugin author thinks of as secret,
	// which for a map or object of secrets is the container rather than each of
	// its entries. Without inheriting it, the leaves carrying the actual values
	// are the ones that come out unmarked.
	parentSensitive bool
	// Field paths a secret reached while the resource's substitutions were resolved, rooted
	// at "spec" for the spec walk and at "metadata" for the metadata one.
	//
	// Separate from the schema's own declarations, a secret interpolated into a field no
	// schema declares as sensitive is visible only at resolution, and without these the
	// change set reports it as plain while the deployment that follows marks it.
	recordedSensitivePaths []string
	currentPath            string
	depth                  int
}

func getField(node *bpcore.MappingNode, fieldName string) *bpcore.MappingNode {
	if node == nil {
		return nil
	}

	if node.Fields != nil {
		return node.Fields[fieldName]
	}

	return nil
}

func getItems(node *bpcore.MappingNode) []*bpcore.MappingNode {
	if node == nil {
		return []*bpcore.MappingNode{}
	}

	return node.Items
}

func getArrayItem(node []*bpcore.MappingNode, index int) *bpcore.MappingNode {
	if node == nil || index >= len(node) {
		return nil
	}

	return node[index]
}

// SortArrayItemsForComparison returns the items in the order two arrays must be in to be
// compared without their order being read as a difference.
//
// Which order that is comes from the schema, since only the resource type knows whether a
// field's order carries meaning. Items are ordered by the field that carries their
// identity when the schema names one, and by their own value when it declares the order
// insignificant. An array the schema says nothing about is returned untouched, so order
// remains significant unless a resource type says otherwise.
func SortArrayItemsForComparison(
	items []*bpcore.MappingNode,
	schema *provider.ResourceDefinitionsSchema,
) []*bpcore.MappingNode {
	if schema == nil {
		return items
	}

	if schema.SortArrayByField != "" {
		return sortArrayItemsByField(items, schema.SortArrayByField)
	}

	if schema.IgnoreItemOrder {
		return sortArrayItemsByValue(items)
	}

	return items
}

// Ordered by each item's own value, for an array the schema declares unordered.
//
// Items that are not scalars sort to the end and keep their relative order. A schema
// declaring the order of an array of objects insignificant has no identity field to
// compare them by, which is what SortArrayByField is for, so there is nothing better to do
// with them than leave them as they are.
func sortArrayItemsByValue(items []*bpcore.MappingNode) []*bpcore.MappingNode {
	if len(items) == 0 {
		return items
	}

	sorted := make([]*bpcore.MappingNode, len(items))
	copy(sorted, items)

	slices.SortStableFunc(sorted, func(a, b *bpcore.MappingNode) int {
		return strings.Compare(scalarSortKey(a), scalarSortKey(b))
	})

	return sorted
}

// Ranked so that scalars order among themselves by value and anything else follows them,
// which keeps a non-scalar from sorting ahead of every scalar on an empty key.
func scalarSortKey(node *bpcore.MappingNode) string {
	if node == nil || node.Scalar == nil {
		return "1"
	}

	return "0" + node.Scalar.ToString()
}

// Returns a sorted copy of the array items based on the
// string value of the specified field. Items without the field or with non-string
// values for the field are sorted to the end.
// This is used for order-independent comparison of arrays like tags.
func sortArrayItemsByField(items []*bpcore.MappingNode, fieldName string) []*bpcore.MappingNode {
	if len(items) == 0 || fieldName == "" {
		return items
	}

	// Create a copy to avoid mutating the original
	sorted := make([]*bpcore.MappingNode, len(items))
	copy(sorted, items)

	slices.SortFunc(sorted, func(a, b *bpcore.MappingNode) int {
		aKey := getFieldStringValue(a, fieldName)
		bKey := getFieldStringValue(b, fieldName)
		return strings.Compare(aKey, bKey)
	})

	return sorted
}

// getFieldStringValue extracts the string value of a field from a mapping node.
// Returns empty string if the node is nil, doesn't have the field, or the field
// is not a string scalar.
func getFieldStringValue(node *bpcore.MappingNode, fieldName string) string {
	if node == nil || node.Fields == nil {
		return ""
	}

	field := node.Fields[fieldName]
	if field == nil || field.Scalar == nil || field.Scalar.StringValue == nil {
		return ""
	}

	return *field.Scalar.StringValue
}

func isMapOrNil(node *bpcore.MappingNode) bool {
	return bpcore.IsNilMappingNode(node) || node.Fields != nil
}

func isArrayOrNil(node *bpcore.MappingNode) bool {
	return bpcore.IsNilMappingNode(node) || node.Items != nil
}

func isScalarOrNil(node *bpcore.MappingNode) bool {
	return bpcore.IsNilMappingNode(node) || node.Scalar != nil
}

// MappingNodeTypeMatchInfo contains information about whether the types of two mapping nodes match
// and the schema to be used for value comparisons.
type MappingNodeTypeMatchInfo struct {
	TypeMatches bool
	Schema      *provider.ResourceDefinitionsSchema
}

func checkMappingNodeTypes(
	currentValue, newValue *bpcore.MappingNode,
	unionSchema *provider.ResourceDefinitionsSchema,
) *MappingNodeTypeMatchInfo {
	if currentValue == nil && newValue == nil {
		return &MappingNodeTypeMatchInfo{
			TypeMatches: true,
			Schema:      unionSchema.OneOf[0],
		}
	}

	if isMapOrNil(currentValue) && bpcore.IsObjectMappingNode(newValue) {
		// For maps and objects, the fields need to be compared with options in the union
		// schema to determine the correct schema to be used for value comparisons.
		// This is important as a field in an object may have "MustRecreate" set to true
		// and if it is misclassified as a key/value pair in a map, it will not be taken into
		// account when determining whether a resource should be recreated.
		// This does not do a deep check on the structure of each field, only a shallow check
		// based on the field names for objects.
		// For maps, no checks are carried out on the structure of the fields,
		// the first map schema present in the union schema will be used.
		// It is best to advise provider plugin developers to use multiple precise object
		// types in unions instead of multiple map definitions to avoid this issue.
		return CheckMappingNodeTypesForFields(getFields(currentValue), getFields(newValue), unionSchema)
	}

	if isArrayOrNil(currentValue) && bpcore.IsArrayMappingNode(newValue) {
		return &MappingNodeTypeMatchInfo{
			TypeMatches: true,
			// This does not guarantee selection of the correct schema in a union with
			// multiple array definitions.
			// This can impact the "MustRecreate" flag set on field schemas to determine
			// whether a resource should be recreated when a specified field changes value.
			// Provider plugin developers should ensure "MustRecreate" is set on the union
			// type so all values that can populate a field are treated the same in this regard.
			Schema: GetArraySchema(unionSchema.OneOf),
		}
	}

	if isScalarOrNil(currentValue) && bpcore.IsScalarMappingNode(newValue) {
		return &MappingNodeTypeMatchInfo{
			TypeMatches: true,
			// This does not guarantee selection of the correct schema in a union with
			// multiple literal definitions.
			// This can impact the "MustRecreate" flag set on field schemas to determine
			// whether a resource should be recreated when a specified field changes value.
			// Provider plugin developers should ensure "MustRecreate" is set on the union
			// type so all values that can populate a field are treated the same in this regard.
			Schema: getScalarSchema(unionSchema.OneOf),
		}
	}

	return &MappingNodeTypeMatchInfo{
		TypeMatches: false,
		Schema:      nil,
	}
}

// CheckMappingNodeTypesForFields compares the fields in two mapping nodes to determine
// whether the types of the fields match the types in the union schema.
func CheckMappingNodeTypesForFields(
	fieldsA, fieldsB map[string]*bpcore.MappingNode,
	unionSchema *provider.ResourceDefinitionsSchema,
) *MappingNodeTypeMatchInfo {
	typeMatches := false
	var schema *provider.ResourceDefinitionsSchema
	// If the union schema has one or more object types, try to match against the object type
	// for a precise match for the structure of the fields before considering a map type.
	objectSchemas := getObjectSchemas(unionSchema.OneOf)
	i := 0
	for !typeMatches && i < len(objectSchemas) {
		objectSchema := objectSchemas[i]
		if (isSuperset(fieldsA, fieldsB) && objectSchemaContainsFields(objectSchema, fieldsA)) ||
			(isSuperset(fieldsB, fieldsA) && objectSchemaContainsFields(objectSchema, fieldsB)) {
			typeMatches = true
			schema = objectSchema
		}

		i += 1
	}

	if !typeMatches {
		// If the union schema has one or more map types, use the first map type to compare
		// the values in the fields.
		// This means that if a union schema has multiple map types, any nested fields
		// with the "MustRecreate" flagged set to true will not be taken into account when
		// determining whether a resource should be recreated.
		// It is best to advise provider plugin developers to use multiple precise object
		// types in unions instead of maps to avoid this issue.
		mapSchema := getMapSchema(unionSchema.OneOf)
		if mapSchema != nil {
			typeMatches = true
			schema = mapSchema
		}
	}

	return &MappingNodeTypeMatchInfo{
		TypeMatches: typeMatches,
		Schema:      schema,
	}
}

func getFields(node *bpcore.MappingNode) map[string]*bpcore.MappingNode {
	if node == nil {
		return nil
	}

	return node.Fields
}

func extractDisplayNameFromResolved(
	resourceMetadata *provider.ResolvedResourceMetadata,
) string {
	if resourceMetadata == nil {
		return ""
	}

	return bpcore.StringValue(resourceMetadata.DisplayName)
}

func extractDisplayNameFromState(
	resourceMetadataState *state.ResourceMetadataState,
) string {
	if resourceMetadataState == nil {
		return ""
	}

	return resourceMetadataState.DisplayName
}

func extractLabelsFromResolved(
	resourceMetadata *provider.ResolvedResourceMetadata,
) *schema.StringMap {
	if resourceMetadata == nil || resourceMetadata.Labels == nil {
		return nil
	}

	return resourceMetadata.Labels
}

func extractLabelsFromState(
	resourceMetadataState *state.ResourceMetadataState,
) map[string]string {
	if resourceMetadataState == nil {
		return nil
	}

	return resourceMetadataState.Labels
}

func extractAnnotationsFromResolved(
	resourceMetadata *provider.ResolvedResourceMetadata,
) *bpcore.MappingNode {
	if resourceMetadata == nil || resourceMetadata.Annotations == nil {
		return nil
	}

	return resourceMetadata.Annotations
}

func extractAnnotationsFromState(
	resourceMetadataState *state.ResourceMetadataState,
) map[string]*bpcore.MappingNode {
	if resourceMetadataState == nil {
		return nil
	}

	return resourceMetadataState.Annotations
}

func extractCustomFromResolved(
	resourceMetadata *provider.ResolvedResourceMetadata,
) *bpcore.MappingNode {
	if resourceMetadata == nil || resourceMetadata.Custom == nil {
		return nil
	}

	return resourceMetadata.Custom
}

func extractCustomFromState(
	resourceMetadataState *state.ResourceMetadataState,
) *bpcore.MappingNode {
	if resourceMetadataState == nil {
		return nil
	}

	return resourceMetadataState.Custom
}

func isSuperset(
	candidateSuperset map[string]*bpcore.MappingNode,
	supersetOf map[string]*bpcore.MappingNode,
) bool {

	for key := range supersetOf {
		if _, ok := candidateSuperset[key]; !ok {
			return false
		}
	}

	return true
}

func objectSchemaContainsFields(
	objectSchema *provider.ResourceDefinitionsSchema,
	fields map[string]*bpcore.MappingNode,
) bool {
	for fieldName := range fields {
		if _, ok := objectSchema.Attributes[fieldName]; !ok {
			return false
		}
	}

	return true
}

func isScalarSchemaType(schemaType provider.ResourceDefinitionsSchemaType) bool {
	return schemaType == provider.ResourceDefinitionsSchemaTypeString ||
		schemaType == provider.ResourceDefinitionsSchemaTypeInteger ||
		schemaType == provider.ResourceDefinitionsSchemaTypeFloat ||
		schemaType == provider.ResourceDefinitionsSchemaTypeBoolean
}

func mustRecreateResource(fieldChanges []provider.FieldChange, resourceInfo *provider.ResourceInfo) bool {
	if resourceInfo.CurrentResourceState == nil {
		return false
	}

	mustRecreate := false
	i := 0
	for !mustRecreate && i < len(fieldChanges) {
		if fieldChanges[i].MustRecreate {
			mustRecreate = true
		}
		i += 1
	}

	return mustRecreate
}

func fieldChangeMustRecreateResource(parentMustRecreate bool, schema *provider.ResourceDefinitionsSchema) bool {
	return (parentMustRecreate || schema.MustRecreate) && !schema.Computed
}

// Matches downwards only. A field is sensitive from a path recorded on itself or on one of
// its containers; a secret in one entry of a map says nothing about its siblings, and
// marking the container would redact all of them.
// Moves recorded sensitivity paths with the items they name.
//
// Recorded indices are positions in the resolved spec, and sorting moves items away from
// them, so a path left alone would mark whichever item now sits at the index it named. That
// is a mark on the wrong item as it hides a value that is not secret and shows one that is.
// Sorting reuses the same nodes, so each item's new position is recoverable.
func contextWithSortedArrayPaths(
	fieldChangeCtx *fieldChangeContext,
	items []*bpcore.MappingNode,
	sortedItems []*bpcore.MappingNode,
) *fieldChangeContext {
	if len(fieldChangeCtx.recordedSensitivePaths) == 0 {
		return fieldChangeCtx
	}

	sortedPositions := sortedItemPositions(items, sortedItems)
	moved := make([]string, 0, len(fieldChangeCtx.recordedSensitivePaths))
	for _, recorded := range fieldChangeCtx.recordedSensitivePaths {
		moved = append(
			moved,
			pathWithSortedIndex(recorded, fieldChangeCtx.currentPath, sortedPositions),
		)
	}

	movedCtx := *fieldChangeCtx
	movedCtx.recordedSensitivePaths = moved

	return &movedCtx
}

// Where each item ended up, by position in the original slice.
//
// Matched by node identity, since sorting reorders the same pointers. Equal values appearing
// twice are interchangeable, so taking positions in order is enough.
func sortedItemPositions(
	items []*bpcore.MappingNode,
	sortedItems []*bpcore.MappingNode,
) []int {
	positions := make([]int, len(items))
	taken := make([]bool, len(sortedItems))
	for i, item := range items {
		positions[i] = i
		for j, sortedItem := range sortedItems {
			if !taken[j] && sortedItem == item {
				positions[i] = j
				taken[j] = true
				break
			}
		}
	}

	return positions
}

// Rewrites the index of a path addressing an item of the array at arrayPath, leaving
// anything below that index in place. Paths naming another array are returned unchanged.
func pathWithSortedIndex(path string, arrayPath string, sortedPositions []int) string {
	if !strings.HasPrefix(path, arrayPath+"[") {
		return path
	}

	indexPart := path[len(arrayPath)+1:]
	before, after, ok := strings.Cut(indexPart, "]")
	if !ok {
		return path
	}

	index, err := strconv.Atoi(before)
	if err != nil || index < 0 || index >= len(sortedPositions) {
		return path
	}

	return fmt.Sprintf(
		"%s[%d]%s",
		arrayPath,
		sortedPositions[index],
		after,
	)
}

// Derives the context for a field one level down.
func childContext(
	parent *fieldChangeContext,
	schema *provider.ResourceDefinitionsSchema,
	currentPath string,
) *fieldChangeContext {
	return &fieldChangeContext{
		fieldsToResolveOnDeploy: parent.fieldsToResolveOnDeploy,
		recordedSensitivePaths:  parent.recordedSensitivePaths,
		parentMustRecreate:      fieldChangeMustRecreateResource(parent.parentMustRecreate, schema),
		parentSensitive:         fieldChangeSensitive(parent, schema),
		currentPath:             currentPath,
		depth:                   parent.depth + 1,
	}
}

func hasRecordedSensitivePath(fieldChangeCtx *fieldChangeContext) bool {
	return isPathRecordedSensitive(
		fieldChangeCtx.recordedSensitivePaths,
		fieldChangeCtx.currentPath,
	)
}

func isPathRecordedSensitive(recordedSensitivePaths []string, path string) bool {
	return slices.ContainsFunc(recordedSensitivePaths, func(recorded string) bool {
		return bpcore.IsPathAtOrBelow(recorded, path)
	})
}

// Unlike MustRecreate, a computed field is not exempt as a value the provider
// computed from a secret is still a secret.
func fieldChangeSensitive(
	fieldChangeCtx *fieldChangeContext,
	schema *provider.ResourceDefinitionsSchema,
) bool {
	return fieldChangeCtx.parentSensitive ||
		schema.Sensitive ||
		hasRecordedSensitivePath(fieldChangeCtx)
}

func getObjectSchemas(
	schemas []*provider.ResourceDefinitionsSchema,
) []*provider.ResourceDefinitionsSchema {
	return core.Filter(schemas, func(schema *provider.ResourceDefinitionsSchema, _ int) bool {
		return schema.Type == provider.ResourceDefinitionsSchemaTypeObject
	})
}

func getMapSchema(
	schemas []*provider.ResourceDefinitionsSchema,
) *provider.ResourceDefinitionsSchema {
	mapSchema := (*provider.ResourceDefinitionsSchema)(nil)
	i := 0
	for mapSchema == nil && i < len(schemas) {
		if schemas[i].Type == provider.ResourceDefinitionsSchemaTypeMap {
			mapSchema = schemas[i]
		}
		i += 1
	}

	return mapSchema
}

// GetArraySchema returns the first array schema in a list of schemas.
func GetArraySchema(
	schemas []*provider.ResourceDefinitionsSchema,
) *provider.ResourceDefinitionsSchema {
	arraySchema := (*provider.ResourceDefinitionsSchema)(nil)
	i := 0
	for arraySchema == nil && i < len(schemas) {
		if schemas[i].Type == provider.ResourceDefinitionsSchemaTypeArray {
			arraySchema = schemas[i]
		}
		i += 1
	}

	return arraySchema
}

func getScalarSchema(
	schemas []*provider.ResourceDefinitionsSchema,
) *provider.ResourceDefinitionsSchema {
	scalarSchema := (*provider.ResourceDefinitionsSchema)(nil)
	i := 0
	for scalarSchema == nil && i < len(schemas) {
		if isScalarSchemaType(schemas[i].Type) {
			scalarSchema = schemas[i]
		}
		i += 1
	}

	return scalarSchema
}

func renderFieldArrayPath(currentPath string, index int) string {
	return fmt.Sprintf("%s[%d]", currentPath, index)
}

func renderMetadataLabelFieldPath(labelKey string) string {
	return substitutions.RenderKeyedFieldPath("metadata.labels", labelKey)
}

func renderMetadataAnnotationFieldPath(annotationKey string) string {
	return substitutions.RenderKeyedFieldPath("metadata.annotations", annotationKey)
}
