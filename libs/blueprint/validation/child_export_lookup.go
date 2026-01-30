package validation

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// ChildExportTypeLookup is a function that looks up a child blueprint's
// export by child include name and export name.
// Returns the export schema if found, nil if the child blueprint couldn't
// be resolved or the export doesn't exist.
// An error is returned only for definitive failures (export name not found
// in a resolved child blueprint).
// The location parameter is used to provide accurate source positioning
// in error diagnostics.
type ChildExportTypeLookup func(
	childName string, exportName string, location *source.Meta,
) (*schema.Export, error)
