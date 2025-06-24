package provider

import (
	nativeerrors "errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

const (
	// ErrorReasonCodeItemTypeProviderNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// there being no provider for a specific resource or data source
	// type.
	ErrorReasonCodeItemTypeProviderNotFound errors.ErrorReasonCode = "provider_not_found"
	// ErrorReasonCodeProviderDataSourceTypeNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the data source provider missing an implementation for a
	// specific data source type.
	ErrorReasonCodeProviderDataSourceTypeNotFound errors.ErrorReasonCode = "data_source_type_not_found"
	// ErrorReasonCodeFunctionNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the function not being found in any of the configured providers.
	ErrorReasonCodeFunctionNotFound errors.ErrorReasonCode = "function_not_found"
	// ErrorReasonCodeProviderFunctionNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the function not being found in a specific provider.
	ErrorReasonCodeProviderFunctionNotFound errors.ErrorReasonCode = "provider_function_not_found"
	// ErrorReasonCodeFunctionAlreadyProvided is provided when the
	// reason for a blueprint spec load error is due to
	// the same function being provided by multiple providers.
	ErrorReasonCodeFunctionAlreadyProvided errors.ErrorReasonCode = "function_already_provided"
	// ErrorReasonCodeInvalidResourceSpecDefinition is provided when the
	// reason for a blueprint spec load error is due to
	// an unknown resource spec schema type being used in the schema definition.
	ErrorReasonCodeInvalidResourceSpecDefinition errors.ErrorReasonCode = "invalid_resource_spec_def"
	// ErrorReasonCodeCustomVariableTypeNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the custom variable type not being found in a specific provider.
	ErrorReasonCodeProviderCustomVariableTypeNotFound errors.ErrorReasonCode = "custom_variable_type_not_found"
	// ErrorReasonCodeLinkImplementationNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the link implementation not being found for a specific resource type pair.
	ErrorReasonCodeLinkImplementationNotFound errors.ErrorReasonCode = "link_implementation_not_found"
)

func errDataSourceTypeProviderNotFound(
	providerNamespace string,
	dataSourceType string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeItemTypeProviderNotFound,
		Err: fmt.Errorf(
			"run failed as the provider with namespace %q was not found for data source type %q",
			providerNamespace,
			dataSourceType,
		),
	}
}

func errProviderDataSourceTypeNotFound(
	dataSourceType string,
	providerNamespace string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeProviderDataSourceTypeNotFound,
		Err: fmt.Errorf(
			"run failed as the provider with namespace %q does not have an implementation for data source type %q",
			providerNamespace,
			dataSourceType,
		),
	}
}

func errCustomVariableTypeProviderNotFound(
	providerNamespace string,
	customVariableType string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeItemTypeProviderNotFound,
		Err: fmt.Errorf(
			"run failed as the provider with namespace %q was not found for custom variable type %q",
			providerNamespace,
			customVariableType,
		),
	}
}

func errProviderCustomVariableTypeNotFound(
	customVariableType string,
	providerNamespace string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeProviderCustomVariableTypeNotFound,
		Err: fmt.Errorf(
			"run failed as the provider with namespace %q does not have an implementation for custom variable type %q",
			providerNamespace,
			customVariableType,
		),
	}
}

func errFunctionNotFound(
	functionName string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeFunctionNotFound,
		Err: fmt.Errorf(
			"run failed as the function %q was not found in any of the configured providers",
			functionName,
		),
	}
}

func errFunctionNotFoundInProvider(
	functionName string,
	providerNamespace string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeProviderFunctionNotFound,
		Err: fmt.Errorf(
			"run failed as the function %q implementation was not found in the provider with namespace %q,"+
				" despite the provider listing it as a function it implements",
			functionName,
			providerNamespace,
		),
	}
}

func errFunctionAlreadyProvided(
	functionName string,
	providedBy string,
	provider string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeFunctionAlreadyProvided,
		Err: fmt.Errorf(
			"run failed as the function %q is already provided by the %q provider,"+
				" but is also implemented by the %q provider",
			functionName,
			providedBy,
			provider,
		),
	}
}

