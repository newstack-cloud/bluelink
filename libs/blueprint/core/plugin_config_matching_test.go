package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PluginConfigMatchingTestSuite struct {
	suite.Suite
}

func dynamicSuffixDefinition() *ConfigDefinition {
	return &ConfigDefinition{
		Fields: map[string]*ConfigFieldDefinition{
			// A literal segment after the placeholder, which is where an
			// unanchored pattern leaks.
			"aws.vpc.<vpc>.cidrBlock": {
				Type:  ScalarTypeString,
				Label: "CIDR Block",
			},
			"aws.aurora.<database>.minACU": {
				Type:  ScalarTypeFloat,
				Label: "Minimum Capacity",
			},
		},
	}
}

func (s *PluginConfigMatchingTestSuite) hasError(config map[string]*ScalarValue) bool {
	return s.hasErrorFor(dynamicSuffixDefinition(), config)
}

func (s *PluginConfigMatchingTestSuite) hasErrorFor(
	definition *ConfigDefinition,
	config map[string]*ScalarValue,
) bool {
	diagnostics, err := ValidateConfigDefinition("test", "provider", config, definition)
	s.Require().NoError(err)

	for _, diagnostic := range diagnostics {
		if diagnostic.Level == DiagnosticLevelError {
			return true
		}
	}

	return false
}

func (s *PluginConfigMatchingTestSuite) Test_matches_a_dynamic_field_with_a_literal_suffix() {
	s.False(s.hasError(map[string]*ScalarValue{
		"aws.vpc.appVpc.cidrBlock": ScalarFromString("10.0.0.0/16"),
	}))
}

// A dynamic field name is matched against the whole config key. Each of these
// was accepted before the pattern was anchored, and then silently ignored by the
// plugin, because the key is where a typo lands.
func (s *PluginConfigMatchingTestSuite) Test_rejects_keys_that_only_contain_a_dynamic_field() {
	for _, key := range []string{
		"aws.vpc.appVpc.cidrBlocks",
		"aws.vpc.appVpc.cidrBlock2",
		"aws.vpc.appVpc.cidrBlock.extra",
		"prefixed.aws.vpc.appVpc.cidrBlock",
	} {
		s.Run(key, func() {
			s.True(
				s.hasError(map[string]*ScalarValue{key: ScalarFromString("10.0.0.0/16")}),
				"%q must not satisfy aws.vpc.<vpc>.cidrBlock",
				key,
			)
		})
	}
}

// A "." in a field definition name is a literal separator, not a wildcard. Only
// the last character of the escape set used to be escaped, so every "." matched
// any character and a key with the separators swapped was accepted.
func (s *PluginConfigMatchingTestSuite) Test_rejects_keys_that_substitute_the_separators() {
	for _, key := range []string{
		"aws.vpc.appVpc_cidrBlock",
		"aws_vpc_appVpc_cidrBlock",
		"awsXvpcYappVpcZcidrBlock",
	} {
		s.Run(key, func() {
			s.True(
				s.hasError(map[string]*ScalarValue{key: ScalarFromString("10.0.0.0/16")}),
				"%q must not satisfy aws.vpc.<vpc>.cidrBlock",
				key,
			)
		})
	}
}

// A whole number authored for a float field arrives as an integer, because a
// value that travels through JSON as 1.0 is re-marshalled as 1. Normalising it
// ahead of validation is what lets the type check, AllowedValues and the SDK
// float validators all keep one notion of the field's type.
func (s *PluginConfigMatchingTestSuite) Test_normalises_a_whole_number_for_a_float_field() {
	for _, literal := range []string{"0.5", "1.0", "1", "16"} {
		s.Run(literal, func() {
			var value ScalarValue
			s.Require().NoError(json.Unmarshal([]byte(literal), &value))

			definition := dynamicSuffixDefinition()
			config := map[string]*ScalarValue{"aws.aurora.appDb.minACU": &value}

			normalised, err := NormaliseConfigValues(config, definition)
			s.Require().NoError(err)
			s.Require().True(
				IsScalarFloat(normalised["aws.aurora.appDb.minACU"]),
				"%s should normalise to a float",
				literal,
			)
			s.False(s.hasErrorFor(definition, normalised))
		})
	}
}

// Normalisation does not reinterpret: only an integer to a float is lossless.
func (s *PluginConfigMatchingTestSuite) Test_normalisation_leaves_other_types_alone() {
	definition := dynamicSuffixDefinition()
	config := map[string]*ScalarValue{
		"aws.aurora.appDb.minACU":  ScalarFromString("not-a-number"),
		"aws.vpc.appVpc.cidrBlock": ScalarFromString("10.0.0.0/16"),
		"undeclared.key":           ScalarFromInt(3),
	}

	normalised, err := NormaliseConfigValues(config, definition)
	s.Require().NoError(err)
	s.True(IsScalarString(normalised["aws.aurora.appDb.minACU"]))
	s.True(IsScalarString(normalised["aws.vpc.appVpc.cidrBlock"]))
	s.True(IsScalarInt(normalised["undeclared.key"]), "no declaration means no conversion")
	s.True(s.hasErrorFor(definition, normalised), "a string is still not a float")
}

// An integer field must not start accepting floats: the conversion is
// one-directional, because the reverse would discard a fraction.
func (s *PluginConfigMatchingTestSuite) Test_integer_field_still_rejects_a_float() {
	definition := &ConfigDefinition{
		Fields: map[string]*ConfigFieldDefinition{
			"capacity": {Type: ScalarTypeInteger, Label: "Capacity"},
		},
	}
	config := map[string]*ScalarValue{"capacity": ScalarFromFloat(1.5)}

	normalised, err := NormaliseConfigValues(config, definition)
	s.Require().NoError(err)
	s.True(
		s.hasErrorFor(definition, normalised),
		"a float must not satisfy an integer field",
	)
}

func TestPluginConfigMatchingTestSuite(t *testing.T) {
	suite.Run(t, new(PluginConfigMatchingTestSuite))
}
