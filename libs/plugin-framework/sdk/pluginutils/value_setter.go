package pluginutils

import (
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValueSetter is a helper struct that can be used to set a value in a
// from a resource spec or other framework-specific use of a mapping node
// to a target API-specific struct used to update
// or create a resource or link in the upstream provider.
type ValueSetter[Target any] struct {
	path         string
	setValueFunc func(
		value *core.MappingNode,
		target Target,
	)
	checkIfChanged bool
	modifiedFields []string
	didSet         bool
}

// ValueSetterOption is a functional option type for configuring
// a ValueSetter.
type ValueSetterOption[Target any] func(*ValueSetter[Target])

// WithValueSetterCheckIfChanged is an option that can be used to
// configure the ValueSetter to check if the value has changed
// before setting it.
// This is useful for avoiding unnecessary updates to the target
// when the current operation is for an update but is only used
// when the setter is configured with a modified fields list
// for the current context.
func WithValueSetterCheckIfChanged[Target any](
	checkIfChanged bool,
) ValueSetterOption[Target] {
	return func(setter *ValueSetter[Target]) {
		setter.checkIfChanged = checkIfChanged
	}
}

// WithValueSetterModifiedFields is an option that can be used to
// configure the ValueSetter with a list of modified fields.
// This is useful for the setter to know which fields have been modified
// in the current context, so it can check if the value has changed
// before setting it, if the setter is configured with
// WithValueSetterCheckIfChanged.
//
// The pathRoot is used to determine the root object for to use in the modified
// field paths. For example, if the pathRoot is "spec",
// any modified fields that have the prefix "spec(\.|\[)" will be
// replaced with "$(\.|\[)" to compare with the value path.
// pathRoot can be set to an empty string if the modified fields are already
// relative to the root of the path, a "$." prefix is added to the modified field paths
// when comparing with the value path.
// pathRoot must be an exact match with the prefix of the modified field paths.
func WithValueSetterModifiedFields[Target any](
	modifiedFields []provider.FieldChange,
	pathRoot string,
) ValueSetterOption[Target] {
	return func(setter *ValueSetter[Target]) {
		setter.modifiedFields = commoncore.Map(
			modifiedFields,
			func(change provider.FieldChange, _ int) string {
				if pathRoot != "" && strings.HasPrefix(change.FieldPath, pathRoot) {
					return "$" + change.FieldPath[len(pathRoot):]
				}

				// A prefix of ["string.literal.name"] should become $["string.literal.name"].
				if strings.HasPrefix(change.FieldPath, "[") {
					return "$" + change.FieldPath
				}

				return "$." + change.FieldPath
			},
		)
	}
}

// NewValueSetter creates a new value setter for the given path and setter function
// that will set the value in the target struct if the path exists in a given value
// mapping node, optionally checking if the value has changed based on a pre-computed
// list of modified fields.
func NewValueSetter[Target any](
	path string,
	setValueFunc func(
		value *core.MappingNode,
		target Target,
	),
	opts ...ValueSetterOption[Target],
) *ValueSetter[Target] {
	setter := &ValueSetter[Target]{
		path:         path,
		setValueFunc: setValueFunc,
	}

	for _, opt := range opts {
		opt(setter)
	}

	return setter
}

// Sets the value in the target struct if the value exists in the given
// mapping node at the specified path. If checkIfChanged is true, it will
// only set the value if the path is in the configured modified fields list.
func (s *ValueSetter[Target]) Set(
	parent *core.MappingNode,
	target Target,
) {
	value, hasValue := GetValueByPath(s.path, parent)
	if !hasValue {
		return
	}

	if s.checkIfChanged {
		hasChanged := slices.Contains(s.modifiedFields, s.path)
		if !hasChanged {
			return
		}
	}

	s.setValueFunc(value, target)
	s.didSet = true
}

// DidSet checks if the value was set by the setter, this is useful
// to determine if an update should take place based on whether or not
// any values have been set on the target.
func (u *ValueSetter[Target]) DidSet() bool {
	return u.didSet
}
