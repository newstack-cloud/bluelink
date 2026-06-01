package lang_test

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/stretchr/testify/suite"
)

type TokenizeSuite struct {
	suite.Suite
}

func (s *TokenizeSuite) Test_tokenizes_a_resource_declaration() {
	src := "version \"2025-05-12\"\n\nresource myTable: aws/dynamodb/table {\n    spec {\n        tableName = \"orders\"\n    }\n}\n"

	tokens, err := lang.Tokenize(src)
	s.Require().NoError(err)
	s.Require().NotEmpty(tokens)

	// The stream is always EOF-terminated.
	s.Assert().Equal(lang.TokenEOF, tokens[len(tokens)-1].Type)

	// Comments and newlines are preserved (unlike the parser), and structural
	// punctuation is surfaced so a CST can be reconstructed.
	s.Assert().True(hasTokenType(tokens, lang.TokenNewline), "expected newline tokens to be preserved")
	s.Assert().True(hasTokenType(tokens, lang.TokenKeywordResource))
	s.Assert().True(hasTokenType(tokens, lang.TokenKeywordSpec))
	s.Assert().True(hasTokenType(tokens, lang.TokenLeftBrace))
	s.Assert().True(hasTokenType(tokens, lang.TokenRightBrace))
	s.Assert().True(hasTokenType(tokens, lang.TokenAssign))
}

func (s *TokenizeSuite) Test_tokens_carry_source_positions() {
	// "resource" begins on line 1, column 1.
	tokens, err := lang.Tokenize("resource myTable: aws/s3/bucket {}\n")
	s.Require().NoError(err)
	s.Require().NotEmpty(tokens)

	first := tokens[0]
	s.Assert().Equal(lang.TokenKeywordResource, first.Type)
	s.Assert().Equal(1, first.Start.Line)
	s.Assert().Equal(1, first.Start.Column)
	s.Assert().Equal(9, first.End.Column)
}

func (s *TokenizeSuite) Test_returns_tokens_and_error_on_lexical_failure() {
	// An unterminated single-line string is a lexical error, but Tokenize must
	// still return a (partial) token stream so a CST can be built mid-edit.
	tokens, err := lang.Tokenize("variable region: string {\n    default = \"oops\n}\n")
	s.Require().Error(err)
	s.Require().NotEmpty(tokens)
	s.Assert().Equal(lang.TokenEOF, tokens[len(tokens)-1].Type)
}

func hasTokenType(tokens []lang.Token, tokenType lang.TokenType) bool {
	for _, tkn := range tokens {
		if tkn.Type == tokenType {
			return true
		}
	}
	return false
}

func TestTokenizeSuite(t *testing.T) {
	suite.Run(t, new(TokenizeSuite))
}
