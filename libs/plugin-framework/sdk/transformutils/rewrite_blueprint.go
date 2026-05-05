package transformutils

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subwalk"
)

// RewriteBlueprintRefs returns a shallow copy of blueprint with every
// substitution-bearing top-level section walked by visitor. Sections walked:
//
//   - Exports[*]      — every value (direct field reference, not in ${..}) and description StringOrSubstitutions
//   - Values[*]       — every value and description StringOrSubstitutions
//   - Include[*]      — path, variables, metadata, description
//   - DataSources[*]  — filter search values, metadata, description
//   - Metadata        — top-level free-form mapping
//
// Sections passed through unchanged (no substitution support as per spec):
//   - Variables, Version
//
// Resources and Transform are also passed through here; the transformer
// owns those separately:
//   - Resources: spec-level rewrites happen inline during per-resource emit,
//     where each emitter has access to resource-specific structural
//     transformations (memory -> memorySize, etc.). The driver replaces
//     this section wholesale with emitted output.
//   - Transform: the transformer strips its own identifier from this list.
//
// Returns a shallow copy — the input blueprint is not mutated, but
// pointer-shared sub-trees that weren't rewritten remain shared.
func RewriteBlueprintRefs(
	blueprint *schema.Blueprint,
	visitor subwalk.SubstitutionVisitor,
) *schema.Blueprint {
	if blueprint == nil {
		return nil
	}

	rewritten := *blueprint
	rewritten.Exports = walkExports(blueprint.Exports, visitor)
	rewritten.Values = walkValues(blueprint.Values, visitor)
	rewritten.Include = walkIncludes(blueprint.Include, visitor)
	rewritten.DataSources = walkDataSources(blueprint.DataSources, visitor)
	rewritten.Metadata = subwalk.WalkMappingNode(blueprint.Metadata, visitor)

	return &rewritten
}

func walkExports(
	exports *schema.ExportMap,
	visitor subwalk.SubstitutionVisitor,
) *schema.ExportMap {
	if exports == nil {
		return nil
	}

	rewritten := &schema.ExportMap{
		Values:     map[string]*schema.Export{},
		SourceMeta: exports.SourceMeta,
	}
	for k, v := range exports.Values {
		rewritten.Values[k] = walkExport(v, visitor)
	}

	return rewritten
}

func walkExport(
	export *schema.Export,
	visitor subwalk.SubstitutionVisitor,
) *schema.Export {
	if export == nil {
		return nil
	}

	rewritten := &schema.Export{
		Type:       export.Type,
		SourceMeta: export.SourceMeta,
	}
	rewritten.Field = walkBareReference(export.Field, visitor)
	rewritten.Description = subwalk.WalkStringOrSubstitutions(
		export.Description,
		visitor,
	)

	return rewritten
}

func walkValues(
	values *schema.ValueMap,
	visitor subwalk.SubstitutionVisitor,
) *schema.ValueMap {
	if values == nil {
		return nil
	}

	rewritten := &schema.ValueMap{
		Values:     map[string]*schema.Value{},
		SourceMeta: values.SourceMeta,
	}
	for k, v := range values.Values {
		rewritten.Values[k] = walkValue(v, visitor)
	}

	return rewritten
}

func walkValue(
	value *schema.Value,
	visitor subwalk.SubstitutionVisitor,
) *schema.Value {
	if value == nil {
		return nil
	}

	rewritten := &schema.Value{
		Type:       value.Type,
		Secret:     value.Secret,
		SourceMeta: value.SourceMeta,
	}
	rewritten.Value = subwalk.WalkMappingNode(
		value.Value,
		visitor,
	)
	rewritten.Description = subwalk.WalkStringOrSubstitutions(
		value.Description,
		visitor,
	)

	return rewritten
}

func walkIncludes(
	includes *schema.IncludeMap,
	visitor subwalk.SubstitutionVisitor,
) *schema.IncludeMap {
	if includes == nil {
		return nil
	}

	rewritten := &schema.IncludeMap{
		Values:     map[string]*schema.Include{},
		SourceMeta: includes.SourceMeta,
	}
	for name, include := range includes.Values {
		rewritten.Values[name] = walkInclude(include, visitor)
	}

	return rewritten
}

func walkInclude(
	include *schema.Include,
	visitor subwalk.SubstitutionVisitor,
) *schema.Include {
	if include == nil {
		return nil
	}

	rewritten := &schema.Include{
		SourceMeta: include.SourceMeta,
	}
	rewritten.Path = subwalk.WalkStringOrSubstitutions(include.Path, visitor)
	rewritten.Variables = subwalk.WalkMappingNode(include.Variables, visitor)
	rewritten.Metadata = subwalk.WalkMappingNode(include.Metadata, visitor)
	rewritten.Description = subwalk.WalkStringOrSubstitutions(
		include.Description,
		visitor,
	)

	return rewritten
}

func walkDataSources(
	dataSources *schema.DataSourceMap,
	visitor subwalk.SubstitutionVisitor,
) *schema.DataSourceMap {
	if dataSources == nil {
		return nil
	}

	rewritten := &schema.DataSourceMap{
		Values:     map[string]*schema.DataSource{},
		SourceMeta: dataSources.SourceMeta,
	}
	for name, ds := range dataSources.Values {
		rewritten.Values[name] = walkDataSource(ds, visitor)
	}

	return rewritten
}

