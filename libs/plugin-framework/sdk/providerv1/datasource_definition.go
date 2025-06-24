package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// DataSourceDefinition is a template to be used for defining data sources
// when creating provider plugins.
// It provides a structure that allows you to define a schema and behaviour
// of a data source.
// This implements the `provider.DataSource` interface and can be used in the same way
// as any other data source implementation used in a provider plugin.
type DataSourceDefinition struct {
	// The type of the data source, prefixed by the namespace of the provider.
	// Example: "aws/lambda/function" for the "aws" provider.
	Type string

	// A human-readable label for the data source type.
	// This will be used in documentation and tooling.
	Label string

	// A summary of the data source type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextSummary string

	// A summary of the data source type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedSummary string

	// A description of the data source type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextDescription string

	// A description of the data source type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedDescription string

	// A list of plain text examples that can be used to
	// demonstrate how to use the data source.
	// This will be used in documentation and tooling.
	PlainTextExamples []string

	// A list of markdown examples that can be used to
	// demonstrate how to use the data source.
	// This will be used in documentation and tooling.
	MarkdownExamples []string

	// Schema definitions for each of the fields that can be exported
	// from a data source.
	Fields map[string]*provider.DataSourceSpecSchema

	// A static map of field names to schemas for filters
	// that can be used to filter in a data source query
	// to the provider.
	// If FilterFieldsFunc is provided, this static list will not be used.
	FilterFields map[string]*provider.DataSourceFilterSchema

	// A function that can be used to dynamically determine a list of fields
	// that can be used to filter in a data source query.
	FilterFieldsFunc func(
		ctx context.Context,
		input *provider.DataSourceGetFilterFieldsInput,
	) (*provider.DataSourceGetFilterFieldsOutput, error)

	// The function that deals with applying filters and retrieving the requested
	// data source from the upstream provider.
	FetchFunc func(
		ctx context.Context,
		input *provider.DataSourceFetchInput,
	) (*provider.DataSourceFetchOutput, error)

	// A function that can be used to carry validation for a data source that goes
	// beyond field schema and filter field validation.
	// When not provided, the plugin will return a result with an empty
	// list of diagnostics.
	CustomValidateFunc func(
		ctx context.Context,
		input *provider.DataSourceValidateInput,
	) (*provider.DataSourceValidateOutput, error)
}

func (d *DataSourceDefinition) GetType(
	ctx context.Context,
	input *provider.DataSourceGetTypeInput,
) (*provider.DataSourceGetTypeOutput, error) {
	return &provider.DataSourceGetTypeOutput{
		Type:  d.Type,
		Label: d.Label,
	}, nil
}

func (d *DataSourceDefinition) GetTypeDescription(
	ctx context.Context,
	input *provider.DataSourceGetTypeDescriptionInput,
) (*provider.DataSourceGetTypeDescriptionOutput, error) {
	return &provider.DataSourceGetTypeDescriptionOutput{
		PlainTextDescription: d.PlainTextDescription,
		MarkdownDescription:  d.FormattedDescription,
		PlainTextSummary:     d.PlainTextSummary,
		MarkdownSummary:      d.FormattedSummary,
	}, nil
}

func (d *DataSourceDefinition) GetExamples(
	ctx context.Context,
	input *provider.DataSourceGetExamplesInput,
) (*provider.DataSourceGetExamplesOutput, error) {
	return &provider.DataSourceGetExamplesOutput{
		PlainTextExamples: d.PlainTextExamples,
		MarkdownExamples:  d.MarkdownExamples,
	}, nil
}

func (d *DataSourceDefinition) CustomValidate(
	ctx context.Context,
	input *provider.DataSourceValidateInput,
) (*provider.DataSourceValidateOutput, error) {
	if d.CustomValidateFunc == nil {
		return &provider.DataSourceValidateOutput{
			Diagnostics: []*core.Diagnostic{},
		}, nil
	}

	return d.CustomValidateFunc(ctx, input)
}

func (d *DataSourceDefinition) GetSpecDefinition(
	ctx context.Context,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	return &provider.DataSourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.DataSourceSpecDefinition{
			Fields: d.Fields,
		},
	}, nil
}

func (d *DataSourceDefinition) GetFilterFields(
	ctx context.Context,
	input *provider.DataSourceGetFilterFieldsInput,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	if d.FilterFieldsFunc != nil {
		return d.FilterFieldsFunc(ctx, input)
	}

	return &provider.DataSourceGetFilterFieldsOutput{
		FilterFields: d.FilterFields,
	}, nil
}

func (d *DataSourceDefinition) Fetch(
	ctx context.Context,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	if d.FetchFunc == nil {
		return nil, errDataSourceFetchFunctionMissing(d.Type)
	}

	return d.FetchFunc(ctx, input)
}
