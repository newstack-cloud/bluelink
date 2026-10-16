package subengine

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// ResolvedValue provides a version of a value for which all ${..}
// substitutions have been applied.
type ResolvedValue struct {
	Type        *schema.ValueTypeWrapper `json:"type"`
	Value       *core.MappingNode        `json:"value"`
	Description *core.MappingNode        `json:"description,omitempty"`
	Secret      *core.ScalarValue        `json:"secret"`
	// SensitivePaths holds the paths within this value that a secret reached, rooted at
	// "value", so "value.password" for a field of an object value.
	//
	// Carried on the value because resolution memoises values. A value that declares nothing
	// but interpolates a secret would otherwise mark only the field whose reference resolved
	// it, with every later reference served from the cache and will allow for leaking secrets
	// used in fields that won't be marked.
	SensitivePaths []string `json:"sensitivePaths,omitempty"`
}

// ResolvedInclude provides a version of a child blueprint
// include for which all ${..} substitutions have been applied.
//
// This is NOT to be confused with a resolved child blueprint
// which is the result of resolving a child blueprint include
// using an implementation of the `includes.ChildResolver` interface.
type ResolvedInclude struct {
	Path        *core.MappingNode `json:"path"`
	Variables   *core.MappingNode `json:"variables,omitempty"`
	Metadata    *core.MappingNode `json:"metadata,omitempty"`
	Description *core.MappingNode `json:"description,omitempty"`
}

// ResolvedExport provides a version of an export
// for which all ${..} substitutions have been applied.
type ResolvedExport struct {
	Type        *schema.ExportTypeWrapper `json:"type"`
	Field       *core.ScalarValue         `json:"field"`
	Description *core.MappingNode         `json:"description,omitempty"`
}
