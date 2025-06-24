package substitutions

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

const (
	// ErrorReasonCodeInvalidReferenceSub is provided when the reason
	// for a blueprint spec load error is due to one or more reference substitutions
	// being invalid.
	ErrorReasonCodeInvalidReferenceSub errors.ErrorReasonCode = "invalid_reference_substitution"
	// ErrorReasonCodeStringOrSubsInvalidType is provided when the reason
	// for a blueprint spec load error is due to a value in a source document
	// that is expected to be a string with 0 or more ${..} substitutions
	// is of a different type.
	ErrorReasonCodeStringOrSubsInvalidType errors.ErrorReasonCode = "invalid_string_or_substitution_type"

	childErrorsFormatStr = "\n\t- %s"
)

func errStringOrSubsInvalidType(
	position source.PositionInfo,
	parentPath string,
) error {
	line := position.GetLine()
	column := position.GetColumn()
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeStringOrSubsInvalidType,
		Err: fmt.Errorf(
			"validation failed due to a value in the source document that is expected to be a string with 0 or more ${..} substitutions "+
				"being of a different type at %q",
			parentPath,
		),
		Line:   &line,
		Column: &column,
	}
}

func errSubstitutions(
	substitutionContext string,
	substitutionErrors []error,
	line *int,
	column *int,
) error {
	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeInvalidReferenceSub,
		Err:         errSubstitutionsMessage(substitutionContext),
		ChildErrors: substitutionErrors,
		Line:        line,
		Column:      column,
	}
}

func errSerialiseSubstitutions(
	substitutionContext string,
	substitutionErrors []error,
) error {
	return &errors.SerialiseError{
		ReasonCode:  ErrorReasonCodeInvalidReferenceSub,
		Err:         errSubstitutionsMessage(substitutionContext),
		ChildErrors: substitutionErrors,
	}
}

func errSubstitutionsMessage(substitutionContext string) error {
	if len(substitutionContext) > 0 {
		return fmt.Errorf(
			"validation failed due to one or more invalid reference substitutions having been provided for \"%s\"",
			substitutionContext,
		)
	}

	return fmt.Errorf(
		"validation failed due to one or more invalid reference substitutions having been provided",
	)
}

func errSerialiseSubstitutionInvalidVariableName(
	variableName string,
) error {
	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid variable name \"%s\" having been provided in a reference substitution",
			variableName,
		),
	}
}

func errSerialiseSubstitutionInvalidDataSourceName(
	dataSourceName string,
) error {
	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid data source name \"%s\" having been provided in a reference substitution",
			dataSourceName,
		),
	}
}

func errSerialiseSubstitutionInvalidDataSourcePath(
	path string,
	dataSourceName string,
) error {
	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid data source path \"%s\" for data source \"%s\" having been provided in a reference substitution",
			path,
			dataSourceName,
		),
	}
}

func errSerialiseSubstitutionInvalidResourceName(
	resourceName string,
) error {
	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid resource name \"%s\" having been provided in a reference substitution",
			resourceName,
		),
	}
}

func errSerialiseSubstitutionInvalidChildName(
	childName string,
) error {
	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid child blueprint name \"%s\" having been provided in a reference substitution",
			childName,
		),
	}
}

func errSerialiseSubstitutionInvalidChildPath(
	path string,
	childName string,
	pathItemErrors []error,
) error {
	childErrStr := ""
	for _, err := range pathItemErrors {
		childErrStr += fmt.Sprintf(childErrorsFormatStr, err.Error())
	}

	if len(path) == 0 {
		return &errors.SerialiseError{
			ReasonCode: ErrorReasonCodeInvalidReferenceSub,
			Err: fmt.Errorf(
				"validation failed due to an empty path having been provided for "+
					"child blueprint \"%s\" in a reference substitution, an exported field must be specified%s",
				childName,
				childErrStr,
			),
		}
	}

	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid path \"%s\" for child blueprint \"%s\" having been provided in a reference substitution%s",
			path,
			childName,
			childErrStr,
		),
	}
}

func errSerialiseSubstitutionInvalidCurrentElementPath(
	path string,
	pathItemErrors []error,
) error {
	childErrStr := ""
	for _, err := range pathItemErrors {
		childErrStr += fmt.Sprintf(childErrorsFormatStr, err.Error())
	}

	if len(path) == 0 {
		return &errors.SerialiseError{
			ReasonCode: ErrorReasonCodeInvalidReferenceSub,
			Err: fmt.Errorf(
				"validation failed due to an empty path having been provided for "+
					"the current element in a reference substitution%s",
				childErrStr,
			),
		}
	}

	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid path \"%s\" for current element having been provided in a reference substitution%s",
			path,
			childErrStr,
		),
	}
}

