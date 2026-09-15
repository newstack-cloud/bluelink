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
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

const skippedTriggerWarning = "the trigger for consumer \"userEvents\" was not recognised " +
	"and has been skipped"

type ChangeStagingDiagnosticsTestSuite struct {
	suite.Suite
}

// A warning raised while the blueprint was transformed has to reach the change set.
func (s *ChangeStagingDiagnosticsTestSuite) Test_carries_a_transform_warning_into_the_change_set() {
	changeSet := s.stageChangesForBlueprintWithTransform()

	s.Require().NotEmpty(
		changeSet.Diagnostics,
		"the change set carries no diagnostics, so a warning that a feature was "+
			"skipped cannot reach the operator",
	)

	messages := []string{}
	for _, diagnostic := range changeSet.Diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	s.Assert().Contains(messages, skippedTriggerWarning)
}

// The level travels with the message, since a client has to tell a warning it should
// surface from an informational note it may not.
func (s *ChangeStagingDiagnosticsTestSuite) Test_carries_the_level_of_a_transform_warning() {
	changeSet := s.stageChangesForBlueprintWithTransform()

	for _, diagnostic := range changeSet.Diagnostics {
		if diagnostic.Message == skippedTriggerWarning {
			s.Assert().Equal(core.DiagnosticLevelWarning, diagnostic.Level)
			return
		}
	}

	s.Fail("the warning was not in the change set at all")
}

func (s *ChangeStagingDiagnosticsTestSuite) stageChangesForBlueprintWithTransform() *changes.BlueprintChanges {
	stateContainer := memstate.NewMemoryStateContainer()

	providers := map[string]provider.Provider{
		"aws": &internal.ProviderMock{
			NamespaceValue: "aws",
			Resources: map[string]provider.Resource{
				"aws/lambda2/function": &internal.Lambda2FunctionResource{},
			},
			Links:               map[string]provider.Link{},
			CustomVariableTypes: map[string]provider.CustomVariableType{},
			DataSources:         map[string]provider.DataSource{},
		},
	}

	loader := NewDefaultLoader(
		providers,
		map[string]transform.SpecTransformer{
			"test-warning-2026": &warningTransformer{},
		},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(true),
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
		"__testdata/container/change-staging/blueprint-transform-diagnostics.yml",
		params,
	)
	s.Require().NoError(err)

	channels := createChangeStagingChannels()
	err = blueprintContainer.StageChanges(
		context.Background(),
		&StageChangesInput{},
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

// A transform that leaves the blueprint alone and reports something it could not act on,
// which is what a transform does with a feature it does not recognise.
type warningTransformer struct {
	internal.CelerityTransformer
}

func (t *warningTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: skippedTriggerWarning,
			},
		},
	}, nil
}

func (t *warningTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-warning-2026", nil
}

func TestChangeStagingDiagnosticsTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeStagingDiagnosticsTestSuite))
}
