package schema

import (
	"os"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"

	"github.com/tailscale/hujson"
	"gopkg.in/yaml.v3"
)

// Loader provides a function that loads a serialised blueprint
// spec either from a file path or a serialised string already
// loaded in memory.
type Loader func(string, SpecFormat) (*Blueprint, error)

// Load deals with loading a blueprint specification
// from a YAML or JSON file on disk.
func Load(specFilePath string, inputFormat SpecFormat) (*Blueprint, error) {
	if inputFormat == YAMLSpecFormat {
		return loadYAMLFromFile(specFilePath)
	}

	return loadJWCCFromFile(specFilePath)
}

func loadYAMLFromFile(specFilePath string) (*Blueprint, error) {
	blueprint := &Blueprint{}
	f, err := os.Open(specFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(blueprint)
	if err != nil {
		return nil, err
	}

	return blueprint, nil
}

func loadJWCCFromFile(specFilePath string) (*Blueprint, error) {
	blueprint := &Blueprint{}
	// In order to use hujson to strip out comments
	// and trailing commas, we need to read the entire file.
	contents, err := os.ReadFile(specFilePath)
	if err != nil {
		return nil, err
	}
	err = unmarshalJWCC(contents, blueprint)
	if err != nil {
		return nil, err
	}

	return blueprint, nil
}

func unmarshalJWCC(contents []byte, blueprint *Blueprint) error {
	standardised, err := hujson.Standardize(contents)
	if err != nil {
		return err
	}

	rootNode := &json.Node{}
	err = json.Unmarshal(standardised, rootNode)
	if err != nil {
		return err
	}
	return populateBlueprintFromJSONNode(string(standardised), rootNode, blueprint)
}

// SpecFormat is the format of a specification
// to be used for transport and storage.
type SpecFormat string

const (
	// JWCCSpecFormat determines that a spec being loaded
	// or exported should be serialised or deserialised
	// in the JSON with Commas and Comments format.
	JWCCSpecFormat SpecFormat = "jwcc"
	// YAMLSpecFormat determines that a spec being loaded
	// or exported should be serialised or deserialised
	// in the YAML format.
	YAMLSpecFormat SpecFormat = "yaml"
)

// LoadString deals with loading a blueprint specification
// from a given YAML or JSON with Commas and Comments string.
func LoadString(spec string, inputFormat SpecFormat) (*Blueprint, error) {
	blueprint := &Blueprint{}

	var err error
	if inputFormat == YAMLSpecFormat {
		err = yaml.Unmarshal([]byte(spec), blueprint)
	} else {
		err = unmarshalJWCC([]byte(spec), blueprint)
	}

	return blueprint, err
}

func populateBlueprintFromJSONNode(docSource string, node *json.Node, blueprint *Blueprint) error {
	linePositions := core.LinePositionsFromSource(docSource)
	nodeMap, isMap := node.Value.(map[string]json.Node)
	if !isMap {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, "blueprint")
	}

	blueprint.Version = &core.ScalarValue{}
	err := core.UnpackValueFromJSONMapNode(
		nodeMap,
		"version",
		blueprint.Version,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	blueprint.Transform = &TransformValueWrapper{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"transform",
		blueprint.Transform,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.Variables = &VariableMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"variables",
		blueprint.Variables,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.Values = &ValueMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"values",
		blueprint.Values,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.Include = &IncludeMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"include",
		blueprint.Include,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.Resources = &ResourceMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"resources",
		blueprint.Resources,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.DataSources = &DataSourceMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"datasources",
		blueprint.DataSources,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.Exports = &ExportMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"exports",
		blueprint.Exports,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	blueprint.Metadata = &core.MappingNode{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"metadata",
		blueprint.Metadata,
		linePositions,
		/* parentPath */ "blueprint",
		/* parentIsRoot */ true,
		/* required */ false,
	)

	return err
}
