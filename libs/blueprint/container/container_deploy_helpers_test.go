package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

func assertDeployMessageOrder(
	actual *actualMessages,
	expected *expectedMessages,
	testSuite *suite.Suite,
) {
	assertResourceUpdateMessageOrder(
		actual.resourceDeployUpdateMessages,
		expected.ResourceDeployUpdateMessages,
		testSuite,
	)
	assertChildUpdateMessageOrder(
		actual.childDeployUpdateMessages,
		expected.ChildDeployUpdateMessages,
		testSuite,
	)
	assertLinkUpdateMessageOrder(
		actual.linkDeployUpdateMessages,
		expected.LinkDeployUpdateMessages,
		testSuite,
	)
	assertDeploymentUpdateMessageOrder(
		actual.deploymentUpdateMessages,
		expected.DeploymentUpdateMessages,
		testSuite,
	)

	if expected.FinishedMessage != nil {
		assertFinishedMessage(
			*actual.finishedMessage,
			*expected.FinishedMessage,
			testSuite,
		)
	}
}

// Assert that the order of messages for each resource from a deployment
// are in the expected order.
//
// This will only compare the following fields:
// - InstanceID
// - ResourceID
// - ResourceName
// - Group
// - Status
// - PreciseStatus
// - FailureReasons
// - Attempt
// - CanRetry
// - UpdateTimestamp (Only checks for presence of timestamp)
// - Durations (Only checks for presence of duration fields)
//
// For all numeric fields where the value will not be compared,
// the value should be set to -1 in the expected messages.
func assertResourceUpdateMessageOrder(
	messages []ResourceDeployUpdateMessage,
	// Each element in the top-level slice represents the expected order of messages for a
	// a given resource.
	expectedPerResource [][]ResourceDeployUpdateMessage,
	testSuite *suite.Suite,
) {
	for _, expectedSequence := range expectedPerResource {
		if len(expectedSequence) > 0 {
			messagesForResource := getMessagesForResource(messages, expectedSequence[0].ResourceName)
			testSuite.Assert().Len(messagesForResource, len(expectedSequence))
			assertResourceMessagesEqual(messagesForResource, expectedSequence, testSuite)
		}
	}
}

func getMessagesForResource(
	messages []ResourceDeployUpdateMessage,
	resourceName string,
) []ResourceDeployUpdateMessage {
	resourceMessages := []ResourceDeployUpdateMessage{}
	for _, message := range messages {
		if message.ResourceName == resourceName {
			resourceMessages = append(resourceMessages, message)
		}
	}
	return resourceMessages
}

func assertResourceMessagesEqual(
	messages []ResourceDeployUpdateMessage,
	expectedMessages []ResourceDeployUpdateMessage,
	testSuite *suite.Suite,
) {
	for i, message := range messages {
		expected := expectedMessages[i]
		assertAllowForDynamicValue(expected.InstanceID, message.InstanceID, message, testSuite)
		assertAllowForDynamicValue(expected.ResourceID, message.ResourceID, message, testSuite)
		testSuite.Assert().Equal(expected.ResourceName, message.ResourceName, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.Status, message.Status, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.PreciseStatus, message.PreciseStatus, "actual message: %+v", message)
		assertSlicesEqual(expected.FailureReasons, message.FailureReasons, testSuite)
		testSuite.Assert().Equal(expected.Attempt, message.Attempt, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.CanRetry, message.CanRetry, "actual message: %+v", message)
		assertTimestampPresent(expected.UpdateTimestamp, message.UpdateTimestamp, testSuite)
		assertResourceMessageDurations(expected.Durations, message.Durations, testSuite)
	}
}

func assertResourceMessageDurations(
	expectedDurations *state.ResourceCompletionDurations,
	actualDurations *state.ResourceCompletionDurations,
	testSuite *suite.Suite,
) {
	if expectedDurations != nil {
		testSuite.Assert().NotNil(actualDurations)
		if expectedDurations.TotalDuration != nil {
			testSuite.Assert().NotNil(actualDurations.TotalDuration)
		}

		if expectedDurations.ConfigCompleteDuration != nil {
			testSuite.Assert().NotNil(actualDurations.ConfigCompleteDuration)
		}

		if expectedDurations.AttemptDurations != nil {
			assertAttemptDurationsPresent(
				expectedDurations.AttemptDurations,
				actualDurations.AttemptDurations,
				testSuite,
			)
		}

	}
}

