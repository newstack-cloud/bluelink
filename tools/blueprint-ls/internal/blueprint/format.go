package blueprint

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
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

// LoadSchemaString parses a document into a blueprint schema for the given
// format.
//
// The blueprint language format has its own parser and is not handled by
// schema.LoadString.
// The container package makes the same dispatch internally when loading a
// blueprint, but does not export it.
func LoadSchemaString(content string, format schema.SpecFormat) (*schema.Blueprint, error) {
	if format == schema.BlueprintLangSpecFormat {
		return lang.ParseString(content)
	}

	return schema.LoadString(content, format)
}
