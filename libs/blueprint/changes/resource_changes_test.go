package changes

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type ResourceChangeGeneratorTestSuite struct {
	resourceChangeGenerator *defaultResourceChangeGenerator
	suite.Suite
}

func (s *ResourceChangeGeneratorTestSuite) SetupSuite() {
	s.resourceChangeGenerator = &defaultResourceChangeGenerator{}
}

func (s *ResourceChangeGeneratorTestSuite) Test_generates_changes_for_existing_resource() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture1(),
		&internal.ExampleComplexResource{},
		[]string{
			"resources.complexResource.spec.itemConfig.endpoints[2]",
			"resources.complexResource.spec.itemConfig.endpoints[4]",
			"resources.complexResource.metadata.annotations[\"test.annotation.v1\"]",
			"resources.complexResource.metadata.custom.url",
		},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, false /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func (s *ResourceChangeGeneratorTestSuite) Test_generates_changes_for_new_resource() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture2(),
		&internal.ExampleComplexResource{},
		[]string{
			"resources.complexResource.spec.itemConfig.endpoints[3]",
			"resources.complexResource.metadata.annotations[\"test.annotation.v1\"]",
			"resources.complexResource.metadata.custom.url",
		},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, false /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func (s *ResourceChangeGeneratorTestSuite) Test_does_not_generate_changes_for_fields_exceeding_max_depth() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture3(),
		&internal.ExampleComplexResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, true /* excludeResourceInfo */))
	s.Require().NoError(err)
}

func (s *ResourceChangeGeneratorTestSuite) Test_generates_changes_for_existing_resource_with_new_resource_type() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture4(),
		&internal.ExampleComplexResource{},
		[]string{
			"resources.complexResource.spec.itemConfig.endpoints[2]",
			"resources.complexResource.spec.itemConfig.endpoints[4]",
			"resources.complexResource.metadata.annotations[\"test.annotation.v1\"]",
			"resources.complexResource.metadata.custom.url",
		},
		nil,
	)
	s.Require().NoError(err)

	err = testhelpers.Snapshot(internal.NormaliseResourceChanges(changes, false /* excludeResourceInfo */))
	s.Require().NoError(err)
}

// Test_no_changes_when_tags_differ_only_in_order verifies that when tags have
// identical key/value pairs but in different order, no changes are detected
// when the schema uses SortArrayByField="key".
func (s *ResourceChangeGeneratorTestSuite) Test_no_changes_when_tags_differ_only_in_order() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture5(),
		&internal.ExampleTaggableResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	// Verify no field changes were detected
	s.Empty(changes.ModifiedFields, "expected no modified fields when tags only differ in order")
	s.Empty(changes.NewFields, "expected no new fields when tags only differ in order")
	s.Empty(changes.RemovedFields, "expected no removed fields when tags only differ in order")

	// Tag fields should be in unchanged fields (at the leaf level)
	// After sorting, both arrays have the same order, so all leaf fields should be unchanged
	s.NotEmpty(changes.UnchangedFields, "expected unchanged fields to be populated")

	// Verify that tag leaf fields are in unchanged fields
	hasTagFields := false
	for _, field := range changes.UnchangedFields {
		if len(field) >= 9 && field[:9] == "spec.tags" {
			hasTagFields = true
			break
		}
	}
	s.True(hasTagFields, "expected tag fields to be in unchanged fields")
}

// Test_detects_tag_value_change_despite_different_order verifies that actual
// changes to tag values are correctly detected even when the array order differs.
func (s *ResourceChangeGeneratorTestSuite) Test_detects_tag_value_change_despite_different_order() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture6(),
		&internal.ExampleTaggableResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	// Should detect the changed tag value (env: staging -> production)
	// After sorting by key, "env" will be at index 0, so the change path should be spec.tags[0].value
	s.NotEmpty(changes.ModifiedFields, "expected modified fields when tag value changed")

	// Verify the specific change was detected
	foundEnvChange := false
	for _, change := range changes.ModifiedFields {
		if change.FieldPath == "spec.tags[0].value" {
			foundEnvChange = true
			s.Equal("staging", *change.PrevValue.Scalar.StringValue, "expected previous value to be 'staging'")
			s.Equal("production", *change.NewValue.Scalar.StringValue, "expected new value to be 'production'")
		}
	}
	s.True(foundEnvChange, "expected to find the env tag value change at spec.tags[0].value")
}

