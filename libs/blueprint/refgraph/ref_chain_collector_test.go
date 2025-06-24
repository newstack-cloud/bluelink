package refgraph

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
	. "gopkg.in/check.v1"
)

type RefChainCollectorTestSuite struct {
	resourceA *schema.Resource
	resourceB *schema.Resource
	resourceC *schema.Resource
	resourceD *schema.Resource
	resourceE *schema.Resource
	resourceF *schema.Resource
	suite.Suite
}

func (s *RefChainCollectorTestSuite) SetupTest(c *C) {
	s.resourceA = createTestResource("resourceA")
	s.resourceB = createTestResource("resourceB")
	s.resourceC = createTestResource("resourceC")
	s.resourceD = createTestResource("resourceD")
	s.resourceE = createTestResource("resourceE")
	s.resourceF = createTestResource("resourceF")
}

func (s *RefChainCollectorTestSuite) Test_detects_circular_references() {

	collector := NewRefChainCollector()
	err := collector.Collect("resources.resourceA", s.resourceA, "", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceB", s.resourceB, "resources.resourceA", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceC", s.resourceC, "resources.resourceB", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceA", s.resourceA, "resources.resourceC", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}

	err = collector.Collect("resources.resourceD", s.resourceD, "", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceE", s.resourceE, "resources.resourceD", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceF", s.resourceF, "resources.resourceE", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceD", s.resourceD, "resources.resourceF", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}

	circularRefs := collector.FindCircularReferences()
	s.Assert().Len(circularRefs, 2)
	s.Assert().Equal("resources.resourceA", circularRefs[0].ElementName)
	s.Assert().Equal("resources.resourceD", circularRefs[1].ElementName)
}

func (s *RefChainCollectorTestSuite) Test_finds_no_circular_references() {
	collector := NewRefChainCollector()
	err := collector.Collect("resources.resourceA", s.resourceA, "", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceB", s.resourceB, "resources.resourceA", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceC", s.resourceC, "resources.resourceB", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}

	err = collector.Collect("resources.resourceD", s.resourceD, "", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceE", s.resourceE, "resources.resourceD", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}
	err = collector.Collect("resources.resourceF", s.resourceF, "resources.resourceE", []string{})
	if err != nil {
		s.FailNow(err.Error())
	}

	circularRefs := collector.FindCircularReferences()
	s.Assert().Len(circularRefs, 0)
}

func createTestResource(id string) *schema.Resource {
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: "bluelink/example"},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"id": {
					Scalar: &core.ScalarValue{
						StringValue: &id,
					},
				},
			},
		},
	}
}

func TestRefChainCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(RefChainCollectorTestSuite))
}
