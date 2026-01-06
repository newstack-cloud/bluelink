package inspectui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
)

// outputJSON outputs the instance state as JSON.
// Per the requirements, the JSON output is simply the raw InstanceState.
func (m *InspectModel) outputJSON() {
	if m.instanceState == nil {
		jsonout.WriteJSON(m.headlessWriter, nil)
		return
	}
	jsonout.WriteJSON(m.headlessWriter, m.instanceState)
}

// outputJSONError outputs an error as JSON.
func (m *InspectModel) outputJSONError(err error) {
	output := jsonout.NewErrorOutput(err)
	jsonout.WriteJSON(m.headlessWriter, output)
}
