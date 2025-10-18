package function

// Definition describes a function that can be used in a blueprint "${..}" substitution.
// This is used to define the parameters and return types of a function that is used
// to validate arguments passed into a function and the return value of a function.
type Definition struct {
	// Name is the name of the function that is used in a blueprint "${..}" substitution.
	// This must be globally unique.
	Name string
	// Summary is a human-readable summary of the function
	// to be used in documentation and tooling when listing
	// functions.
	Summary string
	// FormattedSummary is a human-readable summary of the function
	// that is formatted with markdown.
	FormattedSummary string
	// Description is a human-readable description of the function.
	Description string
	// FormattedDescription is a human-readable description of the function
	// that is formatted with markdown.
	FormattedDescription string
	// Parameters provides a definition of the parameters that are expected
	// to be passed into the function.
	// The order of the parameters is important as it will be used to match
	// arguments passed into the function.
	Parameters []Parameter
	// Return provides a definition of the return type of the function.
	// Return types are always expected as provider functions are expected to
	// be pure functions that return an output based on the input arguments
	// without side effects.
	// Functions can also return other functions that can be shared,
	// this especially useful for function composition and partial application
	// of functions used in mapping over arrays or similar operations.
	Return Return
	// Internal determines whether or not the function is an internal function
	// that should not be exposed to end-users.
	// Some internal functions are required to enable capabilities such as
	// function composition, piping and a small collection of other higher-order
	// functions.
	Internal bool
}

// Parameter is a parameter type definition for arguments
// passed into a function.
type Parameter interface {
	// GetName returns the name of the parameter for functions
	// that support named arguments.
	GetName() string
	// GetLabel returns the usage name for the parameter.
	GetLabel() string
	// GetType returns the type name of the parameter.
	GetType() ValueType
	// GetDescription returns a human-readable description of the parameter
	// that is not formatted.
	GetDescription() string
	// GetFormattedDescription returns a human-readable description of the parameter
	// that is formatted with markdown.
	GetFormattedDescription() string
	// GetAllowNullValue returns whether or
	// not an argument passed in for this parameter can be null.
	GetAllowNullValue() bool
	// GetOptional determines whether or not the value can be omitted.
	GetOptional() bool
}

// ScalarParameter is a parameter type definition for primitive types.
type ScalarParameter struct {
	// Name is the name of the parameter for functions that support named arguments.
	Name string
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Type of the parameter, an argument will be validated
	// against this type.
	Type ValueTypeDefinition
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for this parameter can be null.
	AllowNullValue bool
	// Optional determines whether or not the value can be omitted.
	Optional bool
}

func (p *ScalarParameter) GetName() string {
	return p.Name
}

func (p *ScalarParameter) GetLabel() string {
	return p.Label
}

func (p *ScalarParameter) GetType() ValueType {
	return p.Type.GetType()
}

func (p *ScalarParameter) GetDescription() string {
	return p.Description
}

func (p *ScalarParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *ScalarParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *ScalarParameter) GetOptional() bool {
	return p.Optional
}

// ListParameter is a parameter type definition for lists of values.
type ListParameter struct {
	// Name is the name of the parameter for functions that support named arguments.
	Name string
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Type of elements in the list, an argument will be validated
	// against this type.
	ElementType ValueTypeDefinition
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for this parameter can be null.
	AllowNullValue bool
	// Optional determines whether or not the value can be omitted.
	Optional bool
}

func (p *ListParameter) GetName() string {
	return p.Name
}

func (p *ListParameter) GetLabel() string {
	return p.Label
}

func (p *ListParameter) GetType() ValueType {
	return ValueTypeList
}

func (p *ListParameter) GetDescription() string {
	return p.Description
}

func (p *ListParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *ListParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *ListParameter) GetOptional() bool {
	return p.Optional
}

// MapParameter is a parameter type definition for a mapping of strings to values.
type MapParameter struct {
	// Name is the name of the parameter for functions that support named arguments.
	Name string
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Type of values in the map, an argument will be validated
	// against this type.
	ElementType ValueTypeDefinition
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for this parameter can be null.
	AllowNullValue bool
	// Optional determines whether or not the value can be omitted.
	Optional bool
}

