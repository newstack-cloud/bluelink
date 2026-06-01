package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// ParseBlueprintLangToUnified lexes blueprint-language source and builds a
// UnifiedNode tree in the canonical blueprint shape (the same field-name
// hierarchy the YAML/JWCC converters produce e.g. /resources/<name>/spec/<field>),
// so the existing completion-context, document-symbol and duplicate-key features
// work unchanged.
//
// It is built from the token stream (not the validated schema) so the tree stays
// usable for in-progress edits: the lexer is resilient even when the document
// does not yet parse, which is exactly when completions are needed.
func ParseBlueprintLangToUnified(content string) (*UnifiedNode, error) {
	tokens, _ := lang.Tokenize(content)
	b := &bpBuilder{tokens: tokens}
	root := b.buildRoot()
	fillMissingRanges(root)
	return root, nil
}

// Walks the token stream with a single forward cursor, building the
// canonical UnifiedNode tree. The stream is always EOF-terminated, so the cursor
// never runs past the end of the slice.
type bpBuilder struct {
	tokens []lang.Token
	i      int
}

func (b *bpBuilder) cur() lang.Token {
	return b.tokens[b.i]
}

func (b *bpBuilder) kind() lang.TokenType {
	return b.tokens[b.i].Type
}
func (b *bpBuilder) atEnd() bool {
	return b.tokens[b.i].Type == lang.TokenEOF
}

func (b *bpBuilder) advance() lang.Token {
	tkn := b.tokens[b.i]
	if !b.atEnd() {
		b.i++
	}
	return tkn
}

func (b *bpBuilder) skipSeparators() {
	for !b.atEnd() {
		switch b.kind() {
		case lang.TokenNewline, lang.TokenComma, lang.TokenComment:
			b.advance()
		default:
			return
		}
	}
}

func (b *bpBuilder) skipComments() {
	for b.kind() == lang.TokenComment {
		b.advance()
	}
}

func (b *bpBuilder) buildRoot() *UnifiedNode {
	root := &UnifiedNode{
		Kind:   NodeKindMapping,
		Index:  -1,
		TSKind: "bp_root",
	}
	sections := map[string]*UnifiedNode{}

	for !b.atEnd() {
		b.skipSeparators()
		if b.atEnd() {
			break
		}

		switch b.kind() {
		case lang.TokenKeywordVersion:
			b.appendChild(root, b.parseScalarDirective("version"))
		case lang.TokenKeywordTransform:
			b.appendChild(root, b.parseScalarDirective("transform"))
		case lang.TokenKeywordMetadata:
			b.appendChild(root, b.parseNamedFieldsBlock("metadata", root))
		case lang.TokenKeywordResource:
			b.appendToSection(root, sections, "resources", b.parseResourceDecl)
		case lang.TokenKeywordVariable:
			b.appendToSection(root, sections, "variables", b.parseTypedFieldsDecl)
		case lang.TokenKeywordValue:
			b.appendToSection(root, sections, "values", b.parseTypedFieldsDecl)
		case lang.TokenKeywordData:
			b.appendToSection(root, sections, "datasources", b.parseDataDecl)
		case lang.TokenKeywordInclude:
			b.appendToSection(root, sections, "include", b.parseIncludeDecl)
		case lang.TokenKeywordExport:
			b.appendToSection(root, sections, "exports", b.parseTypedFieldsDecl)
		default:
			// Forgiving: skip any stray token so an in-progress edit elsewhere
			// does not derail the whole tree.
			b.advance()
		}
	}

	return root
}

// Parses a single declaration via decl and appends it under the
// synthetic section mapping (created lazily on first use, in source order).
func (b *bpBuilder) appendToSection(
	root *UnifiedNode,
	sections map[string]*UnifiedNode,
	sectionName string,
	decl func(parent *UnifiedNode) *UnifiedNode,
) {
	section, ok := sections[sectionName]
	if !ok {
		section = &UnifiedNode{
			Kind:      NodeKindMapping,
			Index:     -1,
			FieldName: sectionName,
			Parent:    root,
			TSKind:    "bp_section",
		}
		sections[sectionName] = section
		root.Children = append(root.Children, section)
	}

	node := decl(section)
	if node != nil {
		section.Children = append(section.Children, node)
	}
}

func (b *bpBuilder) appendChild(parent, child *UnifiedNode) {
	if child != nil {
		parent.Children = append(parent.Children, child)
	}
}

func (b *bpBuilder) parseScalarDirective(name string) *UnifiedNode {
	keyword := b.advance()
	b.skipComments()
	node := b.parseValueExpr(nil)
	if node == nil {
		node = scalarNode("", "!!str", tokRange(keyword, keyword), nil)
	}

	b.nameNode(node, name, tokRange(keyword, keyword))
	return node
}

func (b *bpBuilder) parseResourceDecl(parent *UnifiedNode) *UnifiedNode {
	node, nameRange := b.parseDeclHeader(parent)
	b.parseResourceBody(node)
	finalizeNodeRange(node, nameRange)
	return node
}

func (b *bpBuilder) parseTypedFieldsDecl(parent *UnifiedNode) *UnifiedNode {
	node, nameRange := b.parseDeclHeader(parent)
	b.parseFieldsBlock(node)
	finalizeNodeRange(node, nameRange)
	return node
}

// Consumes `<keyword> <name>: <type>` and returns the named
// mapping node (with a synthetic `type` child) ready for its body to be parsed.
func (b *bpBuilder) parseDeclHeader(parent *UnifiedNode) (*UnifiedNode, source.Range) {
	b.advance() // declaration keyword
	name, nameRange := b.parseName()

	node := &UnifiedNode{
		Kind:      NodeKindMapping,
		Index:     -1,
		FieldName: name,
		KeyRange:  rangePtr(nameRange),
		Parent:    parent,
		Range:     nameRange,
		TSKind:    "bp_decl",
	}

	if b.kind() == lang.TokenColon {
		b.advance()
		typeVal, typeRange := b.parseTypeRef()
		typeNode := scalarNode(typeVal, "!!str", typeRange, node)
		b.nameNode(typeNode, "type", typeRange)
		node.Children = append(node.Children, typeNode)
	}

	return node, nameRange
}

// Parses `include <name> "path" { body }`. The path is a
// positional string surfaced as a synthetic `path` scalar.
func (b *bpBuilder) parseIncludeDecl(parent *UnifiedNode) *UnifiedNode {
	b.advance() // include
	name, nameRange := b.parseName()

	node := &UnifiedNode{
		Kind:      NodeKindMapping,
		Index:     -1,
		FieldName: name,
		KeyRange:  rangePtr(nameRange),
		Parent:    parent,
		Range:     nameRange,
		TSKind:    "bp_decl",
	}

	if b.kind() == lang.TokenStringStart {
		pathVal, pathRange := b.consumeString()
		pathNode := scalarNode(pathVal, "!!str", pathRange, node)
		b.nameNode(pathNode, "path", pathRange)
		node.Children = append(node.Children, pathNode)
	}

	b.parseFieldsBlock(node)
	finalizeNodeRange(node, nameRange)
	return node
}

func (b *bpBuilder) parseDataDecl(parent *UnifiedNode) *UnifiedNode {
	node, nameRange := b.parseDeclHeader(parent)
	b.parseDataBody(node)
	finalizeNodeRange(node, nameRange)
	return node
}
