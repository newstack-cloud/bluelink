package schema

import (
	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"gopkg.in/yaml.v3"
)

// Include represents a child blueprint
// include in the specification.
// This provides a method of creating modular blueprints
// that is native to the spec and doesn't require
// a third party plugin to implement. (e.g. a bluelink/blueprint resource type)
type Include struct {
	// The path to the child blueprint on a local or remote file system.
	Path *substitutions.StringOrSubstitutions `yaml:"path" json:"path"`
	// The variables to pass down to the child blueprint.
	Variables *core.MappingNode `yaml:"variables" json:"variables"`
	// Extra metadata to be used by include resolver plugins.
	// An example of this could be the use of fields that provide information
	// about a remote location to download the child blueprint from such as
	// an AWS S3 bucket.
	Metadata         *core.MappingNode                    `yaml:"metadata" json:"metadata"`
	Description      *substitutions.StringOrSubstitutions `yaml:"description" json:"description"`
	SourceMeta       *source.Meta                         `yaml:"-" json:"-"`
	FieldsSourceMeta map[string]*source.Meta              `yaml:"-" json:"-"`
}

func (i *Include) UnmarshalYAML(value *yaml.Node) error {
	i.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}
	i.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromYAMLNode(value)

	type includeAlias Include
	var alias includeAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	i.Path = alias.Path
	i.Variables = alias.Variables
	i.Metadata = alias.Metadata
	i.Description = alias.Description

	return nil
}

func (i *Include) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	i.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromJSONNodeMap(nodeMap, linePositions)

	i.Path = &substitutions.StringOrSubstitutions{}
	err := core.UnpackValueFromJSONMapNode(
		nodeMap,
		"path",
		i.Path,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	i.Variables = &core.MappingNode{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"variables",
		i.Variables,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	i.Metadata = &core.MappingNode{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"metadata",
		i.Metadata,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	i.Description = &substitutions.StringOrSubstitutions{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"description",
		i.Description,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	i.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}
