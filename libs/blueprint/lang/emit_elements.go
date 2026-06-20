package lang

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func emitTransform(b *strings.Builder, bp *schema.Blueprint) error {
	if bp.Transform == nil || len(bp.Transform.Values) == 0 {
		return nil
	}

	values := bp.Transform.Values
	if len(values) == 1 {
		fmt.Fprintf(b, "\ntransform %s\n", quote(values[0]))
		return nil
	}

	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = quote(value)
	}
	fmt.Fprintf(b, "\ntransform [%s]\n", strings.Join(quoted, ", "))

	return nil
}

func emitVariables(b *strings.Builder, bp *schema.Blueprint) error {
	if bp.Variables == nil {
		return nil
	}

	for _, name := range sortedKeys(bp.Variables.Values) {
		variable := bp.Variables.Values[name]
		fmt.Fprintf(b, "\nvariable %s: %s {\n", nameToken(name), string(variable.Type.Value))
		if variable.Description != nil {
			fmt.Fprintf(b, "%sdescription = %s\n", indentUnit, maybeMultilineString(variable.Description.ToString(), 1))
		}
		if variable.Secret != nil {
			fmt.Fprintf(b, "%ssecret = %s\n", indentUnit, variable.Secret.ToString())
		}
		if variable.Default != nil {
			fmt.Fprintf(b, "%sdefault = %s\n", indentUnit, renderScalar(variable.Default))
		}
		if len(variable.AllowedValues) > 0 {
			fmt.Fprintf(b, "%sallowedValues = %s\n", indentUnit, renderScalarList(variable.AllowedValues))
		}
		b.WriteString("}\n")
	}

	return nil
}

func emitValues(b *strings.Builder, bp *schema.Blueprint) error {
	if bp.Values == nil {
		return nil
	}

	for _, name := range sortedKeys(bp.Values.Values) {
		value := bp.Values.Values[name]
		fmt.Fprintf(b, "\nvalue %s: %s {\n", nameToken(name), string(value.Type.Value))
		rendered, err := renderValue(value.Value, 1)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%svalue = %s\n", indentUnit, rendered)
		if value.Description != nil {
			desc, err := renderStringOrSubstitutions(value.Description, 1)
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "%sdescription = %s\n", indentUnit, desc)
		}
		if value.Secret != nil {
			fmt.Fprintf(b, "%ssecret = %s\n", indentUnit, value.Secret.ToString())
		}
		b.WriteString("}\n")
	}

	return nil
}

func emitIncludes(b *strings.Builder, bp *schema.Blueprint) error {
	if bp.Include == nil {
		return nil
	}

	for _, name := range sortedKeys(bp.Include.Values) {
		include := bp.Include.Values[name]
		path, err := renderStringOrSubstitutions(include.Path, 1)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "\ninclude %s %s {\n", nameToken(name), path)
		if include.Description != nil {
			desc, err := renderStringOrSubstitutions(include.Description, 1)
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "%sdescription = %s\n", indentUnit, desc)
		}
		if err := emitNamedBlock(b, "variables", include.Variables); err != nil {
			return err
		}
		if err := emitNamedBlock(b, "metadata", include.Metadata); err != nil {
			return err
		}
		b.WriteString("}\n")
	}

	return nil
}

func emitExports(b *strings.Builder, exports *schema.ExportMap) error {
	if exports == nil {
		return nil
	}

	for _, name := range sortedKeys(exports.Values) {
		export := exports.Values[name]
		fmt.Fprintf(b, "\nexport %s: %s {\n", nameToken(name), string(export.Type.Value))
		if export.Field != nil {
			fmt.Fprintf(b, "%sfield = %s\n", indentUnit, quote(export.Field.ToString()))
		}
		if export.Description != nil {
			desc, err := renderStringOrSubstitutions(export.Description, 1)
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "%sdescription = %s\n", indentUnit, desc)
		}
		b.WriteString("}\n")
	}

	return nil
}

