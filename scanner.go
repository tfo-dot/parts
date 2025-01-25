package main

import (
	"errors"
	"fmt"
	"unicode"
)

type Scanner struct {
	Rules  []Rule
	Source []rune
	Index  int
	Line   int
}

func (s *Scanner) Next() (Token, error) {
	for unicode.IsSpace(s.Peek()) {
		s.Index++
	}

	if s.Peek() == '"' {
		return s.ParseQuote()
	}

	for _, rule := range s.Rules {
		if rule.BaseRule(s.Peek()) {
			rValue, rError := s.ParseRule(rule)

			if rError != nil {
				return rValue, rError
			}

			if rValue.Type == TokenKeyword {
				rValue = s.SplitByKeyword(rValue)
			}

			if rValue.Type == TokenOperator {
				return s.SplitOperators(rValue)
			}

			return rValue, rError
		}
	}

	return Token{}, fmt.Errorf("unknown token %s [%d, pos> %d:%d]", string(s.Peek()), s.Peek(), s.Line, s.Index)
}

func (s Scanner) SplitByKeyword(token Token) Token {
	found := false

	for _, kw := range Keywords {
		if kw == string(token.Value) {
			found = true
			break
		}
	}

	if !found {
		token.Type = TokenIdentifier
	}

	return token
}

func (s *Scanner) SplitOperators(token Token) (Token, error) {
	tokenValue := string(token.Value)
	name, ok := ValidOperators[tokenValue]

	if ok {
		return Token{Type: TokenOperator, Value: []rune(name)}, nil
	}

	for {
		if len(tokenValue) == 0 {
			return Token{}, errors.New("not valid operator")
		}

		tokenValue = tokenValue[0 : len(tokenValue)-1]
		s.Index--
		name, ok := ValidOperators[tokenValue]

		if ok {
			return Token{Type: TokenOperator, Value: []rune(name)}, nil
		}
	}
}

func (s *Scanner) Peek() rune {
	if s.Index < len(s.Source) {
		return s.Source[s.Index]
	}

	return 0
}

func (s *Scanner) ParseQuote() (Token, error) {
	s.Index++
	start := s.Index

	for {
		s.Index++

		if s.Index >= len(s.Source) || (s.Peek() == '"' && s.Source[s.Index-1] != '\\') {
			break
		}
	}

	err := s.CheckBounds("Unterminated string")

	if err != nil {
		return Token{}, err
	}

	if s.Peek() != '"' {
		return Token{}, fmt.Errorf("unterminated string")
	}

	val := s.Source[start:s.Index]

	s.Index++

	return Token{Type: TokenString, Value: val}, nil
}

func (s *Scanner) ParseRule(rule Rule) (Token, error) {
	start := s.Index

	for {
		s.Index++

		if s.Index >= len(s.Source) || !rule.BaseRule(s.Peek()) {
			break
		}
	}

	val := s.Source[start:s.Index]

	return Token{Type: rule.Result, Value: val}, nil
}

func (s *Scanner) CheckBounds(msg string) error {
	if s.Index >= len(s.Source) {
		return fmt.Errorf("[%d] %s: unexpected end of file", s.Line, msg)
	}
	return nil
}
