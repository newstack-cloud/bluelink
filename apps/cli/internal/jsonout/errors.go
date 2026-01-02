package jsonout

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
)

// NewErrorOutput converts an error to an ErrorOutput struct.
func NewErrorOutput(err error) ErrorOutput {
	// Handle validation errors (ClientError with ValidationErrors or ValidationDiagnostics)
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		return newValidationErrorOutput(clientErr)
	}

	// Handle stream errors with diagnostics
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		return newStreamErrorOutput(streamErr)
	}

	// Handle other client errors
	if clientErr, ok := err.(*engineerrors.ClientError); ok {
		return newClientErrorOutput(clientErr)
	}

	// Generic error
	return ErrorOutput{
		Success: false,
		Error: ErrorDetail{
			Type:    "internal",
			Message: err.Error(),
		},
	}
}

func newValidationErrorOutput(clientErr *engineerrors.ClientError) ErrorOutput {
	detail := ErrorDetail{
		Type:       "validation",
		Message:    clientErr.Message,
		StatusCode: clientErr.StatusCode,
	}

	// Convert validation diagnostics
	if len(clientErr.ValidationDiagnostics) > 0 {
		detail.Diagnostics = convertDiagnostics(clientErr.ValidationDiagnostics)
	}

	// Convert validation errors
	if len(clientErr.ValidationErrors) > 0 {
		detail.Validation = make([]ValidationError, len(clientErr.ValidationErrors))
		for i, ve := range clientErr.ValidationErrors {
			detail.Validation[i] = ValidationError{
				Location: ve.Location,
				Message:  ve.Message,
				Type:     ve.Type,
			}
		}
	}

	return ErrorOutput{
		Success: false,
		Error:   detail,
	}
}

func newStreamErrorOutput(streamErr *engineerrors.StreamError) ErrorOutput {
	detail := ErrorDetail{
		Type:    "stream",
		Message: streamErr.Event.Message,
	}

	if len(streamErr.Event.Diagnostics) > 0 {
		detail.Diagnostics = convertDiagnostics(streamErr.Event.Diagnostics)
	}

	return ErrorOutput{
		Success: false,
		Error:   detail,
	}
}

func newClientErrorOutput(clientErr *engineerrors.ClientError) ErrorOutput {
	return ErrorOutput{
		Success: false,
		Error: ErrorDetail{
			Type:       "client",
			Message:    clientErr.Message,
			StatusCode: clientErr.StatusCode,
		},
	}
}

func convertDiagnostics(diagnostics []*core.Diagnostic) []Diagnostic {
	result := make([]Diagnostic, len(diagnostics))
	for i, d := range diagnostics {
		diag := Diagnostic{
			Level:   headless.DiagnosticLevelName(headless.DiagnosticLevelFromCore(d.Level)),
			Message: d.Message,
		}
		if d.Range != nil && d.Range.Start != nil {
			diag.Line = d.Range.Start.Line
			diag.Column = d.Range.Start.Column
		}
		result[i] = diag
	}
	return result
}