// Assert that the order of messages for each child blueprint from a deployment
// are in the expected order.
//
// This will only compare the following fields:
// - ParentInstanceID
// - ChildInstanceID
// - ChildName
// - Group
// - Status
// - FailureReasons
// - UpdateTimestamp (Only checks for presence of timestamp)
// - Durations (Only checks for presence of duration fields)
//
// For all numeric fields where the value will not be compared,
// the value should be set to -1 in the expected messages.
func assertChildUpdateMessageOrder(
	messages []ChildDeployUpdateMessage,
	// Each element in the top-level slice represents the expected order of messages for a
	// a given child blueprint.
	expectedPerChild [][]ChildDeployUpdateMessage,
	testSuite *suite.Suite,
) {
	for _, expectedSequence := range expectedPerChild {
		if len(expectedSequence) > 0 {
			messagesForChild := getMessagesForChild(messages, expectedSequence[0].ChildName)
			testSuite.Assert().Len(messagesForChild, len(expectedSequence))
			assertChildMessagesEqual(messagesForChild, expectedSequence, testSuite)
		}
	}
}

func getMessagesForChild(
	messages []ChildDeployUpdateMessage,
	childName string,
) []ChildDeployUpdateMessage {
	childMessages := []ChildDeployUpdateMessage{}
	for _, message := range messages {
		if message.ChildName == childName {
			childMessages = append(childMessages, message)
		}
	}
	return childMessages
}

func assertChildMessagesEqual(
	messages []ChildDeployUpdateMessage,
	expectedMessages []ChildDeployUpdateMessage,
	testSuite *suite.Suite,
) {
	for i, message := range messages {
		expected := expectedMessages[i]
		assertAllowForDynamicValue(expected.ParentInstanceID, message.ParentInstanceID, message, testSuite)
		assertAllowForDynamicValue(expected.ChildInstanceID, message.ChildInstanceID, message, testSuite)
		testSuite.Assert().Equal(expected.ChildName, message.ChildName, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.Group, message.Group, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.Status, message.Status, "actual message: %+v", message)
		assertSlicesEqual(expected.FailureReasons, message.FailureReasons, testSuite)
		assertTimestampPresent(expected.UpdateTimestamp, message.UpdateTimestamp, testSuite)
		assertChildMessageDurations(expected.Durations, message.Durations, testSuite)
	}
}

func assertChildMessageDurations(
	expectedDurations *state.InstanceCompletionDuration,
	actualDurations *state.InstanceCompletionDuration,
	testSuite *suite.Suite,
) {
	if expectedDurations != nil {
		testSuite.Assert().NotNil(actualDurations)
		if expectedDurations.TotalDuration != nil {
			testSuite.Assert().NotNil(actualDurations.TotalDuration)
		}

		if expectedDurations.PrepareDuration != nil {
			testSuite.Assert().NotNil(actualDurations.PrepareDuration)
		}
	}
}

// Assert that the order of messages for each link from a deployment
// are in the expected order.
//
// This will only compare the following fields:
// - InstanceID
// - LinkID
// - LinkName
// - Status
// - PreciseStatus
// - CurrentStageAttempt
// - CanRetryCurrentStage
// - FailureReasons
// - UpdateTimestamp (Only checks for presence of timestamp)
// - Durations (Only checks for presence of duration fields)
//
// For all numeric fields where the value will not be compared,
// the value should be set to -1 in the expected messages.
func assertLinkUpdateMessageOrder(
	messages []LinkDeployUpdateMessage,
	// Each element in the top-level slice represents the expected order of messages for a
	// a given link.
	expectedPerLink [][]LinkDeployUpdateMessage,
	testSuite *suite.Suite,
) {
	for _, expectedSequence := range expectedPerLink {
		if len(expectedSequence) > 0 {
			messagesForLink := getMessagesForLink(messages, expectedSequence[0].LinkName)
			testSuite.Assert().Len(messagesForLink, len(expectedSequence), "expected: %+v", expectedSequence)
			assertLinkMessagesEqual(messagesForLink, expectedSequence, testSuite)
		}
	}
}

func getMessagesForLink(
	messages []LinkDeployUpdateMessage,
	linkName string,
) []LinkDeployUpdateMessage {
	linkMessages := []LinkDeployUpdateMessage{}
	for _, message := range messages {
		if message.LinkName == linkName {
			linkMessages = append(linkMessages, message)
		}
	}
	return linkMessages
}

