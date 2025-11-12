package processing

import (
	"luen-search-engine/internal/text"
)

type parser struct {
	tokens []token
	idx    int
}

func newParser(token []token) *parser { return &parser{tokens: token} }

// Parse parses a query (string) into an ast
func Parse(rawQuery string, tokenizer *text.Tokenizer) (Node, error) {
	lexer := newLexer(rawQuery, tokenizer)
	tokens, err := lexer.tokens()
	if err != nil {
		return nil, err
	}

	parser := newParser(tokens)
	ast, err := parser.parse()
	if err != nil {
		return nil, err
	}

}
