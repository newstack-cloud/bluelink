package core

import (
	"fmt"
	"strconv"
	"strings"

	json "github.com/coreos/go-json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"gopkg.in/yaml.v3"
)

var (
	// True is a helper var for pointers to booleans such as that
	// of the NoneValue field of a scalar.
	True = true
)

// NoneMarker is a special sentinel type used to represent `none` values
// during substitution resolution and function calls.
// This allows us to differentiate `none` from other "empty" values like nil,
// empty strings, false, 0, etc.
//
// IMPORTANT: This marker is used throughout the resolution pipeline and
// MUST be preserved when converting between Go values and MappingNodes.
// When a function receives a NoneMarker as input, it should typically
// return a NoneMarker to propagate the none value through the chain.
//
// The marker should NEVER appear in serialized blueprint files or protobuf
// messages - it is strictly a runtime construct for the substitution engine.
type NoneMarker struct{}

// noneMarkerInstance is the singleton instance of NoneMarker
// used throughout the codebase to represent none values.
var noneMarkerInstance = &NoneMarker{}

// GetNoneMarker returns the singleton instance of the none marker.
// This should be used when converting none values to Go types and
// when checking if a value is none.
//
// Example usage:
//
//	if value == core.GetNoneMarker() {
//	    // Handle none value
//	}
func GetNoneMarker() *NoneMarker {
	return noneMarkerInstance
}

// IsNoneMarker checks if a given value is the none marker.
// This is the recommended way to check if a value represents none
// in the substitution resolution pipeline.
func IsNoneMarker(value any) bool {
	if value == nil {
		return false
	}
	_, ok := value.(*NoneMarker)
	return ok
}

// ScalarValue represents a scalar value in
// a blueprint specification.
// Pointers are used as empty values such as "", 0 and false
// are valid default values.
// When marshalling, only one value is expected to be set,
// if multiple values are provided for some reason the priority
// is as follows:
// 1. int
// 2. bool
// 3. float64
// 4. bytes
// 5. string
//
// The reason strings are the lowest priority is because
// every other scalar type can be parsed as a string.
//
// IMPORTANT: BytesValue is a runtime-only field and should NEVER appear in
// blueprint files. Bytes are only used internally during substitution resolution
// to efficiently pass binary data between functions (e.g., file() -> base64encode()).
// When a ScalarValue with BytesValue is marshalled to YAML/JSON for blueprint output,
// it is automatically converted to a UTF-8 string. When a blueprint is loaded from
// disk, bytes will never be present - only strings, integers, floats, and booleans.
// This is because MappingNode was originally designed as part of the blueprint schema
// which only supports these primitive types in the specification.
type ScalarValue struct {
	IntValue   *int
	BoolValue  *bool
	FloatValue *float64
	// Runtime-only: never appears in blueprint files, converted to string on marshal
	BytesValue  *[]byte
	StringValue *string
	// Runtime-only: never appears in blueprint files, only used during substitution resolution
	NoneValue  *bool
	SourceMeta *source.Meta
}

// ToString returns the string representation of the scalar value
// that is useful for debugging and logging.
func (v *ScalarValue) ToString() string {
	if v.StringValue != nil {
		return *v.StringValue
	}
	if v.IntValue != nil {
		return fmt.Sprintf("%d", *v.IntValue)
	}
	if v.BoolValue != nil {
		return fmt.Sprintf("%t", *v.BoolValue)
	}
	if v.FloatValue != nil {
		return strconv.FormatFloat(*v.FloatValue, 'f', -1, 64)
	}
	if v.BytesValue != nil {
		return string(*v.BytesValue)
	}
	if v.NoneValue != nil {
		return ""
	}
	return ""
}

// MarshalYAML fulfils the yaml.Marshaler interface
// to marshal a blueprint value into one of the
// supported scalar types.
// NOTE: BytesValue is converted to a UTF-8 string during marshalling.
// Bytes are runtime-only and never appear in serialized blueprints.
func (v *ScalarValue) MarshalYAML() (any, error) {
	if v.StringValue != nil {
		return *v.StringValue, nil
	}
	if v.IntValue != nil {
		return *v.IntValue, nil
	}
	if v.BoolValue != nil {
		return *v.BoolValue, nil
	}
	if v.BytesValue != nil {
		// Convert bytes to UTF-8 string for YAML output.
		// Bytes are runtime-only and should never appear in blueprint files.
		return string(*v.BytesValue), nil
	}
	if v.NoneValue != nil && *v.NoneValue {
		// None values are serialised as empty strings.
		return "", nil
	}
	return *v.FloatValue, nil
}

