package core

import (
	"fmt"
	"regexp"
)

// Secret wraps a configuration value that was declared as holding a secret, so that it
// cannot reach output by accident.
//
// Every way of rendering a Secret reports RedactedPlaceholder rather than the value: the
// fmt verbs, JSON serialisation, and the %#v form. A value only leaves by way of Reveal,
// which is a deliberate and greppable act.
//
// This is the compile-time half of keeping secrets out of output. Masking at the log or
// diagnostic boundary catches a value whose shape is known at that point; nothing there can
// catch a value a plugin has already interpolated into a message it composed. Taking the
// secret as a type that redacts itself means the usual mistake, putting the value into text
// with %s or %v, produces the placeholder instead.
type Secret struct {
	value *ScalarValue
}

// NewSecret wraps a value so that it redacts itself wherever it is rendered.
func NewSecret(value *ScalarValue) Secret {
	return Secret{value: value}
}

// Reveal returns the wrapped value.
//
// The one way out, for the single place it is needed, handing the value to the service it
// configures. Anything that renders the result is back to handling a bare value.
func (s Secret) Reveal() *ScalarValue {
	return s.value
}

// String reports the placeholder, so a Secret in a %s or %v position redacts itself.
func (s Secret) String() string {
	return RedactedPlaceholder
}

// GoString reports the placeholder, so the %#v form redacts as well. Without it, %#v
// prints the struct and the pointer it holds.
func (s Secret) GoString() string {
	return RedactedPlaceholder
}

// Format reports the placeholder for every verb.
//
// String alone covers %s and %v. A numeric verb on a Secret holding an integer, or %q,
// would otherwise fall back to reflection and print the value.
func (s Secret) Format(state fmt.State, verb rune) {
	if verb == 'q' {
		fmt.Fprintf(state, "%q", RedactedPlaceholder)
		return
	}

	fmt.Fprint(state, RedactedPlaceholder)
}

// MarshalJSON reports the placeholder, so a structure holding a Secret can be serialised
// without carrying the value into the output.
func (s Secret) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", RedactedPlaceholder), nil
}

// SecretValue retrieves a config value that this definition declares as holding a secret,
// wrapped so that it redacts itself.
//
// Reports false when the key is absent or when the field that matches it is not declared
// secret, so that a caller reaching for a secret on a field nobody declared as one finds
// nothing rather than quietly receiving a value it will treat as protected.
func (d *ConfigDefinition) SecretValue(config PluginConfig, key string) (Secret, bool) {
	value, hasValue := config.Get(key)
	if !hasValue || d == nil {
		return Secret{}, false
	}

	for fieldName, fieldDef := range d.Fields {
		if fieldDef == nil || !fieldDef.Secret {
			continue
		}

		if configKeyMatchesFieldName(key, fieldName) {
			return NewSecret(value), true
		}
	}

	return Secret{}, false
}

// Reports whether a concrete config key is the field this definition names, which for a
// field whose name is a template is any key the template matches.
func configKeyMatchesFieldName(key string, fieldName string) bool {
	if !IsDynamicFieldName(fieldName) {
		return key == fieldName
	}

	pattern, err := regexp.Compile(
		"^" + CreatePatternForDynamicFieldName(fieldName /* maxPlaceholders */, -1) + "$",
	)
	if err != nil {
		return false
	}

	return pattern.MatchString(key)
}