func assertLinkMessagesEqual(
	messages []LinkDeployUpdateMessage,
	expectedMessages []LinkDeployUpdateMessage,
	testSuite *suite.Suite,
) {
	for i, message := range messages {
		expected := expectedMessages[i]
		assertAllowForDynamicValue(expected.InstanceID, message.InstanceID, message, testSuite)
		assertAllowForDynamicValue(expected.LinkID, message.LinkID, message, testSuite)
		testSuite.Assert().Equal(expected.LinkName, message.LinkName, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.Status, message.Status, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.PreciseStatus, message.PreciseStatus, "actual message: %+v", message)
		assertSlicesEqual(expected.FailureReasons, message.FailureReasons, testSuite)
		testSuite.Assert().Equal(expected.CurrentStageAttempt, message.CurrentStageAttempt, "actual message: %+v", message)
		testSuite.Assert().Equal(expected.CanRetryCurrentStage, message.CanRetryCurrentStage, "actual message: %+v", message)
		assertTimestampPresent(expected.UpdateTimestamp, message.UpdateTimestamp, testSuite)
		assertLinkMessageDurations(expected.Durations, message.Durations, testSuite)
	}
}

func assertLinkMessageDurations(
	expectedDurations *state.LinkCompletionDurations,
	actualDurations *state.LinkCompletionDurations,
	testSuite *suite.Suite,
) {
	if expectedDurations != nil {
		testSuite.Assert().NotNil(actualDurations)
		if expectedDurations.TotalDuration != nil {
			testSuite.Assert().NotNil(actualDurations.TotalDuration)
		}

		if expectedDurations.ResourceAUpdate != nil {
			assertLinkComponentAttemptDurationsPresent(
				expectedDurations.ResourceAUpdate,
				actualDurations.ResourceAUpdate,
				testSuite,
			)
		}

		if expectedDurations.ResourceBUpdate != nil {
			assertLinkComponentAttemptDurationsPresent(
				expectedDurations.ResourceBUpdate,
				actualDurations.ResourceBUpdate,
				testSuite,
			)
		}

		if expectedDurations.IntermediaryResources != nil {
			assertLinkComponentAttemptDurationsPresent(
				expectedDurations.IntermediaryResources,
				actualDurations.IntermediaryResources,
				testSuite,
			)
		}
	}
}

func assertLinkComponentAttemptDurationsPresent(
	expectedDurations *state.LinkComponentCompletionDurations,
	actualDurations *state.LinkComponentCompletionDurations,
	testSuite *suite.Suite,
) {
	if expectedDurations != nil {
		testSuite.Assert().NotNil(actualDurations)
		if expectedDurations.TotalDuration != nil {
			testSuite.Assert().NotNil(actualDurations.TotalDuration)
		}

		if expectedDurations.AttemptDurations != nil {
			assertAttemptDurationsPresent(
				expectedDurations.AttemptDurations,
				actualDurations.AttemptDurations,
				testSuite,
			)
		}
	}
}

// Assert that the order of messages for the top-level deployment
// are in the expected order.
//
// This will compare the following fields:
// - InstanceID
// - Status
// - UpdateTimestamp (Only checks for presence of timestamp)
//
// For all numeric fields where the value will not be compared,
// the value should be set to -1 in the expected messages.
func assertDeploymentUpdateMessageOrder(
	messages []DeploymentUpdateMessage,
	// Each element in the top-level slice represents the expected order of messages for a
	// a deployment.
	expectedPerDeployment [][]DeploymentUpdateMessage,
	testSuite *suite.Suite,
) {
	for _, expectedSequence := range expectedPerDeployment {
		if len(expectedSequence) > 0 {
			testSuite.Assert().Len(messages, len(expectedSequence))
			assertDeploymentMessagesEqual(messages, expectedSequence, testSuite)
		}
	}
}

func assertDeploymentMessagesEqual(
	messages []DeploymentUpdateMessage,
	expectedMessages []DeploymentUpdateMessage,
	testSuite *suite.Suite,
) {
	for i, message := range messages {
		expected := expectedMessages[i]
		assertAllowForDynamicValue(expected.InstanceID, message.InstanceID, message, testSuite)
		testSuite.Assert().Equal(expected.Status, message.Status)
		assertTimestampPresent(expected.UpdateTimestamp, message.UpdateTimestamp, testSuite)
	}
}

