package lang

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// A transient expression tree built while parsing a bare expression
// or a ${..} body. It never reaches the schema: it is lowered to the canonical
// types before leaving the parser.
type expr interface {
	// A marker method to distinguish expr from other node types.
	isExpr()
	meta() *source.Meta
}

type scalarExpr struct {
	value *core.ScalarValue
}

func (s *scalarExpr) isExpr() {}

func (s *scalarExpr) meta() *source.Meta {
	return s.value.SourceMeta
}

type arrayExpr struct {
	elems []expr
	m     *source.Meta
}

func (a *arrayExpr) isExpr() {}

func (a *arrayExpr) meta() *source.Meta {
	return a.m
}

type objectExpr struct {
	entries []objectField
	m       *source.Meta
}

func (o *objectExpr) isExpr() {}

func (o *objectExpr) meta() *source.Meta {
	return o.m
}

type objectField struct {
	key   string
	value expr
	meta  *source.Meta
}

type refExpr struct {
	sub *substitutions.Substitution
}

func (r *refExpr) isExpr() {}

func (r *refExpr) meta() *source.Meta {
	return r.sub.SourceMeta
}

type callExpr struct {
	name string
	args []callArg
	path []*substitutions.SubstitutionPathItem
	m    *source.Meta
}

func (c *callExpr) isExpr() {}

func (c *callExpr) meta() *source.Meta {
	return c.m
}

type callArg struct {
	// Will be an empty string for a positional arg.
	name  string
	value expr
	meta  *source.Meta
}

type opExpr struct {
	fn   substitutions.SubstitutionFunctionName
	args []expr
	m    *source.Meta
}

func (o *opExpr) isExpr() {}

func (o *opExpr) meta() *source.Meta {
	return o.m
}

type interpolationExpr struct {
	parts []interpolationPart
	m     *source.Meta
}

func (i *interpolationExpr) isExpr() {}

func (i *interpolationExpr) meta() *source.Meta {
	return i.m
}

type interpolationPart interface {
	isInterpolationPart()
	meta() *source.Meta
}

type stringPart struct {
	value string
	m     *source.Meta
}

func (s *stringPart) isInterpolationPart() {}

func (s *stringPart) meta() *source.Meta {
	return s.m
}

type substitutionPart struct {
	value expr
	m     *source.Meta
}

func (s *substitutionPart) isInterpolationPart() {}

func (s *substitutionPart) meta() *source.Meta {
	return s.m
}

type noneExpr struct {
	m *source.Meta
}

func (n *noneExpr) isExpr() {}

func (n *noneExpr) meta() *source.Meta {
	return n.m
}
