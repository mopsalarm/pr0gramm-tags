package main

import (
	"unicode"
	"bufio"
	"io"
	"bytes"
)

type Token string

const (
	ILLEGAL Token = "ILLEGAL"
	EOF = "EOF"

	OP_AND = "AND"
	OP_OR = "OR"
	OP_WITHOUT = "WITHOUT"

	PAR_OPEN = "("
	PAR_CLOSE = ")"

	WORD = "WORD"
)

const eof = rune(0)

func isWhitespace(ch rune) bool {
	return unicode.IsSpace(ch)
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsNumber(ch)
}

func isContinueLetter(ch rune) bool {
	return isLetter(ch) || ch == ':'
}

type Scanner struct {
	r *bufio.Reader
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
}

func (s *Scanner) Scan() (Token, string) {
	ch := s.read()

	// If we see whitespace then consume all contiguous whitespace.
	// If we see a letter then consume as an ident or reserved word.
	for isWhitespace(ch) {
		ch = s.read()
	}

	if ch == '(' {
		return PAR_OPEN, "("
	}

	if ch == ')' {
		return PAR_CLOSE, ")"
	}

	if ch == '&' {
		return OP_AND, "&"
	}

	if ch == '-' {
		return OP_WITHOUT, "-"
	}

	if ch == '|' {
		return OP_OR, "|"
	}

	if isLetter(ch) {
		s.unread()
		return s.scanIdentifier()
	}

	if ch == eof {
		return EOF, "EOF"
	}

	return ILLEGAL, string(ch)
}

func (s *Scanner) scanIdentifier() (Token, string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isContinueLetter(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	// If the string matches a keyword then return that keyword.
	switch buf.String() {
	case "and":
		return OP_AND, "and"
	case "or":
		return OP_OR, "or"
	case "minus":
		return OP_WITHOUT, "minus"
	case "without":
		return OP_WITHOUT, "without"
	case "not":
		return OP_WITHOUT, "not"
	}

	// Otherwise return as a regular word.
	return WORD, buf.String()
}
