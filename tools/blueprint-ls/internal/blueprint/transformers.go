package blueprint

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// LoadTransformers deals with loading initial transformers to be used for validating
// and in providing other LSP features for blueprint such as hover information.
//
// The language server uses the deploy engine plugin system to load gRPC transformer
// plugins at a later stage.
func LoadTransformers(ctx context.Context) (map[string]transform.SpecTransformer, error) {
	return map[string]transform.SpecTransformer{}, nil
}
