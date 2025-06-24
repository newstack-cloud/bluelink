package errorsv1

import (
	"errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

// PluginResponseError represents a user-defined error that has ben returned
// by a plugin.
// This is to be used client-side (deploy engine host side) to convert
// error response messages from gRPC to a native error type.
type PluginResponseError struct {
	Code    sharedtypesv1.ErrorCode
	Action  PluginAction
	Message string
	Details any
}

func (e *PluginResponseError) Error() string {
	return fmt.Sprintf("plugin response error: %s", e.Message)
}

// CreateResponseFromError creates a protobuf error response message
// from an error.
// This is to be used server-side (plugin side) to convert errors to messages
// that can be sent over gRPC to the client.
func CreateResponseFromError(inputError error) *sharedtypesv1.ErrorResponse {
	errorResponse := &sharedtypesv1.ErrorResponse{
		Code:    sharedtypesv1.ErrorCode_ERROR_CODE_UNEXPECTED,
		Message: inputError.Error(),
	}

	var retryableError *provider.RetryableError
	var badInputError *provider.BadInputError
	if errors.As(inputError, &retryableError) {
		errorResponse.Code = sharedtypesv1.ErrorCode_ERROR_CODE_TRANSIENT
	} else if errors.As(inputError, &badInputError) {
		errorResponse.Code = sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT
	}

	attachDetailsToErrorResponse(inputError, badInputError, errorResponse)

	return errorResponse
}

func attachDetailsToErrorResponse(
	inputErr error,
	badInputErr *provider.BadInputError,
	errorResponse *sharedtypesv1.ErrorResponse,
) {
	failureReasons := []string{}
	if badInputErr != nil {
		failureReasons = badInputErr.FailureReasons
	}

	errorWithFailureReasons, ok := inputErr.(provider.ErrorFailureReasons)
	if ok {
		failureReasons = errorWithFailureReasons.GetFailureReasons()
	}

	if len(failureReasons) == 0 {
		return
	}

	details := map[string]any{
		"failureReasons": failureReasons,
	}
	// We'll ignore the error for details conversion,
	// worst case scenario is that details has a nil value in the case that
	// it could not be marshalled to a protobuf message.
	pbDetails, _ := pbutils.ConvertInterfaceToProtobuf(details)
	errorResponse.Details = pbDetails
}

// CreateErrorFromResponse creates a native error from a protobuf error response.
// This is to be used client-side (deploy engine host side) to convert
// error response messages from gRPC to a native error type.
func CreateErrorFromResponse(errorResponse *sharedtypesv1.ErrorResponse, action PluginAction) error {
	// We'll ignore the error for details conversion.
	// Returning errors in error creation can get a bit confusing,
	// the worst case scenario in this situation is that details has a nil
	// value in the case that it could not be unmarshalled from the protobuf
	// message.
	details, _ := pbutils.ConvertPBAnyToInterface(errorResponse.Details)

	if errorResponse.Code == sharedtypesv1.ErrorCode_ERROR_CODE_TRANSIENT {
		return &provider.RetryableError{
			ChildError: createPluginResponseError(errorResponse, action, details),
		}
	}

	if errorResponse.Code == sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT {
		badInputErr := &provider.BadInputError{
			ChildError:     createPluginResponseError(errorResponse, action, details),
			FailureReasons: failureReasonsFromErrorResponse(errorResponse, details),
		}
		// For deployment actions, `BadInputError` is not treated in a special way
		// like retry errors are.
		// Instead, we'll wrap the error in the appropriate deployment action error type
		// to ensure that the blueprint framework can handle the error correctly.
		if isDeploymentAction(action) {
			return createDeploymentErrorFromBadInput(badInputErr, action)
		}
		return badInputErr
	}

	return createGeneralErrorFromResponse(errorResponse, action, details)
}

func createGeneralErrorFromResponse(
	errorResponse *sharedtypesv1.ErrorResponse,
	action PluginAction,
	details any,
) error {
	switch action {
	case PluginActionProviderDeployResource:
		return &provider.ResourceDeployError{
			ChildError:     createPluginResponseError(errorResponse, action, details),
			FailureReasons: failureReasonsFromErrorResponse(errorResponse, details),
		}
	case PluginActionProviderDestroyResource:
		return &provider.ResourceDestroyError{
			ChildError:     createPluginResponseError(errorResponse, action, details),
			FailureReasons: failureReasonsFromErrorResponse(errorResponse, details),
		}
	case PluginActionProviderUpdateLinkResourceA:
		return &provider.LinkUpdateResourceAError{
			ChildError:     createPluginResponseError(errorResponse, action, details),
			FailureReasons: failureReasonsFromErrorResponse(errorResponse, details),
		}
	case PluginActionProviderUpdateLinkResourceB:
		return &provider.LinkUpdateResourceBError{
			ChildError:     createPluginResponseError(errorResponse, action, details),
			FailureReasons: failureReasonsFromErrorResponse(errorResponse, details),
		}
	case PluginActionProviderUpdateLinkIntermediaryResources:
		return &provider.LinkUpdateIntermediaryResourcesError{
			ChildError:     createPluginResponseError(errorResponse, action, details),
			FailureReasons: failureReasonsFromErrorResponse(errorResponse, details),
		}
	}

	return createPluginResponseError(errorResponse, action, details)
}

func createPluginResponseError(
	errorResponse *sharedtypesv1.ErrorResponse,
	action PluginAction,
	details any,
) *PluginResponseError {
	return &PluginResponseError{
		Code:    errorResponse.Code,
		Action:  action,
		Message: errorResponse.Message,
		Details: details,
	}
}

// CreateGeneralError creates a general error for a plugin response for errors that are not derived
// from a plugin response.
// This ensures that errors are wrapped in the correct blueprint framework error
// type for actions that the blueprint framework provides error types for.
func CreateGeneralError(
	err error,
	action PluginAction,
) error {
	switch action {
	case PluginActionProviderDeployResource:
		return &provider.ResourceDeployError{
			ChildError:     err,
			FailureReasons: []string{err.Error()},
		}
	case PluginActionProviderDestroyResource:
		return &provider.ResourceDestroyError{
			ChildError:     err,
			FailureReasons: []string{err.Error()},
		}
	case PluginActionProviderUpdateLinkResourceA:
		return &provider.LinkUpdateResourceAError{
			ChildError:     err,
			FailureReasons: []string{err.Error()},
		}
	case PluginActionProviderUpdateLinkResourceB:
		return &provider.LinkUpdateResourceBError{
			ChildError:     err,
			FailureReasons: []string{err.Error()},
		}
	case PluginActionProviderUpdateLinkIntermediaryResources:
		return &provider.LinkUpdateIntermediaryResourcesError{
			ChildError:     err,
			FailureReasons: []string{err.Error()},
		}
	}

	return err
}

func isDeploymentAction(action PluginAction) bool {
	return action == PluginActionProviderDeployResource ||
		action == PluginActionProviderDestroyResource ||
		action == PluginActionProviderUpdateLinkResourceA ||
		action == PluginActionProviderUpdateLinkResourceB ||
		action == PluginActionProviderUpdateLinkIntermediaryResources
}

func createDeploymentErrorFromBadInput(
	badInputErr *provider.BadInputError,
	action PluginAction,
) error {
	switch action {
	case PluginActionProviderDeployResource:
		return &provider.ResourceDeployError{
			ChildError:     badInputErr,
			FailureReasons: badInputErr.FailureReasons,
		}
	case PluginActionProviderDestroyResource:
		return &provider.ResourceDestroyError{
			ChildError:     badInputErr,
			FailureReasons: badInputErr.FailureReasons,
		}
	case PluginActionProviderUpdateLinkResourceA:
		return &provider.LinkUpdateResourceAError{
			ChildError:     badInputErr,
			FailureReasons: badInputErr.FailureReasons,
		}
	case PluginActionProviderUpdateLinkResourceB:
		return &provider.LinkUpdateResourceBError{
			ChildError:     badInputErr,
			FailureReasons: badInputErr.FailureReasons,
		}
	case PluginActionProviderUpdateLinkIntermediaryResources:
		return &provider.LinkUpdateIntermediaryResourcesError{
			ChildError:     badInputErr,
			FailureReasons: badInputErr.FailureReasons,
		}
	}

	return badInputErr
}

func failureReasonsFromErrorResponse(
	errorResponse *sharedtypesv1.ErrorResponse,
	details any,
) []string {
	// Failure reasons are extracted from details if it is a map[string]any
	// and contains a key "failureReasons".
	if detailsMap, isMap := details.(map[string]any); isMap {
		failureReasons, hasFailureReasons := detailsMap["failureReasons"].([]any)
		if hasFailureReasons {
			reasons := make([]string, len(failureReasons))
			for i, reason := range failureReasons {
				reasons[i] = fmt.Sprint(reason)
			}
			return reasons
		}
	}

	return []string{errorResponse.Message}
}

// ErrUnexpectedResponseType is returned when an unexpected response type is returned
// for a plugin action.
func ErrUnexpectedResponseType(action PluginAction) error {
	return fmt.Errorf("unexpected response type returned for action %s", action)
}

// ErrResourceNotDestroyed is returned when a resource is not destroyed.
func ErrResourceNotDestroyed(resourceType string, action PluginAction) error {
	return fmt.Errorf(
		"resource of type %q was not destroyed for action %s, see deployment"+
			" state or application logs for more information",
		resourceType,
		action,
	)
}

// ErrInvalidHostID is returned when an invalid host ID is provided in a request.
// An invalid host ID indicates that a host making calls to a plugin is not the one
// that the plugin was registered with.
func ErrInvalidHostID(hostID string) error {
	return fmt.Errorf("invalid host ID %q", hostID)
}
