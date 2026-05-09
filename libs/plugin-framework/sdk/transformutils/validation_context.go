package transformutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// IsValidationContext reports whether the transformer is being invoked
// for validation only (no change staging, deploy, or destroy will follow).
//
// Plugin authors should call this from transform, link validation and
// abstract-resource validation methods to branch on best-effort behaviour
// when external context required for actions (e.g. build manifests) is
// unavailable in this mode.
func IsValidationContext(transformerCtx transform.Context) bool {
	val, ok := transformerCtx.ContextVariable(core.ValidationContextVariableName)
	if !ok || !core.IsScalarBool(val) {
		return false
	}
	return core.BoolValueFromScalar(val)
}
