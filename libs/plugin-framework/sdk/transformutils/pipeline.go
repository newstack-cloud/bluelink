package transformutils

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subwalk"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// RunTransformPipeline drives the framework's transformer pipeline
// for plugins that don't provide their own TransformFunc implementation.
func RunTransformPipeline(
	inputBlueprint *schema.Blueprint,
	linkGraph linktypes.DeclaredLinkGraph,
	target Target,
	transformerID string,
	registry *TransformerRegistry,
	transformCtx transform.Context,
) (*transform.SpecTransformerTransformOutput, error) {
	if inputBlueprint == nil {
		return nil, fmt.Errorf("input blueprint is required")
	}

	if registry == nil {
		return nil, fmt.Errorf("transformer registry is required")
	}

	aggregator, ok := registry.AggregatorFor(target)
	if !ok {
		return nil, fmt.Errorf(
			"transformer does not support deploy target %q",
			string(target),
		)
	}

	resolved, err := resolveResources(inputBlueprint, linkGraph, registry)
	if err != nil {
		return nil, err
	}

	plan := aggregator(resolved)
	if plan == nil {
		plan = &EmitPlan{}
	}

	resPropRewriter, err := buildChainedRewriter(plan.Primaries, target, registry)
	if err != nil {
		return nil, err
	}

	diagnostics := validateBlueprintReferences(
		inputBlueprint,
		resolved,
		target,
		registry,
	)

	emitted, emitDiagnostics, err := emitPrimaries(plan.Primaries, resPropRewriter, target, registry, transformCtx)
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, emitDiagnostics...)

	parentDiagnostics := mergeSharedParents(emitted, plan.SharedParents)
	diagnostics = append(diagnostics, parentDiagnostics...)

	rewritten := RewriteBlueprintRefs(inputBlueprint, RewriteResourcePropertyRefs(resPropRewriter))

	finalValues, err := mergeValues(rewritten.Values, emitted.derivedValues)
	if err != nil {
		return nil, err
	}

	prunedTransform := stripCurrentTransformerID(rewritten.Transform, transformerID)

	output := assembleBlueprint(rewritten, prunedTransform, finalValues, emitted.resources)
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: output,
		Diagnostics:          diagnostics,
	}, nil
}

// Accumulates per-primary emit results for the driver.
// The shared-parent merge step writes parent resources into
// resources after consuming sharedParentContributions.
type emittedAggregate struct {
	resources                 map[string]*schema.Resource
	derivedValues             map[string]*schema.Value
	sharedParentContributions map[string][]*core.MappingNode
}

func newEmittedAggregate() *emittedAggregate {
	return &emittedAggregate{
		resources:                 map[string]*schema.Resource{},
		derivedValues:             map[string]*schema.Value{},
		sharedParentContributions: map[string][]*core.MappingNode{},
	}
}

func resolveResources(
	blueprint *schema.Blueprint,
	linkGraph linktypes.DeclaredLinkGraph,
	registry *TransformerRegistry,
) ([]ResolvedResource, error) {
	if blueprint.Resources == nil {
		return nil, nil
	}

	resolved := make([]ResolvedResource, 0, len(blueprint.Resources.Values))
	for name, resource := range blueprint.Resources.Values {
		resolvedResource, err := resolveOneResource(name, resource, linkGraph, blueprint, registry)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, resolvedResource)
	}

	return resolved, nil
}

func resolveOneResource(
	name string,
	resource *schema.Resource,
	linkGraph linktypes.DeclaredLinkGraph,
	blueprint *schema.Blueprint,
	registry *TransformerRegistry,
) (ResolvedResource, error) {
	resourceType := resourceTypeOf(resource)
	resolver, ok := registry.ResolverFor(resourceType)
	if !ok {
		return nil, fmt.Errorf(
			"no resolver registered for abstract resource type %q (resource %q)",
			resourceType,
			name,
		)
	}

	resolvedResource, err := resolver(name, resource, linkGraph, blueprint)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve resource %q: %w", name, err)
	}

	return resolvedResource, nil
}

func resourceTypeOf(resource *schema.Resource) string {
	if resource == nil || resource.Type == nil {
		return ""
	}

	return resource.Type.Value
}