func (p *MapParameter) GetName() string {
	return p.Name
}

func (p *MapParameter) GetLabel() string {
	return p.Label
}

func (p *MapParameter) GetType() ValueType {
	return ValueTypeMap
}

func (p *MapParameter) GetDescription() string {
	return p.Description
}

func (p *MapParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *MapParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *MapParameter) GetOptional() bool {
	return p.Optional
}

// ObjectParameter is a parameter type definition for a predefined object structure
// with known attributes.
type ObjectParameter struct {
	// Name is the name of the parameter for functions that support named arguments.
	Name string
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// The type of the object that defines the attributes
	// and their types.
	ObjectValueType ValueTypeDefinition
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for this parameter can be null.
	AllowNullValue bool
	// Optional determines whether or not the value can be omitted.
	Optional bool
}

func (p *ObjectParameter) GetName() string {
	return p.Name
}

func (p *ObjectParameter) GetLabel() string {
	return p.Label
}

func (p *ObjectParameter) GetType() ValueType {
	return ValueTypeObject
}

func (p *ObjectParameter) GetDescription() string {
	return p.Description
}

func (p *ObjectParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *ObjectParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *ObjectParameter) GetOptional() bool {
	return p.Optional
}

// FunctionParameter is a parameter type definition for a function that can be passed
// into another function.
type FunctionParameter struct {
	// Name is the name of the parameter for functions that support named arguments.
	Name string
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Type of function that represents the type signature
	// that defines the parameters and return type of the function.
	FunctionType ValueTypeDefinition
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for this parameter can be null.
	AllowNullValue bool
	// Optional determines whether or not the value can be omitted.
	Optional bool
}

func (p *FunctionParameter) GetName() string {
	return p.Name
}

func (p *FunctionParameter) GetLabel() string {
	return p.Label
}

func (p *FunctionParameter) GetType() ValueType {
	return ValueTypeFunction
}

func (p *FunctionParameter) GetDescription() string {
	return p.Description
}

func (p *FunctionParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *FunctionParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *FunctionParameter) GetOptional() bool {
	return p.Optional
}

// VariadicParameter is a parameter type definition for variadic parameters
// at the end of a parameter list.
// A variadic parameter can be any number of arguments of any or a specific type.
type VariadicParameter struct {
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Type of the parameters, each argument will be validated
	// against this type.
	Type ValueTypeDefinition
	// SingleType determines whether or not the variadic parameters
	// must all be of the same type.
	// This is false by default, meaning that variadic parameters
	// can be of any type.
	// The Type field is only assessed if SingleType is true.
	SingleType bool
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for these parameters can be null.
	AllowNullValue bool
	// Named determines whether or not the variadic parameters
	// are named arguments.
	// Named and positional arguments cannot be mixed.
	// If they are named arguments, all arguments do not need to be
	// of the same type.
	Named bool
}

func (p *VariadicParameter) GetName() string {
	return ""
}

func (p *VariadicParameter) GetLabel() string {
	return p.Label
}

func (p *VariadicParameter) GetType() ValueType {
	return ValueTypeList
}

func (p *VariadicParameter) GetDescription() string {
	return p.Description
}

func (p *VariadicParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *VariadicParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *VariadicParameter) GetOptional() bool {
	return true
}

// AnyParameter is a parameter type definition for any value.
// This can be used for union types as well as parameters that
// can accept any type.
type AnyParameter struct {
	// Name is the name of the parameter for functions that support named arguments.
	Name string
	// Label is the usage name for the parameter.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// UnionTypes is a list of value type definitions that are allowed
	// for the parameter.
	// When provided, an any parameter type is expected to be validated
	// as a union type where the argument must match one of the types.
	UnionTypes []ValueTypeDefinition
	// Description is a human-readable description of
	// the parameter. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the parameter that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// AllowNullValue determines whether or not an argument
	// passed in for this parameter can be null.
	AllowNullValue bool
	// Optional determines whether or not the value can be omitted.
	Optional bool
}

