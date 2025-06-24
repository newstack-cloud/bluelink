package utils

import (
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/spf13/afero"
)

var (
	// ErrUnsupportedBlueprintFormat is the error message for an unsupported blueprint format.
	ErrUnsupportedBlueprintFormat = errors.New("unsupported blueprint format")
)

// BlueprintFormatFromExtension determines the blueprint input
// format based on the file extension.
// An error is returned if the extension is not supported.
func BlueprintFormatFromExtension(filePath string) (schema.SpecFormat, error) {
	if strings.HasSuffix(filePath, ".json") ||
		strings.HasSuffix(filePath, ".jsonc") ||
		strings.HasSuffix(filePath, ".hujson") {
		return schema.JWCCSpecFormat, nil
	} else if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		return schema.YAMLSpecFormat, nil
	}

	parts := strings.Split(filePath, afero.FilePathSeparator)
	fileName := parts[len(parts)-1]
	return "", errUnsupportedBlueprintFormat(fileName)
}

// HasAtLeastOneError checks if there is at least one error
// in the provided diagnostics slice.
func HasAtLeastOneError(diagnostics []*core.Diagnostic) bool {
	return slices.ContainsFunc(diagnostics, func(d *core.Diagnostic) bool {
		return d.Level == core.DiagnosticLevelError
	})
}

// DiagnosticsFromBlueprintValidationError extracts diagnostics from a blueprint
// validation error.
func DiagnosticsFromBlueprintValidationError(
	err error,
	logger core.Logger,
	fallbackToGeneralDiagnostic bool,
) []*core.Diagnostic {
	diagnostics := []*core.Diagnostic{}

	if err == nil {
		logger.Debug("no error to convert to diagnostics")
		return diagnostics
	}

	logger.Debug("converting blueprint validation error to diagnostics", core.ErrorLogField("error", err))
	loadErr, isLoadErr := err.(*bperrors.LoadError)
	if isLoadErr {
		collectLoadErrors(loadErr, &diagnostics, nil, logger)
		return diagnostics
	}

	schemaErr, isSchemaErr := err.(*schema.Error)
	if isSchemaErr {
		collectSchemaError(schemaErr, &diagnostics)
		return diagnostics
	}

	_, isRunErr := err.(*bperrors.RunError)
	if isRunErr {
		// Skip capturing run errors during validation,
		// they are useful at runtime but may appear during validation
		// in loading provider and transformer plugins.
		return diagnostics
	}

	if !fallbackToGeneralDiagnostic {
		// In contexts where the error should be treated differently
		// from a diagnostic, we don't want to produce a diagnostic.
		return diagnostics
	}

	return getGeneralErrorDiagnostics(err)
}

func getGeneralErrorDiagnostics(err error) []*core.Diagnostic {
	level := core.DiagnosticLevelError
	return []*core.Diagnostic{
		{
			Range: &core.DiagnosticRange{
				Start: &source.Meta{Position: source.Position{
					Line:   0,
					Column: 0,
				}},
				End: &source.Meta{Position: source.Position{
					Line:   1,
					Column: 0,
				}},
			},
			Level:   level,
			Message: err.Error(),
		},
	}
}

