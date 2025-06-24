package testutils

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/stretchr/testify/suite"
)

// AssertConfigDefinitionEquals asserts that two core config definitions
// are equal.
// This treats nil and empty slices in the config field definitions
// as equal.
func AssertConfigDefinitionEquals(
	expected *core.ConfigDefinition,
	actual *core.ConfigDefinition,
	testSuite *suite.Suite,
) {
	for key, expectedField := range expected.Fields {
		actualField, ok := actual.Fields[key]
		testSuite.Assert().True(ok)
		testSuite.Assert().Equal(expectedField.Type, actualField.Type)
		testSuite.Assert().Equal(expectedField.Label, actualField.Label)
		testSuite.Assert().Equal(expectedField.Description, actualField.Description)
		testSuite.Assert().Equal(expectedField.Required, actualField.Required)
		testSuite.Assert().Equal(expectedField.DefaultValue, actualField.DefaultValue)
		AssertSlicesEqual(expectedField.Examples, actualField.Examples, testSuite)
		AssertSlicesEqual(expectedField.AllowedValues, actualField.AllowedValues, testSuite)
	}
}

// AssertLinkChangesEquals asserts that two provider link changes
// are equal.
// This treats nil and empty slices in the changes as equal.
func AssertLinkChangesEquals(
	expected *provider.LinkChanges,
	actual *provider.LinkChanges,
	testSuite *suite.Suite,
) {
	AssertSlicesEqual(expected.ModifiedFields, actual.ModifiedFields, testSuite)
	AssertSlicesEqual(expected.NewFields, actual.NewFields, testSuite)
	AssertSlicesEqual(expected.RemovedFields, actual.RemovedFields, testSuite)
	AssertSlicesEqual(expected.UnchangedFields, actual.UnchangedFields, testSuite)
	AssertSlicesEqual(expected.FieldChangesKnownOnDeploy, actual.FieldChangesKnownOnDeploy, testSuite)
}

// AssertInvalidHost asserts that the given error is an invalid host error
// from a plugin method call response.
func AssertInvalidHost(
	respErr error,
	action errorsv1.PluginAction,
	invalidHostID string,
	testSuite *suite.Suite,
) {
	testSuite.Require().Error(respErr)
	pluginRespErr := assertExtractPluginError(respErr, action, testSuite)
	testSuite.Assert().Equal(
		action,
		pluginRespErr.Action,
	)
	testSuite.Assert().Equal(
		sharedtypesv1.ErrorCode_ERROR_CODE_UNEXPECTED,
		pluginRespErr.Code,
	)
	testSuite.Assert().Equal(
		fmt.Sprintf("invalid host ID %q", invalidHostID),
		pluginRespErr.Message,
	)
}

func assertExtractPluginError(
	err error,
	action errorsv1.PluginAction,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	switch action {
	case errorsv1.PluginActionProviderDeployResource:
		return assertExtractDeployResourceError(err, testSuite)
	case errorsv1.PluginActionProviderDestroyResource:
		return assertExtractDestroyResourceError(err, testSuite)
	case errorsv1.PluginActionProviderUpdateLinkResourceA:
		return assertExtractUpdateResourceAError(err, testSuite)
	case errorsv1.PluginActionProviderUpdateLinkResourceB:
		return assertExtractUpdateResourceBError(err, testSuite)
	case errorsv1.PluginActionProviderUpdateLinkIntermediaryResources:
		return assertExtractUpdateIntermediaryResourcesError(err, testSuite)
	default:
		return assertExtractPluginResponseError(err, testSuite)
	}
}

func assertExtractDeployResourceError(
	err error,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	deployErr, isDeployErr := err.(*provider.ResourceDeployError)
	testSuite.Require().True(isDeployErr)
	testSuite.Require().NotNil(deployErr)

	return assertExtractPluginResponseError(
		deployErr.ChildError,
		testSuite,
	)
}

func assertExtractDestroyResourceError(
	err error,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	destroyErr, isDestroyErr := err.(*provider.ResourceDestroyError)
	testSuite.Require().True(isDestroyErr)
	testSuite.Require().NotNil(destroyErr)

	return assertExtractPluginResponseError(
		destroyErr.ChildError,
		testSuite,
	)
}

func assertExtractUpdateResourceAError(
	err error,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	updateResAErr, isUpdateResAErr := err.(*provider.LinkUpdateResourceAError)
	testSuite.Require().True(isUpdateResAErr)
	testSuite.Require().NotNil(updateResAErr)

	return assertExtractPluginResponseError(
		updateResAErr.ChildError,
		testSuite,
	)
}

func assertExtractUpdateResourceBError(
	err error,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	updateResBErr, isUpdateResBErr := err.(*provider.LinkUpdateResourceBError)
	testSuite.Require().True(isUpdateResBErr)
	testSuite.Require().NotNil(updateResBErr)

	return assertExtractPluginResponseError(
		updateResBErr.ChildError,
		testSuite,
	)
}

func assertExtractUpdateIntermediaryResourcesError(
	err error,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	updateIntermediaryErr, isUpdateIntermediaryErr := err.(*provider.LinkUpdateIntermediaryResourcesError)
	testSuite.Require().True(isUpdateIntermediaryErr)
	testSuite.Require().NotNil(updateIntermediaryErr)

	return assertExtractPluginResponseError(
		updateIntermediaryErr.ChildError,
		testSuite,
	)
}

func assertExtractPluginResponseError(
	err error,
	testSuite *suite.Suite,
) *errorsv1.PluginResponseError {
	pluginRespErr, ok := err.(*errorsv1.PluginResponseError)
	testSuite.Require().True(ok)
	testSuite.Require().NotNil(pluginRespErr)
	return pluginRespErr
}

// AssertSlicesEqual asserts that two slices are equal.
// Nil and empty slices are considered equal.
// The order of the elements in the slices must be the same.
func AssertSlicesEqual[Item any](
	expected []Item,
	actual []Item,
	testSuite *suite.Suite,
) {
	if expected != nil {
		expectedCopy := make([]Item, len(expected))
		copy(expectedCopy, expected)

		actualCopy := make([]Item, len(actual))
		copy(actualCopy, actual)

		testSuite.Assert().Equal(expectedCopy, actualCopy)
	} else {
		testSuite.Assert().Empty(actual)
	}
}
