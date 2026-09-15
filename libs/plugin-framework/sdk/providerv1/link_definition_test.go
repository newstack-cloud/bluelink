package providerv1

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type LinkDefinitionTestSuite struct {
	suite.Suite
}

const (
	linkDefResourceTypeA = "aws/lambda/function"
	linkDefResourceTypeB = "aws/rds/proxy"
	linkDefLinkType      = "aws/lambda/function::aws/rds/proxy"
)

// A link definition that leaves one of its functions unset must report a missing function
// rather than dereference the nil field.
//
// These run in the provider plugin process, where a nil dereference takes the whole plugin
// down and every call in flight against it fails with a transport error that names neither
// the link nor the missing function. A provider author who omits a function, which is what
// happens when a link is not migrated after the interface changes, has to get an error that
// says so.
func (s *LinkDefinitionTestSuite) Test_reports_a_missing_stage_changes_function() {
	linkDef := s.linkDefinition()

	output, err := linkDef.StageChanges(
		context.Background(),
		&provider.LinkStageChangesInput{},
	)

	s.Require().Error(err)
	s.Assert().Nil(output)
	s.Assert().Equal(
		"stage changes function missing in link definition for link type "+
			"\""+linkDefLinkType+"\"",
		err.Error(),
		"the error must name the missing function and the link it belongs to",
	)
}

func (s *LinkDefinitionTestSuite) Test_reports_a_missing_update_linked_resources_function() {
	linkDef := s.linkDefinition()

	output, err := linkDef.UpdateLinkedResources(
		context.Background(),
		&provider.LinkUpdateLinkedResourcesInput{},
	)

	s.Require().Error(err)
	s.Assert().Nil(output)
	s.Assert().Equal(
		"update linked resources function missing in link definition for link type "+
			"\""+linkDefLinkType+"\"",
		err.Error(),
	)
}

func (s *LinkDefinitionTestSuite) Test_reports_a_missing_update_intermediary_resources_function() {
	linkDef := s.linkDefinition()

	output, err := linkDef.UpdateIntermediaryResources(
		context.Background(),
		&provider.LinkUpdateIntermediaryResourcesInput{},
	)

	s.Require().Error(err)
	s.Assert().Nil(output)
	s.Assert().Equal(
		"update intermediary resources function missing in link definition for link type "+
			"\""+linkDefLinkType+"\"",
		err.Error(),
	)
}

// A definition that sets its functions must still call them, so the guards cannot be
// satisfied by refusing every call.
func (s *LinkDefinitionTestSuite) Test_calls_the_functions_a_definition_provides() {
	called := map[string]bool{}
	linkDef := s.linkDefinition()
	linkDef.StageChangesFunc = func(
		ctx context.Context,
		input *provider.LinkStageChangesInput,
	) (*provider.LinkStageChangesOutput, error) {
		called["stageChanges"] = true
		return &provider.LinkStageChangesOutput{}, nil
	}
	linkDef.UpdateLinkedResourcesFunc = func(
		ctx context.Context,
		input *provider.LinkUpdateLinkedResourcesInput,
	) (*provider.LinkUpdateLinkedResourcesOutput, error) {
		called["updateLinkedResources"] = true
		return &provider.LinkUpdateLinkedResourcesOutput{}, nil
	}
	linkDef.UpdateIntermediaryResourcesFunc = func(
		ctx context.Context,
		input *provider.LinkUpdateIntermediaryResourcesInput,
	) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
		called["updateIntermediaryResources"] = true
		return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
	}

	_, stageErr := linkDef.StageChanges(
		context.Background(),
		&provider.LinkStageChangesInput{},
	)
	_, updateErr := linkDef.UpdateLinkedResources(
		context.Background(),
		&provider.LinkUpdateLinkedResourcesInput{},
	)
	_, intermediaryErr := linkDef.UpdateIntermediaryResources(
		context.Background(),
		&provider.LinkUpdateIntermediaryResourcesInput{},
	)

	s.Require().NoError(stageErr)
	s.Require().NoError(updateErr)
	s.Require().NoError(intermediaryErr)
	s.Assert().True(called["stageChanges"])
	s.Assert().True(called["updateLinkedResources"])
	s.Assert().True(called["updateIntermediaryResources"])
}

// The priority resource function is optional by design and falls back to the statically
// declared priority resource, so leaving it unset must stay a supported configuration
// rather than becoming an error alongside the others.
func (s *LinkDefinitionTestSuite) Test_falls_back_when_no_priority_resource_function_is_set() {
	linkDef := s.linkDefinition()
	linkDef.PriorityResource = provider.LinkPriorityResourceA

	output, err := linkDef.GetPriorityResource(
		context.Background(),
		&provider.LinkGetPriorityResourceInput{},
	)

	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Assert().Equal(provider.LinkPriorityResourceA, output.PriorityResource)
	s.Assert().Equal(linkDefResourceTypeA, output.PriorityResourceType)
}

func (s *LinkDefinitionTestSuite) linkDefinition() *LinkDefinition {
	return &LinkDefinition{
		ResourceTypeA: linkDefResourceTypeA,
		ResourceTypeB: linkDefResourceTypeB,
	}
}

func TestLinkDefinitionTestSuite(t *testing.T) {
	suite.Run(t, new(LinkDefinitionTestSuite))
}
