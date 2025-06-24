package changes

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type ResourceChangeGeneratorTestSuite struct {
	resourceChangeGenerator *defaultResourceChangeGenerator
	suite.Suite
}

func (s *ResourceChangeGeneratorTestSuite) SetupSuite() {
	s.resourceChangeGenerator = &defaultResourceChangeGenerator{}
}

func (s *ResourceChangeGeneratorTestSuite) Test_generates_changes_for_existing_resource() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture1(),
		&internal.ExampleComplexResource{},
		[]string{
			"resources.complexResource.spec.itemConfig.endpoints[2]",
			"resources.complexResource.spec.itemConfig.endpoints[4]",
			"resources.complexResource.metadata.annotations[\"test.annotation.v1\"]",
			"resources.complexResource.metadata.custom.url",
		},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, false /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func (s *ResourceChangeGeneratorTestSuite) Test_generates_changes_for_new_resource() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture2(),
		&internal.ExampleComplexResource{},
		[]string{
			"resources.complexResource.spec.itemConfig.endpoints[3]",
			"resources.complexResource.metadata.annotations[\"test.annotation.v1\"]",
			"resources.complexResource.metadata.custom.url",
		},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, false /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func (s *ResourceChangeGeneratorTestSuite) Test_does_not_generate_changes_for_fields_exceeding_max_depth() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture3(),
		&internal.ExampleComplexResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, true /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func (s *ResourceChangeGeneratorTestSuite) Test_generates_changes_for_existing_resource_with_new_resource_type() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture4(),
		&internal.ExampleComplexResource{},
		[]string{
			"resources.complexResource.spec.itemConfig.endpoints[2]",
			"resources.complexResource.spec.itemConfig.endpoints[4]",
			"resources.complexResource.metadata.annotations[\"test.annotation.v1\"]",
			"resources.complexResource.metadata.custom.url",
		},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, false /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func TestResourceChangeGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceChangeGeneratorTestSuite))
}