func buildChainedRewriter(
	primaries []ResolvedResource,
	target Target,
	registry *TransformerRegistry,
) (ResourcePropertyRewriter, error) {
	rewriters := make([]ResourcePropertyRewriter, 0, len(primaries))
	for _, primary := range primaries {
		factory, ok := registry.RewriteFactoryFor(target, reflect.TypeOf(primary))
		if !ok {
			return nil, fmt.Errorf(
				"no rewriter factory registered for resolved type %T on target %q",
				primary,
				string(target),
			)
		}
		rewriters = append(rewriters, factory(primary)...)
	}

	return ChainResourcePropertyRewriters(rewriters...), nil
}

func emitPrimaries(
	primaries []ResolvedResource,
	chained ResourcePropertyRewriter,
	target Target,
	registry *TransformerRegistry,
	transformCtx transform.Context,
) (*emittedAggregate, []*core.Diagnostic, error) {
	aggregate := newEmittedAggregate()
	diagnostics := []*core.Diagnostic{}

	for _, primary := range primaries {
		emitter, ok := registry.EmitterFor(target, reflect.TypeOf(primary))
		if !ok {
			return nil, nil, fmt.Errorf(
				"no emitter registered for resolved type %T on target %q",
				primary,
				string(target),
			)
		}

		result, err := emitter(primary, chained, transformCtx)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"emit failed for resource %q: %w",
				primary.ResourceName(),
				err,
			)
		}

		if err := accumulateEmitResult(aggregate, result, primary.ResourceName()); err != nil {
			return nil, nil, err
		}
		if result != nil {
			diagnostics = append(diagnostics, result.Diagnostics...)
		}
	}

	return aggregate, diagnostics, nil
}

func accumulateEmitResult(
	aggregate *emittedAggregate,
	result *EmitResult,
	primaryName string,
) error {
	if result == nil {
		return nil
	}

	for name, resource := range result.Resources {
		if _, exists := aggregate.resources[name]; exists {
			return fmt.Errorf(
				"emit collision: resource %q produced by multiple primaries (last from %q)",
				name,
				primaryName,
			)
		}
		aggregate.resources[name] = resource
	}

	for name, value := range result.DerivedValues {
		if _, exists := aggregate.derivedValues[name]; exists {
			return fmt.Errorf(
				"emit collision: derived value %q produced by multiple primaries (last from %q)",
				name,
				primaryName,
			)
		}
		aggregate.derivedValues[name] = value
	}

	for key, contribution := range result.SharedParentContributions {
		aggregate.sharedParentContributions[key] = append(
			aggregate.sharedParentContributions[key],
			contribution,
		)
	}

	return nil
}

func mergeSharedParents(
	emitted *emittedAggregate,
	sharedParents []SharedParent,
) []*core.Diagnostic {
	diagnostics := []*core.Diagnostic{}

	for _, parent := range sharedParents {
		contributions := emitted.sharedParentContributions[parent.Key]
		merged, conflict := mergeMappingNodesDeep(
			append(
				[]*core.MappingNode{parent.SeedSpec},
				contributions...,
			)...,
		)
		if conflict != "" {
			diagnostics = append(diagnostics, &core.Diagnostic{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"shared parent %q has conflicting contributions at %q; aborting merge",
					parent.Key,
					conflict,
				),
			})
			continue
		}

		emitted.resources[parent.ResourceName] = sharedParentResource(parent, merged)
	}

	return diagnostics
}

func sharedParentResource(parent SharedParent, spec *core.MappingNode) *schema.Resource {
	resource := &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: parent.ResourceType},
		Spec: spec,
	}
	if parent.Annotations != nil {
		resource.Metadata = &schema.Metadata{Custom: parent.Annotations}
	}
	return resource
}

