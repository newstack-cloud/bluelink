package drift

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

const (
	sensitiveInstanceID   = "blueprint-instance-sensitive"
	sensitiveResourceID   = "sensitive-container-resource"
	sensitiveResourceName = "sensitiveContainer"
	sensitiveResourceType = "example/sensitiveContainer"
)

// Drift is reported against a resource type that declares sensitivity on its containers
// rather than on the leaves holding the values. This is how a plugin author declares it,
// marking every entry of a map of secrets is what the container is for.
//
// A fixture marking the leaves instead would pass whether or not sensitivity is inherited,
// which is the trap this suite exists to avoid.
type DriftCheckerSensitivityTestSuite struct {
	stateContainer state.Container
	driftChecker   Checker
	suite.Suite
}

func (s *DriftCheckerSensitivityTestSuite) SetupTest() {
	s.stateContainer = memstate.NewMemoryStateContainer()
	s.Require().NoError(s.populateCurrentState())

	s.driftChecker = NewDefaultChecker(
		s.stateContainer,
		map[string]provider.Provider{
			"example": &internal.ProviderMock{
				NamespaceValue: "example",
				Resources: map[string]provider.Resource{
					sensitiveResourceType: &driftingSensitiveContainerResource{
						externalState: s.externalState(),
					},
				},
				Links:               map[string]provider.Link{},
				CustomVariableTypes: map[string]provider.CustomVariableType{},
				DataSources:         map[string]provider.DataSource{},
				ProviderRetryPolicy: &provider.RetryPolicy{MaxRetries: 1},
			},
		},
		changes.NewDefaultResourceChangeGenerator(),
		core.SystemClock{},
		core.NewNopLogger(),
	)
}

func (s *DriftCheckerSensitivityTestSuite) populateCurrentState() error {
	instanceState := state.InstanceState{
		InstanceID: sensitiveInstanceID,
		Status:     core.InstanceStatusDeployed,
		ResourceIDs: map[string]string{
			sensitiveResourceName: sensitiveResourceID,
		},
		Resources: map[string]*state.ResourceState{
			sensitiveResourceID: {
				ResourceID:    sensitiveResourceID,
				Name:          sensitiveResourceName,
				Type:          sensitiveResourceType,
				InstanceID:    sensitiveInstanceID,
				Status:        core.ResourceStatusCreated,
				PreciseStatus: core.PreciseResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"name": core.MappingNodeFromString("parameter-tree"),
						"values": {
							Fields: map[string]*core.MappingNode{
								"region": core.MappingNodeFromString("us-east-1"),
							},
						},
						"secureValues": {
							Fields: map[string]*core.MappingNode{
								"dbPassword": core.MappingNodeFromString("persisted-password"),
							},
						},
						"credentials": {
							Fields: map[string]*core.MappingNode{
								"username": core.MappingNodeFromString("persisted-user"),
							},
						},
					},
				},
			},
		},
	}

	return s.stateContainer.Instances().Save(context.Background(), instanceState)
}

// What the provider reports the resource actually looks like where every field has drifted.
func (s *DriftCheckerSensitivityTestSuite) externalState() *core.MappingNode {
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"name": core.MappingNodeFromString("parameter-tree-renamed"),
			"values": {
				Fields: map[string]*core.MappingNode{
					"region": core.MappingNodeFromString("eu-west-2"),
				},
			},
			"secureValues": {
				Fields: map[string]*core.MappingNode{
					"dbPassword": core.MappingNodeFromString("external-password"),
				},
			},
			"credentials": {
				Fields: map[string]*core.MappingNode{
					"username": core.MappingNodeFromString("external-user"),
				},
			},
		},
	}
}

func (s *DriftCheckerSensitivityTestSuite) checkDrift() map[string]provider.FieldChange {
	driftState, err := s.driftChecker.CheckResourceDrift(
		context.Background(),
		sensitiveInstanceID,
		sensitiveInstanceID,
		sensitiveResourceID,
		createParams(),
		nil, // taggingConfig
	)
	s.Require().NoError(err)
	s.Require().NotNil(driftState)
	s.Require().NotNil(driftState.Difference)

	byPath := map[string]provider.FieldChange{}
	for _, fieldChange := range driftState.Difference.ModifiedFields {
		byPath[fieldChange.FieldPath] = provider.FieldChange{
			FieldPath: fieldChange.FieldPath,
			Sensitive: fieldChange.Sensitive,
		}
	}

	return byPath
}

// A field whose own schema declares nothing, inside a container that declares sensitivity,
// is sensitive. Change staging inherits it down the walk; drift has to agree, or the same
// value is marked in a changeset and unmarked in a drift record.
func (s *DriftCheckerSensitivityTestSuite) Test_drift_in_a_sensitive_object_is_marked_sensitive() {
	changesByPath := s.checkDrift()

	credentials, found := changesByPath["spec.credentials.username"]
	s.Require().True(found, "drift in a sensitive object was not reported at all")
	s.Assert().True(credentials.Sensitive, "drift in a sensitive object came out unmarked")
}

func (s *DriftCheckerSensitivityTestSuite) Test_drift_in_a_sensitive_map_is_marked_sensitive() {
	changesByPath := s.checkDrift()

	secureValue, found := changesByPath["spec.secureValues.dbPassword"]
	s.Require().True(found, "drift in a sensitive map was not reported at all")
	s.Assert().True(secureValue.Sensitive, "drift in a sensitive map came out unmarked")
}

// Drift inside a map that declares nothing has to be reported too, unmarked. This is the
// case that shows whether the schema walk descends into maps at all.
func (s *DriftCheckerSensitivityTestSuite) Test_drift_in_a_plain_map_is_reported_unmarked() {
	changesByPath := s.checkDrift()

	value, found := changesByPath["spec.values.region"]
	s.Require().True(found, "drift in a map was not reported at all")
	s.Assert().False(value.Sensitive)
}

func (s *DriftCheckerSensitivityTestSuite) Test_drift_in_a_field_declaring_nothing_is_unmarked() {
	changesByPath := s.checkDrift()

	name, found := changesByPath["spec.name"]
	s.Require().True(found, "drift in a plain field was not reported at all")
	s.Assert().False(name.Sensitive)
}

// The sensitive-container fixture reports no external state of its own, so the drifting
// state is supplied here.
type driftingSensitiveContainerResource struct {
	internal.ExampleSensitiveContainerResource
	externalState *core.MappingNode
}

func (r *driftingSensitiveContainerResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: r.externalState,
	}, nil
}

func TestDriftCheckerSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(DriftCheckerSensitivityTestSuite))
}
