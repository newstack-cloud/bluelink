package lang

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// TokenType identifies the kind of a lexical Token produced by the lexer.
type TokenType string

// String returns a human-friendly label for use in diagnostics: the literal
// symbol for punctuation/operators, `keyword "x"` for reserved words, and a
// category name for value-bearing or structural tokens.
func (tt TokenType) String() string {
	if word, ok := keywordWords[tt]; ok {
		return fmt.Sprintf("keyword %q", word)
	}
	if label, ok := tokenTypeLabels[tt]; ok {
		return label
	}
	return string(tt)
}

var tokenTypeLabels = map[TokenType]string{
	TokenLeftBracket:            "'['",
	TokenRightBracket:           "']'",
	TokenLeftParen:              "'('",
	TokenRightParen:             "')'",
	TokenLeftBrace:              "'{'",
	TokenRightBrace:             "'}'",
	TokenColon:                  "':'",
	TokenAssign:                 "'='",
	TokenComma:                  "','",
	TokenPeriod:                 "'.'",
	TokenEq:                     "'=='",
	TokenNeq:                    "'!='",
	TokenLt:                     "'<'",
	TokenGt:                     "'>'",
	TokenLte:                    "'<='",
	TokenGte:                    "'>='",
	TokenAnd:                    "'&&'",
	TokenOr:                     "'||'",
	TokenNot:                    "'!'",
	TokenSlash:                  "'/'",
	TokenStar:                   "'*'",
	TokenIntLiteral:             "integer",
	TokenFloatLiteral:           "float",
	TokenBoolLiteral:            "boolean",
	TokenNoneLiteral:            "none",
	TokenStringStart:            "string",
	TokenStringEnd:              "end of string",
	TokenStringLiteral:          "string",
	TokenMultilineStringLiteral: "multi-line string",
	TokenInterpolationStart:     "'${'",
	TokenInterpolationEnd:       "'}'",
	TokenNewline:                "end of line",
	TokenComment:                "comment",
	TokenIdent:                  "identifier",
	TokenEOF:                    "end of input",
}

// keywordWords is the reverse of keywordTable, used to render reserved-word
// tokens as `keyword "x"` in diagnostics.
var keywordWords = func() map[TokenType]string {
	words := make(map[TokenType]string, len(keywordTable))
	for word, tt := range keywordTable {
		words[tt] = word
	}
	return words
}()

// Token kinds produced by the lexer. Consumers building a concrete syntax tree
// from the token stream (e.g. a language server) will use these.
const (
	TokenLeftBracket            TokenType = "leftBracket"
	TokenRightBracket           TokenType = "rightBracket"
	TokenLeftParen              TokenType = "leftParen"
	TokenRightParen             TokenType = "rightParen"
	TokenLeftBrace              TokenType = "leftBrace"
	TokenRightBrace             TokenType = "rightBrace"
	TokenColon                  TokenType = "colon"
	TokenAssign                 TokenType = "assign"
	TokenComma                  TokenType = "comma"
	TokenPeriod                 TokenType = "period"
	TokenEq                     TokenType = "eq"
	TokenNeq                    TokenType = "neq"
	TokenLt                     TokenType = "lt"
	TokenGt                     TokenType = "gt"
	TokenLte                    TokenType = "lte"
	TokenGte                    TokenType = "gte"
	TokenAnd                    TokenType = "and"
	TokenOr                     TokenType = "or"
	TokenNot                    TokenType = "not"
	TokenSlash                  TokenType = "slash"
	TokenStar                   TokenType = "star"
	TokenIntLiteral             TokenType = "intLiteral"
	TokenFloatLiteral           TokenType = "floatLiteral"
	TokenBoolLiteral            TokenType = "boolLiteral"
	TokenNoneLiteral            TokenType = "noneLiteral"
	TokenStringStart            TokenType = "stringStart"
	TokenStringEnd              TokenType = "stringEnd"
	TokenStringLiteral          TokenType = "stringLiteral"
	TokenMultilineStringLiteral TokenType = "multilineStringLiteral"
	TokenInterpolationStart     TokenType = "interpolationStart"
	TokenInterpolationEnd       TokenType = "interpolationEnd"
	TokenNewline                TokenType = "newline"
	TokenComment                TokenType = "comment"
	TokenIdent                  TokenType = "identifier"
	TokenKeywordVariables       TokenType = "keywordVariables"
	TokenKeywordValues          TokenType = "keywordValues"
	TokenKeywordDatasources     TokenType = "keywordDatasources"
	TokenKeywordResources       TokenType = "keywordResources"
	TokenKeywordChildren        TokenType = "keywordChildren"
	TokenKeywordElem            TokenType = "keywordElem"
	TokenKeywordI               TokenType = "keywordI"
	TokenKeywordVariable        TokenType = "keywordVariable"
	TokenKeywordValue           TokenType = "keywordValue"
	TokenKeywordData            TokenType = "keywordData"
	TokenKeywordResource        TokenType = "keywordResource"
	TokenKeywordInclude         TokenType = "keywordInclude"
	TokenKeywordExport          TokenType = "keywordExport"
	TokenKeywordMetadata        TokenType = "keywordMetadata"
	TokenKeywordSpec            TokenType = "keywordSpec"
	TokenKeywordSelect          TokenType = "keywordSelect"
	TokenKeywordFilter          TokenType = "keywordFilter"
	TokenKeywordForeach         TokenType = "keywordForeach"
	TokenKeywordAs              TokenType = "keywordAs"
	TokenKeywordBy              TokenType = "keywordBy"
	TokenKeywordLabel           TokenType = "keywordLabel"
	TokenKeywordVersion         TokenType = "keywordVersion"
	TokenKeywordTransform       TokenType = "keywordTransform"
	TokenKeywordNot             TokenType = "keywordNot"
	TokenKeywordIn              TokenType = "keywordIn"
	TokenKeywordHas             TokenType = "keywordHas"
	TokenKeywordKey             TokenType = "keywordKey"
	TokenKeywordContains        TokenType = "keywordContains"
	TokenKeywordStarts          TokenType = "keywordStarts"
	TokenKeywordWith            TokenType = "keywordWith"
	TokenKeywordEnds            TokenType = "keywordEnds"
	TokenKeywordString          TokenType = "keywordString"
	TokenKeywordInteger         TokenType = "keywordInteger"
	TokenKeywordFloat           TokenType = "keywordFloat"
	TokenKeywordBoolean         TokenType = "keywordBoolean"
	TokenKeywordArray           TokenType = "keywordArray"
	TokenKeywordObject          TokenType = "keywordObject"
	TokenEOF                    TokenType = "eof"
)

// Token is a single lexical token carrying its source position range. Start is
// inclusive and End is exclusive (the position immediately after the token's
// final rune).
type Token struct {
	Type  TokenType
	Value string
	Start source.Position
	End   source.Position
}
