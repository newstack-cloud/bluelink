package substitutions

import "github.com/newstack-cloud/bluelink/libs/blueprint/source"

// StringOrSubstitution represents a value that can be either
// a string or a substitution represented as ${..}
// from user input.
type StringOrSubstitution struct {
	StringValue       *string
	SubstitutionValue *Substitution
	SourceMeta        *source.Meta
}

// Substitution is a representation of a placeholder provided
// with the ${..} syntax.
type Substitution struct {
	Function           *SubstitutionFunctionExpr
	Variable           *SubstitutionVariable
	ValueReference     *SubstitutionValueReference
	ElemReference      *SubstitutionElemReference
	ElemIndexReference *SubstitutionElemIndexReference
	DataSourceProperty *SubstitutionDataSourceProperty
	ResourceProperty   *SubstitutionResourceProperty
	Child              *SubstitutionChild
	StringValue        *string
	IntValue           *int64
	FloatValue         *float64
	BoolValue          *bool
	NoneValue          bool
	SourceMeta         *source.Meta
}

type SubstitutionVariable struct {
	VariableName string
	SourceMeta   *source.Meta
}

type SubstitutionValueReference struct {
	ValueName  string
	Path       []*SubstitutionPathItem
	SourceMeta *source.Meta
}

type SubstitutionElemReference struct {
	Path       []*SubstitutionPathItem
	SourceMeta *source.Meta
}

type SubstitutionElemIndexReference struct {
	SourceMeta *source.Meta
}

type SubstitutionDataSourceProperty struct {
	DataSourceName    string
	FieldName         string
	PrimitiveArrIndex *int64
	SourceMeta        *source.Meta
}

type SubstitutionResourceProperty struct {
	ResourceName              string
	ResourceEachTemplateIndex *int64
	Path                      []*SubstitutionPathItem
	SourceMeta                *source.Meta
}

type SubstitutionPathItem struct {
	FieldName  string
	ArrayIndex *int64
}

type SubstitutionChild struct {
	ChildName  string
	Path       []*SubstitutionPathItem
	SourceMeta *source.Meta
}

type SubstitutionFunctionExpr struct {
	FunctionName SubstitutionFunctionName
	Arguments    []*SubstitutionFunctionArg
	// Path for values accessed from the function result
	// when the return value is an array or object.
	Path       []*SubstitutionPathItem
	SourceMeta *source.Meta
}

type SubstitutionFunctionArg struct {
	Name       string
	Value      *Substitution
	SourceMeta *source.Meta
}

// SubstitutionFunctionName is a type alias reserved for names
// of functions that can be called in a substitution.
type SubstitutionFunctionName string

// ResolvedSubExprType represents a type of a resolved
// substitution expression defined in a blueprint.
// Can be one of "string", "integer", "float", "boolean", "array" or "object".
type ResolvedSubExprType string

const (
	// ResolvedSubExprTypeString is for a substitution
	// expression that resolves to a string.
	ResolvedSubExprTypeString ResolvedSubExprType = "string"
	// ResolvedSubExprTypeInteger is for a substitution
	// expression that resolves to an integer.
	ResolvedSubExprTypeInteger ResolvedSubExprType = "integer"
	// ResolvedSubExprTypeFloat is for a substitution
	// expression that resolves to a float.
	ResolvedSubExprTypeFloat ResolvedSubExprType = "float"
	// ResolvedSubExprTypeBoolean is for a substitution
	// expression that resolves to a boolean.
	ResolvedSubExprTypeBoolean ResolvedSubExprType = "boolean"
	// ResolvedSubExprTypeArray is for a substitution
	// expression that resolves to an array.
	ResolvedSubExprTypeArray ResolvedSubExprType = "array"
	// ResolvedSubExprTypeObject is for a substitution
	// expression that resolves to an object.
	ResolvedSubExprTypeObject ResolvedSubExprType = "object"
	// ResolvedSubExprTypeFunction is for a substitution
	// expression that resolves to a function.
	ResolvedSubExprTypeFunction ResolvedSubExprType = "function"
	// ResolvedSubExprTypeAny is for a substitution
	// expression that resolves to any type.
	ResolvedSubExprTypeAny ResolvedSubExprType = "any"
)