// UnmarshalYAML fulfils the yaml.Unmarshaler interface
// to unmarshal a parsed blueprint value into one of the
// supported scalar types.
func (v *ScalarValue) UnmarshalYAML(value *yaml.Node) error {
	posInfo := YAMLNodeToPosInfo(value)
	if value.Kind != yaml.ScalarNode {
		return errMustBeScalar(posInfo)
	}

	// Scalar values from regular YAML (non-block) have exact column positions.
	colAccuracy := source.ColumnAccuracyExact
	v.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition:    source.EndSourcePositionFromYAMLScalarNode(value),
		ColumnAccuracy: &colAccuracy,
	}

	// Decode will read floating point numbers as integers
	// and truncate. There probably is a cleaner solution
	// for this but checking for decimal point is simple.
	if !strings.Contains(value.Value, ".") {
		var intVal int
		if err := value.Decode(&intVal); err == nil {
			v.IntValue = &intVal
			return nil
		}
	}

	var boolVal bool
	if err := value.Decode(&boolVal); err == nil {
		v.BoolValue = &boolVal
		return nil
	}

	var floatVal float64
	if err := value.Decode(&floatVal); err == nil {
		v.FloatValue = &floatVal
		return nil
	}

	// String is a superset of all other value types so must
	// be tried last.
	var stringVal string
	if err := value.Decode(&stringVal); err == nil {
		v.StringValue = &stringVal
		return nil
	}

	return errMustBeScalar(posInfo)
}

// MarshalJSON fulfils the json.Marshaler interface
// to marshal a blueprint value into one of the
// supported scalar types.
// NOTE: BytesValue is converted to a UTF-8 string during marshalling.
// Bytes are runtime-only and never appear in serialized blueprints.
func (v *ScalarValue) MarshalJSON() ([]byte, error) {
	if v.StringValue != nil {
		return json.Marshal(*v.StringValue)
	}

	if v.IntValue != nil {
		return json.Marshal(*v.IntValue)
	}

	if v.BoolValue != nil {
		return json.Marshal(*v.BoolValue)
	}

	if v.BytesValue != nil {
		// Convert bytes to UTF-8 string for JSON output.
		// Bytes are runtime-only and should never appear in blueprint files.
		return json.Marshal(string(*v.BytesValue))
	}

	if v.NoneValue != nil && *v.NoneValue {
		// None values are serialised as empty strings.
		return json.Marshal("")
	}

	return json.Marshal(*v.FloatValue)
}

func (v *ScalarValue) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	v.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	// JSON nodes treat all numbers as float64.
	if floatVal, isFloat := node.Value.(float64); isFloat {
		if isIntegral(floatVal) {
			intVal := int(floatVal)
			v.IntValue = &intVal
		} else {
			v.FloatValue = &floatVal
		}
		return nil
	}

	if boolVal, isBool := node.Value.(bool); isBool {
		v.BoolValue = &boolVal
		return nil
	}

	if stringVal, isString := node.Value.(string); isString {
		v.StringValue = &stringVal
		return nil
	}

	return errMustBeScalarWithParentPath(&v.SourceMeta.Position, parentPath)
}

// UnmarshalJSON fulfils the json.Unmarshaler interface
// to unmarshal a parsed blueprint value into one of the
// supported scalar types.
func (v *ScalarValue) UnmarshalJSON(data []byte) error {

	// Decode will read floating point numbers as integers
	// and truncate. There probably is a cleaner solution
	// for this but checking for decimal point is simple.
	if !strings.Contains(string(data), ".") {
		var intVal int
		if err := json.Unmarshal(data, &intVal); err == nil {
			v.IntValue = &intVal
			return nil
		}
	}

	var boolVal bool
	if err := json.Unmarshal(data, &boolVal); err == nil {
		v.BoolValue = &boolVal
		return nil
	}

	var floatVal float64
	if err := json.Unmarshal(data, &floatVal); err == nil {
		v.FloatValue = &floatVal
		return nil
	}

	// String is a superset of all other value types so must
	// be tried last.
	var stringVal string
	if err := json.Unmarshal(data, &stringVal); err == nil {
		v.StringValue = &stringVal
		return nil
	}

	return errMustBeScalar(nil)
}

