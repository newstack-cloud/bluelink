package blueprint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TransformersSuite struct {
	suite.Suite
}

func (s *TransformersSuite) TestLoadTransformers_ReturnsEmptyMap() {
	ctx := context.Background()
	transformers, err := LoadTransformers(ctx)
	s.NoError(err)
	s.Empty(transformers)
}

func TestTransformersSuite(t *testing.T) {
	suite.Run(t, new(TransformersSuite))
}
