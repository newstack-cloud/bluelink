package deployconfig

import (
	"maps"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Config is the Bluelink deploy configuration format, the same shape the
// deploy engine accepts as a blueprint operation config and the CLI writes to
// its converted deploy config file.
type Config struct {
	Providers          map[string]map[string]*core.ScalarValue `json:"providers"`
	Transformers       map[string]map[string]*core.ScalarValue `json:"transformers"`
	ContextVariables   map[string]*core.ScalarValue            `json:"contextVariables"`
	BlueprintVariables map[string]*core.ScalarValue            `json:"blueprintVariables"`
}

// Resolved is a deploy configuration along with the file it came from.
type Resolved struct {
	Config *Config
	// SourcePath is the file the configuration was read from, used for
	// logging and for reporting problems back to the user.
	SourcePath string
}

// Params builds the blueprint parameters for a resolved deploy configuration.
//
// The reserved validation-context flag is always applied last so that it wins
// over anything the deploy configuration declares. Plugins rely on it to tell
// an editor session apart from a real deployment.
func (r *Resolved) Params() core.BlueprintParams {
	if r == nil || r.Config == nil {
		return ValidationParams()
	}

	contextVariables := map[string]*core.ScalarValue{}
	maps.Copy(contextVariables, r.Config.ContextVariables)
	contextVariables[core.ValidationContextVariableName] = core.ScalarFromBool(true)

	return core.NewDefaultParams(
		emptyIfNil(r.Config.Providers),
		emptyIfNil(r.Config.Transformers),
		contextVariables,
		emptyScalarsIfNil(r.Config.BlueprintVariables),
	)
}

// ValidationParams builds the parameters used when no deploy configuration
// could be found. Plugins that require configuration are expected to handle
// its absence in a validation context.
func ValidationParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			core.ValidationContextVariableName: core.ScalarFromBool(true),
		},
		map[string]*core.ScalarValue{},
	)
}

func emptyIfNil(
	values map[string]map[string]*core.ScalarValue,
) map[string]map[string]*core.ScalarValue {
	if values == nil {
		return map[string]map[string]*core.ScalarValue{}
	}
	return values
}

func emptyScalarsIfNil(values map[string]*core.ScalarValue) map[string]*core.ScalarValue {
	if values == nil {
		return map[string]*core.ScalarValue{}
	}
	return values
}
