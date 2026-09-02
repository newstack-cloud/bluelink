package container

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type ChangeStagingLinkContributedFieldTestSuite struct {
	suite.Suite
}

const (
	contributedFieldInstanceID   = "link-contributed-field-instance"
	contributedFieldLinkDataPath = "eventsRule.otherArnValue"
)

// A resource field a link contributes must be reported as changing when the link says the
// value behind it will change.
//
// Both sides of a resource's comparison are composed from link data stored at the last
// deploy, so a contribution about to change is compared against its own old value and
// reports no difference. The link knows better, since it is staged with both endpoints'
// change sets in hand, but it says so against the path in its own data rather than against
// the resource field that path feeds. Without carrying that across, the change set holds a
// link saying the value will change on deploy beside a resource saying the field is
// unchanged, and the field changes silently when the deployment runs.
func (s *ChangeStagingLinkContributedFieldTestSuite) Test_reports_a_contributed_field_the_link_says_will_change() {
	changeSet := s.stageChangesWithContributedField(contributedFieldLinkDataPath)

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Assert().Contains(
		ruleChanges.FieldChangesKnownOnDeploy,
		"spec.otherArn",
		"the link reported the value behind this field as changing on deploy",
	)
	s.Assert().NotContains(
		ruleChanges.UnchangedFields,
		"spec.otherArn",
		"a field reported as changing must not also be reported as unchanged",
	)
}

// A contribution the link says nothing about stays unchanged, so the reclassification is
// driven by what the link reports rather than by the field being link-owned at all.
func (s *ChangeStagingLinkContributedFieldTestSuite) Test_leaves_a_contributed_field_the_link_does_not_touch() {
	changeSet := s.stageChangesWithContributedField("eventsRule.someOtherValue")

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Assert().NotContains(ruleChanges.FieldChangesKnownOnDeploy, "spec.otherArn")
	s.Assert().Contains(ruleChanges.UnchangedFields, "spec.otherArn")
}

func (s *ChangeStagingLinkContributedFieldTestSuite) stageChangesWithContributedField(
	linkReportsChangeAt string,
) *changes.BlueprintChanges {
	stateContainer := memstate.NewMemoryStateContainer()
	s.Require().NoError(seedContributedFieldInstance(stateContainer))

	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &internal.Lambda2FunctionResource{},
				"aws/events2/rule": &eventsRule2Resource{
					Lambda2FunctionResource: &internal.Lambda2FunctionResource{},
				},
			},
			Links: map[string]provider.Link{
				"aws/events2/rule::aws/lambda2/function": &contributingRuleLambda2Link{
					knownOnDeployPath: linkReportsChangeAt,
				},
			},
			CustomVariableTypes: map[string]provider.CustomVariableType{},
			DataSources:         map[string]provider.DataSource{},
		},
	}
	loader := NewDefaultLoader(
		providers,
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint-link-contributed-field.yml",
		params,
	)
	s.Require().NoError(err)

	channels := createChangeStagingChannels()
	err = blueprintContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: contributedFieldInstanceID},
		channels,
		params,
	)
	s.Require().NoError(err)

	for {
		select {
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case <-channels.ResourceChangesChan:
		case changeSet := <-channels.CompleteChan:
			return &changeSet
		case err := <-channels.ErrChan:
			s.Require().NoError(err)
		case <-time.After(defaultDrainTimeout):
			s.Require().NoError(errors.New(timeoutMessage))
		}
	}
}

// The instance as it was left by a previous deployment where the rule holds only what the
// blueprint declares, and the link holds the value it contributed to spec.otherArn along
// with the mapping saying which field of the rule that value belongs to.
func seedContributedFieldInstance(stateContainer state.Container) error {
	return stateContainer.Instances().Save(context.Background(), state.InstanceState{
		InstanceID:   contributedFieldInstanceID,
		InstanceName: "LinkContributedFieldInstance",
		Status:       core.InstanceStatusDeployed,
		ResourceIDs: map[string]string{
			"eventsRule":     "rule-1",
			"linkedFunction": "function-1",
		},
		Resources: map[string]*state.ResourceState{
			"rule-1": {
				ResourceID: "rule-1",
				Name:       "eventsRule",
				Type:       "aws/events2/rule",
				InstanceID: contributedFieldInstanceID,
				Status:     core.ResourceStatusCreated,
				SpecData: core.MappingNodeFields(
					"targetArn",
					core.MappingNodeFromString(
						"arn:aws:lambda:eu-west-2:123456789012:function:sync",
					),
				),
			},
			"function-1": {
				ResourceID: "function-1",
				Name:       "linkedFunction",
				Type:       "aws/lambda2/function",
				InstanceID: contributedFieldInstanceID,
				Status:     core.ResourceStatusCreated,
				SpecData: core.MappingNodeFields(
					"handler",
					core.MappingNodeFromString("src/sync.handler"),
				),
			},
		},
		Links: map[string]*state.LinkState{
			"eventsRule::linkedFunction": {
				LinkID:     "link-1",
				Name:       "eventsRule::linkedFunction",
				InstanceID: contributedFieldInstanceID,
				Status:     core.LinkStatusCreated,
				Data: map[string]*core.MappingNode{
					"eventsRule": core.MappingNodeFields(
						"otherArnValue",
						core.MappingNodeFromString("arn:aws:events:eu-west-2:123456789012:rule/old"),
					),
				},
				ResourceDataMappings: map[string]string{
					"eventsRule::spec.otherArn": contributedFieldLinkDataPath,
				},
			},
		},
	})
}

// Reports one path in its own link data as having a value that will not be known until
// deployment, which is what the framework has to translate into the rule's spec field.
type contributingRuleLambda2Link struct {
	testNoPriorityRuleLambda2Link
	knownOnDeployPath string
}

func (l *contributingRuleLambda2Link) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			FieldChangesKnownOnDeploy: []string{l.knownOnDeployPath},
		},
	}, nil
}

func TestChangeStagingLinkContributedFieldTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeStagingLinkContributedFieldTestSuite))
}