// Test_no_changes_when_nullable_field_has_default_value_in_external_state verifies
// that when a nullable field with a schema default is nil in the persisted spec,
// and GetExternalState returns the default value, no changes are detected.
func (s *ResourceChangeGeneratorTestSuite) Test_no_changes_when_nullable_field_has_default_value_in_external_state() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture7(),
		&internal.ExampleNullableFieldResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	// Verify no new fields were detected for the nullable fields with defaults
	// (delaySeconds and maximumMessageSize should not appear as new fields)
	for _, newField := range changes.NewFields {
		s.NotEqual("spec.delaySeconds", newField.FieldPath,
			"delaySeconds should not be detected as a new field when it equals the default")
		s.NotEqual("spec.maximumMessageSize", newField.FieldPath,
			"maximumMessageSize should not be detected as a new field when it equals the default")
	}

	// The nullable fields with default values should be in unchanged fields
	hasDelaySeconds := false
	hasMaxMessageSize := false
	for _, field := range changes.UnchangedFields {
		if field == "spec.delaySeconds" {
			hasDelaySeconds = true
		}
		if field == "spec.maximumMessageSize" {
			hasMaxMessageSize = true
		}
	}
	s.True(hasDelaySeconds, "delaySeconds should be in unchanged fields")
	s.True(hasMaxMessageSize, "maximumMessageSize should be in unchanged fields")
}

// Test_detects_changes_when_nullable_field_has_non_default_value_in_external_state verifies
// that when a nullable field is nil in the persisted spec but GetExternalState returns
// a NON-default value, the drift is correctly detected.
func (s *ResourceChangeGeneratorTestSuite) Test_detects_changes_when_nullable_field_has_non_default_value_in_external_state() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture8(),
		&internal.ExampleNullableFieldResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	// Should detect the drifted value (delaySeconds = 30 instead of default 0)
	foundDelaySecondsChange := false
	for _, newField := range changes.NewFields {
		if newField.FieldPath == "spec.delaySeconds" {
			foundDelaySecondsChange = true
			s.Nil(newField.PrevValue, "expected previous value to be nil")
			s.NotNil(newField.NewValue, "expected new value to be set")
			s.Equal(30, *newField.NewValue.Scalar.IntValue, "expected new value to be 30")
		}
	}
	s.True(foundDelaySecondsChange, "expected to detect delaySeconds drift when value differs from default")
}

// Test_detects_changes_when_explicit_value_differs_from_external_state verifies
// that when a nullable field has an explicit value in the persisted spec that
// differs from the external state, the change is correctly detected.
func (s *ResourceChangeGeneratorTestSuite) Test_detects_changes_when_explicit_value_differs_from_external_state() {
	changes, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		s.resourceInfoFixture9(),
		&internal.ExampleNullableFieldResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	// Should detect the modified value (delaySeconds: 10 -> 30)
	foundDelaySecondsChange := false
	for _, modifiedField := range changes.ModifiedFields {
		if modifiedField.FieldPath == "spec.delaySeconds" {
			foundDelaySecondsChange = true
			s.NotNil(modifiedField.PrevValue, "expected previous value to be set")
			s.Equal(10, *modifiedField.PrevValue.Scalar.IntValue, "expected previous value to be 10")
			s.NotNil(modifiedField.NewValue, "expected new value to be set")
			s.Equal(30, *modifiedField.NewValue.Scalar.IntValue, "expected new value to be 30")
		}
	}
	s.True(foundDelaySecondsChange, "expected to detect delaySeconds change when explicit value differs from external state")
}

func TestResourceChangeGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceChangeGeneratorTestSuite))
}