// Combines mapping nodes recursively. Identical
// scalar/string-substitution leaves merge cleanly; conflicting leaves or
// shape mismatches return the dotted path of the first conflict.
//
// Items (arrays) are treated as opaque values for conflict purposes: equal
// length and recursively-equal contents merge to the first node, otherwise
// it's a conflict. Conflicts in shared-parent contributions are rare by
// design (siblings should agree on shared fields like runtime stack), so
// the simple equality check is sufficient.
func mergeMappingNodesDeep(nodes ...*core.MappingNode) (*core.MappingNode, string) {
	var merged *core.MappingNode
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if merged == nil {
			merged = node
			continue
		}
		next, conflict := mergeTwoMappingNodes(merged, node, "")
		if conflict != "" {
			return nil, conflict
		}
		merged = next
	}

	return merged, ""
}

func mergeTwoMappingNodes(
	left, right *core.MappingNode,
	path string,
) (*core.MappingNode, string) {
	if left == nil {
		return right, ""
	}
	if right == nil {
		return left, ""
	}

	leftIsObject := left.Fields != nil
	rightIsObject := right.Fields != nil
	if leftIsObject && rightIsObject {
		return mergeObjectFields(left, right, path)
	}
	if leftIsObject || rightIsObject {
		return nil, conflictPath(path, "<shape>")
	}

	if mappingNodesEqual(left, right) {
		return left, ""
	}
	return nil, path
}

func mergeObjectFields(
	left, right *core.MappingNode,
	path string,
) (*core.MappingNode, string) {
	mergedFields := make(map[string]*core.MappingNode, len(left.Fields)+len(right.Fields))
	maps.Copy(mergedFields, left.Fields)
	for key, rightValue := range right.Fields {
		leftValue, exists := mergedFields[key]
		if !exists {
			mergedFields[key] = rightValue
			continue
		}
		merged, conflict := mergeTwoMappingNodes(leftValue, rightValue, conflictPath(path, key))
		if conflict != "" {
			return nil, conflict
		}
		mergedFields[key] = merged
	}

	return &core.MappingNode{Fields: mergedFields}, ""
}

func conflictPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}

func mappingNodesEqual(left, right *core.MappingNode) bool {
	if left == nil || right == nil {
		return left == right
	}

	if left.Scalar != nil || right.Scalar != nil {
		return left.Scalar.Equal(right.Scalar)
	}
	if left.Items != nil || right.Items != nil {
		return mappingItemsEqual(left.Items, right.Items)
	}
	if left.StringWithSubstitutions != nil || right.StringWithSubstitutions != nil {
		return stringSubsEqual(left.StringWithSubstitutions, right.StringWithSubstitutions)
	}

	return left.Fields == nil && right.Fields == nil
}

func mappingItemsEqual(left, right []*core.MappingNode) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !mappingNodesEqual(left[i], right[i]) {
			return false
		}
	}
	return true
}

func stringSubsEqual(left, right *substitutions.StringOrSubstitutions) bool {
	leftStr, leftErr := substitutions.SubstitutionsToString("", left)
	rightStr, rightErr := substitutions.SubstitutionsToString("", right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return leftStr == rightStr
}

func mergeValues(
	rewrittenValues *schema.ValueMap,
	derivedValues map[string]*schema.Value,
) (*schema.ValueMap, error) {
	final := &schema.ValueMap{Values: map[string]*schema.Value{}}
	if rewrittenValues != nil {
		maps.Copy(final.Values, rewrittenValues.Values)
		final.SourceMeta = rewrittenValues.SourceMeta
	}

	for name, value := range derivedValues {
		if _, exists := final.Values[name]; exists {
			return nil, fmt.Errorf(
				"derived value %q collides with a user-defined value of the same name",
				name,
			)
		}
		final.Values[name] = value
	}

	return final, nil
}

func stripCurrentTransformerID(
	transforms *schema.TransformValueWrapper,
	transformerID string,
) *schema.TransformValueWrapper {
	if transforms == nil || len(transforms.Values) == 0 || transformerID == "" {
		return transforms
	}

	if !slices.Contains(transforms.Values, transformerID) {
		return transforms
	}

	filtered := slices.DeleteFunc(slices.Clone(transforms.Values), func(id string) bool {
		return id == transformerID
	})

	return &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values:     filtered,
			SourceMeta: transforms.SourceMeta,
		},
	}
}

