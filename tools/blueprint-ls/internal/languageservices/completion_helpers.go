package languageservices

import (
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// formatValueForInsert formats a string value for insertion based on document format.
// For YAML, values that could be misinterpreted need quoting.
// For JSONC, values are inserted as quoted strings. If hasLeadingQuote is true (user already
// typed the opening quote), only the value and closing quote are inserted.
// If hasLeadingSpace is true (there's already whitespace before cursor), no extra space is added.
func formatValueForInsert(
	value string,
	format docmodel.DocumentFormat,
	hasLeadingQuote bool,
	hasLeadingSpace bool,
) string {
	if format == docmodel.FormatJSONC {
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

	// YAML: check if value needs quoting
	if needsYAMLQuoting(value) {
		return `"` + value + `"`
	}
	return value
}

// stripLeadingQuote returns the prefix without a leading quote and whether one was present.
// This is used for JSONC completions where the user may have typed an opening quote.
func stripLeadingQuote(prefix string) (string, bool) {
	if after, ok := strings.CutPrefix(prefix, `"`); ok {
		return after, true
	}
	return prefix, false
}

// hasLeadingWhitespace checks if there's whitespace immediately before the typed prefix.
// This is used for JSONC completions to avoid adding an extra space when the user
// has already typed one (e.g., `"type": ` vs `"type":`).
func hasLeadingWhitespace(textBefore string, typedPrefixLen int) bool {
	// If there's no typed prefix, check if textBefore ends with whitespace
	if typedPrefixLen == 0 {
		if len(textBefore) > 0 {
			lastChar := textBefore[len(textBefore)-1]
			return lastChar == ' ' || lastChar == '\t'
		}
		return false
	}
	// If there's a typed prefix, check the character just before it
	charBeforePrefix := len(textBefore) - typedPrefixLen
	if charBeforePrefix > 0 {
		char := textBefore[charBeforePrefix-1]
		return char == ' ' || char == '\t'
	}
	return false
}

// needsYAMLQuoting returns true if a string value needs quotes in YAML.
func needsYAMLQuoting(value string) bool {
	if value == "" {
		return true
	}

	// Boolean-like values
	lowerValue := strings.ToLower(value)
	if lowerValue == "true" || lowerValue == "false" ||
		lowerValue == "yes" || lowerValue == "no" ||
		lowerValue == "on" || lowerValue == "off" {
		return true
	}

	// Null-like values
	if lowerValue == "null" || lowerValue == "~" {
		return true
	}

	// Check for special characters that need quoting
	for _, c := range value {
		if c == ':' || c == '#' || c == '[' || c == ']' ||
			c == '{' || c == '}' || c == ',' || c == '&' ||
			c == '*' || c == '!' || c == '|' || c == '>' ||
			c == '\'' || c == '"' || c == '%' || c == '@' {
			return true
		}
	}

	return false
}

// getItemInsertRange returns a range at the current cursor position.
func getItemInsertRange(position *lsp.Position) *lsp.Range {
	return &lsp.Range{
		Start: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
		End: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
	}
}

// getItemInsertRangeWithPrefix returns a range that includes characters already typed.
// This allows the completion to replace what the user has typed so far.
func getItemInsertRangeWithPrefix(position *lsp.Position, prefixLen int) *lsp.Range {
	startChar := position.Character - lsp.UInteger(prefixLen)

	return &lsp.Range{
		Start: lsp.Position{
			Line:      position.Line,
			Character: startChar,
		},
		End: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
	}
}

// getOperatorInsertRange calculates the range for inserting an operator completion.
func getOperatorInsertRange(
	position *lsp.Position,
	insertText string,
	isPrecededByOperator bool,
	operatorElementPosition *source.Position,
) *lsp.Range {
	charCount := utf8.RuneCountInString(insertText)

	// If cursor is right after "operator:" field, insert at cursor position
	if isPrecededByOperator {
		return &lsp.Range{
			Start: lsp.Position{
				Line:      position.Line,
				Character: position.Character,
			},
			End: lsp.Position{
				Line:      position.Line,
				Character: position.Character + lsp.UInteger(charCount),
			},
		}
	}

	// Otherwise, replace from the operator element's start position
	start := lsp.Position{
		Line:      lsp.UInteger(operatorElementPosition.Line) - 1,
		Character: lsp.UInteger(operatorElementPosition.Column) - 1,
	}

	return &lsp.Range{
		Start: start,
		End: lsp.Position{
			Line:      start.Line,
			Character: start.Character + lsp.UInteger(charCount),
		},
	}
}

// formatBracketNotation creates a bracket notation accessor for a key.
func formatBracketNotation(key string, quoteType docmodel.QuoteType) string {
	if quoteType == docmodel.QuoteTypeSingle {
		return "['" + key + "']"
	}
	return `["` + key + `"]`
}

// needsBracketNotation returns true if a key needs bracket notation (contains dots or special chars).
func needsBracketNotation(key string) bool {
	for _, c := range key {
		if c == '.' || c == '[' || c == ']' || c == ' ' || c == '-' {
			return true
		}
	}
	return false
}

// getBracketNotationInsertRange returns the range for inserting bracket notation.
// The range starts 1 character before the current position (to replace the "."
// that preceded the completion trigger) and ends at the current position.
func getBracketNotationInsertRange(position *lsp.Position) *lsp.Range {
	startChar := position.Character
	if startChar > 0 {
		startChar--
	}
	return &lsp.Range{
		Start: lsp.Position{
			Line:      position.Line,
			Character: startChar,
		},
		End: lsp.Position{
			Line:      position.Line,
			Character: position.Character,
		},
	}
}

// mapKeys returns the keys of a map as a slice.
func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// noCompletionsHint returns a single "no completions available" hint item.
func noCompletionsHint(position *lsp.Position, message string) []*lsp.CompletionItem {
	kind := lsp.CompletionItemKindText
	return []*lsp.CompletionItem{
		{
			Label: message,
			Kind:  &kind,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      position.Line,
						Character: position.Character,
					},
					End: lsp.Position{
						Line:      position.Line,
						Character: position.Character,
					},
				},
				NewText: "",
			},
		},
	}
}

