package helpinfo

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type CardinalitySuite struct {
	suite.Suite
}

func (s *CardinalitySuite) Test_FormatLinkCardinality_unconstrained() {
	s.Equal(
		"unconstrained",
		FormatLinkCardinality(provider.LinkCardinality{Min: 0, Max: 0}),
	)
}

func (s *CardinalitySuite) Test_FormatLinkCardinality_exact_match() {
	s.Equal(
		"exactly 1",
		FormatLinkCardinality(provider.LinkCardinality{Min: 1, Max: 1}),
	)
	s.Equal(
		"exactly 3",
		FormatLinkCardinality(provider.LinkCardinality{Min: 3, Max: 3}),
	)
}

func (s *CardinalitySuite) Test_FormatLinkCardinality_at_least() {
	s.Equal(
		"at least 1",
		FormatLinkCardinality(provider.LinkCardinality{Min: 1, Max: 0}),
	)
	s.Equal(
		"at least 2",
		FormatLinkCardinality(provider.LinkCardinality{Min: 2, Max: 0}),
	)
}

func (s *CardinalitySuite) Test_FormatLinkCardinality_at_most() {
	s.Equal(
		"at most 1",
		FormatLinkCardinality(provider.LinkCardinality{Min: 0, Max: 1}),
	)
	s.Equal(
		"at most 5",
		FormatLinkCardinality(provider.LinkCardinality{Min: 0, Max: 5}),
	)
}

func (s *CardinalitySuite) Test_FormatLinkCardinality_bounded_range() {
	s.Equal(
		"1..5",
		FormatLinkCardinality(provider.LinkCardinality{Min: 1, Max: 5}),
	)
	s.Equal(
		"2..3",
		FormatLinkCardinality(provider.LinkCardinality{Min: 2, Max: 3}),
	)
}

func TestCardinalitySuite(t *testing.T) {
	suite.Run(t, new(CardinalitySuite))
}
