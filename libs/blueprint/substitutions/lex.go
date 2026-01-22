package substitutions

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type lexState struct {
	candidateToken    string
	prevChar          rune
	tokens            []*token
	parentSourceStart *source.Meta
	// Used to assign line and column numbers to tokens,
	// lex errors will not be reported with line and column,
	// the parent source context will have to suffice for mapping
	// locations to lex errors.
	relativeLineInfo *source.Meta
	// Used to ignore the parent source column when calculating
	// the absolute column for a token.
	ignoreParentColumn bool
}

type tokenType string

const (
	tokenOpenBracket        tokenType = "openBracket"
	tokenCloseBracket       tokenType = "closeBracket"
	tokenOpenParen          tokenType = "openParen"
	tokenCloseParen         tokenType = "closeParen"
	tokenEquals             tokenType = "equals"
	tokenComma              tokenType = "comma"
	tokenPeriod             tokenType = "period"
	tokenIntLiteral         tokenType = "intLiteral"
	tokenFloatLiteral       tokenType = "floatLiteral"
	tokenBoolLiteral        tokenType = "boolLiteral"
	tokenNoneLiteral        tokenType = "noneLiteral"
	tokenStringLiteral      tokenType = "stringLiteral"
	tokenNameStringLiteral  tokenType = "nameStringLiteral"
	tokenIdent              tokenType = "identifier"
	tokenKeywordVariables   tokenType = "keywordVariables"
	tokenKeywordValues      tokenType = "keywordValues"
	tokenKeywordDatasources tokenType = "keywordDatasources"
	tokenKeywordResources   tokenType = "keywordResources"
	tokenKeywordChildren    tokenType = "keywordChildren"
	tokenKeywordElem        tokenType = "keywordElem"
	tokenKeywordI           tokenType = "keywordI"
	tokenEOF                tokenType = "eof"
)

type token struct {
	tokenType    tokenType
	value        string
	relativeLine int
	relativeCol  int
}

var (
	whiteSpacePattern           = regexp.MustCompile(`\s`)
	boolPattern                 = regexp.MustCompile(`^(true|false)`)
	nonePattern                 = regexp.MustCompile(`^none`)
	lexStringLiteralNamePattern = regexp.MustCompile(`([A-Za-z0-9_-]|\.)+`)
)

func lex(sequence string, parentSourceStart *source.Meta) ([]*token, error) {
	lexState := &lexState{
		candidateToken:    "",
		prevChar:          ' ',
		tokens:            []*token{},
		parentSourceStart: parentSourceStart,
		relativeLineInfo: &source.Meta{
			Position: source.Position{
				Line:   0,
				Column: 0,
			},
		},
	}

	errors := []error{}
	i := 0
	for i < len(sequence) {
		char, width := utf8.DecodeRuneInString(sequence[i:])
		lexUpdateLine(lexState, char)

		if whiteSpacePattern.MatchString(string(char)) {
			i += width
			lexState.relativeLineInfo.Column += 1
			// Use continue to avoid using a complex if else chain
			// for each possible token type.
			continue
		}

		isPunctuation := checkPunctuation(string(char), lexState)
		if isPunctuation {
			i += width
			lexState.relativeLineInfo.Column += 1
			continue
		}

		charsConsumed, bytesConsumed := checkNumber(sequence, i, lexState)
		if bytesConsumed > 0 {
			i += bytesConsumed
			lexState.relativeLineInfo.Column += charsConsumed
			continue
		}

		charsConsumed, bytesConsumed = checkNoneLiteral(sequence, i, lexState)
		if bytesConsumed > 0 {
			i += bytesConsumed
			lexState.relativeLineInfo.Column += charsConsumed
			continue
		}

		charsConsumed, bytesConsumed = checkBoolLiteral(sequence, i, lexState)
		if bytesConsumed > 0 {
			i += bytesConsumed
			lexState.relativeLineInfo.Column += charsConsumed
			continue
		}

		charsConsumed, bytesConsumed = checkIdentifierOrKeyword(sequence, i, lexState)
		if bytesConsumed > 0 {
			i += bytesConsumed
			lexState.relativeLineInfo.Column += charsConsumed
			continue
		}

		charsConsumed, bytesConsumed, err := lexCheckStringLiteral(sequence, i, lexState)
		if err != nil {
			errors = append(errors, err)
		}
		if bytesConsumed > 0 {
			i += bytesConsumed
			lexState.relativeLineInfo.Column += charsConsumed
			continue
		}

		line := toAbsLine(
			lexState.parentSourceStart.Line,
			lexState.relativeLineInfo.Line,
		)
		col := toAbsColumn(
			lexState.parentSourceStart.Column,
			lexState.relativeLineInfo.Column,
			lexState.relativeLineInfo.Line == 0,
			lexState.ignoreParentColumn,
		)
		colAccuracy := determineLexColumnAccuracy(lexState)
		errors = append(errors, errLexUnexpectedChar(line, col, colAccuracy, char))
		i += 1
	}

	if len(errors) > 0 {
		return lexState.tokens, errLexMultiple("lexical analysis failed for substitution", errors)
	}

	return lexState.tokens, nil
}