// completionPrefixInfo holds extracted prefix information for completion filtering.
type completionPrefixInfo struct {
	TypedPrefix     string
	TextBefore      string
	FilterPrefix    string
	HasLeadingQuote bool
	HasLeadingSpace bool
	PrefixLen       int
	PrefixLower     string
}

// extractCompletionPrefix extracts and normalizes prefix information from completion context.
func extractCompletionPrefix(
	completionCtx *docmodel.CompletionContext,
	format docmodel.DocumentFormat,
) completionPrefixInfo {
	info := completionPrefixInfo{}

	if completionCtx != nil && completionCtx.NodeCtx != nil {
		info.TypedPrefix = completionCtx.NodeCtx.GetTypedPrefix()
		info.TextBefore = completionCtx.NodeCtx.TextBefore
	}

	info.FilterPrefix, info.HasLeadingQuote = stripLeadingQuote(info.TypedPrefix)
	if format != docmodel.FormatJSONC {
		info.HasLeadingQuote = false
	}
	info.PrefixLen = len(info.TypedPrefix)
	info.HasLeadingSpace = hasLeadingWhitespace(info.TextBefore, info.PrefixLen)
	info.PrefixLower = strings.ToLower(info.FilterPrefix)

	return info
}

// filterByPrefix returns true if the item should be included based on prefix filtering.
func filterByPrefix(itemLabel string, prefixInfo completionPrefixInfo) bool {
	if len(prefixInfo.FilterPrefix) == 0 {
		return true
	}
	return strings.HasPrefix(strings.ToLower(itemLabel), prefixInfo.PrefixLower)
}
