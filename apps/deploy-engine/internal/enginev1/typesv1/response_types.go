package typesv1

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// ValidationDiagnosticErrors is the data type for validation errors
// that are returned in the response of multiple endpoints.
type ValidationDiagnosticErrors struct {
	Message               string             `json:"message"`
	ValidationDiagnostics []*core.Diagnostic `json:"validationDiagnostics"`
}
