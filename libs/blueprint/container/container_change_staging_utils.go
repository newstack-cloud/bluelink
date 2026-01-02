package container

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

func collectExportChanges(
	changes *changes.IntermediaryBlueprintChanges,
	exports map[string]*schema.Export,
	resolvedExports map[string]*subengine.ResolveResult,
	currentExportsState map[string]*state.ExportState,
) {
	for exportName, resolvedExport := range resolvedExports {
		exportValue := extractExportValue(exportName, currentExportsState)
		sourceFieldPath := getExportSourceFieldPath(exports, exportName)
		collectExportFieldChanges(
			changes,
			exportName,
			sourceFieldPath,
			resolvedExport.Resolved,
			exportValue,
			resolvedExport.ResolveOnDeploy,
		)
	}
}

// getExportSourceFieldPath returns the source field path for an export.
// This is the path within the blueprint that the export references
// (e.g., "resources.myResource.spec.url"), not the target export path.
func getExportSourceFieldPath(exports map[string]*schema.Export, exportName string) string {
	export, hasExport := exports[exportName]
	if !hasExport || export == nil || export.Field == nil || export.Field.StringValue == nil {
		// Fallback to target path if source field path is not available
		return substitutions.RenderFieldPath("exports", exportName)
	}
	return *export.Field.StringValue
}

func extractExportValue(exportName string, exports map[string]*state.ExportState) *core.MappingNode {
	exportState, hasExport := exports[exportName]
	if hasExport {
		return exportState.Value
	}

	return nil
}

func collectExportFieldChanges(
	changes *changes.IntermediaryBlueprintChanges,
	exportName string,
	sourceFieldPath string,
	newExportValue *core.MappingNode,
	currentStateValue *core.MappingNode,
	fieldsToResolveOnDeploy []string,
) {
	if len(fieldsToResolveOnDeploy) > 0 {
		// If any nested values of the export field value can not be known until deploy time,
		// mark the export field as a field to resolve on deploy.
		changes.ResolveOnDeploy = append(
			changes.ResolveOnDeploy,
			substitutions.RenderFieldPath("exports", exportName),
		)
	}

	if core.IsNilMappingNode(newExportValue) &&
		!core.IsNilMappingNode(currentStateValue) &&
		// Do not mark as removed if some parts of the export field value
		// can not be known until deploy time.
		len(fieldsToResolveOnDeploy) == 0 {
		changes.RemovedExports = append(
			changes.RemovedExports,
			substitutions.RenderFieldPath("exports", exportName),
		)
		return
	}

	if !core.IsNilMappingNode(newExportValue) &&
		core.IsNilMappingNode(currentStateValue) {
		changes.NewExports[exportName] = &provider.FieldChange{
			FieldPath: sourceFieldPath,
			PrevValue: nil,
			NewValue:  newExportValue,
		}
		return
	}

	if !core.MappingNodeEqual(newExportValue, currentStateValue) {
		changes.ExportChanges[exportName] = &provider.FieldChange{
			FieldPath: sourceFieldPath,
			PrevValue: currentStateValue,
			NewValue:  newExportValue,
		}
	} else {
		changes.UnchangedExports = append(
			changes.UnchangedExports,
			substitutions.RenderFieldPath("exports", exportName),
		)
	}
}
