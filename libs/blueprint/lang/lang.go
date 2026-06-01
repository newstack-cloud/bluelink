package lang

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// ParseString parses a blueprint language source string and returns
// a standardised blueprint model.
// This returns a structured error with source positions
// on any parse failures.
func ParseString(src string) (*schema.Blueprint, error) {
	return newParser(src).parse()
}

// ParseFile reads a .blueprint / .bp file and returns
// a standardised blueprint model.
// This returns a structured error with source positions
// on any parse failures.
func ParseFile(path string) (*schema.Blueprint, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseString(string(contents))
}

// Tokenize lexes a blueprint language source string into its full token stream,
// terminated by a single TokenEOF token. Unlike ParseString, the stream
// preserves comments and newlines, so callers such as a language server can
// reconstruct a concrete syntax tree for features like completion and document
// symbols.
//
// The token slice is always returned, even on lexical errors. This allows a partial
// CST to be built for an in-progress edit. Any lexical errors are aggregated
// into the returned error, which is a *Errors envelope (the same type
// ParseString returns) or nil when lexing succeeds.
func Tokenize(src string) ([]Token, error) {
	lex := newLexer(src)

	var tokens []Token
	for {
		tkn := lex.nextToken()
		tokens = append(tokens, *tkn)
		if tkn.Type == TokenEOF {
			break
		}
	}

	return tokens, lex.diags.asError()
}
