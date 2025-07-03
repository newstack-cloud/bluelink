package core

import (
	"fmt"
	"maps"
	"strings"
)

// Sum calculates the sum of a list of numbers.
func Sum(numbers []float64) float64 {
	sum := 0.0
	for _, number := range numbers {
		sum += number
	}
	return sum
}

// IsInScalarList checks if a given scalar value is in a list of scalar values.
func IsInScalarList(value *ScalarValue, list []*ScalarValue) bool {
	found := false
	i := 0
	for !found && i < len(list) {
		found = list[i].Equal(value)
		i += 1
	}
	return found
}

// Flatten returns a flattened 2D array of the given type.
func Flatten[Item any](array [][]Item) []Item {
	flattened := []Item{}
	for _, row := range array {
		flattened = append(flattened, row...)
	}
	return flattened
}

// StringValue extracts the string value from a MappingNode.
func StringValue(value *MappingNode) string {
	if value == nil || value.Scalar == nil || value.Scalar.StringValue == nil {
		return ""
	}

	return *value.Scalar.StringValue
}

// BoolValue extracts the boolean value from a MappingNode.
// This will return false if the value is nil or the given mapping node
// contains another type of value.
func BoolValue(value *MappingNode) bool {
	if value == nil || value.Scalar == nil || value.Scalar.BoolValue == nil {
		return false
	}

	return *value.Scalar.BoolValue
}

// IntValue extracts the integer value from a MappingNode.
// This will return 0 if the value is nil or the given mapping node
// contains another type of value.
func IntValue(value *MappingNode) int {
	if value == nil || value.Scalar == nil || value.Scalar.IntValue == nil {
		return 0
	}

	return *value.Scalar.IntValue
}

// FloatValue extracts the float value from a MappingNode.
// This will return 0.0 if the value is nil or the given mapping node
// contains another type of value.
func FloatValue(value *MappingNode) float64 {
	if value == nil || value.Scalar == nil || value.Scalar.FloatValue == nil {
		return 0.0
	}

	return *value.Scalar.FloatValue
}

// StringSliceValue extracts a slice of strings from a MappingNode.
func StringSliceValue(value *MappingNode) []string {
	if value == nil || value.Items == nil {
		return []string{}
	}

	strings := make([]string, len(value.Items))
	for i, item := range value.Items {
		strings[i] = StringValue(item)
	}

	return strings
}

// IntSliceValue extracts a slice of integers from a MappingNode.
func IntSliceValue(value *MappingNode) []int {
	if value == nil || value.Items == nil {
		return []int{}
	}

	ints := make([]int, len(value.Items))
	for i, item := range value.Items {
		ints[i] = IntValue(item)
	}

	return ints
}

// FloatSliceValue extracts a slice of floats from a MappingNode.
func FloatSliceValue(value *MappingNode) []float64 {
	if value == nil || value.Items == nil {
		return []float64{}
	}

	floats := make([]float64, len(value.Items))
	for i, item := range value.Items {
		floats[i] = FloatValue(item)
	}

	return floats
}

// BoolSliceValue extracts a slice of booleans from a MappingNode.
func BoolSliceValue(value *MappingNode) []bool {
	if value == nil || value.Items == nil {
		return []bool{}
	}

	bools := make([]bool, len(value.Items))
	for i, item := range value.Items {
		bools[i] = BoolValue(item)
	}

	return bools
}

// StringMapValue extracts a map of string to string values
// from a MappingNode.
func StringMapValue(value *MappingNode) map[string]string {
	if value == nil || value.Fields == nil {
		return map[string]string{}
	}

	strings := make(map[string]string, len(value.Fields))
	for key, item := range value.Fields {
		strings[key] = StringValue(item)
	}

	return strings
}

// IntMapValue extracts a map of string to int values
// from a MappingNode.
func IntMapValue(value *MappingNode) map[string]int {
	if value == nil || value.Fields == nil {
		return map[string]int{}
	}

	ints := make(map[string]int, len(value.Fields))
	for key, item := range value.Fields {
		ints[key] = IntValue(item)
	}

	return ints
}

// FloatMapValue extracts a map of string to float values
// from a MappingNode.
func FloatMapValue(value *MappingNode) map[string]float64 {
	if value == nil || value.Fields == nil {
		return map[string]float64{}
	}

	floats := make(map[string]float64, len(value.Fields))
	for key, item := range value.Fields {
		floats[key] = FloatValue(item)
	}

	return floats
}

