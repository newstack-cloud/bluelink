package errors

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type ErrorReasonCode string

// ErrorContext provides structured information for error resolution
type ErrorContext struct {
	// Category helps clients group and handle errors
	Category ErrorCategory `json:"category,omitempty"`
	// Reason code from the parent error for programmatic identification
	ReasonCode ErrorReasonCode `json:"reasonCode,omitempty"`
	// Suggested actions the user can take
	SuggestedActions []SuggestedAction `json:"suggestedActions,omitempty"`
	// Additional metadata for context
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ErrorCategory string

const (
	ErrorCategoryProvider       ErrorCategory = "provider"
	ErrorCategoryTransformer    ErrorCategory = "transformer"
	ErrorCategoryFunction       ErrorCategory = "function"
	ErrorCategoryResourceType   ErrorCategory = "resource_type"
	ErrorCategoryVariableType   ErrorCategory = "variable_type"
	ErrorCategoryDataSourceType ErrorCategory = "data_source_type"
	ErrorCategoryExport         ErrorCategory = "export"
)

type SuggestedAction struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority,omitempty"` // 1=highest, 5=lowest
}

// ActionType constants for programmatic identification
type ActionType string

const (
	ActionTypeInstallProvider                ActionType = "install_provider"
	ActionTypeUpdateProvider                 ActionType = "update_provider"
	ActionTypeCheckConfiguration             ActionType = "check_configuration"
	ActionTypeCheckFunctionName              ActionType = "check_function_name"
	ActionTypeCheckResourceType              ActionType = "check_resource_type"
	ActionTypeCheckDataSourceType            ActionType = "check_data_source_type"
	ActionTypeProvideValue                   ActionType = "provide_value"
	ActionTypeAddDefaultValue                ActionType = "add_default_value"
	ActionTypeFixVariableType                ActionType = "fix_variable_type"
	ActionTypeCheckVariableType              ActionType = "check_variable_type"
	ActionTypeContactVariableTypeDeveloper   ActionType = "contact_variable_type_developer"
	ActionTypeAddVariableType                ActionType = "add_variable_type"
	ActionTypeCheckCustomVariableOptions     ActionType = "check_custom_variable_options"
	ActionTypeAddResourceType                ActionType = "add_resource_type"
	ActionTypeContactResourceTypeDeveloper   ActionType = "contact_resource_type_developer"
	ActionTypeCheckAbstractResourceType      ActionType = "check_abstract_resource_type"
	ActionTypeInstallTransformers            ActionType = "install_transformers"
	ActionTypeInstallTransformer             ActionType = "install_transformer"
	ActionTypeCheckTransformers              ActionType = "check_transformers"
	ActionTypeCheckResourceTypeSchema        ActionType = "check_resource_type_schema"
	ActionTypeAddDataSourceFilter            ActionType = "add_data_source_filter"
	ActionTypeAddDataSourceExport            ActionType = "add_data_source_export"
	ActionTypeContactDataSourceTypeDeveloper ActionType = "contact_data_source_type_developer"
	ActionTypeCheckDataSourceFilterFields    ActionType = "check_data_source_filter_fields"
	ActionTypeAddDataSourceType              ActionType = "add_data_source_type"
)

// ErrorReasonCodeAnyTypeWarning is used to tag warning diagnostics
// where a substitution resolves to the "any" type.
const ErrorReasonCodeAnyTypeWarning ErrorReasonCode = "any_type_warning"

type LoadError struct {
	ReasonCode     ErrorReasonCode
	Err            error
	ChildErrors    []error
	Line           *int
	Column         *int
	EndLine        *int
	EndColumn      *int
	ColumnAccuracy *source.ColumnAccuracy
	Context        *ErrorContext `json:"context,omitempty"`
}

func (e *LoadError) Error() string {
	childErrCount := len(e.ChildErrors)
	if childErrCount == 0 {
		return fmt.Sprintf("blueprint load error: %s", e.Err.Error())
	}
	errorsLabel := deriveErrorsLabel(childErrCount)
	return fmt.Sprintf("blueprint load error (%d child %s): %s", childErrCount, errorsLabel, e.Err.Error())
}

func deriveErrorsLabel(errorCount int) string {
	if errorCount == 1 {
		return "error"
	}

	return "errors"
}

type RunError struct {
	ReasonCode  ErrorReasonCode
	Err         error
	ChildErrors []error
	// ChildBlueprintPath is the path to the child blueprint that caused the error.
	// This should be in the following format:
	// "include.<childName>::include.<grandChildName>::..."
	// Rendered as "include.<childName> -> include.<grandChildName> -> ..."
	//
	// This is useful for distinguishing between errors that occur in the parent blueprint
	// and errors that occur in a child blueprint.
	ChildBlueprintPath string
	Context            *ErrorContext `json:"context,omitempty"`
}

func (e *RunError) Error() string {
	childBlueprintPathInfo := renderChildBlueprintPathInfo(e.ChildBlueprintPath)
	childErrCount := len(e.ChildErrors)
	if childErrCount == 0 {
		return fmt.Sprintf("run error%s: %s", childBlueprintPathInfo, e.Err.Error())
	}
	errorsLabel := deriveErrorsLabel(childErrCount)

	return fmt.Sprintf(
		"run error (%d child %s)%s: %s",
		childErrCount,
		errorsLabel,
		childBlueprintPathInfo,
		e.Err.Error(),
	)
}

func renderChildBlueprintPathInfo(childBlueprintPath string) string {
	if childBlueprintPath == "" {
		return ""
	}

	includes := strings.Split(childBlueprintPath, "::")
	displayPath := strings.Join(includes, " -> ")

	return fmt.Sprintf(" (child blueprint path: %s)", displayPath)
}

type SerialiseError struct {
	ReasonCode  ErrorReasonCode
	Err         error
	ChildErrors []error
}

func (e *SerialiseError) Error() string {
	childErrCount := len(e.ChildErrors)
	if childErrCount == 0 {
		return fmt.Sprintf("blueprint serialise error: %s", e.Err.Error())
	}
	errorsLabel := deriveErrorsLabel(childErrCount)
	return fmt.Sprintf("blueprint serialise error (%d child %s): %s", childErrCount, errorsLabel, e.Err.Error())
}

type ExpandedSerialiseError struct {
	ReasonCode  ErrorReasonCode
	Err         error
	ChildErrors []error
}

func (e *ExpandedSerialiseError) Error() string {
	childErrCount := len(e.ChildErrors)
	if childErrCount == 0 {
		return fmt.Sprintf("expanded blueprint serialise error: %s", e.Err.Error())
	}
	errorsLabel := deriveErrorsLabel(childErrCount)
	return fmt.Sprintf("expanded blueprint serialise error (%d child %s): %s", childErrCount, errorsLabel, e.Err.Error())
}
