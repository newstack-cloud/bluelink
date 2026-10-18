package core

// Logger provides a common interface for logging used throughout
// the blueprint framework and layers built on top of it.
type Logger interface {
	// Info logs a message at the info level.
	// This message includes any fields passed into the log call,
	// as well as any fields that have been added to the logger.
	Info(msg string, fields ...LogField)
	// Debug logs a message at the debug level.
	// This message includes any fields passed into the log call,
	// as well as any fields that have been added to the logger.
	Debug(msg string, fields ...LogField)
	// Warn logs a message at the warn level.
	// This message includes any fields passed into the log call,
	// as well as any fields that have been added to the logger.
	Warn(msg string, fields ...LogField)
	// Error logs a message at the error level.
	// This message includes any fields passed into the log call,
	// as well as any fields that have been added to the logger.
	Error(msg string, fields ...LogField)
	// Fatal logs a message at the fatal level.
	// This message includes any fields passed into the log call,
	// as well as any fields that have been added to the logger.
	// After logging the message, the application will exit with
	// a non-zero exit code.
	Fatal(msg string, fields ...LogField)
	// WithFields returns a new logger enrich with the specified
	// fields that will be included in all log messages for the
	// returned logger.
	WithFields(fields ...LogField) Logger
	// Named returns a new logger with the specified name that will
	// be included in all log messages for the returned logger.
	// Multiple nested names will be join with a period.
	// Names are useful as they can be used to filter logs to only surface
	// logs from a specific section of functionality.
	// (e.g. "parent" -> "child" -> "grandchild" would be "parent.child.grandchild")
	Named(name string) Logger
}

// LogField represents a key-value pair that can be used to add
// additional context to a log message.
// This supports log fields that are strings, integers, floats, booleans,
// errors or arrays of the above types.
// Nested structures are not supported, but can be flattened into a
// a set of top-level fields or serialised into a string that can
// be included as a single field.
type LogField struct {
	Type      LogFieldType
	Key       string
	String    string
	Integer   int64
	Float     float64
	Bool      bool
	Err       error
	Interface interface{}
}

// StringLogField creates a new log field with a string value that can
// be used to add additional context to a log message.
func StringLogField(key, value string) LogField {
	return LogField{
		Type:   StringLogFieldType,
		Key:    key,
		String: value,
	}
}

// IntegerLogField creates a new log field with an integer value that can
// be used to add additional context to a log message.
func IntegerLogField(key string, value int64) LogField {
	return LogField{
		Type:    IntegerLogFieldType,
		Key:     key,
		Integer: value,
	}
}

// FloatLogField creates a new log field with a float value that can
// be used to add additional context to a log message.
func FloatLogField(key string, value float64) LogField {
	return LogField{
		Type:  FloatLogFieldType,
		Key:   key,
		Float: value,
	}
}

// BoolLogField creates a new log field with a boolean value that can
// be used to add additional context to a log message.
func BoolLogField(key string, value bool) LogField {
	return LogField{
		Type: BoolLogFieldType,
		Key:  key,
		Bool: value,
	}
}

// ErrorLogField creates a new log field with an error value that can
// be used to add additional context to a log message.
func ErrorLogField(key string, value error) LogField {
	return LogField{
		Type: ErrorLogFieldType,
		Key:  key,
		Err:  value,
	}
}

// StringsLogField creates a new log field with an array of string values
// that can be used to add additional context to a log message.
func StringsLogField(key string, values []string) LogField {
	return LogField{
		Type:      StringsLogFieldType,
		Key:       key,
		Interface: values,
	}
}

// IntegersLogField creates a new log field with an array of integer values
// that can be used to add additional context to a log message.
func IntegersLogField(key string, values []int64) LogField {
	return LogField{
		Type:      IntegersLogFieldType,
		Key:       key,
		Interface: values,
	}
}

// FloatsLogField creates a new log field with an array of float values
// that can be used to add additional context to a log message.
func FloatsLogField(key string, values []float64) LogField {
	return LogField{
		Type:      FloatsLogFieldType,
		Key:       key,
		Interface: values,
	}
}

// BoolsLogField creates a new log field with an array of boolean values
// that can be used to add additional context to a log message.
func BoolsLogField(key string, values []bool) LogField {
	return LogField{
		Type:      BoolsLogFieldType,
		Key:       key,
		Interface: values,
	}
}

// ErrorsLogField creates a new log field with an array of error values
// that can be used to add additional context to a log message.
func ErrorsLogField(key string, values []error) LogField {
	return LogField{
		Type:      ErrorsLogFieldType,
		Key:       key,
		Interface: values,
	}
}

// LogFieldType represents the type of a log field
// that is used to determine which value to use when
// serialising the field to a log message.
type LogFieldType int

const (
	// StringLogFieldType represents a log field with a string value.
	StringLogFieldType LogFieldType = iota
	// IntegerLogFieldType represents a log field with an integer value.
	IntegerLogFieldType
	// FloatLogFieldType represents a log field with a float value.
	FloatLogFieldType
	// BoolLogFieldType represents a log field with a boolean value.
	BoolLogFieldType
	// ErrorLogFieldType represents a log field with an error value.
	ErrorLogFieldType
	// StringsLogFieldType represents a log field with an array of string values.
	StringsLogFieldType
	// IntegersLogFieldType represents a log field with an array of integer values.
	IntegersLogFieldType
	// FloatsLogFieldType represents a log field with an array of float values.
	FloatsLogFieldType
	// BoolsLogFieldType represents a log field with an array of boolean values.
	BoolsLogFieldType
	// ErrorsLogFieldType represents a log field with an array of error values.
	ErrorsLogFieldType
	// logFieldTypeCount is one past the last log field type, so that a type added above it
	// is caught by the test that every type is handled when serialised.
	logFieldTypeCount
)

// NopLogger is an implementation of the core.Logger interface
// that does nothing when log messages are sent to it.
type NopLogger struct{}

// NewNopLogger creates a new instance of the no-op logger
// that will do nothing when log messages are sent to it.
func NewNopLogger() Logger {
	return &NopLogger{}
}

func (l *NopLogger) Info(msg string, fields ...LogField) {
	// no-op does nothing for info logs.
}

func (l *NopLogger) Debug(msg string, fields ...LogField) {
	// no-op does nothing for debug logs.
}

func (l *NopLogger) Warn(msg string, fields ...LogField) {
	// no-op does nothing for warning logs.
}

func (l *NopLogger) Error(msg string, fields ...LogField) {
	// no-op does nothing for error logs.
}

func (l *NopLogger) Fatal(msg string, fields ...LogField) {
	// no-op does nothing for fatal logs.
}

func (l *NopLogger) WithFields(fields ...LogField) Logger {
	return l
}

func (l *NopLogger) Named(name string) Logger {
	return l
}