// Assert that the finished message is as expected.
//
// This will compare the following fields:
// - InstanceID
// - Status
// - FailureReasons
// - FinishTimestamp (Only checks for presence of timestamp)
// - UpdateTimestamp (Only checks for presence of timestamp)
// - Durations (Only checks for presence of duration fields)
//
// For all numeric fields where the value will not be compared,
// the value should be set to -1 in the expected messages.
func assertFinishedMessage(
	message DeploymentFinishedMessage,
	expected DeploymentFinishedMessage,
	testSuite *suite.Suite,
) {
	assertAllowForDynamicValue(expected.InstanceID, message.InstanceID, message, testSuite)
	testSuite.Assert().Equal(expected.Status, message.Status)
	assertSlicesEqual(expected.FailureReasons, message.FailureReasons, testSuite)
	assertTimestampPresent(expected.FinishTimestamp, message.FinishTimestamp, testSuite)
	assertTimestampPresent(expected.UpdateTimestamp, message.UpdateTimestamp, testSuite)
	assertFinishedMessageDurations(expected.Durations, message.Durations, testSuite)
}

func assertFinishedMessageDurations(
	expectedDurations *state.InstanceCompletionDuration,
	actualDurations *state.InstanceCompletionDuration,
	testSuite *suite.Suite,
) {
	if expectedDurations != nil {
		testSuite.Assert().NotNil(actualDurations)
		if expectedDurations.TotalDuration != nil {
			testSuite.Assert().NotNil(actualDurations.TotalDuration)
		}

		if expectedDurations.PrepareDuration != nil {
			testSuite.Assert().NotNil(actualDurations.PrepareDuration)
		}
	}
}

func assertTimestampPresent(
	expectedTimestamp int64,
	actualTimestamp int64,
	testSuite *suite.Suite,
) {
	if expectedTimestamp != 0 {
		testSuite.Assert().NotEqual(int64(0), actualTimestamp)
	}
}

func assertAttemptDurationsPresent(
	expectedDurations []float64,
	actualDurations []float64,
	testSuite *suite.Suite,
) {
	testSuite.Assert().Len(actualDurations, len(expectedDurations))
}

func assertSlicesEqual(
	expected []string,
	actual []string,
	testSuite *suite.Suite,
) {
	if expected != nil {
		expectedCopy := make([]string, len(expected))
		copy(expectedCopy, expected)
		slices.Sort(expectedCopy)

		actualCopy := make([]string, len(actual))
		copy(actualCopy, actual)
		slices.Sort(actualCopy)

		testSuite.Assert().Equal(expectedCopy, actualCopy)
	} else {
		testSuite.Assert().Empty(actual)
	}
}

func assertMapsEqual[Item any](
	expected map[string]Item,
	actual map[string]Item,
	testSuite *suite.Suite,
) {
	if expected != nil {
		testSuite.Assert().Equal(expected, actual)
	} else {
		testSuite.Assert().Empty(actual)
	}
}

func assertInstanceStateEquals(
	expected *state.InstanceState,
	actual *state.InstanceState,
	testSuite *suite.Suite,
) {
	assertAllowForDynamicValue(expected.InstanceID, actual.InstanceID, actual, testSuite)
	assertAllowForDynamicValue(expected.InstanceName, actual.InstanceName, actual, testSuite)
	testSuite.Assert().Equal(expected.Status, actual.Status)
	assertResourceIDsMapKeysEqual(expected.ResourceIDs, actual.ResourceIDs, testSuite)
	assertChildDependenciesEqual(expected.ChildDependencies, actual.ChildDependencies, testSuite)

	for expectedResourceID, expectedResourceState := range expected.Resources {
		// The expected resources can have a placeholder in the form of "{idOf::resourceName}"
		// for the resource map key.
		resourceID, hasResourceID := getResourceIDForAssertion(expectedResourceID, actual.ResourceIDs)
		testSuite.Assert().True(
			hasResourceID,
			"expected resource ID: %s actual resource IDs: %+v",
			expectedResourceID,
			actual.ResourceIDs,
		)
		actualResourceState, ok := actual.Resources[resourceID]
		testSuite.Assert().True(ok)
		assertResourceStateEquals(expectedResourceState, actualResourceState, testSuite)
	}

	for childName, expectedChildState := range expected.ChildBlueprints {
		actualChildState, ok := actual.ChildBlueprints[childName]
		testSuite.Assert().True(ok)
		assertInstanceStateEquals(expectedChildState, actualChildState, testSuite)
	}

	for linkName, expectedLinkState := range expected.Links {
		actualLinkState, ok := actual.Links[linkName]
		testSuite.Assert().True(ok)
		assertLinkStateEquals(expectedLinkState, actualLinkState, testSuite)
	}

	assertTimestampPresent(
		int64(expected.LastStatusUpdateTimestamp),
		int64(actual.LastStatusUpdateTimestamp),
		testSuite,
	)
	assertTimestampPresent(
		int64(expected.LastDeployAttemptTimestamp),
		int64(actual.LastDeployAttemptTimestamp),
		testSuite,
	)
	assertTimestampPresent(
		int64(expected.LastDeployedTimestamp),
		int64(actual.LastDeployedTimestamp),
		testSuite,
	)
	assertFinishedMessageDurations(expected.Durations, actual.Durations, testSuite)
}

