package validationv1

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// CreateValidationRequestPayload represents the payload
// for creating a new blueprint validation.
type CreateValidationRequestPayload struct {
	resolve.BlueprintDocumentInfo
	// Config values for the validation process
	// that will be used in plugins and passed into the blueprint.
	Config *types.BlueprintOperationConfig `json:"config"`
	// LoaderConfig allows opt-in overrides to the validation loader's
	// default behaviour. When omitted, today's defaults apply
	// (transformSpec=false, validateAfterTransform=false).
	LoaderConfig *ValidationLoaderConfig `json:"loaderConfig,omitempty"`
}

// ValidationLoaderConfig carries opt-in flags that override the shared
// validation loader's defaults for a single request.
type ValidationLoaderConfig struct {
	// TransformSpec enables transformer plugins during validation.
	// Requires transformer plugins that handle the validation context.
	TransformSpec *bool `json:"transformSpec,omitempty"`
	// ValidateAfterTransform enables resource validation against the
	// transformed blueprint shape. Has no effect unless TransformSpec is
	// also true.
	ValidateAfterTransform *bool `json:"validateAfterTransform,omitempty"`
}

type diagnosticWithTimestamp struct {
	core.Diagnostic
	Timestamp int64 `json:"timestamp"`
	End       bool  `json:"end"`
}
