package parts

import (
	"fmt"
	"slices"
)

type Scanner struct {
	Rules  []Rule
	Source []rune
	Index  int
	Line   int

	Buffored []Token
}

func (s *Scanner) Next() (Token, error) {
	if len(s.Buffored) != 0 {
		rv := s.Buffored[0]

		s.Buffored = s.Buffored[1:]

		return rv, nil
	}

	if s.Peek() == 0 {
		return Token{Type: TokenInvalid, Value: []rune("EOF")}, nil
	}

	for _, rule := range s.Rules {

		if rule.BaseRule == nil {
			if slices.Contains(rule.ValidChars, s.Peek()) {
				rValue, rError := s.ParseRule(rule)

				if rule.RType == SKIP_RULE {
					return s.Next()
				}

				if rError != nil {
					return Token{}, rError
				}

				if len(rValue) == 0 {
					return s.Next()
				}

				if len(rValue) == 1 {
					return rValue[0], nil
				}

				s.Buffored = rValue[1:]
				return rValue[0], nil
			} else {
				continue
			}
		}

		if rule.BaseRule(s.Peek()) {
			rValue, rError := s.ParseRule(rule)

			if rule.RType == SKIP_RULE {
				return s.Next()
			}

			if rError != nil {
				return Token{}, rError
			}

			if len(rValue) == 0 {
				return s.Next()
			}

			if len(rValue) == 1 {
				return rValue[0], nil
			}

			s.Buffored = rValue[1:]
			return rValue[0], nil
		}
	}

	if s.Peek() == 0 {
		return Token{Type: TokenInvalid, Value: []rune("EOF")}, nil
	}

	return Token{}, fmt.Errorf("unknown token %s [%d, pos> %d:%d]", string(s.Peek()), s.Peek(), s.Line, s.Index)
}

func (s *Scanner) Peek() rune {
	if s.Index < len(s.Source) {
		return s.Source[s.Index]
	}

	return 0
}

func (s *Scanner) ParseRule(rule Rule) ([]Token, error) {
	start := s.Index

	for {
		s.Index++

		outOfBounds := s.Index >= len(s.Source)
		noBaseAndValid := rule.BaseRule == nil && slices.Contains(rule.ValidChars, s.Peek())
		matchesBase :=  rule.BaseRule != nil && rule.BaseRule(s.Peek())
		matchesWhole := rule.Rule == nil || rule.Rule(s.Source[start:s.Index])

		if outOfBounds || !(noBaseAndValid || matchesBase) || !matchesWhole {
			break
		}
	}

	if rule.Process != nil {
		res, err := rule.Process(rule.Mappings, s.Source[start:s.Index])

		if err != nil {
			return []Token{}, err
		}

		return res, nil
	}

	return []Token{{Type: rule.Result, Value: s.Source[start:s.Index]}}, nil
}

func (s *Scanner) CheckBounds(msg string) error {
	if s.Index >= len(s.Source) {
		return fmt.Errorf("[%d] %s: unexpected end of file", s.Line, msg)
	}
	return nil
}
