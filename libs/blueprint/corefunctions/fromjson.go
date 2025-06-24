package corefunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FromJSONFunction provides the implementation of the
// core "fromjson" function defined in the blueprint specification.
type FromJSONFunction struct {
	definition *function.Definition
}

// NewFromJSONFunction creates a new instance of the FromJSONFunction with
// a complete function definition.
func NewFromJSONFunction() provider.Function {
	return &FromJSONFunction{
		definition: &function.Definition{
			Description: "Extracts a value from a serialised JSON string. " +
				"This uses json pointer notation to allow for the extraction of values from complex " +
				"serialised structures. This only works for extracting values when the root of the json string is an object.",
			FormattedDescription: "Extracts a value from a serialised JSON string. " +
				"This uses [json pointer notation](https://datatracker.ietf.org/doc/rfc6901/) " +
				"to allow for the extraction of values from complex serialised structures." +
				" **This only works for extracting values when the root of the json string is an object.**\n\n" +
				"**Examples:**\n\n" +
				"```\n${fromjson(variables.cacheClusterConfig, \"/host\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "jsonString",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "JSON string",
						Type:  function.ValueTypeString,
						Description: "A valid string literal, reference or function" +
							" call yielding the json string to extract values from",
					},
				},
				&function.ScalarParameter{
					Label: "jsonPointer",
					Type: &function.ValueTypeDefinitionScalar{
						Label:       "JSON pointer",
						Type:        function.ValueTypeString,
						Description: "A valid json pointer expression to extract the value from the json string.",
						FormattedDescription: "A valid [json pointer expression](https://datatracker.ietf.org/doc/rfc6901/) " +
							"to extract the value from the json string.",
					},
				},
			},
			Return: &function.AnyReturn{
				Type: function.ValueTypeAny,
				Description: "The value extracted from the json string. " +
					"This can be a primitive value, an array, mapping or object",
			},
		},
	}
}

func (f *FromJSONFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *FromJSONFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var jsonStr string
	var jsonPtr string
	err := input.Arguments.GetMultipleVars(ctx, &jsonStr, &jsonPtr)
	if err != nil {
		return nil, err
	}

	decoded := make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonStr), &decoded)
	if err != nil {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("unable to decode json string: %s", err.Error()),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	value, found := extractJSONValueWithPointer(decoded, jsonPtr)
	if !found {
		return nil, function.NewFuncCallError(
			fmt.Sprintf(
				"unable to extract value from json string using pointer \"%s\"",
				jsonPtr,
			),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: value,
	}, nil
}

func extractJSONValueWithPointer(data map[string]interface{}, pointer string) (interface{}, bool) {
	normalisedPointer := pointer
	// Allow for URI fragment representation.
	if strings.HasPrefix(pointer, "#") {
		normalisedPointer = pointer[1:]
	}

	if normalisedPointer == "" {
		return data, true
	}

	if !strings.HasPrefix(normalisedPointer, "/") {
		return data, false
	}

	referenceTokens := strings.Split(normalisedPointer[1:], "/")
	var current interface{}
	current = data
	found := false
	invalid := false

	i := 0
	for !found && !invalid && i < len(referenceTokens) {
		decodedToken := decodeJSONPointerToken(referenceTokens[i])

		currentKind := reflect.TypeOf(current).Kind()
		if currentKind == reflect.Map {
			currentMap, ok := current.(map[string]interface{})
			if !ok {
				invalid = true
			} else {
				value, exists := currentMap[decodedToken]
				if exists {
					current = value
					found = i == len(referenceTokens)-1
				} else {
					invalid = true
				}
			}
		} else if currentKind == reflect.Slice || currentKind == reflect.Array {
			currentSlice, ok := current.([]interface{})
			if !ok {
				invalid = true
			} else {
				index, err := parseJSONPointerTokenIndex(decodedToken, len(currentSlice))
				if err != nil {
					invalid = true
				} else {
					current = currentSlice[index]
					found = i == len(referenceTokens)-1
				}
			}
		} else {
			// We have reached a leaf node,
			// we can't traverse further so the pointer is invalid.
			invalid = true
		}
		i += 1
	}

	return current, found
}

func decodeJSONPointerToken(referenceToken string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(referenceToken, "~1", "/"),
		"~0",
		"~",
	)
}

func parseJSONPointerTokenIndex(referenceToken string, maxIndex int) (int, error) {
	if referenceToken == "-" {
		// As per the spec, "-" is a special token that refers to the nonexistent
		// element after the last.
		return 0, fmt.Errorf("index out of bounds")
	}

	index, err := strconv.Atoi(referenceToken)
	if err != nil {
		return 0, err
	}

	if index < 0 || index >= maxIndex {
		return 0, fmt.Errorf("index out of bounds")
	}

	return index, nil
}
