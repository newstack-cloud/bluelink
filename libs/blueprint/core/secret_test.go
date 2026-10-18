package core

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Masking at a boundary catches a value whose shape is known at that point. Nothing at a
// boundary can catch a value a plugin has already interpolated into a message it composed,
// so the secret is handed over as a type that redacts itself.
type SecretTestSuite struct {
	suite.Suite
}

func (s *SecretTestSuite) Test_redacts_itself_for_every_verb() {
	secret := NewSecret(ScalarFromString("super-secret-password"))

	for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q", "%d"} {
		s.Assert().NotContains(
			fmt.Sprintf(format, secret),
			"super-secret-password",
			"%s rendered the value",
			format,
		)
		s.Assert().Contains(fmt.Sprintf(format, secret), RedactedPlaceholder)
	}
}

func (s *SecretTestSuite) Test_redacts_itself_when_serialised() {
	serialised, err := json.Marshal(map[string]Secret{
		"apiKey": NewSecret(ScalarFromString("super-secret-password")),
	})
	s.Require().NoError(err)

	s.Assert().NotContains(string(serialised), "super-secret-password")
	s.Assert().Contains(string(serialised), RedactedPlaceholder)
}

// The one way out, so that handing the value to the service it configures stays possible.
func (s *SecretTestSuite) Test_reveals_the_value_when_asked_explicitly() {
	secret := NewSecret(ScalarFromString("super-secret-password"))

	s.Assert().Equal("super-secret-password", StringValueFromScalar(secret.Reveal()))
}

func (s *SecretTestSuite) Test_wraps_a_value_whose_field_is_declared_secret() {
	secret, isSecret := testConfigDefinition.SecretValue(
		PluginConfig{"secretField": ScalarFromString("super-secret-password")},
		"secretField",
	)

	s.Require().True(isSecret)
	s.Assert().Equal(RedactedPlaceholder, secret.String())
	s.Assert().Equal("super-secret-password", StringValueFromScalar(secret.Reveal()))
}

// A field name that is a template matches any key it covers, so a secret declared as
// "aws.config.secretKeys.<region>" is found for a concrete region.
func (s *SecretTestSuite) Test_wraps_a_value_matching_a_secret_field_name_template() {
	fields := map[string]*ConfigFieldDefinition{
		"aws.config.secretKeys.<region>": {
			Type:   ScalarTypeString,
			Secret: true,
		},
	}
	definition := &ConfigDefinition{Fields: fields}

	_, isSecret := definition.SecretValue(
		PluginConfig{
			"aws.config.secretKeys.eu-west-2": ScalarFromString("super-secret-password"),
		},
		"aws.config.secretKeys.eu-west-2",
	)

	s.Assert().True(isSecret)
}

// Reaching for a secret on a field nobody declared as one finds nothing, rather than
// handing back a value the caller would then treat as protected.
func (s *SecretTestSuite) Test_does_not_wrap_a_value_whose_field_is_not_declared_secret() {
	_, isSecret := testConfigDefinition.SecretValue(
		PluginConfig{"stringField": ScalarFromString("a value")},
		"stringField",
	)

	s.Assert().False(isSecret)
}

func TestSecretTestSuite(t *testing.T) {
	suite.Run(t, new(SecretTestSuite))
}
