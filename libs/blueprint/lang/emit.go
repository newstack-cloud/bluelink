package lang

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

const indentUnit = "    "

// Emit serialises a blueprint model into blueprint language (.bp) source text.
// Every top-level construct (version, transform,
// variables, values, includes, resources, datasources and exports) is supported.
func Emit(blueprint *schema.Blueprint) (string, error) {
	if blueprint == nil {
		return "", fmt.Errorf("cannot emit a nil blueprint")
	}

	var b strings.Builder
	if blueprint.Version != nil {
		fmt.Fprintf(&b, "version %s\n", quote(blueprint.Version.ToString()))
	}

	emitters := []func(*strings.Builder, *schema.Blueprint) error{
		emitTransform,
		emitBlueprintMetadata,
		emitVariables,
		emitValues,
		emitIncludes,
		func(b *strings.Builder, bp *schema.Blueprint) error {
			return emitResources(b, bp.Resources)
		},
		emitDataSources,
		func(b *strings.Builder, bp *schema.Blueprint) error {
			return emitExports(b, bp.Exports)
		},
	}
	for _, emit := range emitters {
		if err := emit(&b, blueprint); err != nil {
			return "", err
		}
	}

	return b.String(), nil
}

func emitResources(b *strings.Builder, resources *schema.ResourceMap) error {
	if resources == nil {
		return nil
	}
	for _, name := range sortedKeys(resources.Values) {
		resource := resources.Values[name]
		fmt.Fprintf(b, "\nresource %s: %s {\n", nameToken(name), resourceType(resource))
		if err := emitResourceBody(b, resource); err != nil {
			return err
		}
		b.WriteString("}\n")
	}
	return nil
}

func emitResourceBody(b *strings.Builder, resource *schema.Resource) error {
	if resource.Description != nil {
		desc, err := renderStringOrSubstitutions(resource.Description, 1)
		if err != nil {
			return err
		}

		fmt.Fprintf(b, "%sdescription = %s\n", indentUnit, desc)
	}

	if err := emitResourceCondition(b, resource.Condition); err != nil {
		return err
	}

	emitDependsOn(b, resource.DependsOn)
	if err := emitForEach(b, resource.Each); err != nil {
		return err
	}

	emitLinkSelector(b, resource.LinkSelector)
	if resource.RemovalPolicy != nil {
		fmt.Fprintf(b, "%sremovalPolicy = %s\n", indentUnit, quote(string(resource.RemovalPolicy.Value)))
	}

	if err := emitMetadata(b, resource.Metadata); err != nil {
		return err
	}

	return emitSpec(b, resource.Spec)
}

func emitMetadata(b *strings.Builder, metadata *schema.Metadata) error {
	if metadata == nil {
		return nil
	}

	hasDisplayName := metadata.DisplayName != nil
	hasLabels := metadata.Labels != nil && len(metadata.Labels.Values) > 0
	hasAnnotations := metadata.Annotations != nil && len(metadata.Annotations.Values) > 0
	hasCustom := metadata.Custom != nil
	if !hasDisplayName && !hasLabels && !hasAnnotations && !hasCustom {
		return nil
	}

	b.WriteString(indentUnit + "metadata {\n")
	if hasDisplayName {
		value, err := renderStringOrSubstitutions(metadata.DisplayName, 2)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%sdisplayName = %s\n", indent(2), value)
	}

	if hasLabels {
		b.WriteString(indent(2) + "labels = {\n")
		for _, key := range sortedKeys(metadata.Labels.Values) {
			fmt.Fprintf(b, "%s%s = %s\n", indent(3), keyToken(key), quote(metadata.Labels.Values[key]))
		}
		b.WriteString(indent(2) + "}\n")
	}

	if hasAnnotations {
		if err := emitAnnotations(b, metadata.Annotations); err != nil {
			return err
		}
	}

	if hasCustom {
		custom, err := renderValue(metadata.Custom, 2)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%scustom = %s\n", indent(2), custom)
	}

	b.WriteString(indentUnit + "}\n")

	return nil
}

func emitSpec(b *strings.Builder, spec *core.MappingNode) error {
	b.WriteString(indentUnit + "spec {\n")
	if spec != nil {
		if err := emitObjectFields(b, spec.Fields, 2); err != nil {
			return err
		}
	}

	b.WriteString(indentUnit + "}\n")

	return nil
}

func emitObjectFields(b *strings.Builder, fields map[string]*core.MappingNode, depth int) error {
	for _, key := range sortedKeys(fields) {
		value, err := renderValue(fields[key], depth)
		if err != nil {
			return err
		}

		fmt.Fprintf(b, "%s%s = %s\n", indent(depth), keyToken(key), value)
	}

	return nil
}

func renderValue(node *core.MappingNode, depth int) (string, error) {
	switch {
	case node == nil:
		return "none", nil
	case node.Scalar != nil:
		return renderScalar(node.Scalar), nil
	case node.StringWithSubstitutions != nil:
		return renderStringOrSubstitutions(node.StringWithSubstitutions, depth)
	case node.Fields != nil:
		return renderObject(node.Fields, depth)
	case node.Items != nil:
		return renderArray(node.Items, depth)
	default:
		return "none", nil
	}
}

