package validation

import (
	"context"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ValidateTransforms checks the transform strings for a blueprint when the spec is
// not going to be transformed (e.g. dry run validation), reporting diagnostics for
// transforms that are empty or contain ${..} substitutions.
// This is intended for validation of the transform string list only, checking that
// each transform is available and any validation that requires the transformer
// implementation is carried out at a different stage.
func ValidateTransforms(
	ctx context.Context,
	blueprint *schema.Blueprint,
	specWillBeTransformed bool,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	if specWillBeTransformed || blueprint.Transform == nil {
		// When the spec will be transformed, errors for invalid transforms will be
		// caught on collection of transform implementations, which fails the load
		// with a more precise error than the diagnostics produced here.
		return diagnostics, nil
	}

	for i, transform := range blueprint.Transform.Values {
		validateTransform(&diagnostics, transform, i, blueprint)
	}

	return diagnostics, nil
}

func validateTransform(
	diagnostics *[]*bpcore.Diagnostic,
	transform string,
	transformIndex int,
	blueprint *schema.Blueprint,
) {
	if strings.TrimSpace(transform) == "" {
		*diagnostics = append(*diagnostics, &bpcore.Diagnostic{
			Level:   bpcore.DiagnosticLevelError,
			Message: "A transform can not be empty.",
			Range:   diagnosticRangeFromTransform(transformIndex, blueprint),
		})
		return
	}

	if substitutions.ContainsSubstitution(transform) {
		*diagnostics = append(*diagnostics, &bpcore.Diagnostic{
			Level:   bpcore.DiagnosticLevelError,
			Message: "${..} substitutions can not be used in a transform.",
			Range:   diagnosticRangeFromTransform(transformIndex, blueprint),
		})
		return
	}
}

func diagnosticRangeFromTransform(transformIndex int, blueprint *schema.Blueprint) *bpcore.DiagnosticRange {
	if len(blueprint.Transform.SourceMeta) == 0 {
		return &bpcore.DiagnosticRange{
			Start: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 1,
				},
			},
			End: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 1,
				},
			},
		}
	}

	transformSourceMeta := blueprint.Transform.SourceMeta[transformIndex]
	endSourceMeta := determineTransformEndSourceMeta(
		transformSourceMeta,
		blueprint.Transform,
		transformIndex,
	)

	return &bpcore.DiagnosticRange{
		Start: transformSourceMeta,
		End:   endSourceMeta,
	}
}

func determineTransformEndSourceMeta(
	transformSourceMeta *source.Meta,
	transform *schema.TransformValueWrapper,
	transformIndex int,
) *source.Meta {
	if transformSourceMeta.EndPosition != nil {
		return &source.Meta{
			Position: *transformSourceMeta.EndPosition,
		}
	}

	endSourceMeta := &source.Meta{
		Position: source.Position{
			Line:   transformSourceMeta.Line + 1,
			Column: 1,
		},
	}

	if transformIndex+1 < len(transform.SourceMeta) {
		endSourceMeta = &source.Meta{
			Position: source.Position{
				Line:   transform.SourceMeta[transformIndex+1].Line,
				Column: 1,
			},
		}
	}

	return endSourceMeta
}
