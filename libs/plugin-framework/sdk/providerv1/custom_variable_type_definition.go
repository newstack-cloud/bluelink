package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// CustomVariableTypeDefinition is a template to be used for defining
// custom variable types when creating provider plugins.
// It provides a structure that allows you to define a function or a static list
// of options for variable type options.
// This implements the `provider.CustomVariableType` interface and can be used in the same way
// as any other custom variable type implementation used in a provider plugin.
type CustomVariableTypeDefinition struct {
	// The type of the custom variable type, prefixed by the namespace of the provider.
	// Example: "aws/ec2/instanceType" for the "aws" provider.
	Type string

	// A human-readable label for the custom variable type.
	// Example: "EC2 Instance Type" for the "aws/ec2/instanceType" custom variable type.
	Label string

	// A summary for the custom variable type that is not formatted
	// that can be used to render summaries in contexts that formatting is not supported.
	// This will be used in documentation and tooling when listing custom variable types.
	PlainTextSummary string

	// A summary for the custom variable type that can be formatted using markdown.
	// This will be used in documentation and tooling when listing custom variable types.
	FormattedSummary string

	// A description of the custom variable type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextDescription string

	// A description of the custom variable type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedDescription string

	// A list of examples that are not formatted that can be used to render examples
	// in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextExamples []string

	// A list of examples that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedExamples []string

	// A static map of options for values that are available for the variable type.
	// Each option is keyed by a label, essentially behaving as a runtime enum.
	// If OptionsFunc is defined, this static map will not be used.
	CustomVarTypeOptions map[string]*provider.CustomVariableTypeOption

	// A function that dynamically determines a map of key value pairs for options
	// that the user can choose from for a custom variable type.
	OptionsFunc func(
		ctx context.Context,
		input *provider.CustomVariableTypeOptionsInput,
	) (*provider.CustomVariableTypeOptionsOutput, error)
}

func (c *CustomVariableTypeDefinition) GetType(
	ctx context.Context,
	input *provider.CustomVariableTypeGetTypeInput,
) (*provider.CustomVariableTypeGetTypeOutput, error) {
	return &provider.CustomVariableTypeGetTypeOutput{
		Type:  c.Type,
		Label: c.Label,
	}, nil
}

func (c *CustomVariableTypeDefinition) GetDescription(
	ctx context.Context,
	input *provider.CustomVariableTypeGetDescriptionInput,
) (*provider.CustomVariableTypeGetDescriptionOutput, error) {
	return &provider.CustomVariableTypeGetDescriptionOutput{
		PlainTextDescription: c.PlainTextDescription,
		MarkdownDescription:  c.FormattedDescription,
		PlainTextSummary:     c.PlainTextSummary,
		MarkdownSummary:      c.FormattedSummary,
	}, nil
}

func (c *CustomVariableTypeDefinition) GetExamples(
	ctx context.Context,
	input *provider.CustomVariableTypeGetExamplesInput,
) (*provider.CustomVariableTypeGetExamplesOutput, error) {
	return &provider.CustomVariableTypeGetExamplesOutput{
		PlainTextExamples: c.PlainTextExamples,
		MarkdownExamples:  c.FormattedExamples,
	}, nil
}

func (c *CustomVariableTypeDefinition) Options(
	ctx context.Context,
	input *provider.CustomVariableTypeOptionsInput,
) (*provider.CustomVariableTypeOptionsOutput, error) {
	if c.OptionsFunc != nil {
		return c.OptionsFunc(ctx, input)
	}

	return &provider.CustomVariableTypeOptionsOutput{
		Options: c.CustomVarTypeOptions,
	}, nil
}