func (v *ScalarValue) Equal(otherScalar *ScalarValue) bool {
	if v == nil || otherScalar == nil {
		return false
	}

	if v.StringValue != nil && otherScalar.StringValue != nil {
		return *v.StringValue == *otherScalar.StringValue
	}

	if v.IntValue != nil && otherScalar.IntValue != nil {
		return *v.IntValue == *otherScalar.IntValue
	}

	if v.BoolValue != nil && otherScalar.BoolValue != nil {
		return *v.BoolValue == *otherScalar.BoolValue
	}

	if v.FloatValue != nil && otherScalar.FloatValue != nil {
		return *v.FloatValue == *otherScalar.FloatValue
	}

	if v.NoneValue != nil && otherScalar.NoneValue != nil {
		return *v.NoneValue == *otherScalar.NoneValue
	}

	if v.BytesValue != nil && otherScalar.BytesValue != nil {
		vBytes := *v.BytesValue
		oBytes := *otherScalar.BytesValue
		if len(vBytes) != len(oBytes) {
			return false
		}
		for i := range vBytes {
			if vBytes[i] != oBytes[i] {
				return false
			}
		}
		return true
	}

	return false
}

// StringValueFromScalar extracts a Go string from a string or bytes
// scalar value. If the scalar is nil, an empty string is returned.
// NOTE: If the scalar has BytesValue, it is converted to a UTF-8 string.
func StringValueFromScalar(scalar *ScalarValue) string {
	if scalar == nil {
		return ""
	}

	if scalar.StringValue != nil {
		return *scalar.StringValue
	}

	if scalar.BytesValue != nil {
		return string(*scalar.BytesValue)
	}

	return ""
}

// IntValueFromScalar extracts a Go int from an int
// scalar value. If the scalar is nil, 0 is returned.
func IntValueFromScalar(scalar *ScalarValue) int {
	if scalar == nil {
		return 0
	}

	if scalar.IntValue != nil {
		return *scalar.IntValue
	}

	return 0
}

// BoolValueFromScalar extracts a Go bool from a bool
// scalar value. If the scalar is nil, false is returned.
func BoolValueFromScalar(scalar *ScalarValue) bool {
	if scalar == nil {
		return false
	}

	if scalar.BoolValue != nil {
		return *scalar.BoolValue
	}

	return false
}

// FloatValueFromScalar extracts a Go float64 from a float64
// scalar value. If the scalar is nil, 0.0 is returned.
func FloatValueFromScalar(scalar *ScalarValue) float64 {
	if scalar == nil {
		return 0.0
	}

	if scalar.FloatValue != nil {
		return *scalar.FloatValue
	}

	return 0.0
}

// ScalarFromString creates a scalar value from a string.
func ScalarFromString(value string) *ScalarValue {
	return &ScalarValue{
		StringValue: &value,
	}
}

// ScalarFromBool creates a scalar value from a boolean.
func ScalarFromBool(value bool) *ScalarValue {
	return &ScalarValue{
		BoolValue: &value,
	}
}

// ScalarFromInt creates a scalar value from an integer.
func ScalarFromInt(value int) *ScalarValue {
	return &ScalarValue{
		IntValue: &value,
	}
}

// ScalarFromFloat creates a scalar value from a float.
func ScalarFromFloat(value float64) *ScalarValue {
	return &ScalarValue{
		FloatValue: &value,
	}
}

// NoneScalar creates a scalar value that represents a none value.
func NoneScalar() *ScalarValue {
	return &ScalarValue{
		NoneValue: &True,
	}
}

// IsScalarNil checks if a scalar value is nil or has no value.
func IsScalarNil(scalar *ScalarValue) bool {
	return scalar == nil || (scalar.StringValue == nil &&
		scalar.IntValue == nil &&
		scalar.BoolValue == nil &&
		scalar.FloatValue == nil &&
		scalar.BytesValue == nil &&
		scalar.NoneValue == nil)
}

