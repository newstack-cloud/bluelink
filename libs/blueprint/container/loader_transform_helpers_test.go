package container

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// errorLinksTransformer returns error-level diagnostics from ValidateLinks
// to test that the loader surfaces them as load errors with
// ErrorReasonCodeInvalidTransformLinks.
type errorLinksTransformer struct {
	internal.ServerlessTransformer
}

func (t *errorLinksTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-error-links-2024", nil
}

func (t *errorLinksTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "link between saveOrderFunction and ordersTable is not supported by the transformer",
			},
		},
	}, nil
}

// warningLinksTransformer returns warning-level diagnostics from ValidateLinks
// to test that the loader includes them in ValidationResult.Diagnostics
// without causing an error.
type warningLinksTransformer struct {
	internal.ServerlessTransformer
}

func (t *warningLinksTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-warning-links-2024", nil
}

func (t *warningLinksTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "link between saveOrderFunction and ordersTable may cause issues at deploy time",
			},
		},
	}, nil
}

// generalErrorLinksTransformer returns a Go error from ValidateLinks
// to test that the loader propagates raw errors from transformer link validation.
type generalErrorLinksTransformer struct {
	internal.ServerlessTransformer
}

func (t *generalErrorLinksTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-go-error-links-2024", nil
}

func (t *generalErrorLinksTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return nil, fmt.Errorf("internal transformer error during link validation")
}

// errorEmitTransformer returns error-level diagnostics from Transform
// (the emit phase) to test that the loader promotes these diagnostics into
// load errors with ErrorReasonCodeTransformValidationErrors. This guards
// against silent partial deploys when a transformer signals a fatal
// user-input problem (e.g. an unsupported runtime) via diagnostics rather
// than a Go error.
type errorEmitTransformer struct {
	internal.ServerlessTransformer
}

func (t *errorEmitTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-error-emit-2024", nil
}

func (t *errorEmitTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "unsupported runtime in resource saveOrderFunction",
			},
		},
	}, nil
}

// warningEmitTransformer returns warning-level diagnostics from Transform
// to test that warnings still flow through Load* without halting (regression
// guard for the Load*/Validate* asymmetry: only error-level emit diagnostics
// are promoted to load errors).
type warningEmitTransformer struct {
	internal.ServerlessTransformer
}

func (t *warningEmitTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-warning-emit-2024", nil
}

func (t *warningEmitTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "runtime field is missing, defaulting to nodejs20.x",
			},
		},
	}, nil
}
