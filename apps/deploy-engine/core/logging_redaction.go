package core

import (
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SecretFieldMatcher reports whether a log field's key is one a plugin declared as holding
// a secret.
//
// An interface rather than the registry itself, so that the logger does not depend on
// plugin configuration. The registry is filled as plugin config definitions are read,
// which happens after the logger is built.
type SecretFieldMatcher interface {
	IsSecret(key string) bool
}

// Masks the value of any log field whose key a plugin declared as a secret.
type redactingCore struct {
	zapcore.Core
	matcher SecretFieldMatcher
}

func (c *redactingCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactingCore{
		Core:    c.Core.With(c.masked(fields)),
		matcher: c.matcher,
	}
}

// Adds this core rather than the one it wraps, so that Write goes through the masking
// above rather than around it.
func (c *redactingCore) Check(
	entry zapcore.Entry,
	checked *zapcore.CheckedEntry,
) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}

	return checked
}

func (c *redactingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(entry, c.masked(fields))
}

// Returns a new slice where a field is replaced rather than altering the caller's, which
// zap reuses across entries.
func (c *redactingCore) masked(fields []zapcore.Field) []zapcore.Field {
	if c.matcher == nil {
		return fields
	}

	var replaced []zapcore.Field
	for index, field := range fields {
		if !c.matcher.IsSecret(field.Key) {
			continue
		}

		if replaced == nil {
			replaced = make([]zapcore.Field, len(fields))
			copy(replaced, fields)
		}
		replaced[index] = zap.String(field.Key, bpcore.RedactedPlaceholder)
	}

	if replaced == nil {
		return fields
	}

	return replaced
}
