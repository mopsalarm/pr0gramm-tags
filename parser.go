package main

import (
	"fmt"
	"io"
	"github.com/mopsalarm/go-pr0gramm-tags/store"
)

type IteratorFactory func(string) store.ItemIterator

type buf struct {
	tok Token
	lit string
}

type Parser struct {
	makeIter IteratorFactory
	scanner  *Scanner
	next     buf
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader, makeIter IteratorFactory) *Parser {
	parser := &Parser{
		makeIter: makeIter,
		scanner:  NewScanner(r),
		next:     buf{ILLEGAL, ""},
	}
	parser.buffer()
	return parser
}

func (p *Parser) buffer() {
	if p.next.tok != EOF {
		p.next.tok, p.next.lit = p.scanner.Scan()
	}
}

func (p *Parser) peek() Token {
	return p.next.tok
}

func (p *Parser) consume(expect Token) string {
	tok, lit := p.next.tok, p.next.lit
	if tok != expect {
		panic(fmt.Errorf("Got unexpected token '%s', expected '%s'", tok, expect))
	}

	p.buffer()
	return lit
}

func (p *Parser) Parse() store.ItemIterator {
	result := p.parseOrExpr()
	p.consume(EOF)

	return result
}

func (p *Parser) parseOrExpr() store.ItemIterator {
	var result store.ItemIterator = p.parseAndExpr()

	for p.peek() == OP_OR {
		p.consume(OP_OR)

		second := p.parseAndExpr()
		result = store.NewOrIterator(result, second)
	}

	return result
}

func (p *Parser) parseAndExpr() store.ItemIterator {
	result := p.parseWithoutExpr()

	loop:
	for {
		tok := p.peek()
		switch tok {
		case OP_AND:
			p.consume(OP_AND)
			fallthrough

		case PAR_OPEN:
			fallthrough

		case WORD:
			second := p.parseWithoutExpr()
			result = store.NewAndIterator(result, second)

		default:
			break loop
		}
	}

	return result
}

func (p *Parser) parseWithoutExpr() store.ItemIterator {
	result := p.parseBaseExpr()

	if p.peek() == OP_WITHOUT {
		p.consume(OP_WITHOUT)

		second := p.parseBaseExpr()
		result = store.NewDiffIterator(result, second)
	}

	return result
}

func (p *Parser) parseBaseExpr() store.ItemIterator {
	var result store.ItemIterator

	tok := p.peek()
	switch tok {
	case PAR_OPEN:
		p.consume(PAR_OPEN)
		result = p.parseOrExpr()
		p.consume(PAR_CLOSE)

	case WORD:
		result = p.makeIter(p.consume(WORD))

	case OP_WITHOUT:
		p.consume(OP_WITHOUT)
		result = store.NewDiffIterator(p.makeIter("__all"), p.parseBaseExpr())

	default:
		panic(fmt.Errorf("Found unexpected token '%q'", tok))
	}

	return result
}
