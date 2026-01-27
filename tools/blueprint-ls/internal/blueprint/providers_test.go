package blueprint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProvidersSuite struct {
	suite.Suite
}

func (s *ProvidersSuite) TestLoadProviders_ReturnsValidMap() {
	ctx := context.Background()
	providers, err := LoadProviders(ctx)
	s.NoError(err)
	s.NotNil(providers)
}

func (s *ProvidersSuite) TestLoadProviders_IncludesCoreProvider() {
	ctx := context.Background()
	providers, err := LoadProviders(ctx)
	s.NoError(err)
	s.Contains(providers, "core")
	s.NotNil(providers["core"])
}

func TestProvidersSuite(t *testing.T) {
	suite.Run(t, new(ProvidersSuite))
}