func emitDataSources(b *strings.Builder, bp *schema.Blueprint) error {
	if bp.DataSources == nil {
		return nil
	}

	for _, name := range sortedKeys(bp.DataSources.Values) {
		dataSource := bp.DataSources.Values[name]
		fmt.Fprintf(b, "\ndata %s: %s {\n", nameToken(name), string(dataSource.Type.Value))
		if err := emitDataSourceMetadata(b, dataSource.DataSourceMetadata); err != nil {
			return err
		}
		if err := emitDataSourceFilters(b, dataSource.Filter); err != nil {
			return err
		}
		if err := emitDataSourceExports(b, dataSource.Exports); err != nil {
			return err
		}
		if dataSource.Description != nil {
			desc, err := renderStringOrSubstitutions(dataSource.Description, 1)
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "%sdescription = %s\n", indentUnit, desc)
		}
		b.WriteString("}\n")
	}

	return nil
}

func emitDataSourceFilters(b *strings.Builder, filters *schema.DataSourceFilters) error {
	if filters == nil {
		return nil
	}

	for _, filter := range filters.Filters {
		search, err := renderFilterSearch(filter.Search)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%sfilter %s %s %s\n",
			indentUnit, quote(filter.Field.ToString()), filterOperator(filter.Operator), search)
	}

	return nil
}

func renderFilterSearch(search *schema.DataSourceFilterSearch) (string, error) {
	if search == nil || len(search.Values) == 0 {
		return "", nil
	}

	if len(search.Values) == 1 {
		return renderStringOrSubstitutions(search.Values[0], 1)
	}

	parts := make([]string, len(search.Values))
	for i, value := range search.Values {
		rendered, err := renderStringOrSubstitutions(value, 1)
		if err != nil {
			return "", err
		}
		parts[i] = rendered
	}

	return "[" + strings.Join(parts, ", ") + "]", nil
}

func filterOperator(operator *schema.DataSourceFilterOperatorWrapper) string {
	if operator == nil {
		return "=="
	}

	if operator.Value == schema.DataSourceFilterOperatorEquals {
		// "=" in the blueprint model is represented as "==" in the blueprint language,
		// so that "=" can be used for assignment.
		return "=="
	}

	return string(operator.Value)
}

func emitDataSourceExports(b *strings.Builder, exports *schema.DataSourceFieldExportMap) error {
	if exports == nil {
		return nil
	}

	if exports.ExportAll {
		fmt.Fprintf(b, "%sexport *\n", indentUnit)
		return nil
	}

	for _, name := range sortedKeys(exports.Values) {
		export := exports.Values[name]
		header := nameToken(name)
		if export.AliasFor != nil && export.AliasFor.ToString() != "" {
			header = fmt.Sprintf("%s as %s", quote(export.AliasFor.ToString()), nameToken(name))
		}

		exportType := ""
		if export.Type != nil {
			exportType = ": " + string(export.Type.Value)
		}

		if export.Description != nil {
			desc, err := renderStringOrSubstitutions(export.Description, 2)
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "%sexport %s%s {\n", indentUnit, header, exportType)
			fmt.Fprintf(b, "%sdescription = %s\n", indent(2), desc)
			fmt.Fprintf(b, "%s}\n", indentUnit)
		} else {
			fmt.Fprintf(b, "%sexport %s%s\n", indentUnit, header, exportType)
		}
	}

	return nil
}

func emitResourceCondition(b *strings.Builder, condition *schema.Condition) error {
	if condition == nil {
		return nil
	}

	rendered, err := renderCondition(condition, 1)
	if err != nil {
		return err
	}

	fmt.Fprintf(b, "%scondition = %s\n", indentUnit, rendered)
	return nil
}

func renderCondition(condition *schema.Condition, depth int) (string, error) {
	switch {
	case condition.StringValue != nil:
		return renderStringOrSubstitutions(condition.StringValue, depth)
	case condition.Not != nil:
		inner, err := renderCondition(condition.Not, depth+1)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("{\n%snot = %s\n%s}", indent(depth+1), inner, indent(depth)), nil
	case len(condition.And) > 0:
		return renderConditionList("and", condition.And, depth)
	case len(condition.Or) > 0:
		return renderConditionList("or", condition.Or, depth)
	default:
		return "{}", nil
	}
}

