package processing

import (
	"luen-search-engine/internal/text"
	"strings"
	"unicode"
)

type tokenType int

const (
	tEOF tokenType = iota
	tTerm
	tAnd
	tOr
	tNot
	tLParen
	tRParen
	tLPhrase
	tRPhrase
)

type token struct {
	typ   tokenType
	value string
}

type lexer struct {
	src       string
	tokenizer *text.Tokenizer
}

func newLexer(rawQuery string, tokenizer *text.Tokenizer) *lexer {
	return &lexer{src: rawQuery, tokenizer: tokenizer}
}

// Example: hello and ("world wide" or pain) -> [tTerm tAnd tLParen tLPhrase tTerm tTerm tRPhrase tRParen]
func (l *lexer) tokens() ([]token, error) {
	var out []token
	var builder strings.Builder

	// Function to be called whenever a word ends
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		value := builder.String()
		builder.Reset()

		switch value {
		case "and":
			out = append(out, token{typ: tAnd, value: value})
		case "or":
			out = append(out, token{typ: tOr, value: value})
		case "not":
			out = append(out, token{typ: tNot, value: value})
		default:
			tokenized := l.tokenizer.TokenizeTerm(value)
			if tokenized == "" {
				return
			} else {
				out = append(out, token{typ: tTerm, value: tokenized})
			}

		}
	}

	// TODO: Add phrase/parentheses validation
	// No single parentheses, no closing before opening etc.
	inPhrase := false

	// Iterating over query char by char to detect parentheses etc.
	for _, r := range l.src {
		switch r {
		case '"':
			flush()
			if inPhrase {
				out = append(out, token{typ: tRPhrase, value: "\""})
				inPhrase = false
				continue
			} else {
				out = append(out, token{typ: tLPhrase, value: "\""})
				inPhrase = true
				continue
			}
		case '(':
			flush()
			out = append(out, token{typ: tLParen, value: "("})
			continue
		case ')':
			flush()
			out = append(out, token{typ: tRParen, value: ")"})
			continue
		default:
			if !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
				// this way we're ignoring anything that is not a-z0-9
				flush()
				continue
			}
			builder.WriteRune(r)
		}
	}

	flush()

	return out, nil
}