// BoolMapValue extracts a map of string to bool values
// from a MappingNode.
func BoolMapValue(value *MappingNode) map[string]bool {
	if value == nil || value.Fields == nil {
		return map[string]bool{}
	}

	bools := make(map[string]bool, len(value.Fields))
	for key, item := range value.Fields {
		bools[key] = BoolValue(item)
	}

	return bools
}

// MappingNodeFromString creates a MappingNode from a string value.
func MappingNodeFromString(value string) *MappingNode {
	return &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &value,
		},
	}
}

// MappingNodeFromBool creates a MappingNode from a boolean value.
func MappingNodeFromBool(value bool) *MappingNode {
	return &MappingNode{
		Scalar: &ScalarValue{
			BoolValue: &value,
		},
	}
}

// MappingNodeFromInt creates a MappingNode from an integer value.
func MappingNodeFromInt(value int) *MappingNode {
	return &MappingNode{
		Scalar: &ScalarValue{
			IntValue: &value,
		},
	}
}

// MappingNodeFromFloat creates a MappingNode from a float value.
func MappingNodeFromFloat(value float64) *MappingNode {
	return &MappingNode{
		Scalar: &ScalarValue{
			FloatValue: &value,
		},
	}
}

// MappingNodeFromStringSlice creates a MappingNode from a slice of strings.
func MappingNodeFromStringSlice(values []string) *MappingNode {
	items := make([]*MappingNode, len(values))
	for i, value := range values {
		items[i] = MappingNodeFromString(value)
	}

	return &MappingNode{
		Items: items,
	}
}

// MappingNodeFromIntSlice creates a MappingNode from a slice of integers.
func MappingNodeFromIntSlice(values []int64) *MappingNode {
	items := make([]*MappingNode, len(values))
	for i, value := range values {
		items[i] = MappingNodeFromInt(int(value))
	}

	return &MappingNode{
		Items: items,
	}
}

// MappingNodeFromFloatSlice creates a MappingNode from a slice of floats.
func MappingNodeFromFloatSlice(values []float64) *MappingNode {
	items := make([]*MappingNode, len(values))
	for i, value := range values {
		items[i] = MappingNodeFromFloat(value)
	}

	return &MappingNode{
		Items: items,
	}
}

// MappingNodeFromBoolSlice creates a MappingNode from a slice of booleans.
func MappingNodeFromBoolSlice(values []bool) *MappingNode {
	items := make([]*MappingNode, len(values))
	for i, value := range values {
		items[i] = MappingNodeFromBool(value)
	}

	return &MappingNode{
		Items: items,
	}
}

// MappingNodeFromStringMap creates a MappingNode from a map of string keys to string values.
func MappingNodeFromStringMap(values map[string]string) *MappingNode {
	fields := map[string]*MappingNode{}
	for key, value := range values {
		fields[key] = MappingNodeFromString(value)
	}

	return &MappingNode{
		Fields: fields,
	}
}

// MappingNodeFromIntMap creates a MappingNode from a map of string keys to integer values.
func MappingNodeFromIntMap(values map[string]int64) *MappingNode {
	fields := map[string]*MappingNode{}
	for key, value := range values {
		fields[key] = MappingNodeFromInt(int(value))
	}

	return &MappingNode{
		Fields: fields,
	}
}

// MappingNodeFromFloatMap creates a MappingNode from a map of string keys to float values.
func MappingNodeFromFloatMap(values map[string]float64) *MappingNode {
	fields := map[string]*MappingNode{}
	for key, value := range values {
		fields[key] = MappingNodeFromFloat(value)
	}

	return &MappingNode{
		Fields: fields,
	}
}

// MappingNodeFromBoolMap creates a MappingNode from a map of string keys to boolean values.
func MappingNodeFromBoolMap(values map[string]bool) *MappingNode {
	fields := map[string]*MappingNode{}
	for key, value := range values {
		fields[key] = MappingNodeFromBool(value)
	}

	return &MappingNode{
		Fields: fields,
	}
}

// ResourceElementID generates an element ID for a resource that is used
// primarily for resolving substitutions.
func ResourceElementID(resourceName string) string {
	return fmt.Sprintf("resources.%s", resourceName)
}

