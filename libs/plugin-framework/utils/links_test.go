package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type LinkUtilsTestSuite struct {
	suite.Suite
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_finds_direct_links() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
		"aws/lambda/function::aws/sqs/queue",
		"aws/ec2/instance::aws/ebs/volume",
	}

	result := DeriveLinkableTypes("aws/lambda/function", allLinkTypes)

	s.ElementsMatch([]string{"aws/dynamodb/table", "aws/sqs/queue"}, result)
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_finds_reverse_links() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
	}

	result := DeriveLinkableTypes("aws/dynamodb/table", allLinkTypes)

	s.Equal([]string{"aws/lambda/function"}, result)
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_deduplicates() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
		"aws/lambda/function::aws/dynamodb/table",
	}

	result := DeriveLinkableTypes("aws/lambda/function", allLinkTypes)

	s.Equal([]string{"aws/dynamodb/table"}, result)
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_empty_for_unknown_resource() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
	}

	result := DeriveLinkableTypes("gcp/compute/instance", allLinkTypes)

	s.Empty(result)
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_handles_malformed_link_types() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
		"malformed-link-type",
		"",
	}

	result := DeriveLinkableTypes("aws/lambda/function", allLinkTypes)

	s.Equal([]string{"aws/dynamodb/table"}, result)
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_handles_empty_input() {
	result := DeriveLinkableTypes("aws/lambda/function", []string{})

	s.Empty(result)
}

func (s *LinkUtilsTestSuite) Test_DeriveLinkableTypes_handles_nil_input() {
	result := DeriveLinkableTypes("aws/lambda/function", nil)

	s.Empty(result)
}

func TestLinkUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(LinkUtilsTestSuite))
}
