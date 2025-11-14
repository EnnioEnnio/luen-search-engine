package processing

import (
	"errors"
	"fmt"

	"luen-search-engine/internal/text"
)

// parser implements a recursive-descent query parser over lexer tokens.
type parser struct {
	tokens []token
	idx    int
}

func newParser(token []token) *parser { return &parser{tokens: token} }

// parseToken parses the full token stream into an AST node.
func (p *parser) parseToken() (Node, error) {
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("query contains no searchable terms")
	}

	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if !p.isAtEnd() {
		return nil, fmt.Errorf("unexpected token %q", p.peek().value)
	}

	return node, nil
}

// parseOr parses left-associative OR expressions.
func (p *parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	children := []Node{left}

	for p.match(tOr) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		children = append(children, right)
	}

	if len(children) == 1 {
		return left, nil
	}

	return &OrNode{Children: children}, nil
}

// parseAnd parses explicit AND and implicit adjacency expressions.
func (p *parser) parseAnd() (Node, error) {
	nodes := make([]Node, 0, 2)

	for {
		node, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)

		if p.match(tAnd) {
			if p.isAtEnd() {
				return nil, fmt.Errorf("expected expression after AND")
			}
			continue
		}

		next := p.peek().typ
		if next == tEOF || next == tOr || next == tRParen {
			break
		}

		if p.isAtEnd() {
			break
		}
	}

	if len(nodes) == 1 {
		return nodes[0], nil
	}

	return &AndNode{Children: nodes}, nil
}

// parseUnary handles NOT expressions and delegates to parsePrimary.
func (p *parser) parseUnary() (Node, error) {
	if p.match(tNot) {
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotNode{Child: child}, nil
	}

	return p.parsePrimary()
}

// parsePrimary parses literals, phrases, and parenthesized sub-expressions.
func (p *parser) parsePrimary() (Node, error) {
	if p.isAtEnd() {
		return nil, fmt.Errorf("unexpected end of query")
	}

	switch tok := p.peek(); tok.typ {
	case tTerm:
		p.advance()
		return &TermNode{Token: Token(tok.value)}, nil
	case tLParen:
		p.advance()
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(tRParen, "expected closing ')'"); err != nil {
			return nil, err
		}
		return node, nil
	case tLPhrase:
		return p.parsePhrase()
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.value)
	}
}

// parsePhrase gathers consecutive term tokens between quotes.
func (p *parser) parsePhrase() (Node, error) {
	// consume opening quote
	p.advance()

	tokens := make([]Token, 0, 2)
	for !p.isAtEnd() && p.peek().typ != tRPhrase {
		tok := p.peek()
		if tok.typ != tTerm {
			return nil, fmt.Errorf("phrases can only contain terms, got %q", tok.value)
		}
		tokens = append(tokens, Token(tok.value))
		p.advance()
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty phrase is not allowed")
	}

	if err := p.expect(tRPhrase, "unterminated phrase; missing closing quote"); err != nil {
		return nil, err
	}

	return &PhraseNode{Tokens: tokens}, nil
}

// match consumes the current token if it matches any of the provided types.
func (p *parser) match(types ...tokenType) bool {
	for _, typ := range types {
		if p.peek().typ == typ {
			p.advance()
			return true
		}
	}
	return false
}

// expect enforces that the current token matches typ, returning a friendly error otherwise.
func (p *parser) expect(typ tokenType, errMsg string) error {
	if p.peek().typ != typ {
		return errors.New(errMsg)
	}
	p.advance()
	return nil
}

// advance moves the parser cursor forward and returns the consumed token.
func (p *parser) advance() token {
	if !p.isAtEnd() {
		p.idx++
	}
	return p.tokens[p.idx-1]
}

// peek returns the current token without consuming it.
func (p *parser) peek() token {
	if p.isAtEnd() {
		return token{typ: tEOF}
	}
	return p.tokens[p.idx]
}

// isAtEnd reports whether all tokens have been consumed.
func (p *parser) isAtEnd() bool { return p.idx >= len(p.tokens) }

// Parse converts rawQuery into an AST node using the supplied tokenizer.
func Parse(rawQuery string, tokenizer *text.Tokenizer) (Node, error) {
	// TODO: Move Parse() to main() and make lexer & parser query-agnostic
	lexer := newLexer(rawQuery, tokenizer)
	tokens, err := lexer.tokens()
	if err != nil {
		return nil, err
	}

	parser := newParser(tokens)
	ast, err := parser.parseToken()
	if err != nil {
		return nil, err
	}

	return ast, nil
}
