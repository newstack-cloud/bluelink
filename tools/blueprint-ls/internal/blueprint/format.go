package blueprint

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// DetermineDocFormat determines the document format based on the file extension.
func DetermineDocFormat(docURI lsp.URI) schema.SpecFormat {
	uri := string(docURI)

	if strings.HasSuffix(uri, ".bp") || strings.HasSuffix(uri, ".blueprint") {
		return schema.BlueprintLangSpecFormat
	}

	if strings.HasSuffix(uri, ".jsonc") ||
		strings.HasSuffix(uri, ".hujson") ||
		strings.HasSuffix(uri, ".json") {
		return schema.JWCCSpecFormat
	}

	return schema.YAMLSpecFormat
}
