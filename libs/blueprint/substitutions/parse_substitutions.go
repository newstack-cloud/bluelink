package substitutions

import (
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type interpolationParseState struct {
	parentSourceStart               *source.Meta
	parentContextPrecedingCharCount int
	relativeLineInfo                *source.Meta
	relativeSubStart                *source.Meta
	parsed                          []*StringOrSubstitution
	inPossibleSub                   bool
	inStringLiteral                 bool
	potentialSub                    string
	potentialNonSubStr              string
	prevChar                        rune
	errors                          []error
	outputLineInfo                  bool
	ignoreParentColumn              bool
}

// ParseSubstitutionValues parses a string that can contain interpolated
// references.
func ParseSubstitutionValues(
	substitutionContext, value string,
	parentSourceMeta *source.Meta,
	outputLineInfo bool,
	ignoreParentColumn bool,
	parentContextPrecedingCharCount int,
) ([]*StringOrSubstitution, error) {
	// This is hand-rolled to account for the fact that string literals
	// are supported in the spec for substitutions and they can contain
	// the "${" and "}" syntax, trying to do string splitting won't catch
	// this and regular expressions without lookaheads can't be used to express
	// this.

	// There are no openings for substitutions, the value is just a string literal.
	// This may not be the case if the string literal contains the "${" in which case
	// it will be caught in the process of evaulating every character in sequence.
	if !strings.Contains(value, "${") {
		return []*StringOrSubstitution{
			{
				StringValue: &value,
				SourceMeta:  parentSourceMeta,
			},
		}, nil
	}

	state := &interpolationParseState{
		// To alleviate the frustration of having to deal with only seeing
		// one error at a time per substitution, all errors are collected.
		// The trade-off will not be significant in most use cases as string
		// values that can contain substitutions are not expected to be very long.
		errors:                          []error{},
		parsed:                          []*StringOrSubstitution{},
		parentSourceStart:               parentSourceMeta,
		parentContextPrecedingCharCount: parentContextPrecedingCharCount,
		relativeLineInfo: &source.Meta{
			Position: source.Position{
				Line:   0,
				Column: 0,
			},
		},
		relativeSubStart:   &source.Meta{Position: source.Position{}},
		inPossibleSub:      false,
		inStringLiteral:    false,
		potentialSub:       "",
		potentialNonSubStr: "",
		prevChar:           ' ',
		outputLineInfo:     outputLineInfo,
		ignoreParentColumn: ignoreParentColumn,
	}

	i := 0
	for i < len(value) {
		isOpenSubBracket := checkOpenSubBracket(state, value, i)
		checkStringLiteral(state, value, i)
		isCloseSubBracket := checkCloseSubBracket(state, value, i, substitutionContext)

		char, width := utf8.DecodeRuneInString(value[i:])
		state.prevChar = char
		if !isCloseSubBracket {
			state.potentialNonSubStr += string(char)
		}
		if state.inPossibleSub && !isOpenSubBracket {
			state.potentialSub += string(char)
		}
		updateLineInfo(state, char)

		i += width
	}

	if len(state.potentialNonSubStr) > 0 {
		parentLine := 0
		if parentSourceMeta != nil {
			parentLine = parentSourceMeta.Line
		}

		parentColumn := 0
		if parentSourceMeta != nil {
			parentColumn = parentSourceMeta.Column
		}

		sourceMeta := (*source.Meta)(nil)
		if state.outputLineInfo {
			sourceMeta = &source.Meta{
				Position: source.Position{
					Line: toAbsLine(parentLine, state.relativeLineInfo.Line),
					Column: toAbsColumn(
						parentColumn,
						state.relativeLineInfo.Column-len(state.potentialNonSubStr),
						state.relativeLineInfo.Line == 0,
						state.ignoreParentColumn,
					),
				},
				EndPosition: parentSourceMeta.EndPosition,
			}
		}

		state.parsed = append(state.parsed, &StringOrSubstitution{
			StringValue: &state.potentialNonSubStr,
			SourceMeta:  sourceMeta,
		})
	}

	if len(state.errors) > 0 {
		return nil, prepareSubstitutionErrors(substitutionContext, state)
	}

	return state.parsed, nil
}

func prepareSubstitutionErrors(substitutionContext string, state *interpolationParseState) error {
	line := (*int)(nil)
	if state.parentSourceStart != nil {
		line = &state.parentSourceStart.Line
	}
	column := (*int)(nil)
	if state.parentSourceStart != nil {
		column = &state.parentSourceStart.Column
	}
	return errSubstitutions(
		substitutionContext,
		state.errors,
		line,
		column,
	)
}

func updateLineInfo(state *interpolationParseState, value rune) {
	if value == '\n' {
		state.relativeLineInfo.Line += 1
		state.relativeLineInfo.Column = 0
	} else {
		state.relativeLineInfo.Column += 1
	}
}

func checkOpenSubBracket(state *interpolationParseState, value string, i int) bool {
	char, _ := utf8.DecodeRuneInString(value[i:])
	isOpenSubBracket := state.prevChar == '$' && char == '{' && !state.inStringLiteral
	if isOpenSubBracket {
		// Start of a substitution
		state.inPossibleSub = true
		state.relativeSubStart = &source.Meta{
			Position: source.Position{
				Line:   state.relativeLineInfo.Line,
				Column: state.relativeLineInfo.Column + 1,
			},
		}
		nonSubStr := state.potentialNonSubStr[:len(state.potentialNonSubStr)-1]
		if len(nonSubStr) > 0 {
			sourceMeta := createStringValSourceMeta(state, nonSubStr)
			state.parsed = append(state.parsed, &StringOrSubstitution{
				StringValue: &nonSubStr,
				SourceMeta:  sourceMeta,
			})
		}
	}
	return isOpenSubBracket
}

func createStringValSourceMeta(state *interpolationParseState, stringVal string) *source.Meta {
	if !state.outputLineInfo {
		return nil
	}

	parentLine := 1
	if state.parentSourceStart != nil {
		parentLine = state.parentSourceStart.Line
	}

	parentCol := 1
	if state.parentSourceStart != nil {
		parentCol = state.parentSourceStart.Column
	}

	column := toAbsColumn(
		parentCol,
		// Subtract 1 to account for the "$" code point in the "${"
		// indicating a potential start of a substitution that leads
		// to us taking the previous string value as a string literal.
		state.relativeLineInfo.Column-len(stringVal)-1,
		state.relativeLineInfo.Line == 0,
		state.ignoreParentColumn,
	)

	colAccuracy := determineStateColumnAccuracy(state)
	return &source.Meta{
		Position: source.Position{
			Line:   toAbsLine(parentLine, state.relativeLineInfo.Line),
			Column: column,
		},
		EndPosition: &source.Position{
			Line: toAbsLine(parentLine, state.relativeLineInfo.Line),
			Column: toAbsColumn(
				parentCol,
				// Subtract 1 to account for the "$" code point in the "${"
				// indicating a potential start of a substitution that leads
				// to us taking the previous string value as a string literal.
				state.relativeLineInfo.Column-1,
				state.relativeLineInfo.Line == 0,
				state.ignoreParentColumn,
			),
		},
		ColumnAccuracy: &colAccuracy,
	}
}

func checkStringLiteral(state *interpolationParseState, value string, i int) {
	char, _ := utf8.DecodeRuneInString(value[i:])
	if char == '"' && state.prevChar != '\\' && state.inPossibleSub {
		state.inStringLiteral = !state.inStringLiteral
	}
}

func checkCloseSubBracket(state *interpolationParseState, value string, i int, substitutionContext string) bool {
	char, _ := utf8.DecodeRuneInString(value[i:])
	isCloseSubBracket := char == '}' && state.inPossibleSub && !state.inStringLiteral
	if isCloseSubBracket {
		// End of a substitution
		subSourceStart := toAbsSourceMeta(
			state.parentSourceStart,
			state.relativeSubStart,
			state.ignoreParentColumn,
			state.parentContextPrecedingCharCount,
		)
		parsedSub, err := ParseSubstitution(
			substitutionContext,
			state.potentialSub,
			subSourceStart,
			state.outputLineInfo,
			state.ignoreParentColumn,
		)
		if err != nil {
			state.errors = append(state.errors, err)
		} else {
			sourceMeta := createSubstitutionSourceMeta(state)
			state.parsed = append(state.parsed, &StringOrSubstitution{
				SubstitutionValue: parsedSub,
				SourceMeta:        sourceMeta,
			})
		}
		state.potentialSub = ""
		state.potentialNonSubStr = ""
		state.inPossibleSub = false
	}
	return isCloseSubBracket
}

func createSubstitutionSourceMeta(state *interpolationParseState) *source.Meta {
	if !state.outputLineInfo {
		return nil
	}

	parentLine := 1
	if state.parentSourceStart != nil {
		parentLine = state.parentSourceStart.Line
	}

	parentStartCol := toFinalParentStartCol(
		state.parentSourceStart,
	)

	colAccuracy := determineStateColumnAccuracy(state)
	return &source.Meta{
		Position: source.Position{
			Line: toAbsLine(parentLine, state.relativeSubStart.Line),
			Column: toAbsColumn(
				parentStartCol,
				state.relativeSubStart.Column,
				state.relativeSubStart.Line == 0,
				state.ignoreParentColumn,
			),
		},
		EndPosition: &source.Position{
			Line: toAbsLine(parentLine, state.relativeLineInfo.Line),
			Column: toAbsColumn(
				parentStartCol,
				toFinalEndCol(state.parentSourceStart, state.relativeLineInfo.Column),
				state.relativeLineInfo.Line == 0,
				state.ignoreParentColumn,
			),
		},
		ColumnAccuracy: &colAccuracy,
	}
}

func toFinalParentStartCol(parentSourceStart *source.Meta) int {
	parentStartCol := 1
	if parentSourceStart != nil {
		// Subtract 2 to account for "${" that allows accurate column position reporting
		// for the wrapper Substitution nodes at the top level of parsing
		// a substitution.
		// For example, if there is an error with the substitution as a whole, the column
		// reported should be the start of the "${" that wraps the substitution.
		// This would be reflected to the user by something like
		// range highlighting in an editor.
		parentStartCol = parentSourceStart.Column - 2
	}

	return parentStartCol
}

func toFinalEndCol(parentSourceStart *source.Meta, relativeLineInfoCol int) int {
	if parentSourceStart != nil {
		// Add 3 to include the "${" deducted from the parent start column
		// and include the closing "}" of the substitution.
		return relativeLineInfoCol + 3
	}
	return relativeLineInfoCol
}

func toAbsSourceMeta(
	parentSourceStart, relativeSubStart *source.Meta,
	ignoreParentColumn bool,
	parentPrecedingCharCount int,
) *source.Meta {
	if parentSourceStart == nil {
		return &source.Meta{
			Position: source.Position{
				Line:   relativeSubStart.Line + 1,
				Column: relativeSubStart.Column + 1,
			},
		}
	}

	return &source.Meta{
		Position: source.Position{
			Line: toAbsLine(parentSourceStart.Line, relativeSubStart.Line),
			Column: toAbsColumn(
				parentSourceStart.Column+parentPrecedingCharCount,
				relativeSubStart.Column,
				relativeSubStart.Line == 0,
				ignoreParentColumn,
			),
		},
	}
}

func toAbsColumn(
	parentColumn,
	relativeColumn int,
	sameLineAsParent bool,
	ignoreParentColumn bool,
) int {
	if ignoreParentColumn {
		return relativeColumn
	}

	if sameLineAsParent {
		return parentColumn + relativeColumn

	}
	return relativeColumn
}

func toAbsLine(parentLine, relativeLine int) int {
	return parentLine + relativeLine
}

func determineStateColumnAccuracy(state *interpolationParseState) source.ColumnAccuracy {
	if state.ignoreParentColumn {
		return source.ColumnAccuracyApproximate
	}
	return source.ColumnAccuracyExact
}

// ParseSubstitution parses a string that represents a substitution
// that is the contents of an interpolated "${..}" block.
func ParseSubstitution(
	substitutionContext string,
	substitutionInput string,
	parentSourceStart *source.Meta,
	outputLineInfo bool,
	ignoreParentColumn bool,
) (*Substitution, error) {
	tokens, err := lex(substitutionInput, parentSourceStart)
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens, parentSourceStart, outputLineInfo, ignoreParentColumn)
	return parser.Parse()
}
