package corefunctions

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SupportedDateTimeFormats is a list of valid date/time formats
// supported by the DateTimeFunction.
var SupportedDateTimeFormats = []string{"unix", "rfc3339", "tag", "tagcompact"}

// DateTimeFunction provides the implementation of
// a function that gets the current date/time as per the host
// system's clock in the specified format.
type DateTimeFunction struct {
	definition *function.Definition
	clock      core.Clock
}

// NewDateTimeFunction creates a new instance of the DateTimeFunction with
// a complete function definition.
func NewDateTimeFunction(clock core.Clock) provider.Function {
	return &DateTimeFunction{
		clock: clock,
		definition: &function.Definition{
			Description: "A function that returns the current date/time" +
				" as per the host system's clock in the specified format.\n\n" +
				"All times are normalised to UTC.\n\n" +
				"The \"datetime\" function supports the following date/time formats:\n\n" +
				"- \"unix\" - The number of seconds since the Unix epoch. (e.g. 1611312000)\n" +
				"- \"rfc3339\" - The date/time in RFC3339 format. (e.g. 2023-01-02T15:04:05Z07:00)\n" +
				"- \"tag\" - The date/time in a format suitable for use as a tag. (e.g. 2023-01-02--15-04-05)\n" +
				"- \"tagcompact\" - The date/time in a compact format suitable for use as a tag. (e.g. 20230102150405)\n\n" +
				"\"rfc3339\" is a format derived from ISO 8601.\n\n" +
				"This function is useful for generating timestamps that can be used for tagging and versioning resources. " +
				"(e.g. Docker image tags, S3 object keys, etc.)",
			FormattedDescription: "A function that returns the current date/time" +
				" as per the host system's clock in the specified format.\n\n" +
				"All times are normalised to UTC.\n\n" +
				"The `datetime` function supports the following date/time formats:\n\n" +
				"- `unix` - The number of seconds since the Unix epoch. (e.g. 1611312000)\n" +
				"- `rfc3339` - The date/time in RFC3339 format. (e.g. 2023-01-02T15:04:05Z07:00)\n" +
				"- `tag` - The date/time in a format suitable for use as a tag. (e.g. 2023-01-02--15-04-05)\n" +
				"- `tagcompact` - The date/time in a compact format suitable for use as a tag. (e.g. 20230102150405)\n\n" +
				"_`rfc3339` is a format derived from ISO 8601._\n\n" +
				"This function is useful for generating timestamps that can be used for tagging and versioning resources. " +
				"(e.g. Docker image tags, S3 object keys, etc.)\n\n" +
				"**Examples:**\n\n" +
				"```\n${datetime(\"tag\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "format",
					Type: &function.ValueTypeDefinitionScalar{
						Label:         "string",
						Type:          function.ValueTypeString,
						StringChoices: SupportedDateTimeFormats,
					},
					Description: "The date/time format to return the current date/time in.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "A string representing the current time in the requested format.",
			},
		},
	}
}

func (f *DateTimeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *DateTimeFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var format string
	if err := input.Arguments.GetVar(ctx, 0, &format); err != nil {
		return nil, err
	}

	if !slices.Contains(SupportedDateTimeFormats, format) {
		return nil, function.NewFuncCallError(
			"the requested date/time format is not supported by the \"datetime\" function, "+
				"supported formats include: "+strings.Join(SupportedDateTimeFormats, ", "),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	now := f.clock.Now()

	output := ""
	switch format {
	case "unix":
		output = strconv.FormatInt(now.Unix(), 10)
	case "rfc3339":
		output = now.UTC().Format(time.RFC3339)
	case "tag":
		output = now.UTC().Format("2006-01-02--15-04-05")
	case "tagcompact":
		output = now.UTC().Format("20060102150405")
	}

	return &provider.FunctionCallOutput{
		ResponseData: output,
	}, nil
}