func checkPunctuation(char string, state *lexState) bool {
	switch char {
	case "[":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenOpenBracket,
			value:        "[",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	case "]":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenCloseBracket,
			value:        "]",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	case "(":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenOpenParen,
			value:        "(",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	case ")":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenCloseParen,
			value:        ")",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	case ",":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenComma,
			value:        ",",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	case ".":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenPeriod,
			value:        ".",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	case "=":
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenEquals,
			value:        "=",
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return true
	default:
		return false
	}
}

func checkNumber(sequence string, startPos int, state *lexState) (int, int) {
	char, _ := utf8.DecodeRuneInString(sequence[startPos:])
	if char == '-' || (char >= '0' && char <= '9') {
		charsConsumed, bytesConsumed := takeFloatLiteral(state, sequence, startPos)
		if bytesConsumed > 0 {
			return charsConsumed, bytesConsumed
		}

		charsConsumed, bytesConsumed = takeIntLiteral(state, sequence, startPos)
		return charsConsumed, bytesConsumed
	}
	return 0, 0
}

func checkIdentifierOrKeyword(sequence string, startPos int, state *lexState) (int, int) {
	char, width := utf8.DecodeRuneInString(sequence[startPos:])
	if isIdentStartChar(char) {
		return takeIdentifierOrKeyword(state, sequence, startPos+width, char)
	}
	return 0, 0
}

func lexCheckStringLiteral(sequence string, startPos int, state *lexState) (int, int, error) {
	char, width := utf8.DecodeRuneInString(sequence[startPos:])
	if char == '"' || char == '\'' {
		return takeStringLiteral(state, sequence, startPos+width, char)
	}
	return 0, 0, nil
}

func checkNoneLiteral(sequence string, startPos int, state *lexState) (int, int) {
	char, _ := utf8.DecodeRuneInString(sequence[startPos:])
	if char == 'n' {
		return takeNoneLiteral(state, sequence, startPos)
	}
	return 0, 0
}

func checkBoolLiteral(sequence string, startPos int, state *lexState) (int, int) {
	char, _ := utf8.DecodeRuneInString(sequence[startPos:])
	if char == 't' || char == 'f' {
		return takeBoolLiteral(state, sequence, startPos)
	}
	return 0, 0
}

func isIdentStartChar(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '_'
}

func isIdentChar(char rune) bool {
	return isIdentStartChar(char) || (char >= '0' && char <= '9') || char == '-'
}

func takeFloatLiteral(state *lexState, sequence string, startPos int) (int, int) {
	inPossibleFloat := true
	i := startPos
	sign := ""
	intPart := ""
	passedDecimalPoint := false
	fractionalPart := ""

	for inPossibleFloat && i < len(sequence) {
		char, width := utf8.DecodeRuneInString(sequence[i:])
		if char == '-' && i == startPos {
			sign = "-"
		} else if char == '.' && !passedDecimalPoint {
			passedDecimalPoint = true
		} else if char >= '0' && char <= '9' {
			updateFloatParts(&intPart, &fractionalPart, passedDecimalPoint, char)
		} else {
			inPossibleFloat = false
		}

		i += width
	}

	if !passedDecimalPoint || intPart == "" || fractionalPart == "" {
		// A float literal can not be taken from the current position
		// in the sequence.
		return 0, 0
	}

	value := fmt.Sprintf("%s%s.%s", sign, intPart, fractionalPart)
	state.tokens = append(state.tokens, &token{
		tokenType:    tokenFloatLiteral,
		value:        value,
		relativeLine: state.relativeLineInfo.Line,
		relativeCol:  state.relativeLineInfo.Column,
	})

	return utf8.RuneCountInString(value), len(value)
}

func updateFloatParts(intPart *string, fractionalPart *string, passedDecimalPoint bool, char rune) {
	if passedDecimalPoint {
		*fractionalPart += string(char)
	} else {
		*intPart += string(char)
	}
}

func takeIntLiteral(state *lexState, sequence string, startPos int) (int, int) {
	inPossibleInt := true
	i := startPos
	sign := ""
	numberPart := ""

	for inPossibleInt && i < len(sequence) {
		char, width := utf8.DecodeRuneInString(sequence[i:])
		if char == '-' && i == startPos {
			sign = "-"
		} else if char >= '0' && char <= '9' {
			numberPart += string(char)
		} else {
			inPossibleInt = false
		}

		i += width
	}

	value := fmt.Sprintf("%s%s", sign, numberPart)
	state.tokens = append(state.tokens, &token{
		tokenType:    tokenIntLiteral,
		value:        value,
		relativeLine: state.relativeLineInfo.Line,
		relativeCol:  state.relativeLineInfo.Column,
	})

	return utf8.RuneCountInString(value), len(value)
}

