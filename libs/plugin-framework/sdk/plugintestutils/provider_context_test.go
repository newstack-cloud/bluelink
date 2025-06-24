package plugintestutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type ProviderContextSuite struct {
	suite.Suite
}

func (s *ProviderContextSuite) Test_provider_context_initialisation() {
	providerCtx := NewTestProviderContext(
		"testProvider",
		map[string]*core.ScalarValue{
			"configKey1": core.ScalarFromString("configValue1"),
			"configKey2": core.ScalarFromString("configValue2"),
		},
		map[string]*core.ScalarValue{},
	)
	s.Assert().NotNil(providerCtx, "Provider context should not be nil")

	configValue1, hasConfigValue1 := providerCtx.ProviderConfigVariable("configKey1")
	s.Assert().True(hasConfigValue1, "Provider config should contain 'configKey1'")
	// Check if the value is as expected
	s.Assert().Equal(
		"configValue1",
		core.StringValueFromScalar(
			configValue1,
		),
	)

	configValue2, hasConfigValue2 := providerCtx.ProviderConfigVariable("configKey2")
	s.Assert().True(hasConfigValue2, "Provider config should contain 'configKey2'")
	// Check if the value is as expected
	s.Assert().Equal(
		"configValue2",
		core.StringValueFromScalar(
			configValue2,
		),
	)
}

func TestProviderContextSuite(t *testing.T) {
	suite.Run(t, new(ProviderContextSuite))
}