// IsScalarString checks if a scalar value is a string or bytes.
// NOTE: Bytes are treated as strings because they are automatically
// converted to UTF-8 strings during serialization. This ensures
// validation treats bytes and strings consistently.
func IsScalarString(scalar *ScalarValue) bool {
	return scalar != nil && (scalar.StringValue != nil || scalar.BytesValue != nil)
}

// IsScalarInt checks if a scalar value is an int.
func IsScalarInt(scalar *ScalarValue) bool {
	return scalar != nil && scalar.IntValue != nil
}

// IsScalarBool checks if a scalar value is a bool.
func IsScalarBool(scalar *ScalarValue) bool {
	return scalar != nil && scalar.BoolValue != nil
}

// IsScalarFloat checks if a scalar value is a float.
func IsScalarFloat(scalar *ScalarValue) bool {
	return scalar != nil && scalar.FloatValue != nil
}

// IsScalarBytes checks if a scalar value is a byte array.
// NOTE: Bytes are runtime-only and will never be true for scalars
// loaded from blueprint files, only for values created during
// substitution resolution.
func IsScalarBytes(scalar *ScalarValue) bool {
	return scalar != nil && scalar.BytesValue != nil
}

// IsScalarNone checks if a scalar value is a none value.
func IsScalarNone(scalar *ScalarValue) bool {
	return scalar != nil && scalar.NoneValue != nil && *scalar.NoneValue
}

// BytesValueFromScalar extracts a Go byte slice from a bytes
// scalar value. If the scalar is nil, nil is returned.
// NOTE: This will only return non-nil for runtime-created scalars.
// Bytes never appear in blueprint files and are converted to strings
// during serialization.
func BytesValueFromScalar(scalar *ScalarValue) []byte {
	if scalar == nil {
		return nil
	}

	if scalar.BytesValue != nil {
		return *scalar.BytesValue
	}

	return nil
}

// ScalarFromBytes creates a scalar value from a byte slice.
// NOTE: This should only be used at runtime during substitution resolution.
// Bytes are never persisted to blueprint files and will be converted to
// UTF-8 strings when marshalled to YAML/JSON.
func ScalarFromBytes(value []byte) *ScalarValue {
	return &ScalarValue{
		BytesValue: &value,
	}
}

// TypeFromScalarValue returns the type of a scalar value
// as a ScalarType. If the scalar is nil, an empty string is returned.
// NOTE: Bytes are treated as strings and will return ScalarTypeString.
func TypeFromScalarValue(scalar *ScalarValue) ScalarType {
	if scalar == nil {
		return ""
	}

	if IsScalarInt(scalar) {
		return ScalarTypeInteger
	}

	if IsScalarBool(scalar) {
		return ScalarTypeBool
	}

	if IsScalarFloat(scalar) {
		return ScalarTypeFloat
	}

	// Check for both StringValue and BytesValue
	// Bytes are treated as strings since they convert to UTF-8 on serialization
	if IsScalarString(scalar) {
		return ScalarTypeString
	}

	return ""
}

// ScalarType represents the type of a scalar value that can be
// used in annotation and configuration definitions.
type ScalarType string

const (
	// ScalarTypeString is the type of an element in a spec that is a string.
	ScalarTypeString ScalarType = "string"
	// ScalarTypeInteger is the type of an element in a spec that is an integer.
	ScalarTypeInteger ScalarType = "integer"
	// ScalarTypeFloat is the type of an element in a spec that is a float.
	ScalarTypeFloat ScalarType = "float"
	// ScalarTypeBool is the type of an element in a spec that is a boolean.
	ScalarTypeBool ScalarType = "boolean"
	// ScalarTypeBytes is the type of a runtime-only element that is a byte array.
	// This type will NEVER appear in blueprint specifications and is only used
	// during substitution resolution to pass binary data between functions.
	// When serialized, bytes are always converted to UTF-8 strings.
	ScalarTypeBytes ScalarType = "bytes"
)

func isIntegral(value float64) bool {
	return value == float64(int(value))
}
