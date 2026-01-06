package listui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func (m *MainModel) printHeadlessInstances(instances []state.InstanceSummary, totalCount int) {
	if m.printer == nil {
		return
	}

	w := m.printer.Writer()
	w.PrintlnEmpty()

	if m.searchTerm != "" {
		w.Printf("Blueprint Instances (search: %q)\n", m.searchTerm)
	} else {
		w.Println("Blueprint Instances")
	}
	w.DoubleSeparator(72)

	if len(instances) == 0 {
		w.Println("No instances found.")
		return
	}

	for _, inst := range instances {
		statusText := shared.InstanceStatusHeadlessText(inst.Status)
		timestamp := formatTimestamp(inst.LastDeployedTimestamp)
		w.Printf("  %s\n", inst.InstanceName)
		w.Printf("    ID: %s\n", inst.InstanceID)
		w.Printf("    Status: %s\n", statusText)
		w.Printf("    Last Deployed: %s\n", timestamp)
	}

	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Printf("Total: %d instance(s)\n", totalCount)
	w.PrintlnEmpty()
}

func (m *MainModel) printHeadlessError(err error) {
	if m.printer == nil {
		return
	}
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("ERR List instances failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *MainModel) outputJSON(instances []state.InstanceSummary, totalCount int) {
	if m.headlessWriter == nil {
		return
	}

	items := make([]jsonout.ListInstanceItem, len(instances))
	for i, inst := range instances {
		items[i] = jsonout.ListInstanceItem{
			InstanceID:            inst.InstanceID,
			InstanceName:          inst.InstanceName,
			Status:                inst.Status.String(),
			LastDeployedTimestamp: inst.LastDeployedTimestamp,
		}
	}

	output := jsonout.ListInstancesOutput{
		Success:    true,
		Instances:  items,
		TotalCount: totalCount,
		Search:     m.searchTerm,
	}
	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *MainModel) outputJSONError(err error) {
	if m.headlessWriter == nil {
		return
	}
	jsonout.WriteJSON(m.headlessWriter, jsonout.NewErrorOutput(err))
}

// dispatchHeadlessOutput handles headless output for both success and error cases.
func (m *MainModel) dispatchHeadlessOutput(instances []state.InstanceSummary, totalCount int, err error) {
	if err != nil {
		if m.jsonMode {
			m.outputJSONError(err)
		} else {
			m.printHeadlessError(err)
		}
		return
	}
	if m.jsonMode {
		m.outputJSON(instances, totalCount)
	} else {
		m.printHeadlessInstances(instances, totalCount)
	}
}