func (p *AnyParameter) GetName() string {
	return p.Name
}

func (p *AnyParameter) GetLabel() string {
	return p.Label
}

func (p *AnyParameter) GetType() ValueType {
	return ValueTypeAny
}

func (p *AnyParameter) GetDescription() string {
	return p.Description
}

func (p *AnyParameter) GetFormattedDescription() string {
	return p.FormattedDescription
}

func (p *AnyParameter) GetAllowNullValue() bool {
	return p.AllowNullValue
}

func (p *AnyParameter) GetOptional() bool {
	return p.Optional
}

// Return is a return type definition for the return value of a function.
type Return interface {
	// GetType retrieves the type name of the return value.
	GetType() ValueType
	// GetDescription returns a human-readable description of the return value
	// that is not formatted.
	GetDescription() string
	// GetFormattedDescription returns a human-readable description of the return value
	// that is formatted with markdown.
	GetFormattedDescription() string
}

// ScalarReturn defines a return type for a primitive type.
type ScalarReturn struct {
	// This is the type definition for the scalar return value,
	// this should be a type definition that uses one of the scalar
	// value types such as ValueTypeString, ValueTypeInt32, etc.
	Type ValueTypeDefinition
	// Description is a human-readable description of
	// the return value. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the return value that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (r *ScalarReturn) GetType() ValueType {
	return r.Type.GetType()
}

func (r *ScalarReturn) GetDescription() string {
	return r.Description
}

func (r *ScalarReturn) GetFormattedDescription() string {
	return r.FormattedDescription
}

// ListReturn defines a return type for a list of values
// with a single type.
type ListReturn struct {
	// ElementType is the type definition for the elements in the list.
	ElementType ValueTypeDefinition
	// Description is a human-readable description of
	// the return value. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the return value that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (r *ListReturn) GetType() ValueType {
	return ValueTypeList
}

func (r *ListReturn) GetDescription() string {
	return r.Description
}

func (r *ListReturn) GetFormattedDescription() string {
	return r.FormattedDescription
}

// MapReturn defines a return type for a mapping of strings to values
// with a single type.
type MapReturn struct {
	// ElementType is the type definition for the values in the map.
	ElementType ValueTypeDefinition
	// Description is a human-readable description of
	// the return value. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the return value that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (r *MapReturn) GetType() ValueType {
	return ValueTypeMap
}

func (r *MapReturn) GetDescription() string {
	return r.Description
}

func (r *MapReturn) GetFormattedDescription() string {
	return r.FormattedDescription
}

// ObjectReturn defines a return type for a predefined object structure
// with known attributes.
type ObjectReturn struct {
	// The type of the object that defines the attributes
	// and their types.
	ObjectValueType ValueTypeDefinition
	// Description is a human-readable description of
	// the return value. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the return value that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (r *ObjectReturn) GetType() ValueType {
	return ValueTypeObject
}

func (r *ObjectReturn) GetDescription() string {
	return r.Description
}

func (r *ObjectReturn) GetFormattedDescription() string {
	return r.FormattedDescription
}

// FunctionReturn defines a return type for a function that can be returned
// from another function.
type FunctionReturn struct {
	// FunctionType is the type definition for the function that is returned.
	// This should be a type definition of the function signature
	// that defines the parameters and return type of the function.
	FunctionType ValueTypeDefinition
	// Description is a human-readable description of
	// the return value. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the return value that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (r *FunctionReturn) GetType() ValueType {
	return ValueTypeFunction
}

func (r *FunctionReturn) GetDescription() string {
	return r.Description
}

func (r *FunctionReturn) GetFormattedDescription() string {
	return r.FormattedDescription
}

