package lang

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type tokenType string

// String returns a human-friendly label for use in diagnostics: the literal
// symbol for punctuation/operators, `keyword "x"` for reserved words, and a
// category name for value-bearing or structural tokens.
func (tt tokenType) String() string {
	if word, ok := keywordWords[tt]; ok {
		return fmt.Sprintf("keyword %q", word)
	}
	if label, ok := tokenTypeLabels[tt]; ok {
		return label
	}
	return string(tt)
}

var tokenTypeLabels = map[tokenType]string{
	tokenLeftBracket:            "'['",
	tokenRightBracket:           "']'",
	tokenLeftParen:              "'('",
	tokenRightParen:             "')'",
	tokenLeftBrace:              "'{'",
	tokenRightBrace:             "'}'",
	tokenColon:                  "':'",
	tokenAssign:                 "'='",
	tokenComma:                  "','",
	tokenPeriod:                 "'.'",
	tokenEq:                     "'=='",
	tokenNeq:                    "'!='",
	tokenLt:                     "'<'",
	tokenGt:                     "'>'",
	tokenLte:                    "'<='",
	tokenGte:                    "'>='",
	tokenAnd:                    "'&&'",
	tokenOr:                     "'||'",
	tokenNot:                    "'!'",
	tokenSlash:                  "'/'",
	tokenStar:                   "'*'",
	tokenIntLiteral:             "integer",
	tokenFloatLiteral:           "float",
	tokenBoolLiteral:            "boolean",
	tokenNoneLiteral:            "none",
	tokenStringStart:            "string",
	tokenStringEnd:              "end of string",
	tokenStringLiteral:          "string",
	tokenMultilineStringLiteral: "multi-line string",
	tokenInterpolationStart:     "'${'",
	tokenInterpolationEnd:       "'}'",
	tokenNewline:                "end of line",
	tokenComment:                "comment",
	tokenIdent:                  "identifier",
	tokenEOF:                    "end of input",
}

// keywordWords is the reverse of keywordTable, used to render reserved-word
// tokens as `keyword "x"` in diagnostics.
var keywordWords = func() map[tokenType]string {
	words := make(map[tokenType]string, len(keywordTable))
	for word, tt := range keywordTable {
		words[tt] = word
	}
	return words
}()

const (
	tokenLeftBracket            tokenType = "leftBracket"
	tokenRightBracket           tokenType = "rightBracket"
	tokenLeftParen              tokenType = "leftParen"
	tokenRightParen             tokenType = "rightParen"
	tokenLeftBrace              tokenType = "leftBrace"
	tokenRightBrace             tokenType = "rightBrace"
	tokenColon                  tokenType = "colon"
	tokenAssign                 tokenType = "assign"
	tokenComma                  tokenType = "comma"
	tokenPeriod                 tokenType = "period"
	tokenEq                     tokenType = "eq"
	tokenNeq                    tokenType = "neq"
	tokenLt                     tokenType = "lt"
	tokenGt                     tokenType = "gt"
	tokenLte                    tokenType = "lte"
	tokenGte                    tokenType = "gte"
	tokenAnd                    tokenType = "and"
	tokenOr                     tokenType = "or"
	tokenNot                    tokenType = "not"
	tokenSlash                  tokenType = "slash"
	tokenStar                   tokenType = "star"
	tokenIntLiteral             tokenType = "intLiteral"
	tokenFloatLiteral           tokenType = "floatLiteral"
	tokenBoolLiteral            tokenType = "boolLiteral"
	tokenNoneLiteral            tokenType = "noneLiteral"
	tokenStringStart            tokenType = "stringStart"
	tokenStringEnd              tokenType = "stringEnd"
	tokenStringLiteral          tokenType = "stringLiteral"
	tokenMultilineStringLiteral tokenType = "multilineStringLiteral"
	tokenInterpolationStart     tokenType = "interpolationStart"
	tokenInterpolationEnd       tokenType = "interpolationEnd"
	tokenNewline                tokenType = "newline"
	tokenComment                tokenType = "comment"
	tokenIdent                  tokenType = "identifier"
	tokenKeywordVariables       tokenType = "keywordVariables"
	tokenKeywordValues          tokenType = "keywordValues"
	tokenKeywordDatasources     tokenType = "keywordDatasources"
	tokenKeywordResources       tokenType = "keywordResources"
	tokenKeywordChildren        tokenType = "keywordChildren"
	tokenKeywordElem            tokenType = "keywordElem"
	tokenKeywordI               tokenType = "keywordI"
	tokenKeywordVariable        tokenType = "keywordVariable"
	tokenKeywordValue           tokenType = "keywordValue"
	tokenKeywordData            tokenType = "keywordData"
	tokenKeywordResource        tokenType = "keywordResource"
	tokenKeywordInclude         tokenType = "keywordInclude"
	tokenKeywordExport          tokenType = "keywordExport"
	tokenKeywordMetadata        tokenType = "keywordMetadata"
	tokenKeywordSpec            tokenType = "keywordSpec"
	tokenKeywordSelect          tokenType = "keywordSelect"
	tokenKeywordFilter          tokenType = "keywordFilter"
	tokenKeywordForeach         tokenType = "keywordForeach"
	tokenKeywordAs              tokenType = "keywordAs"
	tokenKeywordBy              tokenType = "keywordBy"
	tokenKeywordLabel           tokenType = "keywordLabel"
	tokenKeywordVersion         tokenType = "keywordVersion"
	tokenKeywordTransform       tokenType = "keywordTransform"
	tokenKeywordNot             tokenType = "keywordNot"
	tokenKeywordIn              tokenType = "keywordIn"
	tokenKeywordHas             tokenType = "keywordHas"
	tokenKeywordKey             tokenType = "keywordKey"
	tokenKeywordContains        tokenType = "keywordContains"
	tokenKeywordStarts          tokenType = "keywordStarts"
	tokenKeywordWith            tokenType = "keywordWith"
	tokenKeywordEnds            tokenType = "keywordEnds"
	tokenKeywordString          tokenType = "keywordString"
	tokenKeywordInteger         tokenType = "keywordInteger"
	tokenKeywordFloat           tokenType = "keywordFloat"
	tokenKeywordBoolean         tokenType = "keywordBoolean"
	tokenKeywordArray           tokenType = "keywordArray"
	tokenKeywordObject          tokenType = "keywordObject"
	tokenEOF                    tokenType = "eof"
)

type token struct {
	tokenType tokenType
	value     string
	pos       source.Position
	endPos    source.Position
}
