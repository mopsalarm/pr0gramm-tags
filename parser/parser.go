package parser

import (
	"fmt"
	"io"
)

type buf struct {
	tok Token
	lit string
}

type Parser struct {
	scanner *Scanner
	next    buf
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader) *Parser {
	parser := &Parser{
		scanner: NewScanner(r),
		next:    buf{tok: ILLEGAL, lit: ""},
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

func (p *Parser) Parse() (node *Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error while parsing: %s", r)
		}
	}()

	if p.peek() == EOF {
		p.consume(EOF)
		node = NewQueryNode("__all")
	} else {
		node = p.parseTopMostExpr()
		p.consume(EOF)
	}

	return
}

func (p *Parser) parseTopMostExpr() *Node {
	return p.parseWithoutExpr()
}

func (p *Parser) parseWithoutExpr() *Node {
	result := p.parseOrExpr()

	for p.peek() == OP_WITHOUT {
		p.consume(OP_WITHOUT)

		second := p.parseOrExpr()
		result = NewOpNode(WITHOUT, result, second)
	}

	return result
}

func (p *Parser) parseOrExpr() *Node {
	var result *Node = p.parseAndExpr()

	for p.peek() == OP_OR {
		p.consume(OP_OR)

		second := p.parseAndExpr()
		result = NewOpNode(OR, result, second)
	}

	return result
}

func (p *Parser) parseAndExpr() *Node {
	result := p.parseBaseExpr()

loop:
	for {
		tok := p.peek()
		switch tok {
		case OP_AND:
			p.consume(OP_AND)
			fallthrough

		case WORD:
			second := p.parseBaseExpr()
			result = NewOpNode(AND, result, second)

		default:
			break loop
		}
	}

	return result
}

func (p *Parser) parseBaseExpr() (result *Node) {
	tok := p.peek()
	switch tok {
	case PAR_OPEN:
		p.consume(PAR_OPEN)
		result = p.parseTopMostExpr()
		p.consume(PAR_CLOSE)

	case WORD:
		result = NewQueryNode(p.consume(WORD))

	case OP_WITHOUT:
		p.consume(OP_WITHOUT)
		result = NewOpNode(NOT, p.parseBaseExpr())

	case OP_NOT:
		p.consume(OP_NOT)
		result = NewOpNode(NOT, p.parseBaseExpr())

	default:
		panic(fmt.Errorf("Found unexpected token '%q'", tok))
	}

	return
}