// AnyReturn defines a return type that allows any value type.
type AnyReturn struct {
	// This is the type definition for a return value that can be any type.
	Type ValueType
	// UnionTypes is a list of value type definitions that are allowed
	// for the return value.
	// When provided, an any return type will be validated as a union type
	// where the return value must match one of the types in the union.
	UnionTypes []ValueTypeDefinition
	// Description is a human-readable description of
	// the return value. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the return value that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (r *AnyReturn) GetType() ValueType {
	return r.Type
}

func (r *AnyReturn) GetDescription() string {
	return r.Description
}

func (r *AnyReturn) GetFormattedDescription() string {
	return r.FormattedDescription
}

// AttributeType provides a wrapper around a value type definition
// that allows specific attributes of an object to be null.
type AttributeType struct {
	// ValueTypeDefinition is the type definition for the attribute.
	Type ValueTypeDefinition
	// AllowNullValue determines whether or not an attribute
	// of an object in a parameter or return type can be null.
	AllowNullValue bool
}

// ValueTypeDefinition is an interface that provides a common
// interface for all value type definitions that can be used
// in parameter and return type definitions.
type ValueTypeDefinition interface {
	// GetType returns the type name of the value type definition.
	GetType() ValueType
	// GetLabel returns the name of the value type definition.
	GetLabel() string
	// GetDescription returns a human-readable description
	// of the value type definition.
	GetDescription() string
	// GetFormattedDescription returns a human-readable description
	// of the value type definition that is formatted with markdown.
	GetFormattedDescription() string
}

// ValueTypeDefinitionScalar is a value type definition
// for scalar (primitive) types.
type ValueTypeDefinitionScalar struct {
	// Type is the value type name for the scalar type.
	Type ValueType
	// Label is the usage name for the value type.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Description is a human-readable description of
	// the value type. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the value type that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// StringChoices is a list of string values that are allowed
	// for the value.
	// This is only taken into account when typeDef.Type is ValueTypeString.
	StringChoices []string
}

func (v *ValueTypeDefinitionScalar) GetType() ValueType {
	return v.Type
}

func (v *ValueTypeDefinitionScalar) GetLabel() string {
	return v.Label
}

func (v *ValueTypeDefinitionScalar) GetDescription() string {
	return v.Description
}

func (v *ValueTypeDefinitionScalar) GetFormattedDescription() string {
	return v.FormattedDescription
}

// ValueTypeDefinitionList is a value type definition
// for lists of values.
type ValueTypeDefinitionList struct {
	// ElementType is the type definition for the elements in the list.
	ElementType ValueTypeDefinition
	// Label is the usage name for the value type.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Description is a human-readable description of
	// the value type. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the value type that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (v *ValueTypeDefinitionList) GetType() ValueType {
	return ValueTypeList
}

func (v *ValueTypeDefinitionList) GetLabel() string {
	return v.Label
}

func (v *ValueTypeDefinitionList) GetDescription() string {
	return v.Description
}

func (v *ValueTypeDefinitionList) GetFormattedDescription() string {
	return v.FormattedDescription
}

// ValueTypeDefinitionMap is a value type definition
// for a mapping of strings to values.
type ValueTypeDefinitionMap struct {
	// ElementType is the type definition for the values in the map.
	ElementType ValueTypeDefinition
	// Label is the usage name for the value type.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Description is a human-readable description of
	// the value type. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the value type that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (v *ValueTypeDefinitionMap) GetType() ValueType {
	return ValueTypeMap
}

func (v *ValueTypeDefinitionMap) GetLabel() string {
	return v.Label
}

func (v *ValueTypeDefinitionMap) GetDescription() string {
	return v.Description
}

func (v *ValueTypeDefinitionMap) GetFormattedDescription() string {
	return v.FormattedDescription
}

// ValueTypeDefinitionObject is a value type definition
// for a predefined object structure with known attributes.
type ValueTypeDefinitionObject struct {
	// AttributeTypes is a map of attribute names to attribute types.
	AttributeTypes map[string]AttributeType
	// Label is the usage name for the value type.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Description is a human-readable description of
	// the value type. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the value type that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
	// Required is a list of attribute names that are required
	// for the object.
	Required []string
}

func (v *ValueTypeDefinitionObject) GetType() ValueType {
	return ValueTypeObject
}