func getResourceIDForAssertion(
	expectedResourceID string,
	actualResourceIDs map[string]string,
) (string, bool) {
	if strings.HasPrefix(expectedResourceID, "{idOf::") && strings.HasSuffix(expectedResourceID, "}") {
		// The resource ID placeholder is in the form of "{idOf::resourceName}"
		// so we can extract the resource name by removing the first 7 characters
		// and the last character.
		resourceName := expectedResourceID[7 : len(expectedResourceID)-1]
		resourceID, hasResourceID := actualResourceIDs[resourceName]
		return resourceID, hasResourceID
	}

	return expectedResourceID, true
}

// Make sure that each resource name in the expected map are also present in the actual map.
// As resource IDs can be dynamically generated, for more robust testing, we will only
// make sure that each expected resource name has an ID mapping.
func assertResourceIDsMapKeysEqual(
	expected map[string]string,
	actual map[string]string,
	testSuite *suite.Suite,
) {
	testSuite.Assert().Len(actual, len(expected))
	expectedNames := []string{}
	for expectedName := range expected {
		expectedNames = append(expectedNames, expectedName)
	}

	actualNames := []string{}
	for actualName := range actual {
		actualNames = append(actualNames, actualName)
	}

	slices.Sort(expectedNames)
	slices.Sort(actualNames)
	testSuite.Assert().Equal(expectedNames, actualNames)
}

func assertResourceStateEquals(
	expected *state.ResourceState,
	actual *state.ResourceState,
	testSuite *suite.Suite,
) {
	assertAllowForDynamicValue(expected.ResourceID, actual.ResourceID, actual, testSuite)
	testSuite.Assert().Equal(expected.Status, actual.Status)
	testSuite.Assert().Equal(expected.PreciseStatus, actual.PreciseStatus)
	testSuite.Assert().Equal(expected.Name, actual.Name)
	testSuite.Assert().Equal(expected.Type, actual.Type)
	testSuite.Assert().Equal(expected.TemplateName, actual.TemplateName)
	assertAllowForDynamicValue(expected.InstanceID, actual.InstanceID, actual, testSuite)
	testSuite.Assert().Equal(expected.SpecData, actual.SpecData)
	testSuite.Assert().Equal(expected.Description, actual.Description)
	assertResourceMetadataEquals(expected.Metadata, actual.Metadata, testSuite)
	assertSlicesEqual(expected.DependsOnResources, actual.DependsOnResources, testSuite)
	assertSlicesEqual(expected.DependsOnChildren, actual.DependsOnChildren, testSuite)
	assertSlicesEqual(expected.FailureReasons, actual.FailureReasons, testSuite)
	assertTimestampPresent(
		int64(expected.LastStatusUpdateTimestamp),
		int64(actual.LastStatusUpdateTimestamp),
		testSuite,
	)
	assertTimestampPresent(
		int64(expected.LastDeployAttemptTimestamp),
		int64(actual.LastDeployAttemptTimestamp),
		testSuite,
	)
	assertTimestampPresent(
		int64(expected.LastDeployedTimestamp),
		int64(actual.LastDeployedTimestamp),
		testSuite,
	)
	assertResourceMessageDurations(expected.Durations, actual.Durations, testSuite)
}

func assertResourceMetadataEquals(
	expected *state.ResourceMetadataState,
	actual *state.ResourceMetadataState,
	testSuite *suite.Suite,
) {
	if expected == nil {
		testSuite.Assert().Nil(actual)
		return
	}

	testSuite.Assert().NotNil(actual)
	testSuite.Assert().Equal(expected.DisplayName, actual.DisplayName)
	assertMapsEqual(expected.Annotations, actual.Annotations, testSuite)
	assertMapsEqual(expected.Labels, actual.Labels, testSuite)
	testSuite.Assert().Equal(expected.Custom, actual.Custom)
}

