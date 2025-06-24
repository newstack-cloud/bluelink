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
}
