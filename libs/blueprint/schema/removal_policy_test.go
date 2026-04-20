package schema

import (
	"encoding/json"

	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

type RemovalPolicyTestSuite struct{}

var _ = Suite(&RemovalPolicyTestSuite{})

func (s *RemovalPolicyTestSuite) Test_parses_retain_value_yaml(c *C) {
	target := &RemovalPolicyWrapper{}
	err := yaml.Unmarshal([]byte("retain\n"), target)
	c.Assert(err, IsNil)
	c.Assert(target.Value, Equals, RemovalPolicyRetain)
	c.Assert(target.SourceMeta, NotNil)
	c.Assert(target.SourceMeta.Line, Equals, 1)
}

func (s *RemovalPolicyTestSuite) Test_parses_delete_value_yaml(c *C) {
	target := &RemovalPolicyWrapper{}
	err := yaml.Unmarshal([]byte("delete\n"), target)
	c.Assert(err, IsNil)
	c.Assert(target.Value, Equals, RemovalPolicyDelete)
}

func (s *RemovalPolicyTestSuite) Test_round_trips_through_json(c *C) {
	original := &RemovalPolicyWrapper{Value: RemovalPolicyRetain}
	data, err := json.Marshal(original)
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, "\"retain\"")

	target := &RemovalPolicyWrapper{}
	err = json.Unmarshal(data, target)
	c.Assert(err, IsNil)
	c.Assert(target.Value, Equals, RemovalPolicyRetain)
}

func (s *RemovalPolicyTestSuite) Test_preserves_unknown_value_for_validation_to_reject(c *C) {
	target := &RemovalPolicyWrapper{}
	err := yaml.Unmarshal([]byte("archive\n"), target)
	c.Assert(err, IsNil)
	// The wrapper itself stores whatever literal was provided so
	// that the validation layer can surface a targeted error.
	c.Assert(string(target.Value), Equals, "archive")
}

func (s *RemovalPolicyTestSuite) Test_valid_removal_policies_list(c *C) {
	c.Assert(ValidRemovalPolicies, DeepEquals, []RemovalPolicy{
		RemovalPolicyDelete,
		RemovalPolicyRetain,
	})
}

func (s *RemovalPolicyTestSuite) Test_get_resource_removal_policy_helper(c *C) {
	c.Assert(GetResourceRemovalPolicy(nil), Equals, "")
	c.Assert(GetResourceRemovalPolicy(&Resource{}), Equals, "")
	c.Assert(
		GetResourceRemovalPolicy(&Resource{
			RemovalPolicy: &RemovalPolicyWrapper{Value: RemovalPolicyRetain},
		}),
		Equals,
		"retain",
	)
}
