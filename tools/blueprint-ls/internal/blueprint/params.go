package blueprint

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/deployconfig"
)

// ValidationParams builds the BlueprintParams used for language server
// interactions with providers and transformers that are not scoped to a
// particular document, such as the registries backing completion and hover.
//
// Document validation resolves deploy configuration instead, so that
// transformer plugins receive the same parameters the CLI would send. See the
// deployconfig package.
func ValidationParams() core.BlueprintParams {
	return deployconfig.ValidationParams()
}