func renderConditionList(keyword string, conditions []*schema.Condition, depth int) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "{\n%s%s = [\n", indent(depth+1), keyword)
	for i, condition := range conditions {
		inner, err := renderCondition(condition, depth+2)
		if err != nil {
			return "", err
		}
		comma := ","
		if i == len(conditions)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "%s%s%s\n", indent(depth+2), inner, comma)
	}

	fmt.Fprintf(&b, "%s]\n%s}", indent(depth+1), indent(depth))

	return b.String(), nil
}

func emitDependsOn(b *strings.Builder, dependsOn *schema.DependsOnList) {
	if dependsOn == nil || len(dependsOn.Values) == 0 {
		return
	}

	quoted := make([]string, len(dependsOn.Values))
	for i, value := range dependsOn.Values {
		quoted[i] = quote(value)
	}

	fmt.Fprintf(b, "%sdependsOn = [%s]\n", indentUnit, strings.Join(quoted, ", "))
}

func emitForEach(b *strings.Builder, each *substitutions.StringOrSubstitutions) error {
	if each == nil {
		return nil
	}

	rendered, err := renderStringOrSubstitutions(each, 1)
	if err != nil {
		return err
	}

	fmt.Fprintf(b, "%sforeach %s\n", indentUnit, rendered)

	return nil
}

func emitLinkSelector(b *strings.Builder, selector *schema.LinkSelector) {
	if selector == nil ||
		selector.ByLabel == nil ||
		len(selector.ByLabel.Values) == 0 {
		return
	}

	b.WriteString(indentUnit + "select by label {\n")
	for _, key := range sortedKeys(selector.ByLabel.Values) {
		fmt.Fprintf(b, "%s%s = %s\n", indent(2), keyToken(key), quote(selector.ByLabel.Values[key]))
	}

	if selector.Exclude != nil && len(selector.Exclude.Values) > 0 {
		quoted := make([]string, len(selector.Exclude.Values))
		for i, value := range selector.Exclude.Values {
			quoted[i] = quote(value)
		}
		fmt.Fprintf(b, "%sexclude = [%s]\n", indent(2), strings.Join(quoted, ", "))
	}

	b.WriteString(indentUnit + "}\n")
}

func emitBlueprintMetadata(b *strings.Builder, bp *schema.Blueprint) error {
	if bp.Metadata == nil || len(bp.Metadata.Fields) == 0 {
		return nil
	}

	b.WriteString("\nmetadata {\n")
	if err := emitObjectFields(b, bp.Metadata.Fields, 1); err != nil {
		return err
	}

	b.WriteString("}\n")

	return nil
}

func emitAnnotations(b *strings.Builder, annotations *schema.StringOrSubstitutionsMap) error {
	b.WriteString(indent(2) + "annotations = {\n")

	for _, key := range sortedKeys(annotations.Values) {
		value, err := renderStringOrSubstitutions(annotations.Values[key], 3)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%s%s = %s,\n", indent(3), keyToken(key), value)
	}

	b.WriteString(indent(2) + "}\n")

	return nil
}

func emitDataSourceMetadata(b *strings.Builder, metadata *schema.DataSourceMetadata) error {
	if metadata == nil {
		return nil
	}

	hasDisplayName := metadata.DisplayName != nil
	hasAnnotations := metadata.Annotations != nil && len(metadata.Annotations.Values) > 0
	hasCustom := metadata.Custom != nil
	if !hasDisplayName && !hasAnnotations && !hasCustom {
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

// Renders a free-form mapping node as a named block (e.g. an
// include's variables or metadata block).
func emitNamedBlock(b *strings.Builder, name string, node *core.MappingNode) error {
	if node == nil || len(node.Fields) == 0 {
		return nil
	}

	fmt.Fprintf(b, "%s%s {\n", indentUnit, name)
	if err := emitObjectFields(b, node.Fields, 2); err != nil {
		return err
	}

	b.WriteString(indentUnit + "}\n")

	return nil
}

func renderScalarList(values []*core.ScalarValue) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = renderScalar(value)
	}

	return "[" + strings.Join(parts, ", ") + "]"
}
