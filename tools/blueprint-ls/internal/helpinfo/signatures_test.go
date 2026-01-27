package helpinfo

import (
	"testing"

	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type SignaturesSuite struct {
	suite.Suite
}

func (s *SignaturesSuite) TestCustomRenderSignatures_EmptyList() {
	result := CustomRenderSignatures([]*lsp.SignatureInformation{})
	s.Empty(result)
}

func (s *SignaturesSuite) TestCustomRenderSignatures_SingleSignature_StringDoc() {
	signatures := []*lsp.SignatureInformation{
		{
			Label:         "len(value: string) -> integer",
			Documentation: "Returns the length of a string",
		},
	}
	result := CustomRenderSignatures(signatures)
	s.Contains(result, "```len(value: string) -> integer```")
	s.Contains(result, "Returns the length of a string")
}

func (s *SignaturesSuite) TestCustomRenderSignatures_SingleSignature_MarkupDoc() {
	signatures := []*lsp.SignatureInformation{
		{
			Label: "concat(a: string, b: string) -> string",
			Documentation: lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: "**Concatenates** two strings",
			},
		},
	}
	result := CustomRenderSignatures(signatures)
	s.Contains(result, "```concat(a: string, b: string) -> string```")
	s.Contains(result, "**Concatenates** two strings")
}

func (s *SignaturesSuite) TestCustomRenderSignatures_SingleSignature_PlainTextMarkup() {
	signatures := []*lsp.SignatureInformation{
		{
			Label: "trim(value: string) -> string",
			Documentation: lsp.MarkupContent{
				Kind:  lsp.MarkupKindPlainText,
				Value: "Plain text description",
			},
		},
	}
	result := CustomRenderSignatures(signatures)
	s.Contains(result, "```trim(value: string) -> string```")
	// PlainText markup is not included (only markdown is)
	s.NotContains(result, "Plain text description")
}

func (s *SignaturesSuite) TestCustomRenderSignatures_MultipleSignatures() {
	signatures := []*lsp.SignatureInformation{
		{
			Label:         "len(value: string) -> integer",
			Documentation: "Returns the length of a string",
		},
		{
			Label:         "len(value: array) -> integer",
			Documentation: "Returns the length of an array",
		},
	}
	result := CustomRenderSignatures(signatures)
	s.Contains(result, "```len(value: string) -> integer```")
	s.Contains(result, "```len(value: array) -> integer```")
	s.Contains(result, "---") // Separator between signatures
}

func (s *SignaturesSuite) TestCustomRenderSignatures_NilDocumentation() {
	signatures := []*lsp.SignatureInformation{
		{
			Label:         "noDoc() -> void",
			Documentation: nil,
		},
	}
	result := CustomRenderSignatures(signatures)
	s.Contains(result, "```noDoc() -> void```")
	// Should not panic and should format correctly
}

func (s *SignaturesSuite) TestCustomRenderSignatures_MixedDocTypes() {
	signatures := []*lsp.SignatureInformation{
		{
			Label:         "func1() -> string",
			Documentation: "String documentation",
		},
		{
			Label: "func2() -> integer",
			Documentation: lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: "Markup documentation",
			},
		},
		{
			Label:         "func3() -> boolean",
			Documentation: nil,
		},
	}
	result := CustomRenderSignatures(signatures)
	s.Contains(result, "```func1() -> string```")
	s.Contains(result, "String documentation")
	s.Contains(result, "```func2() -> integer```")
	s.Contains(result, "Markup documentation")
	s.Contains(result, "```func3() -> boolean```")
}

func TestSignaturesSuite(t *testing.T) {
	suite.Run(t, new(SignaturesSuite))
}
