package container

import (
	"context"
	"os"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type PopulateResourceSpecDefaultsTestSuite struct {
	resourceRegistry resourcehelpers.Registry
	specFixture      *schema.Blueprint
	suite.Suite
}

func (s *PopulateResourceSpecDefaultsTestSuite) SetupSuite() {
	s.resourceRegistry = &internal.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"example/complex": &internal.ExampleComplexResource{},
		},
	}

	specBytes, err := os.ReadFile("__testdata/populate-resource-spec-defaults/blueprint.yml")
	if err != nil {
		s.FailNow(err.Error())
	}
	blueprintStr := string(specBytes)
	blueprint, err := schema.LoadString(blueprintStr, schema.YAMLSpecFormat)
	if err != nil {
		s.FailNow(err.Error())
	}
	s.specFixture = blueprint
}

func (s *PopulateResourceSpecDefaultsTestSuite) Test_populates_defaults_for_resource_spec() {
	blueprintWithDefaultsPopulated, err := PopulateResourceSpecDefaults(
		context.Background(),
		s.specFixture,
		nil,
		s.resourceRegistry,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(blueprintWithDefaultsPopulated)
	s.Require().NoError(err)
}

func TestPopulateResourceSpecDefaultsTestSuite(t *testing.T) {
	suite.Run(t, new(PopulateResourceSpecDefaultsTestSuite))
}
