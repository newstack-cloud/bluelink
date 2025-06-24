package core

import (
	"fmt"
	"strconv"
	"strings"

	json "github.com/coreos/go-json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"gopkg.in/yaml.v3"
)

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
// 4. string
//
// The reason strings are the lowest priority is because
// every other scalar type can be parsed as a string.
type ScalarValue struct {
	IntValue    *int
	BoolValue   *bool
	FloatValue  *float64
	StringValue *string
	SourceMeta  *source.Meta
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
	return ""
}

// MarshalYAML fulfils the yaml.Marshaler interface
// to marshal a blueprint value into one of the
// supported scalar types.
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

	v.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
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

	return false
}

// StringValueFromScalar extracts a Go string from a string
// scalar value. If the scalar is nil, an empty string is returned.
func StringValueFromScalar(scalar *ScalarValue) string {
	if scalar == nil {
		return ""
	}

	if scalar.StringValue != nil {
		return *scalar.StringValue
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

// IsScalarNil checks if a scalar value is nil or has no value.
func IsScalarNil(scalar *ScalarValue) bool {
	return scalar == nil || (scalar.StringValue == nil &&
		scalar.IntValue == nil &&
		scalar.BoolValue == nil &&
		scalar.FloatValue == nil)
}

// IsScalarString checks if a scalar value is a string.
func IsScalarString(scalar *ScalarValue) bool {
	return scalar != nil && scalar.StringValue != nil
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

// TypeFromScalarValue returns the type of a scalar value
// as a ScalarType. If the scalar is nil, an empty string is returned.
func TypeFromScalarValue(scalar *ScalarValue) ScalarType {
	if scalar == nil {
		return ""
	}

	if IsScalarString(scalar) {
		return ScalarTypeString
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
)

func isIntegral(value float64) bool {
	return value == float64(int(value))
}
