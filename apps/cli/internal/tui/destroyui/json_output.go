package destroyui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
)

func (m *DestroyModel) outputJSON() {
	summary := m.buildDestroySummary()

	output := jsonout.DestroyOutput{
		Success:         true,
		InstanceID:      m.instanceID,
		InstanceName:    m.instanceName,
		ChangesetID:     m.changesetID,
		Status:          m.finalStatus.String(),
		InstanceState:   m.postDestroyInstanceState,
		PreDestroyState: m.preDestroyInstanceState,
		Summary:         summary,
	}

	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *DestroyModel) buildDestroySummary() jsonout.DestroySummary {
	elements := m.buildDestroyedElements()

	return jsonout.DestroySummary{
		Destroyed:   len(m.destroyedElements),
		Failed:      len(m.elementFailures),
		Interrupted: len(m.interruptedElements),
		Elements:    elements,
	}
}

func (m *DestroyModel) buildDestroyedElements() []jsonout.DestroyedElement {
	var elements []jsonout.DestroyedElement

	for _, elem := range m.destroyedElements {
		elements = append(elements, jsonout.DestroyedElement{
			Name:   elem.ElementName,
			Path:   elem.ElementPath,
			Type:   elem.ElementType,
			Status: "destroyed",
		})
	}

	for _, elem := range m.elementFailures {
		elements = append(elements, jsonout.DestroyedElement{
			Name:           elem.ElementName,
			Path:           elem.ElementPath,
			Type:           elem.ElementType,
			Status:         "failed",
			FailureReasons: elem.FailureReasons,
		})
	}

	for _, elem := range m.interruptedElements {
		elements = append(elements, jsonout.DestroyedElement{
			Name:   elem.ElementName,
			Path:   elem.ElementPath,
			Type:   elem.ElementType,
			Status: "interrupted",
		})
	}

	return elements
}

func (m *DestroyModel) outputJSONDrift() {
	output := jsonout.DestroyDriftOutput{
		Success:        true,
		DriftDetected:  true,
		InstanceID:     m.instanceID,
		InstanceName:   m.instanceName,
		Message:        m.driftMessage,
		Reconciliation: m.driftResult,
	}

	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *DestroyModel) outputJSONError(err error) {
	output := jsonout.NewErrorOutput(err)
	jsonout.WriteJSON(m.headlessWriter, output)
}