func takeStringLiteral(state *lexState, sequence string, startPos int, quoteChar rune) (int, int, error) {
	inStringLiteral := true
	i := startPos
	prevChar := ' '
	value := ""
	for inStringLiteral && i < len(sequence) {
		char, width := utf8.DecodeRuneInString(sequence[i:])
		if char == quoteChar && prevChar != '\\' {
			inStringLiteral = false
		} else {
			value += string(char)
		}
		prevChar = char
		i += width
	}

	if inStringLiteral && i == len(sequence) {
		line := toAbsLine(
			state.parentSourceStart.Line,
			state.relativeLineInfo.Line,
		)
		col := toAbsColumn(
			state.parentSourceStart.Column,
			state.relativeLineInfo.Column,
			state.relativeLineInfo.Line == 0,
			state.ignoreParentColumn,
		)
		colAccuracy := determineLexColumnAccuracy(state)
		consumed := sequence[startPos:i]
		return utf8.RuneCountInString(consumed), i - startPos, errLexUnexpectedEndOfInput(
			line,
			col,
			colAccuracy,
			"string literal",
		)
	}

	// Differentiate between a string literal and a name string literal
	// to allow the parser to catch errors when unexpected characters are used
	// in a string that is used as a name in a [".."] or ['..'] accessor.
	prevTokenOpenBracket := len(state.tokens) > 0 && state.tokens[len(state.tokens)-1].tokenType == tokenOpenBracket
	if prevTokenOpenBracket && lexStringLiteralNamePattern.MatchString(value) {
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenNameStringLiteral,
			value:        unescapeStringLiteral(value, quoteChar),
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
		return utf8.RuneCountInString(value) + 2, len(value) + 2, nil
	}

	state.tokens = append(state.tokens, &token{
		tokenType:    tokenStringLiteral,
		value:        unescapeStringLiteral(value, quoteChar),
		relativeLine: state.relativeLineInfo.Line,
		relativeCol:  state.relativeLineInfo.Column,
	})

	// Add 2 to account for the quotes.
	return utf8.RuneCountInString(value) + 2, len(value) + 2, nil
}

// unescapeStringLiteral handles escape sequences for the given quote character.
func unescapeStringLiteral(value string, quoteChar rune) string {
	escapeSeq := "\\" + string(quoteChar)
	return strings.ReplaceAll(value, escapeSeq, string(quoteChar))
}

func takeIdentifierOrKeyword(state *lexState, sequence string, restStartPos int, startChar rune) (int, int) {
	inPossibleIdent := true
	i := restStartPos
	value := string(startChar)
	for inPossibleIdent && i < len(sequence) {
		char, width := utf8.DecodeRuneInString(sequence[i:])
		if isIdentChar(char) {
			value += string(char)
		} else {
			inPossibleIdent = false
		}
		i += width
	}

	tType := deriveIdentOrKeywordTokenType(value)
	state.tokens = append(state.tokens, &token{
		tokenType:    tType,
		value:        value,
		relativeLine: state.relativeLineInfo.Line,
		relativeCol:  state.relativeLineInfo.Column,
	})

	return utf8.RuneCountInString(value), len(value)
}

func deriveIdentOrKeywordTokenType(value string) tokenType {
	switch value {
	case "variables":
		return tokenKeywordVariables
	case "values":
		return tokenKeywordValues
	case "datasources":
		return tokenKeywordDatasources
	case "resources":
		return tokenKeywordResources
	case "children":
		return tokenKeywordChildren
	case "elem":
		return tokenKeywordElem
	case "i":
		return tokenKeywordI
	default:
		return tokenIdent
	}
}

func takeNoneLiteral(state *lexState, sequence string, startPos int) (int, int) {
	subSequence := sequence[startPos:]
	value := nonePattern.FindString(subSequence)
	if len(value) > 0 {
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenNoneLiteral,
			value:        value,
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
	}

	return utf8.RuneCountInString(value), len(value)
}

func takeBoolLiteral(state *lexState, sequence string, startPos int) (int, int) {
	subSequence := sequence[startPos:]
	value := boolPattern.FindString(subSequence)
	if len(value) > 0 {
		state.tokens = append(state.tokens, &token{
			tokenType:    tokenBoolLiteral,
			value:        value,
			relativeLine: state.relativeLineInfo.Line,
			relativeCol:  state.relativeLineInfo.Column,
		})
	}

	return utf8.RuneCountInString(value), len(value)
}

func lexUpdateLine(state *lexState, char rune) {
	if char == '\n' {
		state.relativeLineInfo.Line += 1
		state.relativeLineInfo.Column = 1
	}
}

func determineLexColumnAccuracy(state *lexState) ColumnAccuracy {
	if state.ignoreParentColumn {
		// when we are ignoring the parent column, it is usually due to the
		// lack of precision in determining the column number of a token.
		// An example of this is when a YAML scalar node is a block style literal
		// or folded string in the host document and the yaml.v3 library does not provide the
		// starting column number of the beginning of the literal value,
		// only the literal symbol "|" or ">" on the line above the literal value.
		return ColumnAccuracyApproximate
	}

	return ColumnAccuracyExact
}
