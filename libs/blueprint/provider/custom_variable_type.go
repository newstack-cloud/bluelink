package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// CustomVariableType provides the interface for a custom variable type
// that provides convenience variable types with a (usually large) fixed set of possible values.
// A custom variable type should not be used for dynamically sourced values
// external to a blueprint, data sources exist for that purpose.
// All custom variable type values must be of the same primitive type.
type CustomVariableType interface {
	// GetType deals with retrieving the namespaced type for a custom variable type.
	GetType(ctx context.Context, input *CustomVariableTypeGetTypeInput) (*CustomVariableTypeGetTypeOutput, error)
	// GetDescription deals with retrieving the description for a custom variable type in a blueprint spec
	// that can be used for documentation and tooling.
	// Markdown and plain text formats are supported.
	GetDescription(ctx context.Context, input *CustomVariableTypeGetDescriptionInput) (*CustomVariableTypeGetDescriptionOutput, error)
	// Options loads a set of fixed possible values available
	// for the custom variable type.
	// In the returned options, each one is keyed by a label, essentially
	// behaving as a runtime enum.
	Options(ctx context.Context, input *CustomVariableTypeOptionsInput) (*CustomVariableTypeOptionsOutput, error)
	// GetExamples loads a set of examples for how to use the custom
	// variable type in a blueprint.
	GetExamples(ctx context.Context, input *CustomVariableTypeGetExamplesInput) (*CustomVariableTypeGetExamplesOutput, error)
}

// CustomVariableTypeOptionsInput provides the input required to load
// the fixed set of possible values for a custom variable type.
type CustomVariableTypeOptionsInput struct {
	ProviderContext Context
}

// CustomVariableTypeOptionsOutput provides
// the output from loading the fixed set of possible values
// for a custom variable type.
type CustomVariableTypeOptionsOutput struct {
	Options map[string]*CustomVariableTypeOption
}

type CustomVariableTypeOption struct {
	// The value of the option.
	Value *core.ScalarValue
	// A human-readable label for the option.
	Label string
	// A human-readable plain text description for the option.
	Description string
	// A human-readable description for the option
	// that can be formatted in markdown.
	MarkdownDescription string
}

// CustomVariableTypeGetTypeInput provides the input required to
// retrieve the namespaced type for a custom variable type.
type CustomVariableTypeGetTypeInput struct {
	ProviderContext Context
}

// CustomVariableTypeGetTypeOutput provides the output from retrieving the namespaced type
// for a custom variable type.
type CustomVariableTypeGetTypeOutput struct {
	Type string
	// A human-readable label for the custom variable type.
	Label string
}

// CustomVariableTypeGetDescriptionInput provides the input required to
// retrieve a description for a custom variable type.
type CustomVariableTypeGetDescriptionInput struct {
	ProviderContext Context
}

// CustomVariableTypeGetDescriptionOutput provides the output from retrieving the description
// for a custom variable type.
type CustomVariableTypeGetDescriptionOutput struct {
	MarkdownDescription  string
	PlainTextDescription string
	// A short summary of the custom variable type that can be formatted
	// in markdown, this is useful for listing custom variable types in documentation.
	MarkdownSummary string
	// A short summary of the custom variable type in plain text,
	// this is useful for listing custom variable types in documentation.
	PlainTextSummary string
}

// CustomVariableTypeGetExamplesInput provides the input required to
// retrieve examples for a custom variable type.
type CustomVariableTypeGetExamplesInput struct {
	ProviderContext Context
}

// CustomVariableTypeGetExamplesOutput provides the output from retrieving examples
// for a custom variable type.
type CustomVariableTypeGetExamplesOutput struct {
	PlainTextExamples []string
	MarkdownExamples  []string
}
