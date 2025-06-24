package sharedtypesv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// NoResponsePBError returns an error response for when a plugin does not provide
// a valid response to a request.
func NoResponsePBError() *ErrorResponse {
	return &ErrorResponse{
		Message: "Empty response from the plugin",
	}
}

// ToPBDiagnostics converts a slice of core Diagnostics to a slice of protobuf Diagnostics
// for a gRPC plugin response.
func ToPBDiagnostics(diagnostics []*core.Diagnostic) []*Diagnostic {
	pbDiagnostics := make([]*Diagnostic, len(diagnostics))
	for i, diag := range diagnostics {
		pbDiagnostics[i] = &Diagnostic{
			Level:   DiagnosticLevel(diag.Level),
			Message: diag.Message,
			Range:   toPBDiagnosticRange(diag.Range),
		}
	}
	return pbDiagnostics
}

func toPBDiagnosticRange(coreRange *core.DiagnosticRange) *DiagnosticRange {
	columnAccuracy := ColumnAccuracy_COLUMN_ACCURACY_NONE
	if coreRange.ColumnAccuracy != nil {
		columnAccuracy = ColumnAccuracy(*coreRange.ColumnAccuracy)
	}

	return &DiagnosticRange{
		Start:          toPBSourceMetaLocation(coreRange.Start),
		End:            toPBSourceMetaLocation(coreRange.End),
		ColumnAccuracy: columnAccuracy,
	}
}

func toPBSourceMetaLocation(location *source.Meta) *SourceMeta {
	if location == nil {
		return nil
	}

	endPosition := toPBPositionPtr(location.EndPosition)
	startPosition := toPBPosition(&location.Position)
	return &SourceMeta{
		StartPosition: &startPosition,
		EndPosition:   endPosition,
	}
}

func toPBPosition(position *source.Position) SourcePosition {
	if position == nil {
		return SourcePosition{}
	}

	return SourcePosition{
		Line:   int64(position.Line),
		Column: int64(position.Column),
	}
}

func toPBPositionPtr(position *source.Position) *SourcePosition {
	if position == nil {
		return nil
	}

	return &SourcePosition{
		Line:   int64(position.Line),
		Column: int64(position.Column),
	}
}

// ToCoreDiagnostics converts a slice of protobuf Diagnostics to a slice of core Diagnostics
// for a gRPC plugin response.
func ToCoreDiagnostics(diagnostics []*Diagnostic) []*core.Diagnostic {
	coreDiagnostics := make([]*core.Diagnostic, len(diagnostics))
	for i, diag := range diagnostics {
		coreDiagnostics[i] = &core.Diagnostic{
			Level:   core.DiagnosticLevel(diag.Level),
			Message: diag.Message,
			Range:   toCoreDiagnosticRange(diag.Range),
		}
	}
	return coreDiagnostics
}

func toCoreDiagnosticRange(pbRange *DiagnosticRange) *core.DiagnosticRange {
	columnAccuracyPtr := (*substitutions.ColumnAccuracy)(nil)
	if pbRange.ColumnAccuracy != ColumnAccuracy_COLUMN_ACCURACY_NONE {
		columnAccuracy := substitutions.ColumnAccuracy(pbRange.ColumnAccuracy)
		columnAccuracyPtr = &columnAccuracy
	}

	return &core.DiagnosticRange{
		Start:          toCoreSourceMetaLocation(pbRange.Start),
		End:            toCoreSourceMetaLocation(pbRange.End),
		ColumnAccuracy: columnAccuracyPtr,
	}
}

func toCoreSourceMetaLocation(location *SourceMeta) *source.Meta {
	if location == nil {
		return nil
	}

	return &source.Meta{
		Position:    toCorePosition(location.StartPosition),
		EndPosition: toCorePositionPtr(location.EndPosition),
	}
}

func toCorePositionPtr(position *SourcePosition) *source.Position {
	if position == nil {
		return nil
	}

	return &source.Position{
		Line:   int(position.Line),
		Column: int(position.Column),
	}
}

func toCorePosition(position *SourcePosition) source.Position {
	if position == nil {
		return source.Position{}
	}

	return source.Position{
		Line:   int(position.Line),
		Column: int(position.Column),
	}
}

// FromPBResourceTypes converts a slice of protobuf ResourceType to a slice of string
// for a gRPC plugin response.
func FromPBResourceTypes(resourceTypes []*ResourceType) []string {
	types := make([]string, len(resourceTypes))
	for i, resourceType := range resourceTypes {
		types[i] = resourceType.Type
	}
	return types
}
