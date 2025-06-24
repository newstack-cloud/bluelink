package internal

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// RenderErrorTree produces a tree representation of a blueprint error.
func RenderErrorTree(err error) string {
	return renderErrorTree(err, "")
}

func renderErrorTree(err error, indent string) string {
	if loadErr, ok := err.(*errors.LoadError); ok {
		return renderError("LoadError", loadErr.Error(), loadErr.ChildErrors, indent)
	}

	if runErr, ok := err.(*errors.RunError); ok {
		return renderError("RunError", runErr.Error(), runErr.ChildErrors, indent)
	}

	if serialiseErr, ok := err.(*errors.SerialiseError); ok {
		return renderError("SerialiseError", serialiseErr.Error(), serialiseErr.ChildErrors, indent)
	}

	if parseErr, ok := err.(*substitutions.ParseErrors); ok {
		return renderError("ParseError", parseErr.Error(), parseErr.ChildErrors, indent)
	}

	if lexErr, ok := err.(*substitutions.LexErrors); ok {
		return renderError("LexError", lexErr.Error(), lexErr.ChildErrors, indent)
	}

	return renderError("Error", err.Error(), nil, indent)
}

func renderError(errorType string, message string, childErrors []error, indent string) string {
	rendered := strings.Builder{}
	childIndent := indent + "  "
	rendered.WriteString(indent)
	rendered.WriteString(errorType)
	rendered.WriteString("\n")
	rendered.WriteString(childIndent)
	rendered.WriteString("Message: ")
	rendered.WriteString(message)
	rendered.WriteString("\n")

	if childErrors == nil {
		return rendered.String()
	}

	rendered.WriteString(childIndent)
	rendered.WriteString("Children: ")
	renderChildErrors(childErrors, childIndent, &rendered)
	return rendered.String()
}

func renderChildErrors(childErrors []error, indent string, rendered *strings.Builder) {
	for _, child := range childErrors {
		rendered.WriteString("\n")
		rendered.WriteString(renderErrorTree(child, indent))
	}
}
