package serialisation

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schemapb"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// ToSourceMetaPB converts source metadata to its protobuf representation.
//
// Source metadata is absent for blueprints constructed in memory rather than
// parsed from a document, and for elements whose source format does not expose
// a position, so a nil input is expected in those cases.
func ToSourceMetaPB(meta *source.Meta) *schemapb.SourceMeta {
	if meta == nil {
		return nil
	}

	metaPB := &schemapb.SourceMeta{
		Position: toPositionPB(&meta.Position),
	}

	if meta.EndPosition != nil {
		metaPB.EndPosition = toPositionPB(meta.EndPosition)
	}

	if meta.ColumnAccuracy != nil {
		accuracy := schemapb.ColumnAccuracy(*meta.ColumnAccuracy)
		metaPB.ColumnAccuracy = &accuracy
	}

	return metaPB
}

// FromSourceMetaPB converts source metadata from its protobuf representation.
func FromSourceMetaPB(metaPB *schemapb.SourceMeta) *source.Meta {
	if metaPB == nil {
		return nil
	}

	meta := &source.Meta{}
	if position := fromPositionPB(metaPB.Position); position != nil {
		meta.Position = *position
	}

	meta.EndPosition = fromPositionPB(metaPB.EndPosition)

	if metaPB.ColumnAccuracy != nil {
		accuracy := source.ColumnAccuracy(*metaPB.ColumnAccuracy)
		meta.ColumnAccuracy = &accuracy
	}

	return meta
}

// ToFieldsSourceMetaPB converts a map of field positions to its protobuf
// representation, returning nil for an empty map so that blueprints without
// position information do not carry an empty map across the wire.
func ToFieldsSourceMetaPB(fields map[string]*source.Meta) map[string]*schemapb.SourceMeta {
	if len(fields) == 0 {
		return nil
	}

	fieldsPB := make(map[string]*schemapb.SourceMeta, len(fields))
	for name, meta := range fields {
		if metaPB := ToSourceMetaPB(meta); metaPB != nil {
			fieldsPB[name] = metaPB
		}
	}

	return fieldsPB
}

// FromFieldsSourceMetaPB converts a map of field positions from its protobuf
// representation.
func FromFieldsSourceMetaPB(fieldsPB map[string]*schemapb.SourceMeta) map[string]*source.Meta {
	if len(fieldsPB) == 0 {
		return nil
	}

	fields := make(map[string]*source.Meta, len(fieldsPB))
	for name, metaPB := range fieldsPB {
		if meta := FromSourceMetaPB(metaPB); meta != nil {
			fields[name] = meta
		}
	}

	return fields
}

func toPositionPB(position *source.Position) *schemapb.Position {
	if position == nil {
		return nil
	}

	return &schemapb.Position{
		Line:   int64(position.Line),
		Column: int64(position.Column),
	}
}

func fromPositionPB(positionPB *schemapb.Position) *source.Position {
	if positionPB == nil {
		return nil
	}

	return &source.Position{
		Line:   int(positionPB.Line),
		Column: int(positionPB.Column),
	}
}
