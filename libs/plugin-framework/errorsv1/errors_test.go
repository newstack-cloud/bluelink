package errorsv1

import (
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/stretchr/testify/suite"
)

type ErrorsTestSuite struct {
	suite.Suite
}

func (s *ErrorsTestSuite) Test_create_response_from_retryable_error() {
	errorResponse := CreateResponseFromError(
		&provider.RetryableError{
			ChildError: errors.New("A transient error"),
		},
	)
	s.Assert().Equal(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_TRANSIENT,
			Message: "retryable error: A transient error",
		},
		errorResponse,
	)
}

func (s *ErrorsTestSuite) Test_create_response_from_bad_input_error() {
	errorResponse := CreateResponseFromError(
		&provider.BadInputError{
			ChildError: errors.New("Invalid input provided"),
			FailureReasons: []string{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Assert().Equal(
		sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
		errorResponse.Code,
	)
	s.Assert().Equal(
		"bad input provided with 2 input errors",
		errorResponse.Message,
	)
	value, err := pbutils.ConvertPBAnyToInterface(errorResponse.Details)
	s.Require().NoError(err)

	s.Assert().Equal(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
		value,
	)
}

func (s *ErrorsTestSuite) Test_create_retryable_error_from_response() {
	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_TRANSIENT,
			Message: "retryable error: A transient error",
		},
		PluginActionProviderGetConfigDefinition,
	)
	s.Assert().Equal(
		&provider.RetryableError{
			ChildError: &PluginResponseError{
				Code:    sharedtypesv1.ErrorCode_ERROR_CODE_TRANSIENT,
				Action:  PluginActionProviderGetConfigDefinition,
				Message: "retryable error: A transient error",
			},
		},
		goError,
	)
}

func (s *ErrorsTestSuite) Test_create_bad_input_error_from_response() {
	errorDetails, err := pbutils.ConvertInterfaceToProtobuf(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Require().NoError(err)

	failureReasonsStrSlice := []string{
		"Invalid field 'id' in input",
		"Invalid field 'label' in input",
	}

	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
			Message: "Invalid input provided with 2 input errors",
			Details: errorDetails,
		},
		PluginActionProviderGetConfigDefinition,
	)
	s.Assert().Equal(
		&provider.BadInputError{
			ChildError: &PluginResponseError{
				Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
				Action:  PluginActionProviderGetConfigDefinition,
				Message: "Invalid input provided with 2 input errors",
				Details: map[string]any{
					"failureReasons": []any{
						"Invalid field 'id' in input",
						"Invalid field 'label' in input",
					},
				},
			},
			FailureReasons: failureReasonsStrSlice,
		},
		goError,
	)
}

func (s *ErrorsTestSuite) Test_create_resource_deployment_bad_input_error_from_response() {
	errorDetails, err := pbutils.ConvertInterfaceToProtobuf(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Require().NoError(err)

	failureReasonsStrSlice := []string{
		"Invalid field 'id' in input",
		"Invalid field 'label' in input",
	}

	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
			Message: "Invalid input provided with 2 input errors",
			Details: errorDetails,
		},
		PluginActionProviderDeployResource,
	)
	s.Assert().Equal(
		&provider.ResourceDeployError{
			ChildError: &provider.BadInputError{
				ChildError: &PluginResponseError{
					Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
					Action:  PluginActionProviderDeployResource,
					Message: "Invalid input provided with 2 input errors",
					Details: map[string]any{
						"failureReasons": []any{
							"Invalid field 'id' in input",
							"Invalid field 'label' in input",
						},
					},
				},
				FailureReasons: failureReasonsStrSlice,
			},
			FailureReasons: failureReasonsStrSlice,
		},
		goError,
	)
}

