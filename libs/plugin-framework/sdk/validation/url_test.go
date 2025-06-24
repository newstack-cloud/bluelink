package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type URLValidationSuite struct {
	suite.Suite
}

func (s *URLValidationSuite) Test_valid_web_urls() {
	validURLs := []*core.ScalarValue{
		core.ScalarFromString("https://example.com"),
		core.ScalarFromString("http://example.com"),
		core.ScalarFromString("https://subdomain.example.com/path?query=1#fragment"),
		core.ScalarFromString("http://localhost:8080"),
	}

	for _, url := range validURLs {
		diagnostics := IsWebURL()("exampleField", url)
		s.Assert().Empty(diagnostics)
	}
}

func (s *URLValidationSuite) Test_invalid_web_urls() {
	invalidURLs := []*core.ScalarValue{
		core.ScalarFromString("not-a-url"),
		core.ScalarFromString("http://"),
		core.ScalarFromString("https://"),
		core.ScalarFromString("ftp://example.com"),
		core.ScalarFromString("http://example.com:invalidport"),
		core.ScalarFromString(""),
		core.ScalarFromInt(4034), // Invalid type
	}

	for _, url := range invalidURLs {
		diagnostics := IsWebURL()("exampleField", url)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *URLValidationSuite) Test_valid_https_urls() {
	validURLs := []*core.ScalarValue{
		core.ScalarFromString("https://example.com"),
		core.ScalarFromString("https://subdomain.example.com/path?query=1#fragment"),
	}

	for _, url := range validURLs {
		diagnostics := IsHTTPSURL()("exampleField", url)
		s.Assert().Empty(diagnostics)
	}
}

func (s *URLValidationSuite) Test_invalid_https_urls() {
	invalidURLs := []*core.ScalarValue{
		core.ScalarFromString("http://example.com"),
		core.ScalarFromString("ftp://example.com"),
		core.ScalarFromString("not-a-url"),
		core.ScalarFromString("https://"),
		core.ScalarFromFloat(4034.4029), // Invalid type
	}

	for _, url := range invalidURLs {
		diagnostics := IsHTTPSURL()("exampleField", url)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *URLValidationSuite) Test_valid_http_urls() {
	validURLs := []*core.ScalarValue{
		core.ScalarFromString("http://example.com"),
		core.ScalarFromString("http://subdomain.example.com/path?query=1#fragment"),
	}

	for _, url := range validURLs {
		diagnostics := IsHTTPURL()("exampleField", url)
		s.Assert().Empty(diagnostics)
	}
}

func (s *URLValidationSuite) Test_invalid_http_urls() {
	invalidURLs := []*core.ScalarValue{
		core.ScalarFromString("https://example.com"),
		core.ScalarFromString("ftp://example.com"),
		core.ScalarFromString("not-a-url"),
		core.ScalarFromString("http://"),
		core.ScalarFromBool(false), // Invalid type
	}

	for _, url := range invalidURLs {
		diagnostics := IsHTTPURL()("exampleField", url)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func (s *URLValidationSuite) Test_valid_urls_custom_schemes() {
	validURLs := []*core.ScalarValue{
		core.ScalarFromString("custom://example.com"),
		core.ScalarFromString("custom://subdomain.example.com/path?query=1#fragment"),
		core.ScalarFromString("custom2://example.com"),
	}

	allowedSchemes := []string{"custom", "custom2"}

	for _, url := range validURLs {
		diagnostics := IsURL(allowedSchemes)("exampleField", url)
		s.Assert().Empty(diagnostics)
	}
}

func (s *URLValidationSuite) Test_invalid_urls_custom_schemes() {
	invalidURLs := []*core.ScalarValue{
		core.ScalarFromString("http://example.com"),
		core.ScalarFromString("https://example.com"),
		core.ScalarFromString("ftp://example.com"),
		core.ScalarFromString("not-a-url"),
		core.ScalarFromString("custom://"),
		core.ScalarFromInt(4034), // Invalid type
	}

	allowedSchemes := []string{"custom", "custom2"}

	for _, url := range invalidURLs {
		diagnostics := IsURL(allowedSchemes)("exampleField", url)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	}
}

func TestURLValidationSuite(t *testing.T) {
	suite.Run(t, new(URLValidationSuite))
}
