package deployui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
)

func (m *DeployModel) outputJSON() {
	summary := m.buildDeploySummary()

	output := jsonout.DeployOutput{
		Success:          true,
		InstanceID:       m.instanceID,
		InstanceName:     m.instanceName,
		ChangesetID:      m.changesetID,
		Status:           m.finalStatus.String(),
		InstanceState:    m.postDeployInstanceState,
		PreRollbackState: m.preRollbackState,
		Summary:          summary,
	}

	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *DeployModel) buildDeploySummary() jsonout.DeploySummary {
	elements := m.buildDeployedElements()

	return jsonout.DeploySummary{
		Successful:           len(m.successfulElements),
		Failed:               len(m.elementFailures),
		Interrupted:          len(m.interruptedElements),
		SkippedRollbackItems: m.skippedRollbackItems,
		Elements:             elements,
	}
}

func (m *DeployModel) buildDeployedElements() []jsonout.DeployedElement {
	var elements []jsonout.DeployedElement

	for _, elem := range m.successfulElements {
		elements = append(elements, jsonout.DeployedElement{
			Name:   elem.ElementName,
			Path:   elem.ElementPath,
			Type:   elem.ElementType,
			Status: "success",
			Action: elem.Action,
		})
	}

	for _, elem := range m.elementFailures {
		elements = append(elements, jsonout.DeployedElement{
			Name:           elem.ElementName,
			Path:           elem.ElementPath,
			Type:           elem.ElementType,
			Status:         "failed",
			FailureReasons: elem.FailureReasons,
		})
	}

	for _, elem := range m.interruptedElements {
		elements = append(elements, jsonout.DeployedElement{
			Name:   elem.ElementName,
			Path:   elem.ElementPath,
			Type:   elem.ElementType,
			Status: "interrupted",
		})
	}

	return elements
}

func (m *DeployModel) outputJSONDrift() {
	output := jsonout.DeployDriftOutput{
		Success:        true,
		DriftDetected:  true,
		InstanceID:     m.instanceID,
		InstanceName:   m.instanceName,
		ChangesetID:    m.driftBlockedChangesetID,
		Message:        m.driftMessage,
		Reconciliation: m.driftResult,
	}

	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *DeployModel) outputJSONError(err error) {
	output := jsonout.NewErrorOutput(err)
	jsonout.WriteJSON(m.headlessWriter, output)
}
