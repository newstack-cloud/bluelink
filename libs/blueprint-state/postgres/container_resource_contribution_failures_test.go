package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type PostgresStateContainerContributionFailuresTestSuite struct {
	container state.Container
	connPool  *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerContributionFailuresTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

// Every test leaves the resource as it found it, since the seeded resource is shared with
// the other suites and a failure left behind would be read by them as their own.
func (s *PostgresStateContainerContributionFailuresTestSuite) TearDownTest() {
	ctx := context.Background()
	for _, layerDepth := range []int{0, 2} {
		_ = s.container.Resources().RemoveContributionFailure(ctx, existingResourceID, layerDepth)
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerContributionFailuresTestSuite) Test_saves_a_contribution_failure_against_a_resource() {
	resources := s.container.Resources()
	failure := postgresContributionFailureFixture(
		0,
		"the role could not be given what its links contribute",
	)

	err := resources.SaveContributionFailure(context.Background(), existingResourceID, failure)
	s.Require().NoError(err)

	resourceState, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)
	s.Require().Len(resourceState.LinkContributionFailures, 1)
	s.Assert().Equal(failure, resourceState.LinkContributionFailures[0])
}

// The resource's own deployment is a separate outcome, and a reader cannot tell a resource
// that failed to deploy from one that deployed and was then left without its contributions
// if saving one disturbs the other.
func (s *PostgresStateContainerContributionFailuresTestSuite) Test_saving_a_failure_leaves_the_resources_own_status_alone() {
	resources := s.container.Resources()

	before, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)

	err = resources.SaveContributionFailure(
		context.Background(),
		existingResourceID,
		postgresContributionFailureFixture(0, "the role could not be written"),
	)
	s.Require().NoError(err)

	after, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)
	s.Assert().Equal(before.Status, after.Status)
	s.Assert().Equal(before.PreciseStatus, after.PreciseStatus)
	s.Assert().Equal(before.FailureReasons, after.FailureReasons)
	s.Assert().Equal(before.Drifted, after.Drifted)
	s.Assert().Equal(before.LastDeployedTimestamp, after.LastDeployedTimestamp)
}

func (s *PostgresStateContainerContributionFailuresTestSuite) Test_replaces_the_failure_held_for_the_same_layer() {
	resources := s.container.Resources()
	ctx := context.Background()

	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, postgresContributionFailureFixture(0, "first attempt"),
	))
	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, postgresContributionFailureFixture(0, "second attempt"),
	))

	resourceState, err := resources.Get(ctx, existingResourceID)
	s.Require().NoError(err)
	s.Require().Len(
		resourceState.LinkContributionFailures,
		1,
		"a layer failing twice accumulated an entry per attempt",
	)
	s.Assert().Equal(
		[]string{"second attempt"},
		resourceState.LinkContributionFailures[0].Reasons,
	)
}

func (s *PostgresStateContainerContributionFailuresTestSuite) Test_removes_only_the_layer_that_was_applied() {
	resources := s.container.Resources()
	ctx := context.Background()

	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, postgresContributionFailureFixture(0, "the placement link"),
	))
	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, postgresContributionFailureFixture(2, "the access link"),
	))

	s.Require().NoError(resources.RemoveContributionFailure(ctx, existingResourceID, 0))

	resourceState, err := resources.Get(ctx, existingResourceID)
	s.Require().NoError(err)
	s.Require().Len(
		resourceState.LinkContributionFailures,
		1,
		"applying one layer cleared a failure belonging to another",
	)
	s.Assert().Equal(2, resourceState.LinkContributionFailures[0].LayerDepth)
}

func (s *PostgresStateContainerContributionFailuresTestSuite) Test_holds_nothing_once_the_last_failure_is_removed() {
	resources := s.container.Resources()
	ctx := context.Background()

	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, postgresContributionFailureFixture(0, "the placement link"),
	))
	s.Require().NoError(resources.RemoveContributionFailure(ctx, existingResourceID, 0))

	resourceState, err := resources.Get(ctx, existingResourceID)
	s.Require().NoError(err)
	s.Assert().Empty(resourceState.LinkContributionFailures)
}

// The common case is a layer that applies successfully having never failed, so clearing one
// that is not held must not be reported as a mistake.
func (s *PostgresStateContainerContributionFailuresTestSuite) Test_removing_a_layer_that_never_failed_is_not_an_error() {
	resources := s.container.Resources()

	err := resources.RemoveContributionFailure(context.Background(), existingResourceID, 0)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerContributionFailuresTestSuite) Test_reports_resource_not_found_for_saving_a_failure() {
	resources := s.container.Resources()

	err := resources.SaveContributionFailure(
		context.Background(),
		nonExistentResourceID,
		postgresContributionFailureFixture(0, "the role could not be written"),
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerContributionFailuresTestSuite) Test_reports_resource_not_found_for_removing_a_failure() {
	resources := s.container.Resources()

	err := resources.RemoveContributionFailure(context.Background(), nonExistentResourceID, 0)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func postgresContributionFailureFixture(
	layerDepth int,
	reason string,
) state.ResourceLinkContributionFailure {
	return state.ResourceLinkContributionFailure{
		LayerDepth: layerDepth,
		Reasons:    []string{reason},
		UnappliedContributions: []state.UnappliedLinkContribution{
			{
				LinkName:  "saveOrderFunction::ordersRole",
				FieldPath: "spec.policies",
				Reason:    "the value to inject does not match the selector that would locate it",
			},
		},
		ContributingLinks: []string{"saveOrderFunction::ordersRole"},
		Timestamp:         1678901234,
	}
}

func TestPostgresStateContainerContributionFailuresTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerContributionFailuresTestSuite))
}
