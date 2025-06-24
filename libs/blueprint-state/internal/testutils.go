package internal

import (
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// AssertInstanceStatesEqual asserts that the actual instance state is equal to the expected instance state.
// This normalises nil slice and map fields to empty slices and maps as they are considered
// equivalent in this context.
func AssertInstanceStatesEqual(expected, actual *state.InstanceState, s *suite.Suite) {
	s.Assert().Equal(expected.InstanceID, actual.InstanceID)
	s.Assert().Equal(expected.Status, actual.Status)
	assertMapsEqual(expected.ResourceIDs, actual.ResourceIDs, s)
	assertChildDependenciesEqual(expected.ChildDependencies, actual.ChildDependencies, s)
	assertResourcesEqual(expected.Resources, actual.Resources, s)
	assertChildrenEqual(expected.ChildBlueprints, actual.ChildBlueprints, s)
	assertLinksEqual(expected.Links, actual.Links, s)

	s.Assert().Equal(expected.LastStatusUpdateTimestamp, actual.LastStatusUpdateTimestamp)
	s.Assert().Equal(expected.LastDeployAttemptTimestamp, actual.LastDeployAttemptTimestamp)
	s.Assert().Equal(expected.LastDeployedTimestamp, actual.LastDeployedTimestamp)
	s.Assert().Equal(expected.Durations, actual.Durations, s)
}

func assertResourcesEqual(expected, actual map[string]*state.ResourceState, s *suite.Suite) {
	s.Assert().Len(actual, len(expected))
	for resourceName, expectedResourceState := range expected {
		actualResourceState, ok := actual[resourceName]
		s.Assert().True(ok)
		AssertResourceStatesEqual(expectedResourceState, actualResourceState, s)
	}
}

// AssertResourceStatesEqual asserts that the actual resource state is equal to the expected resource state.
// This normalises nil slice and map fields to empty slices and maps as they are considered
// equivalent in this context.
func AssertResourceStatesEqual(expected, actual *state.ResourceState, s *suite.Suite) {
	s.Assert().Equal(expected.ResourceID, actual.ResourceID)
	s.Assert().Equal(expected.Status, actual.Status)
	s.Assert().Equal(expected.PreciseStatus, actual.PreciseStatus)
	s.Assert().Equal(expected.Name, actual.Name)
	s.Assert().Equal(expected.Type, actual.Type)
	s.Assert().Equal(expected.TemplateName, actual.TemplateName)
	s.Assert().Equal(expected.InstanceID, actual.InstanceID)
	s.Assert().Equal(expected.SpecData, actual.SpecData)
	s.Assert().Equal(expected.Description, actual.Description)
	assertResourceMetadataEqual(expected.Metadata, actual.Metadata, s)
	assertSlicesEqual(expected.DependsOnResources, actual.DependsOnResources, s)
	assertSlicesEqual(expected.DependsOnChildren, actual.DependsOnChildren, s)
	assertSlicesEqual(expected.FailureReasons, actual.FailureReasons, s)
	s.Assert().Equal(expected.LastStatusUpdateTimestamp, actual.LastStatusUpdateTimestamp)
	s.Assert().Equal(expected.LastDeployAttemptTimestamp, actual.LastDeployAttemptTimestamp)
	s.Assert().Equal(expected.LastDeployedTimestamp, actual.LastDeployedTimestamp)
	s.Assert().Equal(expected.Durations, actual.Durations)
}

func assertResourceMetadataEqual(
	expected *state.ResourceMetadataState,
	actual *state.ResourceMetadataState,
	s *suite.Suite,
) {
	if expected == nil {
		s.Assert().True(isEmptyResourceMetadata(actual))
		return
	}

	s.Assert().NotNil(actual)
	s.Assert().Equal(expected.DisplayName, actual.DisplayName)
	assertMapsEqual(expected.Annotations, actual.Annotations, s)
	assertMapsEqual(expected.Labels, actual.Labels, s)
	s.Assert().Equal(expected.Custom, actual.Custom)
}

func isEmptyResourceMetadata(actual *state.ResourceMetadataState) bool {
	return actual == nil || (actual.DisplayName == "" &&
		len(actual.Annotations) == 0 &&
		len(actual.Labels) == 0 &&
		actual.Custom == nil)
}

// AssertResourceDriftEqual asserts that the actual resource drift state is equal to the expected resource drift state.
// This normalises nil slice and map fields to empty slices and maps as they are considered
// equivalent in this context.
func AssertResourceDriftEqual(expected, actual *state.ResourceDriftState, s *suite.Suite) {
	s.Assert().Equal(expected.ResourceID, actual.ResourceID)
	s.Assert().Equal(expected.ResourceName, actual.ResourceName)
	s.Assert().Equal(expected.SpecData, actual.SpecData)
	s.Assert().Equal(expected.Timestamp, actual.Timestamp)
	assertResourceDriftDiffEqual(expected.Difference, actual.Difference, s)
}

func assertResourceDriftDiffEqual(expected, actual *state.ResourceDriftChanges, s *suite.Suite) {
	if expected == nil {
		s.Assert().Nil(actual)
		return
	}

	s.Assert().NotNil(actual)
	s.Assert().Equal(
		orderResourceDriftFieldChanges(expected.NewFields),
		orderResourceDriftFieldChanges(actual.NewFields),
	)
	s.Assert().Equal(
		orderResourceDriftFieldChanges(expected.ModifiedFields),
		orderResourceDriftFieldChanges(actual.ModifiedFields),
	)
	assertSlicesEqual(expected.RemovedFields, actual.RemovedFields, s)
	assertSlicesEqual(expected.UnchangedFields, actual.UnchangedFields, s)
}

func assertChildrenEqual(expected, actual map[string]*state.InstanceState, s *suite.Suite) {
	for childName, expectedChildState := range expected {
		actualChildState, ok := actual[childName]
		s.Assert().True(ok)
		AssertInstanceStatesEqual(expectedChildState, actualChildState, s)
	}
}

func assertLinksEqual(expected, actual map[string]*state.LinkState, s *suite.Suite) {
	s.Assert().Len(actual, len(expected))
	for linkName, expectedLink := range expected {
		actualV, ok := actual[linkName]
		s.Assert().True(ok)
		AssertLinkStatesEqual(expectedLink, actualV, s)
	}
}

// AssertLinkStatesEqual asserts that the actual link state is equal to the expected link state.
// This normalises nil slice and map fields to empty slices and maps as they are considered
// equivalent in this context.
func AssertLinkStatesEqual(
	expected *state.LinkState,
	actual *state.LinkState,
	s *suite.Suite,
) {
	s.Assert().Equal(expected.LinkID, actual.LinkID)
	s.Assert().Equal(expected.Status, actual.Status)
	s.Assert().Equal(expected.PreciseStatus, actual.PreciseStatus)
	s.Assert().Equal(expected.Name, actual.Name)
	s.Assert().Equal(expected.InstanceID, actual.InstanceID)
	assertMapsEqual(expected.Data, actual.Data, s)
	assertSlicesEqual(expected.FailureReasons, actual.FailureReasons, s)
	s.Assert().Equal(expected.LastStatusUpdateTimestamp, actual.LastStatusUpdateTimestamp)
	s.Assert().Equal(expected.LastDeployAttemptTimestamp, actual.LastDeployAttemptTimestamp)
	s.Assert().Equal(expected.LastDeployedTimestamp, actual.LastDeployedTimestamp)
	s.Assert().Equal(expected.Durations, actual.Durations)
}

func assertChildDependenciesEqual(expected, actual map[string]*state.DependencyInfo, s *suite.Suite) {
	s.Assert().Len(actual, len(expected))
	for k, v := range expected {
		actualV, ok := actual[k]
		s.Assert().True(ok)
		assertDependencyInfoEquals(v, actualV, s)
	}
}

func assertDependencyInfoEquals(expected, actual *state.DependencyInfo, s *suite.Suite) {
	s.Assert().Len(actual.DependsOnChildren, len(expected.DependsOnChildren))
	for i, v := range expected.DependsOnChildren {
		s.Assert().Equal(v, actual.DependsOnChildren[i])
	}

	s.Assert().Len(actual.DependsOnResources, len(expected.DependsOnResources))
	for i, v := range expected.DependsOnResources {
		s.Assert().Equal(v, actual.DependsOnResources[i])
	}
}

func assertSlicesEqual(
	expected []string,
	actual []string,
	s *suite.Suite,
) {
	if expected != nil {
		expectedCopy := make([]string, len(expected))
		copy(expectedCopy, expected)
		slices.Sort(expectedCopy)

		actualCopy := make([]string, len(actual))
		copy(actualCopy, actual)
		slices.Sort(actualCopy)

		s.Assert().Equal(expectedCopy, actualCopy)
	} else {
		s.Assert().Empty(actual)
	}
}

func assertMapsEqual[Item any](expected, actual map[string]Item, s *suite.Suite) {
	if expected != nil {
		s.Assert().Equal(expected, actual)
	} else {
		s.Assert().Empty(actual)
	}
}

func orderResourceDriftFieldChanges(
	fieldChanges []*state.ResourceDriftFieldChange,
) []*state.ResourceDriftFieldChange {
	orderedFieldChanges := make([]*state.ResourceDriftFieldChange, len(fieldChanges))
	copy(orderedFieldChanges, fieldChanges)
	slices.SortFunc(orderedFieldChanges, func(a, b *state.ResourceDriftFieldChange) int {
		if a.FieldPath < b.FieldPath {
			return -1
		}

		if a.FieldPath > b.FieldPath {
			return 1
		}

		return 0
	})
	return orderedFieldChanges
}

// AssertInstanceStatusInfo asserts that the actual instance status
// info is equal to the expected instance status info.
func AssertInstanceStatusInfo(
	expected state.InstanceStatusInfo,
	actual state.InstanceState,
	s *suite.Suite,
) {
	s.Assert().Equal(expected.Status, actual.Status)
	s.Assert().Equal(*expected.LastDeployedTimestamp, actual.LastDeployedTimestamp)
	s.Assert().Equal(*expected.LastDeployAttemptTimestamp, actual.LastDeployAttemptTimestamp)
	s.Assert().Equal(*expected.LastStatusUpdateTimestamp, actual.LastStatusUpdateTimestamp)
	s.Assert().Equal(expected.Durations, actual.Durations)
}

// CreateTestInstanceStatusInfo creates instance status info for testing status updates.
func CreateTestInstanceStatusInfo() state.InstanceStatusInfo {
	lastDeployedTimestamp := 1739660528
	lastDeployAttemptTimestamp := 1739660528
	lastStatusUpdateTimestamp := 1739660528
	prepareDuration := 1000.0
	totalDuration := 2000.0
	return state.InstanceStatusInfo{
		Status:                     core.InstanceStatusDeployFailed,
		LastDeployedTimestamp:      &lastDeployedTimestamp,
		LastDeployAttemptTimestamp: &lastDeployAttemptTimestamp,
		LastStatusUpdateTimestamp:  &lastStatusUpdateTimestamp,
		Durations: &state.InstanceCompletionDuration{
			PrepareDuration: &prepareDuration,
			TotalDuration:   &totalDuration,
		},
	}
}

// AssertResourceStatusInfo asserts that the actual resource status
// info is equal to the expected resource status info.
func AssertResourceStatusInfo(
	expected state.ResourceStatusInfo,
	actual state.ResourceState,
	s *suite.Suite,
) {
	s.Assert().Equal(expected.Status, actual.Status)
	s.Assert().Equal(expected.PreciseStatus, actual.PreciseStatus)
	s.Assert().Equal(*expected.LastDeployedTimestamp, actual.LastDeployedTimestamp)
	s.Assert().Equal(*expected.LastDeployAttemptTimestamp, actual.LastDeployAttemptTimestamp)
	s.Assert().Equal(*expected.LastStatusUpdateTimestamp, actual.LastStatusUpdateTimestamp)
	s.Assert().Equal(expected.FailureReasons, actual.FailureReasons)
	s.Assert().Equal(expected.Durations, actual.Durations)
}

// CreateTestResourceStatusInfo creates resource status info for testing status updates.
func CreateTestResourceStatusInfo() state.ResourceStatusInfo {
	lastDeployedTimestamp := 1739660528
	lastDeployAttemptTimestamp := 1739660528
	lastStatusUpdateTimestamp := 1739660528
	configCompleteDuration := 1000.0
	totalDuration := 2000.0
	attemptDurations := []float64{2000.0}
	return state.ResourceStatusInfo{
		Status:                     core.ResourceStatusCreateFailed,
		LastDeployedTimestamp:      &lastDeployedTimestamp,
		LastDeployAttemptTimestamp: &lastDeployAttemptTimestamp,
		LastStatusUpdateTimestamp:  &lastStatusUpdateTimestamp,
		FailureReasons:             []string{"Failed to create resource due to network error"},
		Durations: &state.ResourceCompletionDurations{
			ConfigCompleteDuration: &configCompleteDuration,
			TotalDuration:          &totalDuration,
			AttemptDurations:       attemptDurations,
		},
	}
}

// IsEmptyDriftState checks if all the values in a given drift state
// are empty.
func IsEmptyDriftState(actual state.ResourceDriftState) bool {
	return actual.ResourceID == "" &&
		actual.ResourceName == "" &&
		actual.SpecData == nil &&
		actual.Difference == nil &&
		actual.Timestamp == nil
}

// AssertLinkStatusInfo asserts that the actual link status
// info is equal to the expected link status info.
func AssertLinkStatusInfo(
	expected state.LinkStatusInfo,
	actual state.LinkState,
	s *suite.Suite,
) {
	s.Assert().Equal(expected.Status, actual.Status)
	s.Assert().Equal(expected.PreciseStatus, actual.PreciseStatus)
	s.Assert().Equal(*expected.LastDeployedTimestamp, actual.LastDeployedTimestamp)
	s.Assert().Equal(*expected.LastDeployAttemptTimestamp, actual.LastDeployAttemptTimestamp)
	s.Assert().Equal(*expected.LastStatusUpdateTimestamp, actual.LastStatusUpdateTimestamp)
	s.Assert().Equal(expected.FailureReasons, actual.FailureReasons)
	s.Assert().Equal(expected.Durations, actual.Durations)
}

// CreateTestLinkStatusInfo creates link status info for testing status updates.
func CreateTestLinkStatusInfo() state.LinkStatusInfo {
	lastDeployedTimestamp := 1739660528
	lastDeployAttemptTimestamp := 1739660528
	lastStatusUpdateTimestamp := 1739660528
	totalDuration := 2000.0
	resourceAAttemptDurations := []float64{2000.0}
	return state.LinkStatusInfo{
		Status:                     core.LinkStatusCreateFailed,
		LastDeployedTimestamp:      &lastDeployedTimestamp,
		LastDeployAttemptTimestamp: &lastDeployAttemptTimestamp,
		LastStatusUpdateTimestamp:  &lastStatusUpdateTimestamp,
		FailureReasons:             []string{"Failed to update resource A due to network error"},
		Durations: &state.LinkCompletionDurations{
			TotalDuration: &totalDuration,
			ResourceAUpdate: &state.LinkComponentCompletionDurations{
				AttemptDurations: resourceAAttemptDurations,
				TotalDuration:    &totalDuration,
			},
		},
	}
}

const (
	envVarsField = "variables.environment"
)

// SaveAllExportsInput returns a map of export states for testing
// behaviour to save exports.
func SaveAllExportsInput() map[string]*state.ExportState {
	return map[string]*state.ExportState{
		"environment": {
			Value: core.MappingNodeFromString("production"),
			Type:  schema.ExportTypeString,
			Field: envVarsField,
		},
		"region": {
			Value: core.MappingNodeFromString("us-west-1"),
			Type:  schema.ExportTypeString,
			Field: "variables.region",
		},
		"exampleId": {
			Value: core.MappingNodeFromString("exampleId"),
			Type:  schema.ExportTypeString,
			Field: "spec.id",
		},
	}
}

// SaveSingleExportInput returns a single export state for testing
// behaviour to save a single export.
func SaveSingleExportInput() state.ExportState {
	return state.ExportState{
		Value: core.MappingNodeFromString("exampleId"),
		Type:  schema.ExportTypeString,
		Field: "spec.id",
	}
}

// SaveMetadataInput returns a map of metadata for testing
// behaviour to save metadata.
func SaveMetadataInput() map[string]*core.MappingNode {
	return map[string]*core.MappingNode{
		"build":    core.MappingNodeFromString("esbuild"),
		"otherKey": core.MappingNodeFromString("otherValue"),
	}
}
