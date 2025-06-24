package changes

import (
	"context"
	"fmt"
	"slices"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
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
			currentPath:             "metadata",
			depth:                   0,
		},
	)
	collectComputedFields(
		changes,
		specDefinitionOutput.SpecDefinition.Schema,
		&fieldChangeContext{
			currentPath: "spec",
			depth:       0,
		},
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

	if schema.Computed {
		// Change staging is not supported for computed fields.
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
		})
		return
	}

	if bpcore.IsNilMappingNode(scalarInCurrentState) &&
		!bpcore.ScalarMappingNodeEqual(scalarInNewSpec, scalarInCurrentState) {
		changes.NewFields = append(changes.NewFields, provider.FieldChange{
			FieldPath:    fieldChangeCtx.currentPath,
			PrevValue:    nil,
			NewValue:     scalarInNewSpec,
			MustRecreate: fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
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

	for i, newValue := range newSpecItems {
		currentValue := getArrayItem(currentStateItems, i)
		collectSpecFieldChanges(
			changes,
			schema.Items,
			newValue,
			currentValue,
			&fieldChangeContext{
				fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
				parentMustRecreate:      fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
				currentPath:             renderFieldArrayPath(fieldChangeCtx.currentPath, i),
				depth:                   fieldChangeCtx.depth + 1,
			},
		)
	}

	if len(newSpecItems) < len(currentStateItems) {
		for i := len(newSpecItems); i < len(currentStateItems); i += 1 {
			collectSpecFieldChanges(
				changes,
				schema.Items,
				nil,
				getArrayItem(currentStateItems, i),
				&fieldChangeContext{
					fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
					parentMustRecreate:      fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
					currentPath:             renderFieldArrayPath(fieldChangeCtx.currentPath, i),
					depth:                   fieldChangeCtx.depth + 1,
				},
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
			&fieldChangeContext{
				fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
				parentMustRecreate:      fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
				currentPath:             substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName),
				depth:                   fieldChangeCtx.depth + 1,
			},
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
			&fieldChangeContext{
				fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
				parentMustRecreate:      fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
				currentPath:             substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName),
				depth:                   fieldChangeCtx.depth + 1,
			},
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
				&fieldChangeContext{
					fieldsToResolveOnDeploy: fieldChangeCtx.fieldsToResolveOnDeploy,
					parentMustRecreate:      fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
					currentPath:             substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName),
					depth:                   fieldChangeCtx.depth + 1,
				},
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
			// Ensure that if the field with the union type changes requires a resource to be recreated,
			// this information is passed down to all child values.
			parentMustRecreate: fieldChangeMustRecreateResource(fieldChangeCtx.parentMustRecreate, schema),
			currentPath:        fieldChangeCtx.currentPath,
			depth:              fieldChangeCtx.depth,
		},
	)
}

func collectMetadataFieldChanges(
	changes *provider.Changes,
	newResourceMetadata *provider.ResolvedResourceMetadata,
	currentResourceMetadata *state.ResourceMetadataState,
	fieldChangeCtx *fieldChangeContext,
) {
	newDisplayName := extractDisplayNameFromResolved(newResourceMetadata)
	currentDisplayName := extractDisplayNameFromState(currentResourceMetadata)
	if currentDisplayName != "" && newDisplayName != currentDisplayName {
		changes.ModifiedFields = append(changes.ModifiedFields, provider.FieldChange{
			FieldPath: "metadata.displayName",
			PrevValue: &bpcore.MappingNode{
				Scalar: &bpcore.ScalarValue{StringValue: &currentDisplayName},
			},
			NewValue: &bpcore.MappingNode{
				Scalar: &bpcore.ScalarValue{StringValue: &newDisplayName},
			},
			MustRecreate: false,
		})
	} else if newDisplayName != currentDisplayName {
		changes.NewFields = append(changes.NewFields, provider.FieldChange{
			FieldPath: "metadata.displayName",
			PrevValue: nil,
			NewValue: &bpcore.MappingNode{
				Scalar: &bpcore.ScalarValue{StringValue: &newDisplayName},
			},
			MustRecreate: false,
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
			})
		}

		if !hasCurrentValue {
			changes.NewFields = append(changes.NewFields, provider.FieldChange{
				FieldPath:    currentPath,
				PrevValue:    nil,
				NewValue:     &bpcore.MappingNode{Scalar: &bpcore.ScalarValue{StringValue: &newValue}},
				MustRecreate: false,
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
			})
		}

		if !hasCurrentValue {
			changes.NewFields = append(changes.NewFields, provider.FieldChange{
				FieldPath:    currentPath,
				PrevValue:    nil,
				NewValue:     newValue,
				MustRecreate: false,
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

func collectComputedFields(
	changes *provider.Changes,
	schema *provider.ResourceDefinitionsSchema,
	fieldChangeCtx *fieldChangeContext,
) {
	if schema.Computed {
		changes.ComputedFields = append(changes.ComputedFields, fieldChangeCtx.currentPath)
		return
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeObject {
		for fieldName, fieldSchema := range schema.Attributes {
			collectComputedFields(
				changes,
				fieldSchema,
				&fieldChangeContext{
					currentPath: substitutions.RenderFieldPath(fieldChangeCtx.currentPath, fieldName),
					depth:       fieldChangeCtx.depth + 1,
				},
			)
		}
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeMap {
		collectComputedFields(
			changes,
			schema.MapValues,
			&fieldChangeContext{
				currentPath: substitutions.RenderFieldPath(fieldChangeCtx.currentPath, "<key>"),
				depth:       fieldChangeCtx.depth + 1,
			},
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeArray {
		collectComputedFields(
			changes,
			schema.Items,
			&fieldChangeContext{
				// 0 is a placeholder for any array index.
				currentPath: renderFieldArrayPath(fieldChangeCtx.currentPath, 0),
				depth:       fieldChangeCtx.depth + 1,
			},
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeUnion {
		for _, unionSchema := range schema.OneOf {
			collectComputedFields(
				changes,
				unionSchema,
				fieldChangeCtx,
			)
		}
	}
}

type fieldChangeContext struct {
	fieldsToResolveOnDeploy []string
	parentMustRecreate      bool
	currentPath             string
	depth                   int
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
	return fmt.Sprintf("metadata.labels[\"%s\"]", labelKey)
}

func renderMetadataAnnotationFieldPath(annotationKey string) string {
	return fmt.Sprintf("metadata.annotations[\"%s\"]", annotationKey)
}
