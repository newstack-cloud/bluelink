package stageui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
)

func (m *StageModel) outputJSON() {
	summary := m.buildChangeSummary()

	output := jsonout.StageOutput{
		Success:      true,
		ChangesetID:  m.changesetID,
		InstanceID:   m.instanceID,
		InstanceName: m.instanceName,
		Changes:      m.completeChanges,
		Summary:      summary,
	}

	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *StageModel) buildChangeSummary() jsonout.ChangeSummary {
	resourceSummary := jsonout.ResourceSummary{}
	childSummary := jsonout.ChildSummary{}
	linkSummary := jsonout.LinkSummary{}

	for _, item := range m.items {
		switch item.Type {
		case ItemTypeResource:
			resourceSummary.Total += 1
			countResourceAction(&resourceSummary, item.Action)
		case ItemTypeChild:
			childSummary.Total += 1
			countChildAction(&childSummary, item.Action)
		case ItemTypeLink:
			linkSummary.Total += 1
			countLinkAction(&linkSummary, item.Action)
		}
	}

	exportNew, exportModified, exportRemoved, exportUnchanged := countExportChanges(m.completeChanges)

	return jsonout.ChangeSummary{
		Resources: resourceSummary,
		Children:  childSummary,
		Links:     linkSummary,
		Exports: jsonout.ExportSummary{
			Total:     exportNew + exportModified + exportRemoved + exportUnchanged,
			New:       exportNew,
			Modified:  exportModified,
			Removed:   exportRemoved,
			Unchanged: exportUnchanged,
		},
	}
}

func countResourceAction(summary *jsonout.ResourceSummary, action ActionType) {
	switch action {
	case ActionCreate:
		summary.Create += 1
	case ActionUpdate:
		summary.Update += 1
	case ActionDelete:
		summary.Delete += 1
	case ActionRecreate:
		summary.Recreate += 1
	}
}

func countChildAction(summary *jsonout.ChildSummary, action ActionType) {
	switch action {
	case ActionCreate:
		summary.Create += 1
	case ActionUpdate:
		summary.Update += 1
	case ActionDelete:
		summary.Delete += 1
	}
}

func countLinkAction(summary *jsonout.LinkSummary, action ActionType) {
	switch action {
	case ActionCreate:
		summary.Create += 1
	case ActionUpdate:
		summary.Update += 1
	case ActionDelete:
		summary.Delete += 1
	}
}

func (m *StageModel) outputJSONDrift() {
	output := jsonout.StageDriftOutput{
		Success:        true,
		DriftDetected:  true,
		InstanceID:     m.instanceID,
		InstanceName:   m.instanceName,
		Message:        m.driftMessage,
		Reconciliation: m.driftResult,
	}

	jsonout.WriteJSON(m.headlessWriter, output)
}

func (m *StageModel) outputJSONError(err error) {
	output := jsonout.NewErrorOutput(err)
	jsonout.WriteJSON(m.headlessWriter, output)
}
