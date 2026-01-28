package languageservices

import (
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// CompletionFormatter handles format-specific completion item formatting.
// It abstracts the differences between YAML and JSONC completion item generation.
type CompletionFormatter interface {
	// FormatValue formats a value for insertion (handles quoting, spacing).
	// hasLeadingQuote indicates the user already typed an opening quote.
	// hasLeadingSpace indicates there's whitespace before the cursor.
	FormatValue(value string, hasLeadingQuote bool, hasLeadingSpace bool) string

	// FormatKey formats a key/field name for insertion (adds colon, quotes as needed).
	// Returns the text to insert for a field name completion.
	FormatKey(keyName string) string

	// FormatArrayItem formats an array item value for insertion.
	FormatArrayItem(value string) string

	// GetInsertRange calculates the insertion range based on typed prefix length.
	GetInsertRange(position *lsp.Position, prefixLen int) *lsp.Range

	// NeedsQuoting returns whether a value needs quoting in this format.
	NeedsQuoting(value string) bool

	// GetFormat returns the document format this formatter handles.
	GetFormat() docmodel.DocumentFormat
}

// YAMLFormatter implements CompletionFormatter for YAML documents.
// YAML uses indentation-based structure and has special quoting rules.
type YAMLFormatter struct{}

// FormatValue formats a value for YAML insertion.
// Values that could be misinterpreted (booleans, nulls, special chars) need quoting.
func (f *YAMLFormatter) FormatValue(value string, _, _ bool) string {
	if f.NeedsQuoting(value) {
		return `"` + value + `"`
	}
	return value
}

// FormatKey formats a key name for YAML insertion.
// YAML keys are followed by ": " (colon and space).
func (f *YAMLFormatter) FormatKey(keyName string) string {
	return keyName + ": "
}

// FormatArrayItem formats an array item for YAML insertion.
// YAML uses "- " prefix for block-style sequence items.
func (f *YAMLFormatter) FormatArrayItem(value string) string {
	if f.NeedsQuoting(value) {
		return `"` + value + `"`
	}
	return value
}

// GetInsertRange returns the range to replace with the completion.
func (f *YAMLFormatter) GetInsertRange(position *lsp.Position, prefixLen int) *lsp.Range {
	return getItemInsertRangeWithPrefix(position, prefixLen)
}

// NeedsQuoting returns true if a value needs quoting in YAML.
func (f *YAMLFormatter) NeedsQuoting(value string) bool {
	return needsYAMLQuoting(value)
}

// GetFormat returns FormatYAML.
func (f *YAMLFormatter) GetFormat() docmodel.DocumentFormat {
	return docmodel.FormatYAML
}

// JSONCFormatter implements CompletionFormatter for JSONC documents.
// JSONC uses delimiter-based structure and always quotes string values.
type JSONCFormatter struct{}

// FormatValue formats a value for JSONC insertion.
// If hasLeadingQuote is true, user already typed the opening quote.
// If hasLeadingSpace is true, there's already whitespace before the cursor.
func (f *JSONCFormatter) FormatValue(value string, hasLeadingQuote bool, hasLeadingSpace bool) string {
	if hasLeadingQuote {
		// User already typed the opening quote, just insert value + closing quote
		return value + `"`
	}
	if hasLeadingSpace {
		// There's already whitespace before cursor, just insert quoted value
		return `"` + value + `"`
	}
	// JSONC: insert as quoted string with leading space after the colon
	return ` "` + value + `"`
}

// FormatKey formats a key name for JSONC insertion.
// JSONC keys are quoted and followed by ": ".
func (f *JSONCFormatter) FormatKey(keyName string) string {
	return `"` + keyName + `": `
}

// FormatArrayItem formats an array item for JSONC insertion.
// JSONC array items are quoted strings.
func (f *JSONCFormatter) FormatArrayItem(value string) string {
	return `"` + value + `"`
}

// GetInsertRange returns the range to replace with the completion.
func (f *JSONCFormatter) GetInsertRange(position *lsp.Position, prefixLen int) *lsp.Range {
	return getItemInsertRangeWithPrefix(position, prefixLen)
}

// NeedsQuoting returns true - JSONC always requires quotes for string values.
func (f *JSONCFormatter) NeedsQuoting(_ string) bool {
	return true
}

// GetFormat returns FormatJSONC.
func (f *JSONCFormatter) GetFormat() docmodel.DocumentFormat {
	return docmodel.FormatJSONC
}

// NewCompletionFormatter creates the appropriate formatter for a document format.
func NewCompletionFormatter(format docmodel.DocumentFormat) CompletionFormatter {
	switch format {
	case docmodel.FormatJSONC:
		return &JSONCFormatter{}
	default:
		return &YAMLFormatter{}
	}
}