func (v *ValueTypeDefinitionObject) GetLabel() string {
	return v.Label
}

func (v *ValueTypeDefinitionObject) GetDescription() string {
	return v.Description
}

func (v *ValueTypeDefinitionObject) GetFormattedDescription() string {
	return v.FormattedDescription
}

// ValueTypeDefinitionFunction is a value type definition
// for a function that can be passed into and returned from
// other functions.
type ValueTypeDefinitionFunction struct {
	// Definition is the function definition that describes the
	// parameters and return type of an anonymous function
	// that can be passed into and returned from other functions.
	Definition Definition
	// Label is the usage name for the value type.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Description is a human-readable description of
	// the value type. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the value type that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (v *ValueTypeDefinitionFunction) GetType() ValueType {
	return ValueTypeFunction
}

func (v *ValueTypeDefinitionFunction) GetLabel() string {
	return v.Label
}

func (v *ValueTypeDefinitionFunction) GetDescription() string {
	return v.Description
}

func (v *ValueTypeDefinitionFunction) GetFormattedDescription() string {
	return v.FormattedDescription
}

// ValueTypeDefinitionAny is a value type definition
// for an argument or return value that can have any type.
type ValueTypeDefinitionAny struct {
	// Type is the value type name for an any type.
	Type ValueType
	// UnionTypes is a list of value type definitions that are allowed
	// for the value.
	// When provided, an any value type will be validated as a union type
	// where the return value must match one of the types in the union.
	UnionTypes []ValueTypeDefinition
	// Label is the usage name for the value type.
	// This will appear in logs and in tooling such as
	// the language server.
	Label string
	// Description is a human-readable description of
	// the value type. This will appear in logs and in
	// tooling such as the language server.
	Description string
	// FormattedDescription is a human-readable description of
	// the value type that is formatted with markdown.
	// This will appear in usage documentation, logs and in
	// tooling such as the language server.
	FormattedDescription string
}

func (v *ValueTypeDefinitionAny) GetType() ValueType {
	return ValueTypeAny
}

func (v *ValueTypeDefinitionAny) GetLabel() string {
	return v.Label
}

func (v *ValueTypeDefinitionAny) GetDescription() string {
	return v.Description
}

func (v *ValueTypeDefinitionAny) GetFormattedDescription() string {
	return v.FormattedDescription
}

// ValueType is used as an enum for the value types that
// are supported for parameters and return types for
// provider functions.
type ValueType string

const (
	// ValueTypeString is for strings.
	ValueTypeString ValueType = "string"

	// ValueTypeBytes is for byte arrays.
	ValueTypeBytes ValueType = "bytes"

	// ValueTypeInt32 is for 32-bit signed integers.
	ValueTypeInt32 ValueType = "int32"

	// ValueTypeInt64 is for 64-bit signed integers.
	ValueTypeInt64 ValueType = "int64"

	// ValueTypeUint32 is for 32-bit unsigned integers.
	ValueTypeUint32 ValueType = "uint32"

	// ValueTypeUint64 is for 64-bit unsigned integers.
	ValueTypeUint64 ValueType = "uint64"

	// ValueTypeFloat32 is for 32-bit floating point numbers.
	ValueTypeFloat32 ValueType = "float32"

	// ValueTypeFloat64 is for 64-bit floating point numbers.
	ValueTypeFloat64 ValueType = "float64"

	// ValueTypeBool is for boolean values.
	ValueTypeBool ValueType = "bool"

	// ValueTypeList is for lists of values.
	ValueTypeList ValueType = "list"

	// ValueTypeMap is for maps of values.
	ValueTypeMap ValueType = "map"

	// ValueTypeObject is for objects with a predefined
	// structure with known attributes.
	ValueTypeObject ValueType = "object"

	// ValueTypeFunction is for functions that can be passed
	// into and returned from other functions.
	ValueTypeFunction ValueType = "function"

	// ValueTypeAny is for a parameter or return value
	// that can have any type.
	ValueTypeAny ValueType = "any"
)
