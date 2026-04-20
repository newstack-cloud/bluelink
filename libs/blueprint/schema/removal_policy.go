package schema

import (
	"fmt"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/jsonutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"gopkg.in/yaml.v3"
)

// RemovalPolicyWrapper provides a struct that holds a removal policy
// value for a resource along with source metadata for error reporting.
type RemovalPolicyWrapper struct {
	Value      RemovalPolicy
	SourceMeta *source.Meta
}

func (t *RemovalPolicyWrapper) MarshalYAML() (interface{}, error) {
	return t.Value, nil
}

func (t *RemovalPolicyWrapper) UnmarshalYAML(value *yaml.Node) error {
	t.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
	}

	t.Value = RemovalPolicy(value.Value)
	return nil
}

func (t *RemovalPolicyWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(t.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (t *RemovalPolicyWrapper) UnmarshalJSON(data []byte) error {
	var policyVal string
	err := json.Unmarshal(data, &policyVal)
	if err != nil {
		return err
	}

	t.Value = RemovalPolicy(policyVal)

	return nil
}

func (t *RemovalPolicyWrapper) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	t.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)
	stringVal := node.Value.(string)
	t.Value = RemovalPolicy(stringVal)
	return nil
}

// RemovalPolicy represents the policy that controls what happens
// to the underlying resource in the provider when a resource is removed
// from a blueprint.
// See the blueprint specification for the full semantics.
type RemovalPolicy string

func (p RemovalPolicy) Equal(compareWith RemovalPolicy) bool {
	return p == compareWith
}

const (
	// RemovalPolicyDelete indicates that the underlying resource should
	// be destroyed in the provider when the resource is removed
	// from the blueprint.
	// This is the default behaviour when no removal policy is set.
	RemovalPolicyDelete RemovalPolicy = "delete"
	// RemovalPolicyRetain indicates that the underlying resource should
	// be left untouched in the provider when the resource is removed
	// from the blueprint.
	// The resource will still be removed from the blueprint's managed
	// state along with any internal bookkeeping associated with it.
	RemovalPolicyRetain RemovalPolicy = "retain"
)

var (
	// ValidRemovalPolicies lists all valid removal policy values
	// for clean validation of the removalPolicy field on a resource.
	ValidRemovalPolicies = []RemovalPolicy{
		RemovalPolicyDelete,
		RemovalPolicyRetain,
	}
)
