package provider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	. "gopkg.in/check.v1"
)

type LinkRegistryTestSuite struct {
	linkRegistry LinkRegistry
}

var _ = Suite(&LinkRegistryTestSuite{})

func (s *LinkRegistryTestSuite) SetUpTest(c *C) {
	s.linkRegistry = NewLinkRegistry(
		map[string]Provider{
			"test": &testStubLinkProvider{
				linkTypes: []string{
					"test/exampleResource::test/otherExampleResource",
				},
			},
		},
	)
}

func (s *LinkRegistryTestSuite) Test_get_link_for_implemented_link_type(c *C) {
	link, err := s.linkRegistry.Link(
		context.TODO(),
		"test/exampleResource",
		"test/otherExampleResource",
	)
	c.Assert(err, IsNil)
	c.Assert(link, NotNil)
}

func (s *LinkRegistryTestSuite) Test_reports_missing_link_for_link_type_not_in_provider(c *C) {
	// Plugin-backed providers hand back a client stub for any link type, the
	// registry must rule them out with the link types they report instead of
	// claiming a link that belongs to another plugin or a transformer.
	_, err := s.linkRegistry.Link(
		context.TODO(),
		"celerity/bucket",
		"celerity/consumer",
	)
	c.Assert(err, NotNil)
	runErr, isRunErr := err.(*errors.RunError)
	c.Assert(isRunErr, Equals, true)
	c.Assert(runErr.ReasonCode, Equals, ErrorReasonCodeLinkImplementationNotFound)
}

// testStubLinkProvider mirrors the behaviour of a provider plugin client,
// where a link client stub is returned for any resource type pair, regardless
// of whether the plugin implements the link.
type testStubLinkProvider struct {
	Provider
	linkTypes []string
}

func (p *testStubLinkProvider) Link(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (Link, error) {
	return &stubLink{}, nil
}

func (p *testStubLinkProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	return p.linkTypes, nil
}

type stubLink struct {
	Link
}
