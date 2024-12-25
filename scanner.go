package parts

import "fmt"

type Scanner struct {
	Rules  []Rule
	Source []rune
	Index  int
	Line   int
}

func (s *Scanner) Next() (Token, error) {
	if s.Peek() == '"' {
		return s.ParseQuote()
	}

	for _, rule := range s.Rules {
		if rule.BaseRule(s.Peek()) {
			return s.ParseRule(rule)
		}
	}

	return Token{}, fmt.Errorf("unknown token")
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
