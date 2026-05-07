package transformutils

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// ResourceConfigVariable reads a resource-scoped or service-scoped
// transformer config variable, preferring the resource-scoped form.
//
// For prefix "aws.dynamodb", resourceName "ordersTable", key "billingMode":
//  1. tries "aws.dynamodb.ordersTable.billingMode"
//  2. falls back to "aws.dynamodb.billingMode"
//  3. returns (nil, false) if neither is set; caller decides default.
func ResourceConfigVariable(
	transformCtx transform.Context,
	prefix string,
	resourceName string,
	key string,
) (*core.ScalarValue, error) {
	resourceKey := fmt.Sprintf("%s.%s.%s", prefix, resourceName, key)
	serviceKey := fmt.Sprintf("%s.%s", prefix, key)

	if value, ok := transformCtx.TransformerConfigVariable(resourceKey); ok {
		return value, nil
	}

	if value, ok := transformCtx.TransformerConfigVariable(serviceKey); ok {
		return value, nil
	}

	return nil, fmt.Errorf("config variable not found: %s", resourceKey)
}

// AppEnv is a helper function to read the "appEnv" context variable,
// which is commonly used to determine the deployment environment
// (e.g., "dev", "staging", "prod").
// It returns an empty string if the variable is not set.
func AppEnv(transformCtx transform.Context) string {
	if value, ok := transformCtx.ContextVariable("appEnv"); ok {
		return core.StringValueFromScalar(value)
	}

	return ""
}
