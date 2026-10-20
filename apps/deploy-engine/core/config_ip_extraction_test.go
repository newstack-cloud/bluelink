package core_test

import (
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/stretchr/testify/suite"
)

// Config holds the default as a literal so that it does not depend on the middleware
// that consumes it. This is what keeps the two from drifting.
type IPExtractionConfigSuite struct {
	suite.Suite
}

func (s *IPExtractionConfigSuite) Test_the_configured_default_is_a_source_the_middleware_knows() {
	config, err := core.LoadConfig()
	s.Require().NoError(err)

	s.Assert().Equal(
		string(httpmw.DirectExtractionSource),
		config.IPExtractionSource,
		"the default must name a source the middleware implements",
	)
}

// Defaulting to a source that trusts a forwarded header would let any caller reached
// directly choose the address recorded against its own actions.
func (s *IPExtractionConfigSuite) Test_the_default_does_not_trust_a_forwarded_header() {
	config, err := core.LoadConfig()
	s.Require().NoError(err)

	s.Assert().NotEqual(
		string(httpmw.AWSALBExtractionSource),
		config.IPExtractionSource,
	)
}

func TestIPExtractionConfigSuite(t *testing.T) {
	suite.Run(t, new(IPExtractionConfigSuite))
}
