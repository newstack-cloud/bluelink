package validation

import (
	"fmt"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// A description is prose for authors. It is persisted in state and served from the API as
// written, and nothing consumes it, so a secret reaching one is an authoring mistake rather
// than a value that needs carrying. Warned about rather than marked as marking would mean a
// redaction flag beside every description in state which adds complexity for an unlikely
// author mistake.
//
// Covers the declarations the blueprint author writes, `secret` on a variable or a value.
// A field a provider declares sensitive is not covered, since reading that needs the
// resource and data source schemas walked, and the nudge is weakest where the author
// declared nothing.
func checkDescriptionForSecrets(
	usedIn string,
	description *substitutions.StringOrSubstitutions,
	valCtx *ValidationContext,
	diagnostics *[]*bpcore.Diagnostic,
) {
	if description == nil {
		return
	}

	for _, stringOrSub := range description.Values {
		secretName := secretReferencedBy(stringOrSub.SubstitutionValue, valCtx)
		if secretName == "" {
			continue
		}

		*diagnostics = append(*diagnostics, &bpcore.Diagnostic{
			Level: bpcore.DiagnosticLevelWarning,
			Message: fmt.Sprintf(
				"the description of %s interpolates %q, which is declared as a secret, "+
					"descriptions are stored and served as written and are never redacted",
				usedIn,
				secretName,
			),
			Range: bpcore.DiagnosticRangeFromSourceMeta(stringOrSub.SourceMeta, nil),
		})
	}
}

// Returns the name of the first secret the substitution reads, or an empty string. Function
// arguments are followed, so a secret passed through a call is still found.
func secretReferencedBy(sub *substitutions.Substitution, valCtx *ValidationContext) string {
	if sub == nil {
		return ""
	}

	if sub.Variable != nil && secretVariable(sub.Variable.VariableName, valCtx) {
		return substitutions.RenderFieldPath("variables", sub.Variable.VariableName)
	}

	if sub.ValueReference != nil && secretValue(sub.ValueReference.ValueName, valCtx) {
		return substitutions.RenderFieldPath("values", sub.ValueReference.ValueName)
	}

	if sub.Function == nil {
		return ""
	}

	for _, argument := range sub.Function.Arguments {
		if secretName := secretReferencedBy(argument.Value, valCtx); secretName != "" {
			return secretName
		}
	}

	return ""
}

func secretVariable(variableName string, valCtx *ValidationContext) bool {
	variable := variableFromSchema(variableName, valCtx)
	return variable != nil && bpcore.BoolValueFromScalar(variable.Secret)
}

func secretValue(valueName string, valCtx *ValidationContext) bool {
	value := valueFromSchema(valueName, valCtx)
	return value != nil && bpcore.BoolValueFromScalar(value.Secret)
}

func variableFromSchema(variableName string, valCtx *ValidationContext) *schema.Variable {
	if valCtx == nil || valCtx.BpSchema == nil || valCtx.BpSchema.Variables == nil {
		return nil
	}

	return valCtx.BpSchema.Variables.Values[variableName]
}

func valueFromSchema(valueName string, valCtx *ValidationContext) *schema.Value {
	if valCtx == nil || valCtx.BpSchema == nil || valCtx.BpSchema.Values == nil {
		return nil
	}

	return valCtx.BpSchema.Values.Values[valueName]
}