// Renders a string-or-substitutions value. A lone
// substitution is emitted as a bare expression (e.g. variables.x, fn(...)); any
// value containing literal text is emitted as a quoted, interpolated string.
func renderStringOrSubstitutions(sos *substitutions.StringOrSubstitutions, depth int) (string, error) {
	if len(sos.Values) == 1 && sos.Values[0].SubstitutionValue != nil {
		return renderSubstitution(sos.Values[0].SubstitutionValue)
	}
	value, err := substitutions.SubstitutionsToString("", sos)
	if err != nil {
		return "", err
	}
	return maybeMultilineString(value, depth), nil
}

// Emits a string containing newlines as a triple-quoted
// block (whose closing """ indentation sets the strip width), otherwise as a
// regular quoted string.
func maybeMultilineString(s string, depth int) string {
	if !strings.Contains(s, "\n") {
		return quote(s)
	}

	pad := indent(depth + 1)
	var b strings.Builder
	b.WriteString("\"\"\"\n")
	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(pad + line + "\n")
	}

	b.WriteString(pad + "\"\"\"")

	return b.String()
}

// Renders a lone substitution as a bare expression. Literal
// substitutions are rendered directly because the SubstitutionToString only
// handles references and function calls.
func renderSubstitution(sub *substitutions.Substitution) (string, error) {
	switch {
	case sub.StringValue != nil:
		return quote(*sub.StringValue), nil
	case sub.IntValue != nil:
		return strconv.FormatInt(*sub.IntValue, 10), nil
	case sub.FloatValue != nil:
		return formatFloat(*sub.FloatValue), nil
	case sub.BoolValue != nil:
		return strconv.FormatBool(*sub.BoolValue), nil
	case sub.NoneValue:
		return "none", nil
	default:
		return substitutions.SubstitutionToString("", sub)
	}
}

func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func renderObject(fields map[string]*core.MappingNode, depth int) (string, error) {
	if len(fields) == 0 {
		return "{}", nil
	}

	var b strings.Builder
	b.WriteString("{\n")
	keys := sortedKeys(fields)
	for i, key := range keys {
		value, err := renderValue(fields[key], depth+1)
		if err != nil {
			return "", err
		}
		comma := ","
		if i == len(keys)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "%s%s = %s%s\n", indent(depth+1), keyToken(key), value, comma)
	}

	b.WriteString(indent(depth) + "}")

	return b.String(), nil
}

func renderArray(items []*core.MappingNode, depth int) (string, error) {
	if len(items) == 0 {
		return "[]", nil
	}

	var b strings.Builder
	b.WriteString("[\n")
	for i, item := range items {
		value, err := renderValue(item, depth+1)
		if err != nil {
			return "", err
		}
		comma := ","
		if i == len(items)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "%s%s%s\n", indent(depth+1), value, comma)
	}

	b.WriteString(indent(depth) + "]")

	return b.String(), nil
}

func renderScalar(scalar *core.ScalarValue) string {
	switch {
	case scalar.StringValue != nil:
		return quote(*scalar.StringValue)
	case scalar.IntValue != nil:
		return strconv.Itoa(*scalar.IntValue)
	case scalar.BoolValue != nil:
		return strconv.FormatBool(*scalar.BoolValue)
	case scalar.FloatValue != nil:
		return strconv.FormatFloat(*scalar.FloatValue, 'f', -1, 64)
	case scalar.NoneValue != nil && *scalar.NoneValue:
		return "none"
	default:
		return "none"
	}
}

func resourceType(resource *schema.Resource) string {
	if resource.Type == nil {
		return ""
	}
	return resource.Type.Value
}

func quote(value string) string {
	return strconv.Quote(value)
}

func keyToken(name string) string {
	if isBareIdent(name) {
		return name
	}

	return quote(name)
}

// Renders a declaration name (resource, variable, …). Unlike object keys and type
// identifiers, declaration names cannot be bare reserved keywords, so those are quoted.
func nameToken(name string) string {
	if !isBareIdent(name) || reservedKeywords[name] {
		return quote(name)
	}
	return name
}

// The blueprintlang keywords that cannot be used as bare
// declaration names (mirrors the lexer's keyword table).
var reservedKeywords = map[string]bool{
	"variable": true, "value": true, "data": true, "resource": true,
	"include": true, "export": true, "metadata": true, "spec": true,
	"select": true, "filter": true, "foreach": true, "as": true, "by": true,
	"label": true, "version": true, "transform": true, "not": true, "in": true,
	"has": true, "key": true, "contains": true, "starts": true, "with": true,
	"ends": true, "string": true, "integer": true, "float": true,
	"boolean": true, "array": true, "object": true, "variables": true,
	"values": true, "datasources": true, "resources": true, "children": true,
	"elem": true, "i": true,
}

func isBareIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_',
			r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func indent(depth int) string {
	return strings.Repeat(indentUnit, depth)
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
