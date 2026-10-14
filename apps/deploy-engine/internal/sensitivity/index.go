// Package sensitivity answers whether a field of a resource or data source holds a value
// its plugin declared sensitive.
//
// Sensitivity is declared on the field a plugin author thinks of as secret, which for a map
// or object of secrets is the container rather than each of its entries, so answering the
// question means walking a schema and inheriting what the ancestors declared. The engine is
// the only component holding both the provider registries and the field paths that appear in
// state, changesets and drift records, which is why this lives here.
//
// Nothing here redacts. It marks, and leaves what to do about it to the caller.
package sensitivity

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/drift"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
)

// Index reports which fields of a type hold sensitive values, from the spec schemas the
// provider registries expose.
//
// Schemas are fetched once per type and kept: a plugin's schema is fixed for as long as the
// plugin is running, and the registry call may cross a process boundary.
type Index struct {
	resources         resourcehelpers.Registry
	dataSources       provider.DataSourceRegistry
	mu                sync.RWMutex
	resourceSchemas   map[string]*provider.ResourceDefinitionsSchema
	dataSourceSchemas map[string]map[string]*provider.DataSourceSpecSchema
}

// NewIndex returns an index over the given registries.
func NewIndex(
	resources resourcehelpers.Registry,
	dataSources provider.DataSourceRegistry,
) *Index {
	return &Index{
		resources:         resources,
		dataSources:       dataSources,
		resourceSchemas:   map[string]*provider.ResourceDefinitionsSchema{},
		dataSourceSchemas: map[string]map[string]*provider.DataSourceSpecSchema{},
	}
}

// ResourceField reports whether a field of a resource's spec holds a sensitive value.
//
// The path is the form that appears in field changes and resource state mappings, with the
// "spec" prefix optional: "spec.secureValues.dbPassword" and "secureValues.dbPassword"
// address the same field. A path addressing anything other than the spec, such as a
// metadata field change, is not sensitive as far as this index is concerned; see the note on
// the walk below.
//
// A field is sensitive when it or any of its ancestors is declared sensitive, so marking a
// map of secrets marks everything in it. Computed fields are deliberately not exempt as a
// value a provider computed from a secret is still a secret.
func (i *Index) ResourceField(
	ctx context.Context,
	resourceType string,
	fieldPath string,
) (bool, error) {
	schema, err := i.resourceSchema(ctx, resourceType)
	if err != nil {
		return false, err
	}

	return resourceFieldSensitive(schema, fieldPath), nil
}

// DataSourceField reports whether an exported field of a data source holds a sensitive
// value.
//
// Data source specs hold a flat set of fields of primitives or arrays of primitives, so
// there is no nesting to inherit through.
func (i *Index) DataSourceField(
	ctx context.Context,
	dataSourceType string,
	fieldName string,
) (bool, error) {
	fields, err := i.dataSourceFields(ctx, dataSourceType)
	if err != nil {
		return false, err
	}

	field, found := fields[fieldName]
	if !found || field == nil {
		return false, nil
	}

	return field.Sensitive, nil
}

func (i *Index) resourceSchema(
	ctx context.Context,
	resourceType string,
) (*provider.ResourceDefinitionsSchema, error) {
	i.mu.RLock()
	cached, found := i.resourceSchemas[resourceType]
	i.mu.RUnlock()
	if found {
		return cached, nil
	}

	output, err := i.resources.GetSpecDefinition(
		ctx,
		resourceType,
		&provider.ResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get the spec definition for resource type %q: %w",
			resourceType,
			err,
		)
	}

	var schema *provider.ResourceDefinitionsSchema
	if output != nil && output.SpecDefinition != nil {
		schema = output.SpecDefinition.Schema
	}

	i.mu.Lock()
	i.resourceSchemas[resourceType] = schema
	i.mu.Unlock()

	return schema, nil
}