func errSerialiseSubstitutionInvalidPathItem(pathItem *SubstitutionPathItem) error {
	if pathItem.ArrayIndex == nil {
		return &errors.SerialiseError{
			ReasonCode: ErrorReasonCodeInvalidReferenceSub,
			Err: fmt.Errorf(
				"validation failed due to invalid path item \"%s\" having been provided in a reference substitution",
				pathItem.FieldName,
			),
		}
	}

	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to invalid index accessor path item [\"%d\"] having been provided in a reference substitution",
			*pathItem.ArrayIndex,
		),
	}
}

func errSerialiseSubstitutionFunctionArgValueMissing() error {
	return &errors.SerialiseError{
		ReasonCode: ErrorReasonCodeInvalidReferenceSub,
		Err: fmt.Errorf(
			"validation failed due to a missing function argument value having been provided in a reference substitution",
		),
	}
}

func errLexUnexpectedEndOfInput(
	line int,
	column int,
	columnAccuracy ColumnAccuracy,
	evaluatingTokenType string,
) error {
	message := fmt.Sprintf(
		"validation failed due to an unexpected end of input having "+
			"been encountered in a reference substitution when evaluating \"%s\"",
		evaluatingTokenType,
	)
	return errLexError(message, line, column, columnAccuracy)
}

func errLexUnexpectedChar(
	line int,
	column int,
	columnAccuracy ColumnAccuracy,
	char rune,
) error {
	message := fmt.Sprintf(
		"validation failed due to an unexpected character \"%s\" having "+
			"been encountered in a reference substitution",
		string(char),
	)
	return errLexError(message, line, column, columnAccuracy)
}

// ParseError is an error that is returned
// during failure to parse a reference substitution.
type ParseError struct {
	token   *token
	message string
	Line    int
	Column  int
	// ColumnAccuracy gives a hint as to how accurate the column numbers
	// are likely to be in a host document that contains reference substitutions.
	ColumnAccuracy ColumnAccuracy
}

func (e *ParseError) Error() string {
	var errStr string
	if e.token != nil {
		errStr = fmt.Sprintf("parse error at column %d with token type %s: %s", e.Column, e.token.tokenType, e.message)
	} else {
		errStr = fmt.Sprintf("parse error at end of input: %s", e.message)
	}
	return errStr
}

func errParseError(t *token, message string, line int, col int, colAccuracy ColumnAccuracy) error {
	return &ParseError{
		token:          t,
		message:        message,
		Line:           line,
		Column:         col,
		ColumnAccuracy: colAccuracy,
	}
}

type ParseErrors struct {
	message     string
	ChildErrors []error
}

func (e *ParseErrors) Error() string {
	errStr := fmt.Sprintf("parse errors: %s", e.message)
	for _, err := range e.ChildErrors {
		errStr += fmt.Sprintf(childErrorsFormatStr, err.Error())
	}
	return errStr
}

func errParseErrorMultiple(message string, errors []error) error {
	return &ParseErrors{
		message:     message,
		ChildErrors: errors,
	}
}

type LexErrors struct {
	message     string
	ChildErrors []error
}

func (e *LexErrors) Error() string {
	errStr := fmt.Sprintf("lex errors: %s", e.message)
	for _, err := range e.ChildErrors {
		errStr += fmt.Sprintf(childErrorsFormatStr, err.Error())
	}
	return errStr
}

func errLexMultiple(message string, errors []error) error {
	return &LexErrors{
		message:     message,
		ChildErrors: errors,
	}
}

// LexError is an error that is returned
// during failure to during lexical analysis
// of a reference substitution.
type LexError struct {
	message string
	Line    int
	Column  int
	// ColumnAccuracy gives a hint as to how accurate the column numbers
	// are likely to be in a host document that contains reference substitutions.
	ColumnAccuracy ColumnAccuracy
}

func (e *LexError) Error() string {
	return fmt.Sprintf("lex error at column %d: %s", e.Column, e.message)
}

func errLexError(message string, line int, col int, colAccuracy ColumnAccuracy) error {
	return &LexError{
		message:        message,
		Line:           line,
		Column:         col,
		ColumnAccuracy: colAccuracy,
	}
}

// ColumnAccuracy is an enum that describes the accuracy of column numbers
// in a host document that contains reference substitutions.
// This is used to give a hint to the user as to how accurate the column numbers
// are likely to be in a host document for diagnostics that will be displayed to a user.
type ColumnAccuracy int

const (
	// ColumnAccuracyExact indicates that column numbers are accurate.
	ColumnAccuracyExact ColumnAccuracy = 1
	// ColumnAccuracyApproximate indicates that column numbers are approximate.
	ColumnAccuracyApproximate ColumnAccuracy = 2
)
