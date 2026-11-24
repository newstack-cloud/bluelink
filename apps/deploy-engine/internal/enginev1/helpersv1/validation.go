package helpersv1

import (
	"github.com/go-playground/validator/v10"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/inputvalidation"
)

// ValidateRequestBody is the request body validator shared between
// endpoints.
var ValidateRequestBody *validator.Validate

// SetupRequestBodyValidator sets up the request body validator
// for all endpoints.
func SetupRequestBodyValidator() {
	ValidateRequestBody = validator.New()
	ValidateRequestBody.RegisterTagNameFunc(inputvalidation.JSONTagNameFunc)
	// Only fails for an empty tag or a nil function, which these can't be.
	_ = ValidateRequestBody.RegisterValidation(
		inputvalidation.TagPreciseResourceStatus,
		func(field validator.FieldLevel) bool {
			_, valid := ParsePreciseResourceStatus(field.Field().String())
			return valid
		},
	)
	_ = ValidateRequestBody.RegisterValidation(
		inputvalidation.TagPreciseLinkStatus,
		func(field validator.FieldLevel) bool {
			_, valid := ParsePreciseLinkStatus(field.Field().String())
			return valid
		},
	)
}
