package schema

import (
	"fmt"
	"strings"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"gopkg.in/yaml.v3"
)

// Condition represents a condition that can be used
// to determine if a resource should be created.
type Condition struct {
	// A list of conditions that must all be true.
	And []*Condition `yaml:"and,omitempty" json:"and,omitempty"`
	// A list of conditions where at least one must be true.
	Or []*Condition `yaml:"or,omitempty" json:"or,omitempty"`
	// A condition that will be negated.
	Not *Condition `yaml:"not,omitempty" json:"not,omitempty"`
	// A condition expression that is expected
	// to be a substitution that resolves to a boolean.
	StringValue *substitutions.StringOrSubstitutions `yaml:"-" json:"-"`
	SourceMeta  *source.Meta                         `yaml:"-" json:"-"`
}

func (c *Condition) UnmarshalYAML(value *yaml.Node) error {
	c.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}

	if value.Kind == yaml.ScalarNode {
		subOrStringVal := &substitutions.StringOrSubstitutions{}
		if err := value.Decode(subOrStringVal); err == nil {
			c.StringValue = subOrStringVal
			return nil
		} else {
			return err
		}
	}

	type conditionAlias Condition
	var alias conditionAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	c.And = alias.And
	c.Or = alias.Or
	c.Not = alias.Not

	if (len(c.And) > 0 && len(c.Or) > 0) ||
		(len(c.Or) > 0 && c.Not != nil) ||
		(len(c.And) > 0 && c.Not != nil) {
		return errInvalidResourceCondition(value)
	}

	return nil
}

func (c *Condition) MarshalYAML() (interface{}, error) {
	if c.StringValue != nil {
		return c.StringValue, nil
	}

	type conditionAlias Condition
	var alias conditionAlias
	alias.And = c.And
	alias.Or = c.Or
	alias.Not = c.Not
	return alias, nil
}

func (c *Condition) UnmarshalJSON(data []byte) error {
	if strings.HasPrefix(string(data), "\"") {
		stringVal := &substitutions.StringOrSubstitutions{}
		if err := json.Unmarshal(data, &stringVal); err == nil {
			c.StringValue = stringVal
			return nil
		} else {
			return err
		}
	}

	type conditionAlias Condition
	var alias conditionAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	c.And = alias.And
	c.Or = alias.Or
	c.Not = alias.Not

	if (len(c.And) > 0 && len(c.Or) > 0) ||
		(len(c.Or) > 0 && c.Not != nil) ||
		(len(c.And) > 0 && c.Not != nil) {
		return errInvalidResourceCondition(nil)
	}

	return nil
}

func (c *Condition) MarshalJSON() ([]byte, error) {
	if c.StringValue != nil {
		return json.Marshal(c.StringValue)
	}

	type conditionAlias Condition
	var alias conditionAlias
	alias.And = c.And
	alias.Or = c.Or
	alias.Not = c.Not
	return json.Marshal(alias)
}

func (c *Condition) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	c.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	nodeMap, isMap := node.Value.(map[string]json.Node)
	if isMap {
		andConditions, hasAnd := nodeMap["and"]
		if hasAnd {
			return c.andConditionsFromJSONNode(
				andConditions,
				linePositions,
				parentPath,
			)
		}

		orConditions, hasOr := nodeMap["or"]
		if hasOr {
			return c.orConditionsFromJSONNode(
				orConditions,
				linePositions,
				parentPath,
			)
		}

		notCondition, hasNot := nodeMap["not"]
		if hasNot {
			return c.notConditionFromJSONNode(
				notCondition,
				linePositions,
				parentPath,
			)
		}

		return nil
	}

	_, isString := node.Value.(string)
	if isString {
		c.StringValue = &substitutions.StringOrSubstitutions{}
		err := c.StringValue.FromJSONNode(
			node,
			linePositions,
			parentPath,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Condition) andConditionsFromJSONNode(
	andConditions json.Node,
	linePositions []int,
	parentPath string,
) error {
	andPath := core.CreateJSONNodePath(
		"and",
		parentPath,
		false, // parentIsRoot
	)
	conditions, err := c.conditionsFromJSONNode(
		andConditions,
		linePositions,
		andPath,
	)
	if err != nil {
		return err
	}

	c.And = conditions
	return nil
}

func (c *Condition) orConditionsFromJSONNode(
	orConditions json.Node,
	linePositions []int,
	parentPath string,
) error {
	orPath := core.CreateJSONNodePath(
		"or",
		parentPath,
		false, // parentIsRoot
	)
	conditions, err := c.conditionsFromJSONNode(
		orConditions,
		linePositions,
		orPath,
	)
	if err != nil {
		return err
	}

	c.Or = conditions
	return nil
}

func (c *Condition) notConditionFromJSONNode(
	notCondition json.Node,
	linePositions []int,
	parentPath string,
) error {
	notPath := core.CreateJSONNodePath(
		"not",
		parentPath,
		false, // parentIsRoot
	)
	condition := &Condition{}
	err := condition.FromJSONNode(
		&notCondition,
		linePositions,
		notPath,
	)
	if err != nil {
		return err
	}

	c.Not = condition
	return nil
}

func (c *Condition) conditionsFromJSONNode(
	listNode json.Node,
	linePositions []int,
	parentPath string,
) ([]*Condition, error) {
	nodeSlice, ok := listNode.Value.([]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(&listNode, linePositions)
		return nil, errInvalidArray(&position, parentPath)
	}

	conditions := make([]*Condition, len(nodeSlice))
	for i, node := range nodeSlice {
		condition := &Condition{}
		key := fmt.Sprintf("%d", i)
		conditionPath := core.CreateJSONNodePath(key, parentPath, false)
		err := condition.FromJSONNode(
			&node,
			linePositions,
			conditionPath,
		)
		if err != nil {
			return nil, err
		}

		conditions[i] = condition
	}

	return conditions, nil
}
