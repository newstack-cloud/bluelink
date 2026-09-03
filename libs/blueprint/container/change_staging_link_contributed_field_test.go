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

// A link created by this deployment has nothing in state saying what it contributes, so
// the resource it contributes to reports nothing about the field at all, rather than
// reporting it wrongly.
//
// Composition draws on link data recorded at the last deploy. The link itself is reported
// through NewOutboundLinks, so the deployment is not wholly silent about it, but the
// effect on the resource's spec goes missing: a field that appears when the deployment
// runs is absent from every category of the resource's changes beforehand. The link
// declaring the mappings it will produce is the only account of it that exists before the
// link has run.
func (s *ChangeStagingLinkContributedFieldTestSuite) Test_reports_a_contributed_field_declared_by_a_new_link() {
	changeSet := s.stageChangesWithNewLinkDeclaring(
		map[string]string{
			"eventsRule::spec.otherArn": contributedFieldLinkDataPath,
		},
	)

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Assert().Contains(
		ruleChanges.FieldChangesKnownOnDeploy,
		"spec.otherArn",
		"a field the new link declared and reports as changing must be reported",
	)
	s.Assert().Equal(
		"eventsRule::linkedFunction",
		ruleChanges.LinkOwnedFields["spec.otherArn"],
		"the reported field must be attributed to the link that declared it",
	)
}

// A declared contribution to a field the resource's changes account for in no category at
// all still has to be reported.
//
// A field the blueprint does not declare and no link in state contributes is composed into
// neither side of the resource's comparison, so it is not reported as unchanged and cannot
// be reached by moving fields between categories. It has to be added, or a field that
// appears when the deployment runs is absent from the change set entirely.
func (s *ChangeStagingLinkContributedFieldTestSuite) Test_reports_a_declared_field_absent_from_the_resource_comparison() {
	changeSet := s.stageChangesWithNewLinkReporting(
		map[string]string{
			"eventsRule::spec.unwrittenArn": "eventsRule.neverWrittenValue",
		},
		"eventsRule.neverWrittenValue",
	)

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Assert().Contains(
		ruleChanges.FieldChangesKnownOnDeploy,
		"spec.unwrittenArn",
		"a declared contribution the link reports as changing must be reported",
	)
	s.Assert().Equal(
		"eventsRule::linkedFunction",
		ruleChanges.LinkOwnedFields["spec.unwrittenArn"],
	)
}

// Declaring a target and then contributing nothing to it is allowed, so a declaration on
// its own does not put a field into the change set. The link has to report the path behind
// it as changing, which is the same gate a contribution already held in state passes.
func (s *ChangeStagingLinkContributedFieldTestSuite) Test_leaves_a_field_a_new_link_declares_but_does_not_touch() {
	changeSet := s.stageChangesWithNewLinkDeclaring(
		map[string]string{
			"eventsRule::spec.unwrittenArn": "eventsRule.neverWrittenValue",
		},
	)

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Assert().NotContains(ruleChanges.FieldChangesKnownOnDeploy, "spec.unwrittenArn")
	s.Assert().NotContains(ruleChanges.LinkOwnedFields, "spec.unwrittenArn")
}

func (s *ChangeStagingLinkContributedFieldTestSuite) stageChangesWithContributedField(
	linkReportsChangeAt string,
) *changes.BlueprintChanges {
	return s.stageChangesForLink(
		seedContributedFieldInstance,
		&contributingRuleLambda2Link{knownOnDeployPath: linkReportsChangeAt},
	)
}

// Staged against an instance holding no link, which is what state looks like when the
// link is created by the deployment being staged. The link reports the same data path as
// the existing-link cases, so what differs is only where its mappings come from.
func (s *ChangeStagingLinkContributedFieldTestSuite) stageChangesWithNewLinkDeclaring(
	declaredMappings map[string]string,
) *changes.BlueprintChanges {
	return s.stageChangesWithNewLinkReporting(declaredMappings, contributedFieldLinkDataPath)
}

func (s *ChangeStagingLinkContributedFieldTestSuite) stageChangesWithNewLinkReporting(
	declaredMappings map[string]string,
	linkReportsChangeAt string,
) *changes.BlueprintChanges {
	return s.stageChangesForLink(
		seedContributedFieldInstanceWithoutLink,
		&contributingRuleLambda2Link{
			knownOnDeployPath: linkReportsChangeAt,
			declaredMappings:  declaredMappings,
		},
	)
}

func (s *ChangeStagingLinkContributedFieldTestSuite) stageChangesForLink(
	seedInstance func(state.Container) error,
	ruleLambdaLink provider.Link,
) *changes.BlueprintChanges {
	stateContainer := memstate.NewMemoryStateContainer()
	s.Require().NoError(seedInstance(stateContainer))

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
				"aws/events2/rule::aws/lambda2/function": ruleLambdaLink,
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
	return saveContributedFieldInstance(stateContainer, map[string]*state.LinkState{
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
	})
}

// The same instance with the link absent, so nothing in state records what it contributes.
func seedContributedFieldInstanceWithoutLink(stateContainer state.Container) error {
	return saveContributedFieldInstance(stateContainer, map[string]*state.LinkState{})
}

func saveContributedFieldInstance(
	stateContainer state.Container,
	links map[string]*state.LinkState,
) error {
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
		Links: links,
	})
}

// Reports one path in its own link data as having a value that will not be known until
// deployment, which is what the framework has to translate into the rule's spec field.
type contributingRuleLambda2Link struct {
	testNoPriorityRuleLambda2Link
	knownOnDeployPath string
	declaredMappings  map[string]string
}

func (l *contributingRuleLambda2Link) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			FieldChangesKnownOnDeploy: []string{l.knownOnDeployPath},
			ResourceDataMappings:      l.declaredMappings,
		},
	}, nil
}

func TestChangeStagingLinkContributedFieldTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeStagingLinkContributedFieldTestSuite))
}