func walkDataSource(
	ds *schema.DataSource,
	visitor subwalk.SubstitutionVisitor,
) *schema.DataSource {
	if ds == nil {
		return nil
	}

	rewritten := &schema.DataSource{
		Type:             ds.Type,
		SourceMeta:       ds.SourceMeta,
		FieldsSourceMeta: ds.FieldsSourceMeta,
	}
	rewritten.Filter = walkDataSourceFilters(ds.Filter, visitor)
	rewritten.Description = subwalk.WalkStringOrSubstitutions(
		ds.Description,
		visitor,
	)
	rewritten.DataSourceMetadata = walkDataSourceMetadata(ds.DataSourceMetadata, visitor)

	return rewritten
}

func walkDataSourceFilters(
	filters *schema.DataSourceFilters,
	visitor subwalk.SubstitutionVisitor,
) *schema.DataSourceFilters {
	if filters == nil {
		return nil
	}

	rewritten := &schema.DataSourceFilters{
		Filters: make([]*schema.DataSourceFilter, len(filters.Filters)),
	}
	for i, filter := range filters.Filters {
		rewritten.Filters[i] = walkDataSourceFilter(filter, visitor)
	}

	return rewritten
}

func walkDataSourceFilter(
	filter *schema.DataSourceFilter,
	visitor subwalk.SubstitutionVisitor,
) *schema.DataSourceFilter {
	if filter == nil {
		return nil
	}

	rewritten := &schema.DataSourceFilter{
		Field:      filter.Field,
		Operator:   filter.Operator,
		SourceMeta: filter.SourceMeta,
	}
	rewritten.Search = walkDataSourceFilterSearch(filter.Search, visitor)

	return rewritten
}

func walkDataSourceFilterSearch(
	search *schema.DataSourceFilterSearch,
	visitor subwalk.SubstitutionVisitor,
) *schema.DataSourceFilterSearch {
	if search == nil {
		return nil
	}

	rewritten := &schema.DataSourceFilterSearch{
		Values:     make([]*substitutions.StringOrSubstitutions, len(search.Values)),
		SourceMeta: search.SourceMeta,
	}

	for i, v := range search.Values {
		rewritten.Values[i] = subwalk.WalkStringOrSubstitutions(v, visitor)
	}

	return rewritten
}

func walkDataSourceMetadata(
	metadata *schema.DataSourceMetadata,
	visitor subwalk.SubstitutionVisitor,
) *schema.DataSourceMetadata {
	if metadata == nil {
		return nil
	}

	rewritten := &schema.DataSourceMetadata{
		SourceMeta:       metadata.SourceMeta,
		FieldsSourceMeta: metadata.FieldsSourceMeta,
	}
	rewritten.DisplayName = subwalk.WalkStringOrSubstitutions(metadata.DisplayName, visitor)
	rewritten.Annotations = walkStringSubMap(metadata.Annotations, visitor)
	rewritten.Custom = subwalk.WalkMappingNode(metadata.Custom, visitor)

	return rewritten
}

func walkStringSubMap(
	m *schema.StringOrSubstitutionsMap,
	visitor subwalk.SubstitutionVisitor,
) *schema.StringOrSubstitutionsMap {
	if m == nil {
		return nil
	}

	rewritten := &schema.StringOrSubstitutionsMap{
		Values:     map[string]*substitutions.StringOrSubstitutions{},
		SourceMeta: m.SourceMeta,
	}
	for k, v := range m.Values {
		rewritten.Values[k] = subwalk.WalkStringOrSubstitutions(v, visitor)
	}

	return rewritten
}

func walkBareReference(
	bareRef *core.ScalarValue,
	visitor subwalk.SubstitutionVisitor,
) *core.ScalarValue {
	if bareRef == nil || !core.IsScalarString(bareRef) {
		return nil
	}
	raw := *bareRef.StringValue

	sub, err := parseBareSubstitution(raw, bareRef.SourceMeta)
	if err != nil || sub == nil {
		// Not a valid substitution-compatible reference; return as-is.
		return bareRef
	}

	rewritten := visitor(sub)
	if rewritten == nil || rewritten == sub {
		return bareRef
	}

	serialised, err := serialiseBareSubstitution(rewritten)
	if err != nil {
		return bareRef
	}

	return core.ScalarFromString(serialised)
}

func parseBareSubstitution(raw string, meta *source.Meta) (*substitutions.Substitution, error) {
	substitutionStr := fmt.Sprintf("${%s}", raw)

	parsed, err := substitutions.ParseSubstitutionValues(
		"bare-reference",
		substitutionStr,
		meta,
		/* outputLineInfo */ false,
		/* ignoreParentColumn */ false,
		/* parentContextPrecedingCharCount */ 0,
	)
	if err != nil || len(parsed) != 1 || parsed[0].SubstitutionValue == nil {
		return nil, err
	}

	return parsed[0].SubstitutionValue, nil
}

func serialiseBareSubstitution(sub *substitutions.Substitution) (string, error) {
	s, err := substitutions.SubstitutionsToString(
		"",
		&substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: sub,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}"), nil
}