func assertLinkStateEquals(
	expected *state.LinkState,
	actual *state.LinkState,
	testSuite *suite.Suite,
) {
	assertAllowForDynamicValue(expected.LinkID, actual.LinkID, actual, testSuite)
	testSuite.Assert().Equal(expected.Status, actual.Status)
	testSuite.Assert().Equal(expected.PreciseStatus, actual.PreciseStatus)
	testSuite.Assert().Equal(expected.Name, actual.Name)
	assertAllowForDynamicValue(expected.InstanceID, actual.InstanceID, actual, testSuite)
	assertMapsEqual(expected.Data, actual.Data, testSuite)
	assertMapsEqual(expected.ResourceDataMappings, actual.ResourceDataMappings, testSuite)
	assertSlicesEqual(expected.FailureReasons, actual.FailureReasons, testSuite)
	assertTimestampPresent(
		int64(expected.LastStatusUpdateTimestamp),
		int64(actual.LastStatusUpdateTimestamp),
		testSuite,
	)
	assertTimestampPresent(
		int64(expected.LastDeployAttemptTimestamp),
		int64(actual.LastDeployAttemptTimestamp),
		testSuite,
	)
	assertTimestampPresent(
		int64(expected.LastDeployedTimestamp),
		int64(actual.LastDeployedTimestamp),
		testSuite,
	)
	assertLinkMessageDurations(expected.Durations, actual.Durations, testSuite)
}

func assertChildDependenciesEqual(
	expected map[string]*state.DependencyInfo,
	actual map[string]*state.DependencyInfo,
	testSuite *suite.Suite,
) {
	testSuite.Assert().Len(actual, len(expected))
	for expectedChildName, expectedDependencyInfo := range expected {
		actualDependencyInfo, ok := actual[expectedChildName]
		testSuite.Assert().True(ok)

		assertDependencyInfoEqual(expectedDependencyInfo, actualDependencyInfo, testSuite)
	}
}

func assertDependencyInfoEqual(
	expected *state.DependencyInfo,
	actual *state.DependencyInfo,
	testSuite *suite.Suite,
) {
	assertSlicesEqual(expected.DependsOnResources, actual.DependsOnResources, testSuite)
	assertSlicesEqual(expected.DependsOnChildren, actual.DependsOnChildren, testSuite)
}

func assertAllowForDynamicValue(
	expected string,
	actual string,
	value interface{},
	testSuite *suite.Suite,
) {
	if expected == "{dynamicValue}" {
		testSuite.Assert().NotEmpty(actual, "actual containing value: %+v", value)
	} else {
		testSuite.Assert().Equal(expected, actual, "actual containing value: %+v", value)
	}
}

func createBlueprintDeployFixture(
	deployType string,
	fixtureNo int,
	loader Loader,
	params core.BlueprintParams,
	blueprintFormat schema.SpecFormat,
) (blueprintDeployFixture, error) {
	extension := "yml"
	if blueprintFormat == schema.JWCCSpecFormat {
		extension = "jsonc"
	}

	blueprintContainer, err := loader.Load(
		context.Background(),
		fmt.Sprintf("__testdata/container/%s/blueprint%d.%s", deployType, fixtureNo, extension),
		params,
	)
	if err != nil {
		return blueprintDeployFixture{}, err
	}

	expectedMessagesFilePath := fmt.Sprintf(
		"__testdata/container/%s/expected-messages/blueprint%d.json",
		deployType,
		fixtureNo,
	)
	expectedInstanceStateFilePath := fmt.Sprintf(
		"__testdata/container/%s/expected-state/blueprint%d.json",
		deployType,
		fixtureNo,
	)
	return createBlueprintDeployFixtureFromFile(
		blueprintContainer,
		expectedMessagesFilePath,
		expectedInstanceStateFilePath,
	)
}

func createBlueprintDeployFixtureFromFile(
	container BlueprintContainer,
	expectedMessagesFilePath string,
	expectedInstanceStateFilePath string,
) (blueprintDeployFixture, error) {
	expectedMessages, err := loadExpectedMessagesFromFile(expectedMessagesFilePath)
	if err != nil {
		return blueprintDeployFixture{}, err
	}

	expectedInstanceState, err := internal.LoadInstanceState(expectedInstanceStateFilePath)
	if err != nil {
		// Expected instance state is not required for all tests.
		return blueprintDeployFixture{
			blueprintContainer: container,
			expected:           expectedMessages,
		}, nil
	}

	return blueprintDeployFixture{
		blueprintContainer:    container,
		expected:              expectedMessages,
		expectedInstanceState: expectedInstanceState,
	}, nil
}