func assembleBlueprint(
	rewritten *schema.Blueprint,
	prunedTransform *schema.TransformValueWrapper,
	finalValues *schema.ValueMap,
	emittedResources map[string]*schema.Resource,
) *schema.Blueprint {
	output := *rewritten
	output.Transform = prunedTransform
	output.Values = finalValues
	output.Resources = &schema.ResourceMap{Values: emittedResources}
	if rewritten.Resources != nil {
		output.Resources.SourceMeta = rewritten.Resources.SourceMeta
	}
	return &output
}

// Walks every substitution in the input
// blueprint and checks each SubstitutionResourceProperty against the
// capability matrix entry for its named resource's resolved type. Unknown
// abstract paths produce warning diagnostics; errors do not short-circuit
// the pipeline.
func validateBlueprintReferences(
	blueprint *schema.Blueprint,
	resolved []ResolvedResource,
	target Target,
	registry *TransformerRegistry,
) []*core.Diagnostic {
	resolvedByName := indexResolvedByName(resolved)
	diagnostics := []*core.Diagnostic{}
	visitor := referenceValidatorVisitor(resolvedByName, target, registry, &diagnostics)

	walkResourceSpecs(blueprint, visitor)
	_ = RewriteBlueprintRefs(blueprint, visitor)

	return diagnostics
}

func walkResourceSpecs(blueprint *schema.Blueprint, visitor subwalk.SubstitutionVisitor) {
	if blueprint.Resources == nil {
		return
	}
	for _, resource := range blueprint.Resources.Values {
		if resource == nil {
			continue
		}
		_ = subwalk.WalkMappingNode(resource.Spec, visitor)
	}
}

func indexResolvedByName(resolved []ResolvedResource) map[string]ResolvedResource {
	index := make(map[string]ResolvedResource, len(resolved))
	for _, r := range resolved {
		index[r.ResourceName()] = r
	}
	return index
}

func referenceValidatorVisitor(
	resolvedByName map[string]ResolvedResource,
	target Target,
	registry *TransformerRegistry,
	diagnostics *[]*core.Diagnostic,
) subwalk.SubstitutionVisitor {
	seen := map[string]bool{}
	return func(sub *substitutions.Substitution) *substitutions.Substitution {
		if sub == nil || sub.ResourceProperty == nil {
			return nil
		}
		ref := sub.ResourceProperty
		key := referenceKey(ref)
		if seen[key] {
			return nil
		}
		seen[key] = true

		diag := validateReference(ref, resolvedByName, target, registry)
		if diag != nil {
			*diagnostics = append(*diagnostics, diag)
		}
		return nil
	}
}

func validateReference(
	ref *substitutions.SubstitutionResourceProperty,
	resolvedByName map[string]ResolvedResource,
	target Target,
	registry *TransformerRegistry,
) *core.Diagnostic {
	resolved, ok := resolvedByName[ref.ResourceName]
	if !ok {
		return nil
	}

	caps, ok := registry.CapabilitiesFor(target, reflect.TypeOf(resolved))
	if !ok {
		return nil
	}

	if referencePathSupported(ref, caps.SupportedAbstractPaths) {
		return nil
	}

	pathStr, err := substitutions.SubResourcePropertyToString(ref)
	if err != nil {
		pathStr = ref.ResourceName
	}
	return &core.Diagnostic{
		Level: core.DiagnosticLevelWarning,
		Message: fmt.Sprintf(
			"reference %q targets abstract path not declared in transformer capabilities for target %q",
			pathStr,
			string(target),
		),
	}
}

func referenceKey(ref *substitutions.SubstitutionResourceProperty) string {
	key, err := substitutions.SubResourcePropertyToString(ref)
	if err != nil {
		return ref.ResourceName
	}
	return key
}

func referencePathSupported(
	ref *substitutions.SubstitutionResourceProperty,
	supportedPaths []string,
) bool {
	for _, supported := range supportedPaths {
		if matchSupportedPath(ref, supported) {
			return true
		}
	}
	return false
}

func matchSupportedPath(
	ref *substitutions.SubstitutionResourceProperty,
	supported string,
) bool {
	if strings.Contains(supported, "[*]") {
		return matchPathPattern(ref, parsePathPattern(supported))
	}
	if supported == "" {
		return PathExact(ref)
	}
	return PathExact(ref, strings.Split(supported, ".")...)
}
