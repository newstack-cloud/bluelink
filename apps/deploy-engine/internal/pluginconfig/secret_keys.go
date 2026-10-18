package pluginconfig

import (
	"regexp"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// SecretKeyRegistry records which plugin config keys hold secrets, so that a log entry
// carrying one of them under its own key can be masked.
//
// This is purely for config key-based redaction, a value a plugin has already placed
// inside free text is not masked.
// Only the plugin knows which part of a message it composed came from a secret.
type SecretKeyRegistry struct {
	mu       sync.RWMutex
	keys     map[string]struct{}
	patterns []*regexp.Regexp
}

// NewSecretKeyRegistry creates an empty registry, to be filled as plugin config
// definitions are read.
func NewSecretKeyRegistry() *SecretKeyRegistry {
	return &SecretKeyRegistry{
		keys: map[string]struct{}{},
	}
}

// Register records a config field name as holding a secret.
//
// A name that is a template for dynamic field names, "aws.config.regionKMSKeys.<region>",
// is held as a pattern, since the keys it matches are only known once a configuration
// supplies them.
func (r *SecretKeyRegistry) Register(fieldName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !core.IsDynamicFieldName(fieldName) {
		r.keys[fieldName] = struct{}{}
		return
	}

	// Anchored, so a template matches a whole key rather than appearing anywhere inside one.
	pattern, err := regexp.Compile(
		"^" + core.CreatePatternForDynamicFieldName(fieldName /* maxPlaceholders */, -1) + "$",
	)
	if err != nil {
		// A definition the engine could not turn into a pattern is one it cannot match
		// against, and refusing to log at all would be worse than not masking this field.
		return
	}

	r.patterns = append(r.patterns, pattern)
}

// IsSecret reports whether a config key was registered as holding a secret.
func (r *SecretKeyRegistry) IsSecret(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, found := r.keys[key]; found {
		return true
	}

	for _, pattern := range r.patterns {
		if pattern.MatchString(key) {
			return true
		}
	}

	return false
}
