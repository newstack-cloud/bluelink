package provider

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/stretchr/testify/assert"
)

func TestProviderErrorsWithContext(t *testing.T) {
	t.Run("DataSourceTypeProviderNotFound", func(t *testing.T) {
		err := errDataSourceTypeProviderNotFound("aws", "s3_bucket")
		runErr, ok := err.(*errors.RunError)
		assert.True(t, ok)
		assert.NotNil(t, runErr.Context)
		assert.Equal(t, errors.ErrorCategoryProviderMissing, runErr.Context.Category)
		assert.Len(t, runErr.Context.SuggestedActions, 2)
		assert.Equal(t, string(errors.ActionTypeInstallProvider), runErr.Context.SuggestedActions[0].Type)
		assert.Contains(t, runErr.Error(), "provider \"aws\" not found for data source type \"s3_bucket\"")
	})

	t.Run("ProviderDataSourceTypeNotFound", func(t *testing.T) {
		err := errProviderDataSourceTypeNotFound("s3_bucket", "aws")
		runErr, ok := err.(*errors.RunError)
		assert.True(t, ok)
		assert.NotNil(t, runErr.Context)
		assert.Equal(t, errors.ErrorCategoryProviderIncompatible, runErr.Context.Category)
		assert.Contains(t, runErr.Error(), "provider \"aws\" does not support data source type \"s3_bucket\"")
	})

	t.Run("FunctionNotFound", func(t *testing.T) {
		err := errFunctionNotFound("myFunction")
		runErr, ok := err.(*errors.RunError)
		assert.True(t, ok)
		assert.NotNil(t, runErr.Context)
		assert.Equal(t, errors.ErrorCategoryFunctionNotFound, runErr.Context.Category)
		assert.Contains(t, runErr.Error(), "function \"myFunction\" not found")
		assert.Equal(t, "myFunction", runErr.Context.Metadata["functionName"])
		assert.Equal(t, string(errors.ActionTypeCheckFunctionName), runErr.Context.SuggestedActions[0].Type)
	})

	t.Run("ConciseErrorMessage", func(t *testing.T) {
		err := errDataSourceTypeProviderNotFound("aws", "s3_bucket")
		runErr, ok := err.(*errors.RunError)
		assert.True(t, ok)

		// Error message should be concise and single-line
		assert.Equal(t, "provider \"aws\" not found for data source type \"s3_bucket\"", runErr.Err.Error())
	})
}