func (i *Index) dataSourceFields(
	ctx context.Context,
	dataSourceType string,
) (map[string]*provider.DataSourceSpecSchema, error) {
	i.mu.RLock()
	cached, found := i.dataSourceSchemas[dataSourceType]
	i.mu.RUnlock()
	if found {
		return cached, nil
	}

	output, err := i.dataSources.GetSpecDefinition(
		ctx,
		dataSourceType,
		&provider.DataSourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get the spec definition for data source type %q: %w",
			dataSourceType,
			err,
		)
	}

	fields := map[string]*provider.DataSourceSpecSchema{}
	if output != nil && output.SpecDefinition != nil && output.SpecDefinition.Fields != nil {
		fields = output.SpecDefinition.Fields
	}

	i.mu.Lock()
	i.dataSourceSchemas[dataSourceType] = fields
	i.mu.Unlock()

	return fields, nil
}

// Walks a spec schema along a field path, carrying down what the ancestors declared.
//
// A path that leaves the schema keeps the sensitivity it inherited rather than losing it,
// because the paths handed to this walk come from state and field changes rather than from
// the schema, and the two drift apart: state outlives the schema version that wrote it, and
// a provider's computed state often carries keys it never declared. Treating a path that
// does not resolve as "not sensitive" would unmark exactly the values inside a container
// someone took the trouble to mark.
//
// Paths rooted outside the spec, metadata field changes among them, resolve to nothing and
// so come back unmarked. That is the right answer for a different reason: spec schemas
// cannot declare metadata sensitive, so a secret reaching metadata is not the responsibility
// of the index to mark.
func resourceFieldSensitive(
	schema *provider.ResourceDefinitionsSchema,
	fieldPath string,
) bool {
	if schema == nil {
		return false
	}

	segments := fieldPathSegments(fieldPath)

	return walkSensitive(schema, segments, false)
}

func walkSensitive(
	schema *provider.ResourceDefinitionsSchema,
	segments []string,
	inherited bool,
) bool {
	if schema == nil {
		return inherited
	}

	sensitive := inherited || schema.Sensitive
	if len(segments) == 0 {
		return sensitive
	}

	// A union is sensitive at this position if any variant that can hold the rest of the
	// path says so, which mirrors how change staging treats a union of which one variant is
	// secret.
	if len(schema.OneOf) > 0 {
		for _, variant := range schema.OneOf {
			if walkSensitive(variant, segments, sensitive) {
				return true
			}
		}

		return sensitive
	}

	name, selectorCount := drift.SplitSegmentSelectors(segments[0])

	current := schema
	if name != "" {
		next := namedChild(current, name)
		if next == nil {
			return sensitive
		}

		current = next
		sensitive = sensitive || current.Sensitive
	}

	for range selectorCount {
		next := selectedChild(current)
		if next == nil {
			return sensitive
		}

		current = next
		sensitive = sensitive || current.Sensitive
	}

	return walkSensitive(current, segments[1:], sensitive)
}

// A named segment addresses an attribute of an object, or any value of a map, since a map's
// keys are not known to the schema.
func namedChild(
	schema *provider.ResourceDefinitionsSchema,
	name string,
) *provider.ResourceDefinitionsSchema {
	if attribute, found := schema.Attributes[name]; found {
		return attribute
	}

	return schema.MapValues
}

// A selector addresses an item of an array or, where a key is given in brackets, a value of
// a map.
func selectedChild(
	schema *provider.ResourceDefinitionsSchema,
) *provider.ResourceDefinitionsSchema {
	if schema.Items != nil {
		return schema.Items
	}

	return schema.MapValues
}

// Splits a field path into the segments to walk, dropping the optional "spec" root.
//
// The root is dropped by name rather than by trimming the string, so a field called
// something like "specVersion" is not mistaken for the root with a suffix.
func fieldPathSegments(fieldPath string) []string {
	segments := []string{}
	for _, segment := range drift.SplitFieldPathSegments(fieldPath) {
		if segment != "" {
			segments = append(segments, segment)
		}
	}

	if len(segments) == 0 {
		return segments
	}

	name, _ := drift.SplitSegmentSelectors(segments[0])
	if name != "spec" {
		return segments
	}

	// Any selectors on the root segment still address something, as in spec["key"], so only
	// the name itself is removed.
	withoutRoot := strings.TrimPrefix(segments[0], name)
	if withoutRoot == "" {
		return segments[1:]
	}

	return append([]string{withoutRoot}, segments[1:]...)
}
