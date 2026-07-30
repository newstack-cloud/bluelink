package transform

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TransformerContextTestSuite struct {
	suite.Suite
}

func (s *TransformerContextTestSuite) Test_context_from_nil_params_yields_empty_values() {
	transformerCtx := NewTransformerContextFromParams("celerity", nil)

	configVar, hasConfigVar := transformerCtx.TransformerConfigVariable("deployTarget")
	s.Assert().False(hasConfigVar)
	s.Assert().Nil(configVar)

	contextVar, hasContextVar := transformerCtx.ContextVariable("blueprintDir")
	s.Assert().False(hasContextVar)
	s.Assert().Nil(contextVar)

	s.Assert().Empty(transformerCtx.TransformerConfigVariables())
	s.Assert().Empty(transformerCtx.ContextVariables())
}

func TestTransformerContextTestSuite(t *testing.T) {
	suite.Run(t, new(TransformerContextTestSuite))
}
