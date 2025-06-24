package serialisation

import "github.com/newstack-cloud/bluelink/libs/blueprint/schema"

// ExpandedBlueprintSerialiser is an interface that represents a service
// that serialises and deserialises expanded blueprints.
// (An expanded blueprint is a representation with substitutions
// expanded into an AST-like structure.)
type ExpandedBlueprintSerialiser interface {
	Marshal(blueprint *schema.Blueprint) ([]byte, error)
	Unmarshal(data []byte) (*schema.Blueprint, error)
}