func errLinkImplementationNotFound(
	resourceTypeA string,
	resourceTypeB string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeLinkImplementationNotFound,
		Err: fmt.Errorf(
			"link implementation for resource types %q and %q was not found",
			resourceTypeA,
			resourceTypeB,
		),
	}
}

// IsLinkImplementationNotFoundError returns true if an error
// is for the case when a link implementation is not found in the registered
// providers.
func IsLinkImplementationNotFoundError(err error) bool {
	var runErr *errors.RunError
	if nativeerrors.As(err, &runErr) {
		if runErr.ReasonCode == ErrorReasonCodeLinkImplementationNotFound {
			return true
		}
	}

	return false
}

// ErrUnknownResourceDefSchemaType is returned when the schema definition for a resource type
// contains an unknown resource definition schema type.
func ErrUnknownResourceDefSchemaType(
	specType ResourceDefinitionsSchemaType,
	resourceType string,
) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidResourceSpecDefinition,
		Err: fmt.Errorf(
			"validation failed due to an unknown resource definitions schema type %q "+
				"used in the schema definition for resource type %q",
			specType,
			resourceType,
		),
	}
}

// RetryableError is an error that indicates a transient error that can be retried.
// This is a part of the API for provider resources, data sources and custom variable types.
// When a retryable error is returned from a provider resource, data source or custom variable type.
// The operation is retried after a delay based on a configured backoff/retry strategy
// that is configured globally or at the provider level, the framework/host tool will
// provide some reasonable defaults.
//
// The message in ChildError will be used as the failure reason persisted in the state
// when in the context of a resource deployment.
type RetryableError struct {
	// The underlying error for the action that can be retried.
	ChildError error
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error: %s", e.ChildError.Error())
}

// AsRetryableError returns true if the error is a retryable error.
// Whether or not there will be another retry depends on the retry policy
// configured for the provider or globally.
// This will assign the error to the target.
func AsRetryableError(err error, target **RetryableError) bool {
	return nativeerrors.As(err, target)
}

// IsRetryableError returns true if the error is a retryable error.
// Whether or not there will be another retry depends on the retry policy
// configured for the provider or globally.
func IsRetryableError(err error) bool {
	var retryErr *RetryableError
	return nativeerrors.As(err, &retryErr)
}

// ResourceDeployError is an error that indicates a failure to deploy a resource.
// This is a part of the API for provider resources that should be returned when a resource
// fails to deploy, this will cause the operation to fail and the state of the resource will
// be marked as failed with the failure reasons provided.
// This should only be used for errors that can not be retried, the RetryableError
// should be used for transient errors that can be retried.
type ResourceDeployError struct {
	FailureReasons []string
	ChildError     error
}

func (e *ResourceDeployError) Error() string {
	if len(e.FailureReasons) == 1 {
		return fmt.Sprintf("resource deployment failed: %s", e.FailureReasons[0])
	}

	return fmt.Sprintf("resource deployment failed with %d failures", len(e.FailureReasons))
}

func (e *ResourceDeployError) GetFailureReasons() []string {
	return e.FailureReasons
}

// AsResourceDeployError returns true if the error is a resource deploy error
// and assigns the error to the target.
func AsResourceDeployError(err error, target **ResourceDeployError) bool {
	return nativeerrors.As(err, target)
}

// ResourceDestroyError is an error that indicates a failure to destroy a resource.
// This is a part of the API for provider resources that should be returned when an attempt
// to remove a resource fails, this will cause the operation to fail and the state of the resource will
// be marked as failed with the failure reasons provided.
// This should only be used for errors that can not be retried, the RetryableError
// should be used for transient errors that can be retried.
type ResourceDestroyError struct {
	FailureReasons []string
	ChildError     error
}

func (e *ResourceDestroyError) Error() string {
	if len(e.FailureReasons) == 1 {
		return fmt.Sprintf("resource removal failed: %s", e.FailureReasons[0])
	}

	return fmt.Sprintf("resource removal failed with %d failures", len(e.FailureReasons))
}

func (e *ResourceDestroyError) GetFailureReasons() []string {
	return e.FailureReasons
}

