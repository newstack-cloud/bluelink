package inputvalidation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidationError is a struct that represents a validation error
// that can be returned in responses to clients.
type ValidationError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
	Type     string `json:"type"`
}

// FormattedValidationError is a struct that represents a formatted
// validation error that can be returned in responses to clients.
type FormattedValidationError struct {
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors"`
}

// FormatValidationErrors is a function that takes a slice of validation errors
// and returns a slice of more readable validation errors that can be returned
// to clients.
func FormatValidationErrors(errors validator.ValidationErrors) *FormattedValidationError {
	validationErrors := make([]ValidationError, len(errors))
	for i, err := range errors {
		validationErrors[i] = ValidationError{
			Location: stripNestedAnonymousStructs(
				stripLocationPrefix(err.Namespace()),
			),
			Message: validationErrorMessage(err),
			Type:    err.Tag(),
		}
	}
	return &FormattedValidationError{
		Message: "request body input validation failed",
		Errors:  validationErrors,
	}
}

// HTTPValidationError is a function that writes a slice of validation errors
// to the response writer in a JSON format.
func HTTPValidationError(w http.ResponseWriter, errors validator.ValidationErrors) {
	formatted := FormatValidationErrors(errors)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(formatted)
}

func stripLocationPrefix(namespace string) string {
	structPrefixEnd := strings.Index(namespace, ".")
	if structPrefixEnd == -1 {
		return namespace
	}

	// Include the preceding "." to indicate that the field is nested within
	// a top-level object.
	return namespace[structPrefixEnd:]
}

func stripNestedAnonymousStructs(namespace string) string {
	parts := strings.Split(namespace, ".")
	// regular fields will use the JSON tag name in "lowerCamelCase"
	// form, anonymous nested structs will use the struct name in "UpperCamelCase" form.
	// Nested structs are an implementation detail of Go and should not be exposed to the
	// client that is using the HTTP API.
	partsWithoutAnonymousStructs := core.Filter(
		parts,
		func(part string, _ int) bool {
			return len(part) == 0 || !unicode.IsUpper(rune(part[0]))
		},
	)
	return strings.Join(partsWithoutAnonymousStructs, ".")
}

func validationErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "missing required value"
	case "oneof":
		return fmt.Sprintf("the value must be one of the following: %s", err.Param())
	default:
		return "the specified field is invalid"
	}
}