// ToLogicalResourceName converts a resource element ID to a logical resource name
// (e.g. "resources.resource1" -> "resource1").
func ToLogicalResourceName(resourceElementID string) string {
	return strings.TrimPrefix(resourceElementID, "resources.")
}

// VariableElementID generates an element ID for a variable that is used
// primarily for resolving substitutions.
func VariableElementID(variableName string) string {
	return fmt.Sprintf("variables.%s", variableName)
}

// ValueElementID generates an element ID for a value that is used
// primarily for resolving substitutions.
func ValueElementID(valueName string) string {
	return fmt.Sprintf("values.%s", valueName)
}

// ChildElementID generates an element ID for a child blueprint that is used
// primarily for resolving substitutions.
func ChildElementID(childName string) string {
	return fmt.Sprintf("children.%s", childName)
}

// ToLogicalChildName converts a child element ID to a logical child name
// (e.g. "children.child1" -> "child1").
func ToLogicalChildName(childElementID string) string {
	return strings.TrimPrefix(childElementID, "children.")
}

// DataSourceElementID generates an element ID for a data source that is used
// primarily for resolving substitutions.
func DataSourceElementID(dataSourceName string) string {
	return fmt.Sprintf("datasources.%s", dataSourceName)
}

// ExportElementID generates an element ID for a blueprint export that is used
// primarily for resolving substitutions.
func ExportElementID(dataSourceName string) string {
	return fmt.Sprintf("exports.%s", dataSourceName)
}

// ElementPropertyPath generates a property path for a given element ID and property name.
func ElementPropertyPath(elementID string, propertyName string) string {
	if strings.HasPrefix(propertyName, "[") {
		return fmt.Sprintf("%s%s", elementID, propertyName)
	}
	return fmt.Sprintf("%s.%s", elementID, propertyName)
}

// ExpandedResourceName generates a resource name with an index appended to it
// for resources expanded from a resource template.
func ExpandedResourceName(resourceTemplateName string, index int) string {
	return fmt.Sprintf("%s_%d", resourceTemplateName, index)
}

// LogicalLinkName generates a logical link name for a given pair of resource names
// in the given order.
// (e.g. "resourceA::resourceB").
func LogicalLinkName(resourceAName string, resourceBName string) string {
	return fmt.Sprintf("%s::%s", resourceAName, resourceBName)
}

// LinkType generates a link type identifier for a given pair of resource types
// in the given order.
// (e.g. "aws/lambda/function::aws/dynamodb/table").
func LinkType(resourceTypeA string, resourceTypeB string) string {
	return LogicalLinkName(resourceTypeA, resourceTypeB)
}

// ReplaceSpecWithRoot replaces the "spec." prefix in a path with "$.".
// This is primarily useful for resource specs during merge/overlay operations
// for merging computed fields into a resource spec and overlaying link data mappings
// for drift checking.
func ReplaceSpecWithRoot(path string) string {
	if strings.HasPrefix(path, "spec.") {
		withoutSpec := path[5:]
		return fmt.Sprintf("$.%s", withoutSpec)
	}

	// If the path does not start with "spec.", it could be "spec[".
	// An example of this would be in a path such as "spec[\"ids.v1\"].arns[0]".
	withoutSpec := strings.TrimPrefix(path, "spec")
	return fmt.Sprintf("$%s", withoutSpec)
}

// AddRootToPath adds a root prefix to a path.
// If the path starts with a bracket (e.g. "[\"myField.v1\"]"), it adds a
// "$" prefix to it (e.g. "$[\"myField.v1\"]"). Otherwise, it adds a "$." prefix
// to it (e.g. "$.myField").
func AddRootToPath(path string) string {
	if strings.HasPrefix(path, "[") {
		return fmt.Sprintf("$%s", path)
	}

	return fmt.Sprintf("$.%s", path)
}

// MergeNativeMaps merges multiple maps of string keys to values into a single map.
// Each subsequent map will overwrite the keys of the previous maps.
func MergeNativeMaps[Value any](nativeMaps ...map[string]Value) map[string]Value {
	merged := make(map[string]Value)
	for _, current := range nativeMaps {
		if len(current) > 0 {
			maps.Copy(merged, current)
		}
	}
	return merged
}
