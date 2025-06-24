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
}

type diagnosticWithTimestamp struct {
	core.Diagnostic
	Timestamp int64 `json:"timestamp"`
	End       bool  `json:"end"`
}