func collectLoadErrors(
	err *bperrors.LoadError,
	diagnostics *[]*core.Diagnostic,
	parentLoadErr *bperrors.LoadError,
	logger core.Logger,
) {
	logger.Debug(
		"load error",
		core.StringLogField("error", err.Error()),
		core.IntegerLogField("child error count", int64(len(err.ChildErrors))),
	)

	if len(err.ChildErrors) == 0 {
		level := core.DiagnosticLevelError
		line, col := positionFromParentLoadError(parentLoadErr)
		*diagnostics = append(*diagnostics, &core.Diagnostic{
			Range: &core.DiagnosticRange{
				Start: &source.Meta{Position: source.Position{
					Line:   line,
					Column: col,
				}},
				End: &source.Meta{Position: source.Position{
					Line:   line + 1,
					Column: 0,
				}},
			},
			Level:   level,
			Message: err.Error(),
		})
	}

	for _, childErr := range err.ChildErrors {
		logger.Debug("child error type", core.StringLogField("type", reflect.TypeOf(err.Err).String()))
		childLoadErr, isLoadErr := childErr.(*bperrors.LoadError)
		if isLoadErr {
			logger.Debug("child load error", core.StringLogField("error", childLoadErr.Error()))
			collectLoadErrors(childLoadErr, diagnostics, err, logger)
		}

		childSchemaErr, isSchemaErr := childErr.(*schema.Error)
		if isSchemaErr {
			logger.Debug("child schema error", core.StringLogField("error", childSchemaErr.Error()))
			collectSchemaError(childSchemaErr, diagnostics)
		}

		childParseErrs, isParseErrs := childErr.(*substitutions.ParseErrors)
		if isParseErrs {
			collectParseErrors(childParseErrs, diagnostics, err)
		}

		childParseErr, isParseErr := childErr.(*substitutions.ParseError)
		if isParseErr {
			collectParseError(childParseErr, diagnostics)
		}

		childCoreErr, isCoreErr := childErr.(*core.Error)
		if isCoreErr {
			collectCoreError(childCoreErr, diagnostics, err)
		}

		childLexErrors, isLexErrs := childErr.(*substitutions.LexErrors)
		if isLexErrs {
			collectLexErrors(childLexErrors, diagnostics, err)
		}

		childLexError, isLexErr := childErr.(*substitutions.LexError)
		if isLexErr {
			collectLexError(childLexError, diagnostics)
		}

		_, isRunErr := childErr.(*bperrors.RunError)
		if isRunErr {
			// Skip capturing run errors during validation,
			// they are useful at runtime but may appear during validation
			// in loading provider and transformer plugins.
			return
		}

		if !isLoadErr && !isSchemaErr && !isParseErrs &&
			!isParseErr && !isCoreErr && !isLexErrs && !isLexErr {
			collectGeneralError(childErr, diagnostics, err)
		}
	}
}

func collectParseErrors(
	errs *substitutions.ParseErrors,
	diagnostics *[]*core.Diagnostic,
	parentLoadError *bperrors.LoadError,
) {
	for _, childErr := range errs.ChildErrors {
		childParseErr, isParseError := childErr.(*substitutions.ParseError)
		if isParseError {
			collectParseError(childParseErr, diagnostics)
		}
	}

	if len(errs.ChildErrors) == 0 {
		level := core.DiagnosticLevelError
		line, col := positionFromParentLoadError(parentLoadError)
		*diagnostics = append(*diagnostics, &core.Diagnostic{
			Range: &core.DiagnosticRange{
				Start: &source.Meta{Position: source.Position{
					Line:   line,
					Column: col,
				}},
				End: &source.Meta{Position: source.Position{
					Line:   line + 1,
					Column: 0,
				}},
			},
			Level:   level,
			Message: errs.Error(),
		})
	}
}

func collectParseError(
	err *substitutions.ParseError,
	diagnostics *[]*core.Diagnostic,
) {
	level := core.DiagnosticLevelError
	*diagnostics = append(*diagnostics, &core.Diagnostic{
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{
				Line:   err.Line,
				Column: err.Column,
			}},
			End: &source.Meta{Position: source.Position{
				Line:   err.Line + 1,
				Column: 0,
			}},
		},
		Level:   level,
		Message: err.Error(),
	})
}

func collectCoreError(
	err *core.Error,
	diagnostics *[]*core.Diagnostic,
	parentLoadErr *bperrors.LoadError,
) {
	line, col := positionFromCoreError(err, parentLoadErr)
	level := core.DiagnosticLevelError
	*diagnostics = append(*diagnostics, &core.Diagnostic{
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{
				Line:   line,
				Column: col,
			}},
			End: &source.Meta{Position: source.Position{
				Line:   line + 1,
				Column: 0,
			}},
		},
		Level:   level,
		Message: err.Error(),
	})
}

