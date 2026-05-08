package transformutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

const testValidationCtxTransformerID = "celerity-test-2026"

type IsValidationContextTestSuite struct {
	suite.Suite
}

func (s *IsValidationContextTestSuite) Test_returns_true_when_reserved_key_set_to_true() {
	transformerCtx := newTransformerContextWithContextVars(
		testValidationCtxTransformerID,
		map[string]*core.ScalarValue{
			core.ValidationContextVariableName: core.ScalarFromBool(true),
		},
	)

	s.Assert().True(IsValidationContext(transformerCtx))
}

func (s *IsValidationContextTestSuite) Test_returns_false_when_reserved_key_set_to_false() {
	transformerCtx := newTransformerContextWithContextVars(
		testValidationCtxTransformerID,
		map[string]*core.ScalarValue{
			core.ValidationContextVariableName: core.ScalarFromBool(false),
		},
	)

	s.Assert().False(IsValidationContext(transformerCtx))
}

func (s *IsValidationContextTestSuite) Test_returns_false_when_reserved_key_absent() {
	transformerCtx := newTransformerContextWithContextVars(
		testValidationCtxTransformerID,
		map[string]*core.ScalarValue{
			"someOtherVar": core.ScalarFromString("value"),
		},
	)

	s.Assert().False(IsValidationContext(transformerCtx))
}

func (s *IsValidationContextTestSuite) Test_returns_false_when_reserved_key_has_wrong_scalar_type() {
	transformerCtx := newTransformerContextWithContextVars(
		testValidationCtxTransformerID,
		map[string]*core.ScalarValue{
			core.ValidationContextVariableName: core.ScalarFromString("true"),
		},
	)

	s.Assert().False(IsValidationContext(transformerCtx))
}

func (s *IsValidationContextTestSuite) Test_returns_false_when_no_context_vars_set() {
	transformerCtx := newTransformerContextWithContextVars(
		testValidationCtxTransformerID,
		map[string]*core.ScalarValue{},
	)

	s.Assert().False(IsValidationContext(transformerCtx))
}

func TestIsValidationContextTestSuite(t *testing.T) {
	suite.Run(t, new(IsValidationContextTestSuite))
}

func newTransformerContextWithContextVars(
	namespace string,
	contextVars map[string]*core.ScalarValue,
) transform.Context {
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		contextVars,
		map[string]*core.ScalarValue{},
	)
	return transform.NewTransformerContextFromParams(namespace, params)
}
