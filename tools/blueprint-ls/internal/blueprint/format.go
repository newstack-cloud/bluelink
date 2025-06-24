package blueprint

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// DetermineDocFormat determines the document format based on the file extension.
func DetermineDocFormat(docURI lsp.URI) schema.SpecFormat {
	if strings.HasSuffix(string(docURI), ".jsonc") ||
		strings.HasSuffix(string(docURI), ".hujson") ||
		strings.HasSuffix(string(docURI), ".json") {
		return schema.JWCCSpecFormat
	}

	return schema.YAMLSpecFormat
}