// AsResourceDestroyError returns true if the error is a resource destroy error
// and assigns the error to the target.
func AsResourceDestroyError(err error, target **ResourceDestroyError) bool {
	return nativeerrors.As(err, target)
}

// LinkUpdateResourceAError is an error that indicates a failure to update
// resource A in a link relationship.
type LinkUpdateResourceAError struct {
	FailureReasons []string
	ChildError     error
}

func (e *LinkUpdateResourceAError) Error() string {
	if len(e.FailureReasons) == 1 {
		return fmt.Sprintf("link resource A update failed: %s", e.FailureReasons[0])
	}

	return fmt.Sprintf("link resource A update failed with %d failures", len(e.FailureReasons))
}

func (e *LinkUpdateResourceAError) GetFailureReasons() []string {
	return e.FailureReasons
}

// AsLinkUpdateResourceAError returns true if the error is a link update resource A error
// and assigns the error to the target.
func AsLinkUpdateResourceAError(err error, target **LinkUpdateResourceAError) bool {
	return nativeerrors.As(err, target)
}

// LinkUpdateResourceBError is an error that indicates a failure to update
// resource B in a link relationship.
type LinkUpdateResourceBError struct {
	FailureReasons []string
	ChildError     error
}

func (e *LinkUpdateResourceBError) Error() string {
	if len(e.FailureReasons) == 1 {
		return fmt.Sprintf("link resource B update failed: %s", e.FailureReasons[0])
	}

	return fmt.Sprintf("link resource B update failed with %d failures", len(e.FailureReasons))
}

func (e *LinkUpdateResourceBError) GetFailureReasons() []string {
	return e.FailureReasons
}

// AsLinkUpdateResourceBError returns true if the error is a link update resource B error
// and assigns the error to the target.
func AsLinkUpdateResourceBError(err error, target **LinkUpdateResourceBError) bool {
	return nativeerrors.As(err, target)
}

// LinkUpdateIntermediaryResourcesError is an error that indicates a failure to update
// intermediary resources in a link relationship.
type LinkUpdateIntermediaryResourcesError struct {
	FailureReasons []string
	ChildError     error
}

func (e *LinkUpdateIntermediaryResourcesError) Error() string {
	if len(e.FailureReasons) == 1 {
		return fmt.Sprintf("link intermediary resources update failed: %s", e.FailureReasons[0])
	}

	return fmt.Sprintf("link intermediary resources update failed with %d failures", len(e.FailureReasons))
}

func (e *LinkUpdateIntermediaryResourcesError) GetFailureReasons() []string {
	return e.FailureReasons
}

// AsLinkUpdateIntermediaryResourcesError returns true if
// the error is a link update intermediary resources error
// and assigns the error to the target.
func AsLinkUpdateIntermediaryResourcesError(
	err error,
	target **LinkUpdateIntermediaryResourcesError,
) bool {
	return nativeerrors.As(err, target)
}

// BadInputError is an error that indicates an error due to unexpected user input.
// This is primarily used to allow the framework to distinguish between unexpected errors
// and errors that are due to user input.
// This should be used by provider plugin implementations when the error is due to
// user input that can not be recovered from, applications built on top of the blueprint framework
// should ensure that this distinction is relayed to the user in a meaningful way.
type BadInputError struct {
	// The underlying error for that describes the bad input.
	ChildError     error
	FailureReasons []string
}

func (e *BadInputError) Error() string {
	if len(e.FailureReasons) == 1 {
		return fmt.Sprintf("bad input error: %s", e.FailureReasons[0])
	}

	return fmt.Sprintf("bad input provided with %d input errors", len(e.FailureReasons))
}

func (e *BadInputError) GetFailureReasons() []string {
	return e.FailureReasons
}

// AsBadInputError returns true if the error is a bad input error
// and assigns the error to the target.
func AsBadInputError(err error, target **BadInputError) bool {
	return nativeerrors.As(err, target)
}

// ErrorFailureReasons is an interface that should be implemented by errors
// that provide a list of failure reasons.
type ErrorFailureReasons interface {
	GetFailureReasons() []string
}
