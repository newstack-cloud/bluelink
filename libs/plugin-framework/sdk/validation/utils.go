package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

func invalidTypeDiagnostics(
	fieldName string,
	value *core.ScalarValue,
	typeLabel string,
) []*core.Diagnostic {
	return []*core.Diagnostic{
		{
			Level: core.DiagnosticLevelError,
			Message: fmt.Sprintf(
				"%s must be %s, but got %s.",
				fieldName,
				typeLabel,
				core.TypeFromScalarValue(value),
			),
			Range: toDiagnosticRange(value.SourceMeta, nil),
		},
	}
}

func toDiagnosticRange(
	start *source.Meta,
	nextLocation *source.Meta,
) *core.DiagnosticRange {
	if start == nil {
		return &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{
				Line:   1,
				Column: 1,
			}},
			End: &source.Meta{Position: source.Position{
				Line:   1,
				Column: 1,
			}},
		}
	}

	endSourceMeta := determineEndSourceMeta(
		start,
		nextLocation,
	)

	return &core.DiagnosticRange{
		Start: start,
		End:   endSourceMeta,
	}
}

func determineEndSourceMeta(
	start *source.Meta,
	nextLocation *source.Meta,
) *source.Meta {
	if start.EndPosition != nil {
		return &source.Meta{
			Position: *start.EndPosition,
		}
	}

	endSourceMeta := &source.Meta{Position: source.Position{
		Line:   start.Line + 1,
		Column: 1,
	}}
	if nextLocation != nil {
		endSourceMeta = &source.Meta{Position: source.Position{
			Line:   nextLocation.Line,
			Column: nextLocation.Column,
		}}
	}

	return endSourceMeta
}

func typeFromMappingNode(
	value *core.MappingNode,
) string {
	if value == nil {
		return "null"
	}

	if core.IsScalarMappingNode(value) {
		return string(core.TypeFromScalarValue(value.Scalar))
	}

	if core.IsArrayMappingNode(value) {
		return "array"
	}

	if core.IsObjectMappingNode(value) {
		return "object"
	}

	if value.StringWithSubstitutions != nil {
		// A string with substitutions is a special case
		// that can be a string literal, a string interpolation or a
		// substitution that resolves to another type.
		return "any"
	}

	return "unknown"
}
