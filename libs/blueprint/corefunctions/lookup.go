package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// LookupFunction provides the implementation of
// a function that looks up a value in a mapping by key
// or an object by attribute name.
// This function will always require a default value to be passed in,
// `.` or `[]` accessors should be used to access values if a default value is not required.
type LookupFunction struct {
	definition *function.Definition
}

// NewLookupFunction creates a new instance of the LookupFunction with
// a complete function definition.
func NewLookupFunction() provider.Function {
	return &LookupFunction{
		definition: &function.Definition{
			Description: "A function that retrieves a value from a map/object using a key, " +
				"with a required default value to be returned if the key is not found.\n" +
				"You should use this function instead of \".\" or \"[]\" accessors if you " +
				"need to provide a default value if the key is not found.",
			FormattedDescription: "A function that retrieves a value from a map/object using a key, " +
				"with a required default value to be returned if the key is not found.\n" +
				"You should use this function instead of `.` or `[]` accessors if you " +
				"need to provide a default value if the key is not found.\n\n" +
				"**Examples:**\n\n" +
				"```\n${lookup(datasources.network.subnets, \"subnet-1234\", \"default\")\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "objectOrMap",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionObject{
							Label: "object",
						},
						&function.ValueTypeDefinitionMap{
							Label: "mapping",
							ElementType: &function.ValueTypeDefinitionAny{
								Label: "any",
								Type:  function.ValueTypeAny,
							},
						},
					},
					Description: "A mapping or object to extract the value from.",
				},
				&function.ScalarParameter{
					Label: "key",
					Type: &function.ValueTypeDefinitionScalar{
						Type: function.ValueTypeString,
					},
					Description: "The key to extract the value from.",
				},
				&function.AnyParameter{
					Label:       "default",
					Description: "The default value to return if the key is not found.",
				},
			},
			Return: &function.AnyReturn{
				Type:        function.ValueTypeAny,
				Description: "The value from the mapping or object or the default value if the key is not found.",
			},
		},
	}
}

func (f *LookupFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *LookupFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// Get first argument to check for none marker
	firstArg, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}

	// If the input map/object is none, return none
	if core.IsNoneMarker(firstArg) {
		return &provider.FunctionCallOutput{
			ResponseData: core.GetNoneMarker(),
		}, nil
	}

	var mapping map[string]any
	var key string
	var defaultVal any
	if err := input.Arguments.GetMultipleVars(ctx, &mapping, &key, &defaultVal); err != nil {
		return nil, err
	}

	value, ok := mapping[key]
	if !ok {
		return &provider.FunctionCallOutput{
			ResponseData: defaultVal,
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: value,
	}, nil
}
