// Provides functions that convert between the mapping node
// representation of values and native Go types to be used in
// function calls.

package subengine

import (
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// GoValueToMappingNode converts a Go value to a mapping node
// to be used in resolving substitutions for values returned
// by function calls.
// This uses reflection for the conversion process,
// reflection is generally slow but is the only way to effectively
// convert Go values to mapping nodes.
// There shouldn't be noticeable impact on performance unless very large, complex
// structures are being converted or a blueprint contains 100s of function calls.
func GoValueToMappingNode(value any) *core.MappingNode {
	// Check for none marker first - convert back to none scalar
	if core.IsNoneMarker(value) {
		return &core.MappingNode{
			Scalar: core.NoneScalar(),
		}
	}

	finalValue := dereferencePointer(value)

	typeofValue := reflect.TypeOf(finalValue)
	kindOfValue := typeofValue.Kind()

	if isInt(kindOfValue) {
		intValue := value.(int)
		return toIntMappingNode(intValue)
	}

	if isFloat(kindOfValue) {
		floatValue := value.(float64)
		return toFloatMappingNode(floatValue)
	}

	if kindOfValue == reflect.String {
		stringValue := value.(string)
		return toStringMappingNode(stringValue)
	}

	if kindOfValue == reflect.Bool {
		boolValue := value.(bool)
		return toBoolMappingNode(boolValue)
	}

	if kindOfValue == reflect.Slice {
		// Special handling for byte slices ([]byte) - store as bytes
		if typeofValue.Elem().Kind() == reflect.Uint8 {
			byteSlice := value.([]byte)
			return toBytesMappingNode(byteSlice)
		}
		return toSliceMappingNode(value)
	}

	if kindOfValue == reflect.Map {
		return toMapMappingNode(value)
	}

	if kindOfValue == reflect.Struct {
		return toStructMappingNode(value)
	}

	return nil
}

// MappingNodeToGoValue converts a mapping node to a Go value
// to be used as arguments for function calls as functions expect
// Go values as arguments.
func MappingNodeToGoValue(node *core.MappingNode) any {
	if node.Scalar != nil {
		return toGoScalar(node.Scalar)
	}

	if node.Items != nil {
		return toGoSlice(node.Items)
	}

	if node.Fields != nil {
		return toGoMap(node.Fields)
	}

	return nil
}

func toGoScalar(Scalar *core.ScalarValue) any {
	// Check for none value first - return the special marker
	if core.IsScalarNone(Scalar) {
		return core.GetNoneMarker()
	}

	if Scalar.IntValue != nil {
		return *Scalar.IntValue
	}

	if Scalar.FloatValue != nil {
		return *Scalar.FloatValue
	}

	if Scalar.BytesValue != nil {
		return *Scalar.BytesValue
	}

	if Scalar.StringValue != nil {
		return *Scalar.StringValue
	}

	if Scalar.BoolValue != nil {
		return *Scalar.BoolValue
	}

	return nil
}

func toGoSlice(items []*core.MappingNode) interface{} {
	slice := []interface{}{}
	for _, item := range items {
		slice = append(slice, MappingNodeToGoValue(item))
	}

	return slice
}

func toGoMap(fields map[string]*core.MappingNode) interface{} {
	mapping := map[string]interface{}{}
	for key, value := range fields {
		mapping[key] = MappingNodeToGoValue(value)
	}

	return mapping
}

func toIntMappingNode(value int) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{
			IntValue: &value,
		},
	}
}

func toFloatMappingNode(value float64) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{
			FloatValue: &value,
		},
	}
}

func toStringMappingNode(value string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{
			StringValue: &value,
		},
	}
}

func toBoolMappingNode(value bool) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{
			BoolValue: &value,
		},
	}
}

func toBytesMappingNode(value []byte) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{
			BytesValue: &value,
		},
	}
}

func toSliceMappingNode(value interface{}) *core.MappingNode {
	valueOfSlice := reflect.ValueOf(value)
	items := []*core.MappingNode{}
	for i := 0; i < valueOfSlice.Len(); i += 1 {
		items = append(items, GoValueToMappingNode(valueOfSlice.Index(i).Interface()))
	}

	return &core.MappingNode{
		Items: items,
	}
}

func toMapMappingNode(value interface{}) *core.MappingNode {
	valueOfMap := reflect.ValueOf(value)
	fields := map[string]*core.MappingNode{}
	for _, key := range valueOfMap.MapKeys() {
		keyString := key.String()
		fields[keyString] = GoValueToMappingNode(valueOfMap.MapIndex(key).Interface())
	}

	return &core.MappingNode{
		Fields: fields,
	}
}

func toStructMappingNode(value interface{}) *core.MappingNode {
	valueOfStruct := reflect.ValueOf(value)
	fields := map[string]*core.MappingNode{}
	for i := 0; i < valueOfStruct.NumField(); i += 1 {
		field := valueOfStruct.Field(i)
		fieldName := valueOfStruct.Type().Field(i).Name
		fields[fieldName] = GoValueToMappingNode(field.Interface())
	}

	return &core.MappingNode{
		Fields: fields,
	}
}

func isInt(kind reflect.Kind) bool {
	// Unsigned integers are currently not supported as the substitution language
	// does not have a concept of signed and unsigned integers.
	return kind == reflect.Int ||
		kind == reflect.Int8 ||
		kind == reflect.Int16 ||
		kind == reflect.Int32 ||
		kind == reflect.Int64
}

func isFloat(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}

func dereferencePointer(value interface{}) interface{} {
	typeofValue := reflect.TypeOf(value)
	kindOfValue := typeofValue.Kind()

	if kindOfValue == reflect.Ptr {
		return reflect.ValueOf(value).Elem().Interface()
	}

	return value
}
