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

type ChangeStagingUnappliedLinkFieldsTestSuite struct {
	suite.Suite
}

const (
	unappliedFieldsInstanceID = "unapplied-link-fields-instance"
	unappliedFieldsLinkName   = "eventsRule::linkedFunction"
	// The path the seeded link actually holds a value at, so a mapping pointing anywhere
	// else is one the link's data cannot satisfy.
	unappliedFieldsHeldPath = "eventsRule.otherArnValue"
	missingLinkDataReason   = "the link's data holds no value at this path"
)

// A contribution recorded against a resource that cannot be composed into its spec has to
// reach the change set.
func (s *ChangeStagingUnappliedLinkFieldsTestSuite) Test_reports_a_contribution_the_links_data_cannot_satisfy() {
	changeSet := s.stageChangesWithMappings(map[string]string{
		"eventsRule::spec.otherArn": "eventsRule.absentValue",
	})

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Require().Len(
		ruleChanges.UnappliedLinkFields,
		1,
		"the contribution the link's data holds no value for must be reported",
	)
	s.Assert().Equal(
		provider.UnappliedLinkField{
			LinkName:  unappliedFieldsLinkName,
			FieldPath: "spec.otherArn",
			Reason:    missingLinkDataReason,
		},
		ruleChanges.UnappliedLinkFields[0],
		"the report must name the link, the resource field and why it could not be composed",
	)
}

// The same unresolvable contribution is composed into both sides of the resource's
// comparison, so both report it. The change set has to state the problem once.
func (s *ChangeStagingUnappliedLinkFieldsTestSuite) Test_reports_a_contribution_once_though_both_sides_compose_it() {
	changeSet := s.stageChangesWithMappings(map[string]string{
		"eventsRule::spec.otherArn": "eventsRule.absentValue",
	})

	ruleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Assert().Equal(
		1,
		s.countReportsFor(ruleChanges.UnappliedLinkFields, "spec.otherArn"),
		"the current and desired sides both compose this contribution, "+
			"it must not be reported twice",
	)
}

// Reporting a contribution once must not collapse distinct contributions of the same link.
func (s *ChangeStagingUnappliedLinkFieldsTestSuite) Test_reports_every_unresolved_field_of_the_same_link() {
	changeSet := s.stageChangesWithMappings(map[string]string{
		"eventsRule::spec.otherArn":  "eventsRule.absentValue",
		"eventsRule::spec.secondArn": "eventsRule.alsoAbsentValue",
	})

	ruleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Assert().Equal(
		1,
		s.countReportsFor(ruleChanges.UnappliedLinkFields, "spec.otherArn"),
	)
	s.Assert().Equal(
		1,
		s.countReportsFor(ruleChanges.UnappliedLinkFields, "spec.secondArn"),
		"a second field of the same link is a distinct contribution and must be reported",
	)
}

// A contribution that composes is not a problem to report, so the change set stays silent
// about it. Without this, every link-owned field would be reported as unapplied and the
// report would say nothing.
func (s *ChangeStagingUnappliedLinkFieldsTestSuite) Test_reports_nothing_when_the_contribution_composes() {
	changeSet := s.stageChangesWithMappings(map[string]string{
		"eventsRule::spec.otherArn": unappliedFieldsHeldPath,
	})

	ruleChanges, hasRuleChanges := changeSet.ResourceChanges["eventsRule"]
	s.Require().True(hasRuleChanges, "the rule was dropped from the change set")
	s.Assert().Empty(
		ruleChanges.UnappliedLinkFields,
		"the link's data holds a value for this contribution, nothing is unapplied",
	)
	s.Assert().Equal(
		unappliedFieldsLinkName,
		ruleChanges.LinkOwnedFields["spec.otherArn"],
		"a contribution that composes is still attributed to the link that made it",
	)
}

func (s *ChangeStagingUnappliedLinkFieldsTestSuite) countReportsFor(
	unapplied []provider.UnappliedLinkField,
	fieldPath string,
) int {
	count := 0
	for _, field := range unapplied {
		if field.LinkName == unappliedFieldsLinkName && field.FieldPath == fieldPath {
			count++
		}
	}

	return count
}

func (s *ChangeStagingUnappliedLinkFieldsTestSuite) stageChangesWithMappings(
	mappings map[string]string,
) *changes.BlueprintChanges {
	stateContainer := memstate.NewMemoryStateContainer()
	s.Require().NoError(seedUnappliedFieldsInstance(stateContainer, mappings))

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
					// A path the link reports nothing about, so what the change set says
					// about these fields comes from composition alone.
					knownOnDeployPath: "eventsRule.someOtherValue",
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
		&StageChangesInput{InstanceID: unappliedFieldsInstanceID},
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

// An instance left by a previous deployment where the link holds one value and records the
// given mappings against the rule, so that a mapping pointing anywhere other than
// unappliedFieldsHeldPath is one that composition cannot satisfy.
func seedUnappliedFieldsInstance(
	stateContainer state.Container,
	mappings map[string]string,
) error {
	return stateContainer.Instances().Save(context.Background(), state.InstanceState{
		InstanceID:   unappliedFieldsInstanceID,
		InstanceName: "UnappliedLinkFieldsInstance",
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
				InstanceID: unappliedFieldsInstanceID,
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
				InstanceID: unappliedFieldsInstanceID,
				Status:     core.ResourceStatusCreated,
				SpecData: core.MappingNodeFields(
					"handler",
					core.MappingNodeFromString("src/sync.handler"),
				),
			},
		},
		Links: map[string]*state.LinkState{
			unappliedFieldsLinkName: {
				LinkID:     "link-1",
				Name:       unappliedFieldsLinkName,
				InstanceID: unappliedFieldsInstanceID,
				Status:     core.LinkStatusCreated,
				Data: map[string]*core.MappingNode{
					"eventsRule": core.MappingNodeFields(
						"otherArnValue",
						core.MappingNodeFromString(
							"arn:aws:events:eu-west-2:123456789012:rule/old",
						),
					),
				},
				ResourceDataMappings: mappings,
			},
		},
	})
}

func TestChangeStagingUnappliedLinkFieldsTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeStagingUnappliedLinkFieldsTestSuite))
}
