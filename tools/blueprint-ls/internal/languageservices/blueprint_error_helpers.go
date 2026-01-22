package languageservices

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// Helper interface to handle extracting the location of different kinds of
// blueprint errors in one way.
type blueprintErrorLocation interface {
	Line() *int
	Column() *int
	EndLine() *int
	EndColumn() *int
	UseColumnAccuracy() bool
	ColumnAccuracy() *substitutions.ColumnAccuracy
}

//////////////////////////////////////////////////////////
// Error location wrapper for errors.LoadError
//////////////////////////////////////////////////////////

type blueprintErrorLocationLoadErr struct {
	err *errors.LoadError
}

func (l blueprintErrorLocationLoadErr) Line() *int {
	if l.err == nil {
		return nil
	}
	return l.err.Line
}

func (l blueprintErrorLocationLoadErr) Column() *int {
	if l.err == nil {
		return nil
	}
	return l.err.Column
}

func (l blueprintErrorLocationLoadErr) EndLine() *int {
	if l.err == nil {
		return nil
	}
	return l.err.EndLine
}

func (l blueprintErrorLocationLoadErr) EndColumn() *int {
	if l.err == nil {
		return nil
	}
	return l.err.EndColumn
}

func (l blueprintErrorLocationLoadErr) UseColumnAccuracy() bool {
	if l.err == nil {
		return false
	}
	return l.err.ColumnAccuracy != nil
}

func (l blueprintErrorLocationLoadErr) ColumnAccuracy() *substitutions.ColumnAccuracy {
	if l.err == nil || l.err.ColumnAccuracy == nil {
		return nil
	}
	// source.ColumnAccuracy and substitutions.ColumnAccuracy are the same type
	ca := substitutions.ColumnAccuracy(*l.err.ColumnAccuracy)
	return &ca
}

//////////////////////////////////////////////////////////
// Error location wrapper for schema.Error
//////////////////////////////////////////////////////////

type blueprintErrorLocationSchemaErr struct {
	err *schema.Error
}

func (l blueprintErrorLocationSchemaErr) Line() *int {
	return l.err.SourceLine
}

func (l blueprintErrorLocationSchemaErr) Column() *int {
	return l.err.SourceColumn
}

func (l blueprintErrorLocationSchemaErr) EndLine() *int {
	return nil
}

func (l blueprintErrorLocationSchemaErr) EndColumn() *int {
	return nil
}

func (l blueprintErrorLocationSchemaErr) UseColumnAccuracy() bool {
	return false
}

func (l blueprintErrorLocationSchemaErr) ColumnAccuracy() *substitutions.ColumnAccuracy {
	return nil
}

////////////////////////////////////////////////////////////////
// Error location wrapper for load substitutions.ParseError
////////////////////////////////////////////////////////////////

type blueprintErrorLocationParseErr struct {
	err *substitutions.ParseError
}

func (l blueprintErrorLocationParseErr) Line() *int {
	return &l.err.Line
}

func (l blueprintErrorLocationParseErr) Column() *int {
	return &l.err.Column
}

func (l blueprintErrorLocationParseErr) EndLine() *int {
	return nil
}

func (l blueprintErrorLocationParseErr) EndColumn() *int {
	return nil
}

func (l blueprintErrorLocationParseErr) UseColumnAccuracy() bool {
	return true
}

func (l blueprintErrorLocationParseErr) ColumnAccuracy() *substitutions.ColumnAccuracy {
	return &l.err.ColumnAccuracy
}

////////////////////////////////////////////////////////////////
// Error location wrapper for load substitutions.LexError
////////////////////////////////////////////////////////////////

type blueprintErrorLocationLexErr struct {
	err *substitutions.LexError
}

func (l blueprintErrorLocationLexErr) Line() *int {
	return &l.err.Line
}

func (l blueprintErrorLocationLexErr) Column() *int {
	return &l.err.Column
}

func (l blueprintErrorLocationLexErr) EndLine() *int {
	return nil
}

func (l blueprintErrorLocationLexErr) EndColumn() *int {
	return nil
}

func (l blueprintErrorLocationLexErr) UseColumnAccuracy() bool {
	return true
}

func (l blueprintErrorLocationLexErr) ColumnAccuracy() *substitutions.ColumnAccuracy {
	return &l.err.ColumnAccuracy
}

////////////////////////////////////////////////////////////////
// Error location wrapper for load core.Error
////////////////////////////////////////////////////////////////

type blueprintErrorLocationCoreErr struct {
	err *core.Error
}

func (l blueprintErrorLocationCoreErr) Line() *int {
	return l.err.SourceLine
}

func (l blueprintErrorLocationCoreErr) Column() *int {
	return l.err.SourceColumn
}

func (l blueprintErrorLocationCoreErr) EndLine() *int {
	return nil
}

func (l blueprintErrorLocationCoreErr) EndColumn() *int {
	return nil
}

func (l blueprintErrorLocationCoreErr) UseColumnAccuracy() bool {
	return false
}

func (l blueprintErrorLocationCoreErr) ColumnAccuracy() *substitutions.ColumnAccuracy {
	return nil
}
