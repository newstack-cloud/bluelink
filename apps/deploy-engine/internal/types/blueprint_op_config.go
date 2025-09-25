package types

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// BlueprintOperationConfig is the data type for configuration that can be provided
// in HTTP requests for actions that are carried out for blueprints.
// These values will be merged with the default values either defined in
// plugins or in the blueprint itself.
type BlueprintOperationConfig struct {
	Providers          map[string]map[string]*core.ScalarValue `json:"providers"`
	Transformers       map[string]map[string]*core.ScalarValue `json:"transformers"`
	ContextVariables   map[string]*core.ScalarValue            `json:"contextVariables"`
	BlueprintVariables map[string]*core.ScalarValue            `json:"blueprintVariables"`
	Dependencies       map[string]string                       `json:"dependencies,omitempty"`
}