func loadExpectedMessagesFromFile(
	expectedMessagesFilePath string,
) (*expectedMessages, error) {
	expectedMessagesBytes, err := os.ReadFile(expectedMessagesFilePath)
	if err != nil {
		return nil, err
	}

	expectedMessages := &expectedMessages{}
	err = json.Unmarshal(expectedMessagesBytes, expectedMessages)
	if err != nil {
		return nil, err
	}

	return expectedMessages, nil
}

func loadBlueprintChangesFromFile(
	changesFilePath string,
) (*changes.BlueprintChanges, error) {
	changesFileBytes, err := os.ReadFile(changesFilePath)
	if err != nil {
		return nil, err
	}

	changes := &changes.BlueprintChanges{}
	err = json.Unmarshal(changesFileBytes, changes)
	if err != nil {
		return nil, err
	}

	return changes, nil
}

func populateCurrentState(
	fixtureInstances []int,
	stateContainer state.Container,
	containerTestType string,
) error {
	for _, instanceNo := range fixtureInstances {
		err := populateBlueprintCurrentState(
			stateContainer,
			fmt.Sprintf("blueprint-instance-%d", instanceNo),
			instanceNo,
			containerTestType,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func populateBlueprintCurrentState(
	stateContainer state.Container,
	instanceID string,
	blueprintNo int,
	containerTestType string,
) error {
	blueprintCurrentState, err := internal.LoadInstanceState(
		fmt.Sprintf(
			"__testdata/container/%s/current-state/blueprint%d.json",
			containerTestType,
			blueprintNo,
		),
	)
	if err != nil {
		return err
	}
	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprintCurrentState,
	)
	if err != nil {
		return err
	}

	blueprintChildCurrentState, err := internal.LoadInstanceState(
		fmt.Sprintf(
			"__testdata/container/%s/current-state/blueprint%d-child-core-infra.json",
			containerTestType,
			blueprintNo,
		),
	)
	if err != nil {
		return err
	}

	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprintChildCurrentState,
	)
	if err != nil {
		return err
	}

	return stateContainer.Children().Attach(
		context.Background(),
		instanceID,
		blueprintChildCurrentState.InstanceID,
		"coreInfra",
	)
}

type blueprintDeployFixture struct {
	blueprintContainer    BlueprintContainer
	expected              *expectedMessages
	expectedInstanceState *state.InstanceState
}

type expectedMessages struct {
	ResourceDeployUpdateMessages [][]ResourceDeployUpdateMessage `json:"resourceDeployUpdateMessages"`
	ChildDeployUpdateMessages    [][]ChildDeployUpdateMessage    `json:"childDeployUpdateMessages"`
	LinkDeployUpdateMessages     [][]LinkDeployUpdateMessage     `json:"linkDeployUpdateMessages"`
	DeploymentUpdateMessages     [][]DeploymentUpdateMessage     `json:"deploymentUpdateMessages"`
	FinishedMessage              *DeploymentFinishedMessage      `json:"finishedMessage,omitempty"`
}

type actualMessages struct {
	resourceDeployUpdateMessages []ResourceDeployUpdateMessage
	childDeployUpdateMessages    []ChildDeployUpdateMessage
	linkDeployUpdateMessages     []LinkDeployUpdateMessage
	deploymentUpdateMessages     []DeploymentUpdateMessage
	finishedMessage              *DeploymentFinishedMessage
}

type staticResourceSubstitutionResolver struct {
	resolvedResource *provider.ResolvedResource
}

func (s *staticResourceSubstitutionResolver) ResolveInResource(
	ctx context.Context,
	resourceName string,
	resource *schema.Resource,
	resolveTargetInfo *subengine.ResolveResourceTargetInfo,
) (*subengine.ResolveInResourceResult, error) {
	return &subengine.ResolveInResourceResult{
		ResolvedResource: s.resolvedResource,
		ResolveOnDeploy:  []string{},
	}, nil
}

type dynamicIncludeSubstitutionResolver struct {
	resolvedIncludeFactory func(include *schema.Include) *subengine.ResolvedInclude
}

func (s *dynamicIncludeSubstitutionResolver) ResolveInInclude(
	ctx context.Context,
	includeName string,
	include *schema.Include,
	resolveTargetInfo *subengine.ResolveIncludeTargetInfo,
) (*subengine.ResolveInIncludeResult, error) {
	return &subengine.ResolveInIncludeResult{
		ResolvedInclude: s.resolvedIncludeFactory(include),
		ResolveOnDeploy: []string{},
	}, nil
}

type stubBlueprintContainerLoader struct {
	deployEventSequence []*DeployEvent
}

func (s *stubBlueprintContainerLoader) Load(
	ctx context.Context,
	blueprintSpecFile string,
	params core.BlueprintParams,
) (BlueprintContainer, error) {
	return &stubBlueprintContainer{
		deployEventSequence: s.deployEventSequence,
	}, nil
}

func (s *stubBlueprintContainerLoader) Validate(
	ctx context.Context,
	blueprintSpecFile string,
	params core.BlueprintParams,
) (*ValidationResult, error) {
	return nil, nil
}

func (s *stubBlueprintContainerLoader) LoadString(
	ctx context.Context,
	blueprintSpec string,
	inputFormat schema.SpecFormat,
	params core.BlueprintParams,
) (BlueprintContainer, error) {
	return &stubBlueprintContainer{
		deployEventSequence: s.deployEventSequence,
	}, nil
}

func (s *stubBlueprintContainerLoader) ValidateString(
	ctx context.Context,
	blueprintSpec string,
	inputFormat schema.SpecFormat,
	params core.BlueprintParams,
) (*ValidationResult, error) {
	return nil, nil
}

func (s *stubBlueprintContainerLoader) LoadFromSchema(
	ctx context.Context,
	blueprintSchema *schema.Blueprint,
	params core.BlueprintParams,
) (BlueprintContainer, error) {
	return &stubBlueprintContainer{
		deployEventSequence: s.deployEventSequence,
	}, nil
}

func (s *stubBlueprintContainerLoader) ValidateFromSchema(
	ctx context.Context,
	blueprintSchema *schema.Blueprint,
	params core.BlueprintParams,
) (*ValidationResult, error) {
	return nil, nil
}

type stubBlueprintContainer struct {
	deployEventSequence []*DeployEvent
}

func (c *stubBlueprintContainer) StageChanges(
	ctx context.Context,
	input *StageChangesInput,
	channels *ChangeStagingChannels,
	paramOverrides core.BlueprintParams,
) error {
	return nil
}

func (c *stubBlueprintContainer) Deploy(
	ctx context.Context,
	input *DeployInput,
	channels *DeployChannels,
	paramOverrides core.BlueprintParams,
) error {
	go func() {
		for _, event := range c.deployEventSequence {
			if event.ChildUpdateEvent != nil {
				channels.ChildUpdateChan <- *event.ChildUpdateEvent
			}

			if event.ResourceUpdateEvent != nil {
				channels.ResourceUpdateChan <- *event.ResourceUpdateEvent
			}

			if event.LinkUpdateEvent != nil {
				channels.LinkUpdateChan <- *event.LinkUpdateEvent
			}

			if event.DeploymentUpdateEvent != nil {
				channels.DeploymentUpdateChan <- *event.DeploymentUpdateEvent
			}

			if event.FinishEvent != nil {
				channels.FinishChan <- *event.FinishEvent
			}
		}
	}()

	return nil
}

func (c *stubBlueprintContainer) Destroy(
	ctx context.Context,
	input *DestroyInput,
	channels *DeployChannels,
	paramOverrides core.BlueprintParams,
) {
	// do nothing, this is a stub to fulfil the BlueprintContainer interface.
}

func (c *stubBlueprintContainer) SpecLinkInfo() links.SpecLinkInfo {
	return nil
}

func (c *stubBlueprintContainer) BlueprintSpec() speccore.BlueprintSpec {
	return nil
}

func (c *stubBlueprintContainer) RefChainCollector() refgraph.RefChainCollector {
	return nil
}

func (c *stubBlueprintContainer) ResourceTemplates() map[string]string {
	return map[string]string{}
}

func (c *stubBlueprintContainer) Diagnostics() []*core.Diagnostic {
	return []*core.Diagnostic{}
}

func (c *stubBlueprintContainer) CheckReconciliation(
	ctx context.Context,
	input *CheckReconciliationInput,
	paramOverrides core.BlueprintParams,
) (*ReconciliationCheckResult, error) {
	return nil, nil
}

func (c *stubBlueprintContainer) ApplyReconciliation(
	ctx context.Context,
	input *ApplyReconciliationInput,
	paramOverrides core.BlueprintParams,
) (*ApplyReconciliationResult, error) {
	return nil, nil
}