func collectLexErrors(
	errs *substitutions.LexErrors,
	diagnostics *[]*core.Diagnostic,
	parentLoadErr *bperrors.LoadError,
) {
	for _, childErr := range errs.ChildErrors {
		childLexErr, isLexError := childErr.(*substitutions.LexError)
		if isLexError {
			collectLexError(childLexErr, diagnostics)
		}
	}

	if len(errs.ChildErrors) == 0 {
		line, col := positionFromParentLoadError(parentLoadErr)
		level := core.DiagnosticLevelError
		*diagnostics = append(*diagnostics, &core.Diagnostic{
			Range: &core.DiagnosticRange{
				Start: &source.Meta{Position: source.Position{
					Line:   line,
					Column: col,
				}},
				End: &source.Meta{Position: source.Position{
					Line:   line + 1,
					Column: 0,
				}},
			},
			Level:   level,
			Message: errs.Error(),
		})
	}
}

func collectLexError(
	err *substitutions.LexError,
	diagnostics *[]*core.Diagnostic,
) {
	line, col := positionFromLexError(err)
	level := core.DiagnosticLevelError
	*diagnostics = append(*diagnostics, &core.Diagnostic{
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{
				Line:   line,
				Column: col,
			}},
			End: &source.Meta{Position: source.Position{
				Line:   line + 1,
				Column: 0,
			}},
		},
		Level:   level,
		Message: err.Error(),
	})
}

func positionFromLexError(
	err *substitutions.LexError,
) (int, int) {
	col := 0
	if err.ColumnAccuracy == substitutions.ColumnAccuracyExact {
		col = err.Column
	}

	return err.Line, col
}

func collectSchemaError(
	err *schema.Error,
	diagnostics *[]*core.Diagnostic,
) {
	line, col := positionFromSchemaError(err)
	level := core.DiagnosticLevelError
	*diagnostics = append(*diagnostics, &core.Diagnostic{
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{
				Line:   line,
				Column: col,
			}},
			End: &source.Meta{Position: source.Position{
				Line:   line + 1,
				Column: 0,
			}},
		},
		Level:   level,
		Message: err.Error(),
	})
}

func positionFromSchemaError(
	err *schema.Error,
) (int, int) {
	line := 0
	if err.SourceLine != nil {
		line = *err.SourceLine
	}

	col := 0
	if err.SourceColumn != nil {
		col = *err.SourceColumn
	}

	return line, col
}

func positionFromParentLoadError(
	parentLoadErr *bperrors.LoadError,
) (int, int) {
	parentMissingLines := parentLoadErr == nil || parentLoadErr.Line == nil && parentLoadErr.Column == nil

	if parentMissingLines {
		return 0, 0
	}

	return *parentLoadErr.Line, *parentLoadErr.Column
}

func positionFromCoreError(
	err *core.Error,
	parentLoadErr *bperrors.LoadError,
) (int, int) {
	errMissingLines := err.SourceLine == nil && err.SourceColumn == nil
	parentMissingLines := parentLoadErr == nil || parentLoadErr.Line == nil && parentLoadErr.Column == nil

	if errMissingLines && parentMissingLines {
		return 0, 0
	}

	if errMissingLines {
		return *parentLoadErr.Line, *parentLoadErr.Column
	}

	return *err.SourceLine, *err.SourceColumn
}

func collectGeneralError(err error, diagnostics *[]*core.Diagnostic, parentLoadError *bperrors.LoadError) {
	level := core.DiagnosticLevelError
	line, col := positionFromParentLoadError(parentLoadError)
	*diagnostics = append(*diagnostics, &core.Diagnostic{
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{
				Line:   line,
				Column: col,
			}},
			End: &source.Meta{Position: source.Position{
				Line:   line + 1,
				Column: 0,
			}},
		},
		Level:   level,
		Message: err.Error(),
	})
}
