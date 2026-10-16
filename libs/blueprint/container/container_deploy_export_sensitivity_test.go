package container

import (
	"context"
	"errors"
	"fmt"
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

// An exported secret is served as it is, since a blueprint exports one precisely so other
// blueprints can consume it, so the persisted flag is all a client has to redact on.
//
// Marking has to be precise in both directions. A missed mark displays a secret; a spurious
// one hides a value that is not secret, and an export marked from an unrelated secret
// resolved on the way would do exactly that.
type ContainerDeployExportSensitivityTestSuite struct {
	exports map[string]*state.ExportState
	suite.Suite
}

// Deployed once: every test reads the same persisted exports and none of them mutate state.
func (s *ContainerDeployExportSensitivityTestSuite) SetupSuite() {
	s.exports, _ = s.deployInto(memstate.NewMemoryStateContainer(), "")
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_marks_an_export_of_a_secret_variable() {
	s.Assert().True(s.exports["dbPassword"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_marks_an_export_of_a_secret_value() {
	s.Assert().True(s.exports["adminToken"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_leaves_an_export_of_a_plain_variable_unmarked() {
	s.Assert().False(s.exports["environment"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_leaves_an_export_of_a_plain_resource_field_unmarked() {
	s.Assert().False(s.exports["functionId"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_does_not_mark_from_a_secret_in_a_sibling_field() {
	s.Assert().False(
		s.exports["connectionHost"].Sensitive,
		"an export was marked from a secret in a sibling field of the value it read",
	)
}

// The case the whole item exists for: a secret that reached a field by interpolation, with
// nothing in any schema saying the field is sensitive. Resolution memoises values and
// resources alike, so the mark has to survive a cache hit to arrive here at all.
func (s *ContainerDeployExportSensitivityTestSuite) Test_marks_an_interpolated_secret_in_a_value() {
	s.Assert().True(s.exports["connectionPassword"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_marks_an_interpolated_secret_in_a_resource_spec() {
	s.Assert().True(s.exports["functionHandler"].Sensitive)
}

// Deploying again must not quietly unmark an export. A deployment resolves only the
// resources it is touching, so a reference to one it skipped reads a cache the resolver
// never filled. The mark survives because what is cached for a skipped resource is the
// partially resolved one from the change set, which carries its recorded paths.
func (s *ContainerDeployExportSensitivityTestSuite) Test_a_redeploy_keeps_the_marks_on_its_exports() {
	stateContainer := memstate.NewMemoryStateContainer()
	first, instanceID := s.deployInto(stateContainer, "")
	s.Require().True(first["functionHandler"].Sensitive)

	second, _ := s.deployInto(stateContainer, instanceID)

	s.Assert().True(second["functionHandler"].Sensitive, "a resource mark was lost on redeploy")
	s.Assert().True(second["connectionPassword"].Sensitive, "a value mark was lost on redeploy")
	s.Assert().False(second["functionId"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) Test_staged_export_changes_carry_the_same_marks() {
	staged := s.stageExportChanges()

	s.Assert().True(staged["dbPassword"].Sensitive)
	s.Assert().True(staged["adminToken"].Sensitive)
	s.Assert().True(staged["functionHandler"].Sensitive)
	s.Assert().True(staged["connectionPassword"].Sensitive)
	s.Assert().False(staged["environment"].Sensitive)
	s.Assert().False(staged["connectionHost"].Sensitive)
}

func (s *ContainerDeployExportSensitivityTestSuite) stageExportChanges() map[string]provider.FieldChange {
	stagingContainer, params := s.loadExportSensitivityBlueprint(memstate.NewMemoryStateContainer())

	staged, err := stageChangesForTest(context.Background(), stagingContainer, params)
	s.Require().NoError(err)
	// functionId exports a computed field, which cannot be known until deploy time, so it
	// defers rather than appearing in the change set.
	s.Require().Len(staged.NewExports, 6)

	return staged.NewExports
}

func (s *ContainerDeployExportSensitivityTestSuite) deployInto(
	stateContainer state.Container,
	instanceID string,
) (map[string]*state.ExportState, string) {
	// Staging and deploying are separate requests to the engine and so separate containers,
	// each with its own resolver caches. Sharing one here would leave the value cache warm
	// for the export resolution and hide what a cold one records.
	stagingContainer, params := s.loadExportSensitivityBlueprint(stateContainer)
	deployChanges, err := stageChangesForTest(context.Background(), stagingContainer, params)
	s.Require().NoError(err)

	blueprintContainer, _ := s.loadExportSensitivityBlueprint(stateContainer)

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		deployInputFor(instanceID, deployChanges),
		channels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage, err := drainDeployChannels(channels)
	s.Require().NoError(err)
	s.Require().NotNil(finishedMessage)
	s.Require().Contains(
		[]core.InstanceStatus{core.InstanceStatusDeployed, core.InstanceStatusUpdated},
		finishedMessage.Status,
		fmt.Sprintf("deployment failed: %v", finishedMessage.FailureReasons),
	)

	savedExports, err := stateContainer.Exports().GetAll(
		context.Background(),
		finishedMessage.InstanceID,
	)
	s.Require().NoError(err)
	s.Require().Len(savedExports, 7)

	return savedExports, finishedMessage.InstanceID
}

func (s *ContainerDeployExportSensitivityTestSuite) loadExportSensitivityBlueprint(
	stateContainer state.Container,
) (BlueprintContainer, core.BlueprintParams) {
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
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderResourceStabilityPollingConfig(&ResourceStabilityPollingConfig{
			PollingInterval: 10 * time.Millisecond,
			PollingTimeout:  1 * time.Second,
		}),
		WithLoaderLogger(core.NewNopLogger()),
	)

	environment := "production"
	dbPassword := "super-secret-password"
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			"environment": {StringValue: &environment},
			"dbPassword":  {StringValue: &dbPassword},
		},
	)

	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-export-sensitivity.yml",
		params,
	)
	s.Require().NoError(err)

	return blueprintContainer, params
}

func deployInputFor(instanceID string, deployChanges *changes.BlueprintChanges) *DeployInput {
	if instanceID != "" {
		return &DeployInput{InstanceID: instanceID, Changes: deployChanges}
	}

	return &DeployInput{InstanceName: "ExportSensitivityInstance", Changes: deployChanges}
}

func drainDeployChannels(channels *DeployChannels) (*DeploymentFinishedMessage, error) {
	for {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case <-channels.DeploymentUpdateChan:
		case msg := <-channels.FinishChan:
			return &msg, nil
		case err := <-channels.ErrChan:
			return nil, err
		case <-time.After(defaultDrainTimeout):
			return nil, errors.New(timeoutMessage)
		}
	}
}

func TestContainerDeployExportSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerDeployExportSensitivityTestSuite))
}
