package sharedtypesv1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	anypb "google.golang.org/protobuf/types/known/anypb"
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
func ToPBDiagnostics(diagnostics []*core.Diagnostic) ([]*Diagnostic, error) {
	pbDiagnostics := make([]*Diagnostic, len(diagnostics))
	for i, diag := range diagnostics {
		diagContext, err := toPBDiagnosticContext(diag.Context)
		if err != nil {
			return nil, err
		}

		pbDiagnostics[i] = &Diagnostic{
			Level:   DiagnosticLevel(diag.Level),
			Message: diag.Message,
			Range:   toPBDiagnosticRange(diag.Range),
			Context: diagContext,
		}
	}
	return pbDiagnostics, nil
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

func toPBDiagnosticContext(coreContext *errors.ErrorContext) (*DiagnosticContext, error) {
	if coreContext == nil {
		return nil, nil
	}

	var metadata *anypb.Any
	var err error
	if coreContext.Metadata != nil {
		metadata, err = pbutils.ConvertInterfaceToProtobuf(coreContext.Metadata)
	}
	if err != nil {
		return nil, err
	}

	return &DiagnosticContext{
		Category:         string(coreContext.Category),
		ReasonCode:       string(coreContext.ReasonCode),
		SuggestedActions: toPBDiagnosticSuggestedActions(coreContext.SuggestedActions),
		Metadata:         metadata,
	}, nil
}

func toPBDiagnosticSuggestedActions(coreActions []errors.SuggestedAction) []*DiagnosticSuggestedAction {
	pbActions := make([]*DiagnosticSuggestedAction, len(coreActions))
	for i, action := range coreActions {
		pbActions[i] = &DiagnosticSuggestedAction{
			Type:        string(action.Type),
			Title:       action.Title,
			Description: action.Description,
			Priority:    int32(action.Priority),
		}
	}
	return pbActions
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
func ToCoreDiagnostics(diagnostics []*Diagnostic) ([]*core.Diagnostic, error) {
	coreDiagnostics := make([]*core.Diagnostic, len(diagnostics))
	for i, diag := range diagnostics {
		coreContext, err := toCoreDiagnosticContext(diag.Context)
		if err != nil {
			return nil, err
		}

		coreDiagnostics[i] = &core.Diagnostic{
			Level:   core.DiagnosticLevel(diag.Level),
			Message: diag.Message,
			Range:   toCoreDiagnosticRange(diag.Range),
			Context: coreContext,
		}
	}
	return coreDiagnostics, nil
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

func toCoreDiagnosticContext(pbContext *DiagnosticContext) (*errors.ErrorContext, error) {
	if pbContext == nil {
		return nil, nil
	}

	var metadataAny any
	var err error
	if pbContext.Metadata != nil {
		metadataAny, err = pbutils.ConvertPBAnyToInterface(pbContext.Metadata)
	}
	if err != nil {
		return nil, err
	}

	metadata, isMetadataMap := metadataAny.(map[string]any)
	if !isMetadataMap {
		return nil, fmt.Errorf("metadata is expected to be a map of string to any")
	}

	return &errors.ErrorContext{
		Category:         errors.ErrorCategory(pbContext.Category),
		ReasonCode:       errors.ErrorReasonCode(pbContext.ReasonCode),
		SuggestedActions: toCoreDiagnosticSuggestedActions(pbContext.SuggestedActions),
		Metadata:         metadata,
	}, nil
}

func toCoreDiagnosticSuggestedActions(pbActions []*DiagnosticSuggestedAction) []errors.SuggestedAction {
	actions := make([]errors.SuggestedAction, len(pbActions))
	for i, pbAction := range pbActions {
		actions[i] = errors.SuggestedAction{
			Type:        pbAction.Type,
			Title:       pbAction.Title,
			Description: pbAction.Description,
			Priority:    int(pbAction.Priority),
		}
	}

	return actions
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