func (s *ErrorsTestSuite) Test_create_resource_destruction_bad_input_error_from_response() {
	errorDetails, err := pbutils.ConvertInterfaceToProtobuf(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Require().NoError(err)

	failureReasonsStrSlice := []string{
		"Invalid field 'id' in input",
		"Invalid field 'label' in input",
	}

	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
			Message: "Invalid input provided with 2 input errors",
			Details: errorDetails,
		},
		PluginActionProviderDestroyResource,
	)
	s.Assert().Equal(
		&provider.ResourceDestroyError{
			ChildError: &provider.BadInputError{
				ChildError: &PluginResponseError{
					Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
					Action:  PluginActionProviderDestroyResource,
					Message: "Invalid input provided with 2 input errors",
					Details: map[string]any{
						"failureReasons": []any{
							"Invalid field 'id' in input",
							"Invalid field 'label' in input",
						},
					},
				},
				FailureReasons: failureReasonsStrSlice,
			},
			FailureReasons: failureReasonsStrSlice,
		},
		goError,
	)
}

func (s *ErrorsTestSuite) Test_create_link_resource_a_update_bad_input_error_from_response() {
	errorDetails, err := pbutils.ConvertInterfaceToProtobuf(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Require().NoError(err)

	failureReasonsStrSlice := []string{
		"Invalid field 'id' in input",
		"Invalid field 'label' in input",
	}

	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
			Message: "Invalid input provided with 2 input errors",
			Details: errorDetails,
		},
		PluginActionProviderUpdateLinkResourceA,
	)
	s.Assert().Equal(
		&provider.LinkUpdateResourceAError{
			ChildError: &provider.BadInputError{
				ChildError: &PluginResponseError{
					Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
					Action:  PluginActionProviderUpdateLinkResourceA,
					Message: "Invalid input provided with 2 input errors",
					Details: map[string]any{
						"failureReasons": []any{
							"Invalid field 'id' in input",
							"Invalid field 'label' in input",
						},
					},
				},
				FailureReasons: failureReasonsStrSlice,
			},
			FailureReasons: failureReasonsStrSlice,
		},
		goError,
	)
}

func (s *ErrorsTestSuite) Test_create_link_resource_b_update_bad_input_error_from_response() {
	errorDetails, err := pbutils.ConvertInterfaceToProtobuf(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Require().NoError(err)

	failureReasonsStrSlice := []string{
		"Invalid field 'id' in input",
		"Invalid field 'label' in input",
	}

	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
			Message: "Invalid input provided with 2 input errors",
			Details: errorDetails,
		},
		PluginActionProviderUpdateLinkResourceB,
	)
	s.Assert().Equal(
		&provider.LinkUpdateResourceBError{
			ChildError: &provider.BadInputError{
				ChildError: &PluginResponseError{
					Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
					Action:  PluginActionProviderUpdateLinkResourceB,
					Message: "Invalid input provided with 2 input errors",
					Details: map[string]any{
						"failureReasons": []any{
							"Invalid field 'id' in input",
							"Invalid field 'label' in input",
						},
					},
				},
				FailureReasons: failureReasonsStrSlice,
			},
			FailureReasons: failureReasonsStrSlice,
		},
		goError,
	)
}

func (s *ErrorsTestSuite) Test_create_link_intermediary_resources_update_bad_input_error_from_response() {
	errorDetails, err := pbutils.ConvertInterfaceToProtobuf(
		map[string]any{
			"failureReasons": []any{
				"Invalid field 'id' in input",
				"Invalid field 'label' in input",
			},
		},
	)
	s.Require().NoError(err)

	failureReasonsStrSlice := []string{
		"Invalid field 'id' in input",
		"Invalid field 'label' in input",
	}

	goError := CreateErrorFromResponse(
		&sharedtypesv1.ErrorResponse{
			Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
			Message: "Invalid input provided with 2 input errors",
			Details: errorDetails,
		},
		PluginActionProviderUpdateLinkIntermediaryResources,
	)
	s.Assert().Equal(
		&provider.LinkUpdateIntermediaryResourcesError{
			ChildError: &provider.BadInputError{
				ChildError: &PluginResponseError{
					Code:    sharedtypesv1.ErrorCode_ERROR_CODE_BAD_INPUT,
					Action:  PluginActionProviderUpdateLinkIntermediaryResources,
					Message: "Invalid input provided with 2 input errors",
					Details: map[string]any{
						"failureReasons": []any{
							"Invalid field 'id' in input",
							"Invalid field 'label' in input",
						},
					},
				},
				FailureReasons: failureReasonsStrSlice,
			},
			FailureReasons: failureReasonsStrSlice,
		},
		goError,
	)
}

func TestErrorsTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsTestSuite))
}
