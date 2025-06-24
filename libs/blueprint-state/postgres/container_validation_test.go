package postgres

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

const (
	existingBlueprintValidationID    = "96e38b14-2316-4614-9684-8384b4a0db11"
	nonExistentBlueprintValidationID = "0b04331f-de5d-44ce-9b74-0a39a3ef3c50"
)

type PostgresBlueprintValidationTestSuite struct {
	container                       *StateContainer
	connPool                        *pgxpool.Pool
	saveBlueprintValidationFixtures map[int]internal.SaveBlueprintValidationFixture
	suite.Suite
}

func (s *PostgresBlueprintValidationTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "blueprint-validations")
	saveFixtures, err := internal.SetupSaveBlueprintValidationFixtures(
		dirPath,
	)
	s.Require().NoError(err)
	s.saveBlueprintValidationFixtures = saveFixtures
}

func (s *PostgresBlueprintValidationTestSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresBlueprintValidationTestSuite) Test_retrieve_existing_blueprint_validation() {
	ctx := context.Background()
	blueprintValidation, err := s.container.Validation().Get(
		ctx,
		existingBlueprintValidationID,
	)
	s.Require().NoError(err)

	s.Require().Equal(expectedExistingBlueprintValidation, blueprintValidation)
}

func (s *PostgresBlueprintValidationTestSuite) Test_fails_to_retrieve_non_existent_blueprint_validation() {
	ctx := context.Background()
	_, err := s.container.Validation().Get(ctx, nonExistentBlueprintValidationID)
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.BlueprintValidationNotFound)
	s.Require().True(isNotFoundErr)
	s.Require().Equal(nonExistentBlueprintValidationID, notFoundErr.ID)
	s.Require().Equal(
		fmt.Sprintf(
			"blueprint validation request with ID %s not found",
			nonExistentBlueprintValidationID,
		),
		notFoundErr.Error(),
	)
}

func (s *PostgresBlueprintValidationTestSuite) Test_saves_blueprint_validation() {
	fixture := s.saveBlueprintValidationFixtures[1]

	validation := s.container.Validation()
	err := validation.Save(
		context.Background(),
		fixture.Validation,
	)
	s.Require().NoError(err)

	savedValidation, err := validation.Get(
		context.Background(),
		fixture.Validation.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedValidation)
	s.Assert().Equal(fixture.Validation, savedValidation)
}

func (s *PostgresBlueprintValidationTestSuite) Test_update_existing_blueprint_validation() {
	// Fixture 2 represents a blueprint validation that already exists
	// but with a change made to the "status" field.
	fixture := s.saveBlueprintValidationFixtures[2]

	validation := s.container.Validation()
	err := validation.Save(
		context.Background(),
		fixture.Validation,
	)
	s.Require().NoError(err)

	savedValidation, err := validation.Get(
		context.Background(),
		fixture.Validation.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedValidation)
	s.Assert().Equal(fixture.Validation, savedValidation)
}

func (s *PostgresBlueprintValidationTestSuite) Test_cleans_up_old_blueprint_validations() {
	err := s.container.Validation().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	for _, id := range validationsShouldBeCleanedUp {
		_, err := s.container.Validation().Get(
			context.Background(),
			id,
		)
		s.Require().Error(err)

		notFoundErr, isNotFoundErr := err.(*manage.BlueprintValidationNotFound)
		s.Require().True(isNotFoundErr)
		s.Assert().Equal(
			fmt.Sprintf("blueprint validation request with ID %s not found", id),
			notFoundErr.Error(),
		)
	}

	for _, id := range validationsShouldNotBeCleanedUp {
		blueprintValidation, err := s.container.Validation().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, blueprintValidation.ID)
	}
}

var expectedExistingBlueprintValidation = &manage.BlueprintValidation{
	ID:                existingBlueprintValidationID,
	Status:            manage.BlueprintValidationStatusValidated,
	BlueprintLocation: "s3://bluelink-test/project1/project.blueprint.yml",
	Created:           1745496983,
}

// Seed blueprint validations that should be cleaned up.
var validationsShouldBeCleanedUp = []string{
	"138a9325-b30b-4953-ac4e-5e049c5f3e8a",
	"b20604cd-48a2-4bb8-b8fd-a46ea4e2c25f",
	"caa1b4d4-5b52-4703-a6b0-6d067e98dcea",
	"1d574b3d-5674-4643-9ebf-19bcdfc90128",
}

// Seed blueprint validations that should not be cleaned up.
// This must not include the IDs of any dynamically generated validations
// in the test runs.
var validationsShouldNotBeCleanedUp = []string{
	"96e38b14-2316-4614-9684-8384b4a0db11",
}

func TestPostgresBlueprintValidationTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresBlueprintValidationTestSuite))
}
