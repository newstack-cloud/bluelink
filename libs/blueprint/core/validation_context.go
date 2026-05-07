package core

// ValidationContextVariableName is the reserved context-variable key that
// signals to transformer plugins that the current load is for validation
// only and that no actions (change staging, deploy, destroy) will follow.
//
// Host applications (e.g. deploy-engine, blueprint-ls) set this in their
// context-variables map before constructing BlueprintParams for a
// validation-only loader call:
//
//	contextVars[core.ValidationContextVariableName] =
//	    core.ScalarFromBool(true)
//
// Transformer plugin authors read it ergonomically via the helper at
// libs/plugin-framework/sdk/transformutils.IsValidationContext.
//
// The "__bluelink_…__" prefix marks this as a Bluelink-reserved internal
// name — do not rely on the literal string in plugin or host code; import
// this constant.
const ValidationContextVariableName = "__bluelink_is_validation_context__"
