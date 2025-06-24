package memfile

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	existingBluepritnValidationID    = "96e38b14-2316-4614-9684-8384b4a0db11"
	nonExistentBlueprintValidationID = "0b04331f-de5d-44ce-9b74-0a39a3ef3c50"
)

type MemFileStateContainerValidationSuite struct {
	container              *StateContainer
	stateDir               string
	fs                     afero.Fs
	saveValidationFixtures map[int]internal.SaveBlueprintValidationFixture
	suite.Suite
}

func (s *MemFileStateContainerValidationSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// Use a low max guide file size of 100 bytes to trigger the logic that splits
	// blueprint validation state across multiple chunk files.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "blueprint-validations")
	fixtures, err := internal.SetupSaveBlueprintValidationFixtures(
		dirPath,
	)
	s.Require().NoError(err)
	s.saveValidationFixtures = fixtures
}

func (s *MemFileStateContainerValidationSuite) Test_retrieve_blueprint_validation() {
	validationStore := s.container.Validation()
	blueprintValidation, err := validationStore.Get(
		context.Background(),
		existingBluepritnValidationID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(blueprintValidation)
	err = testhelpers.Snapshot(blueprintValidation)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerValidationSuite) Test_fails_to_retrieve_non_existent_blueprint_validation() {
	validationStore := s.container.Validation()

	_, err := validationStore.Get(
		context.Background(),
		nonExistentBlueprintValidationID,
	)
	s.Require().Error(err)
	validationNotFoundErr, isValidationNotFoundErr := err.(*manage.BlueprintValidationNotFound)
	s.Assert().True(isValidationNotFoundErr)
	s.Assert().EqualError(
		validationNotFoundErr,
		fmt.Sprintf(
			"blueprint validation request with ID %s not found",
			nonExistentBlueprintValidationID,
		),
	)
}

func (s *MemFileStateContainerValidationSuite) Test_saves_new_blueprint_validation() {
	fixture := s.saveValidationFixtures[1]

	validationStore := s.container.Validation()
	err := validationStore.Save(
		context.Background(),
		fixture.Validation,
	)
	s.Require().NoError(err)

	savedValidation, err := validationStore.Get(
		context.Background(),
		fixture.Validation.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedValidation)
	s.Assert().Equal(fixture.Validation, savedValidation)

	s.assertPersistedBlueprintValidation(fixture.Validation)
}

func (s *MemFileStateContainerValidationSuite) Test_updates_existing_blueprint_validation() {
	fixture := s.saveValidationFixtures[2]

	validationStore := s.container.Validation()
	err := validationStore.Save(
		context.Background(),
		fixture.Validation,
	)
	s.Require().NoError(err)

	savedValidation, err := validationStore.Get(
		context.Background(),
		fixture.Validation.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedValidation)
	s.Assert().Equal(fixture.Validation, savedValidation)

	s.assertPersistedBlueprintValidation(fixture.Validation)
}

func (s *MemFileStateContainerValidationSuite) Test_cleans_up_old_validations() {
	err := s.container.Validation().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	assertBlueprintValidationsCleanedUp(
		s.container,
		&s.Suite,
	)

	// Assert that the blueprint validations are cleaned up when loading a fresh
	// state container from file, ensuring that the cleanup
	// operation was persisted correctly.
	s.assertBlueprintValidationCleanupPersisted()
}

func (s *MemFileStateContainerValidationSuite) assertPersistedBlueprintValidation(
	expected *manage.BlueprintValidation,
) {
	// Check that the blueprint validation state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the validation request.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	validationStore := container.Validation()
	persistedValidation, err := validationStore.Get(
		context.Background(),
		expected.ID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		expected,
		persistedValidation,
	)
}

func (s *MemFileStateContainerValidationSuite) assertBlueprintValidationCleanupPersisted() {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	assertBlueprintValidationsCleanedUp(
		container,
		&s.Suite,
	)
}

func assertBlueprintValidationsCleanedUp(
	container *StateContainer,
	s *suite.Suite,
) {
	for _, id := range validationsShouldBeCleanedUp {
		_, err := container.Validation().Get(
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
		blueprintValidation, err := container.Validation().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, blueprintValidation.ID)
	}
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

func TestMemFileStateContainerValidationSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerValidationSuite))
}
