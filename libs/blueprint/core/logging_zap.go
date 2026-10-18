package core

import (
	"go.uber.org/zap"
)

type loggerFromZap struct {
	zapLogger *zap.Logger
}

// NewLoggerFromZap creates a new instance of a blueprint framework logger
// that is backed by a zap logger.
func NewLoggerFromZap(zapLogger *zap.Logger) Logger {
	return &loggerFromZap{
		zapLogger,
	}
}

func (l *loggerFromZap) Debug(message string, fields ...LogField) {
	l.zapLogger.Debug(message, convertLogFieldsToZap(fields)...)
}

func (l *loggerFromZap) Info(message string, fields ...LogField) {
	l.zapLogger.Info(message, convertLogFieldsToZap(fields)...)
}

func (l *loggerFromZap) Warn(message string, fields ...LogField) {
	l.zapLogger.Warn(message, convertLogFieldsToZap(fields)...)
}

func (l *loggerFromZap) Error(message string, fields ...LogField) {
	l.zapLogger.Error(message, convertLogFieldsToZap(fields)...)
}

func (l *loggerFromZap) Fatal(message string, fields ...LogField) {
	l.zapLogger.Fatal(message, convertLogFieldsToZap(fields)...)
}

func (l *loggerFromZap) WithFields(fields ...LogField) Logger {
	return &loggerFromZap{
		zapLogger: l.zapLogger.With(convertLogFieldsToZap(fields)...),
	}
}

func (l *loggerFromZap) Named(name string) Logger {
	return &loggerFromZap{
		zapLogger: l.zapLogger.Named(name),
	}
}

func convertLogFieldsToZap(fields []LogField) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		zapFields = append(zapFields, convertLogFieldToZap(field))
	}
	return zapFields
}

func convertLogFieldToZap(field LogField) zap.Field {
	switch field.Type {
	case StringLogFieldType:
		return zap.String(field.Key, field.String)
	case IntegerLogFieldType:
		return zap.Int64(field.Key, field.Integer)
	case FloatLogFieldType:
		return zap.Float64(field.Key, field.Float)
	case BoolLogFieldType:
		return zap.Bool(field.Key, field.Bool)
	case ErrorLogFieldType:
		return zap.Error(field.Err)
	case StringsLogFieldType:
		strings, ok := field.Interface.([]string)
		if !ok {
			strings = []string{}
		}
		return zap.Strings(field.Key, strings)
	case IntegersLogFieldType:
		integers, ok := field.Interface.([]int64)
		if !ok {
			integers = []int64{}
		}
		return zap.Int64s(field.Key, integers)
	case FloatsLogFieldType:
		floats, ok := field.Interface.([]float64)
		if !ok {
			floats = []float64{}
		}
		return zap.Float64s(field.Key, floats)
	case BoolsLogFieldType:
		bools, ok := field.Interface.([]bool)
		if !ok {
			bools = []bool{}
		}
		return zap.Bools(field.Key, bools)
	case ErrorsLogFieldType:
		errors, ok := field.Interface.([]error)
		if !ok {
			errors = []error{}
		}
		return zap.Errors(field.Key, errors)
	default:
		// Withheld rather than serialised. A field whose type this does not handle is one
		// nothing here knows the shape of, and zap.Any would write whatever it holds, which
		// for a type added without a case above could be a resource spec or link data. The
		// placeholder keeps the field visible so the omission is noticed.
		return zap.String(field.Key, RedactedPlaceholder)
	}
}

// LogFieldTypes lists every log field type the adapter handles explicitly.
//
// Exported so that adding a type without handling it fails a test rather than silently
// falling through to the placeholder above.
func LogFieldTypes() []LogFieldType {
	return []LogFieldType{
		StringLogFieldType,
		IntegerLogFieldType,
		FloatLogFieldType,
		BoolLogFieldType,
		ErrorLogFieldType,
		StringsLogFieldType,
		IntegersLogFieldType,
		FloatsLogFieldType,
		BoolsLogFieldType,
		ErrorsLogFieldType,
	}
}
