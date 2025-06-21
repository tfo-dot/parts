package parts

import (
	"errors"
	"fmt"
)

type Parser struct {
	Scanner *Scanner

	LastToken Token
	Rules     []ParserRule
	PostFix   []PostFixRule

	Literals   []Literal
	Meta       map[string]string
	ModulePath string
}

func (p *Parser) ParseAll() ([]Bytecode, error) {
	bytecode := make([]Bytecode, 0)

	for !(p.LastToken.Type == TokenInvalid && string(p.LastToken.Value) == "EOF") {
		temp, err := p.parse()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while parsing whole code"), err)
		}

		bytecode = append(bytecode, temp...)
	}

	return bytecode, nil
}

func (p *Parser) parse() ([]Bytecode, error) {
	for _, rule := range p.Rules {
		if rule.Rule(p) {
			if rule.AdvanceToken {
				if _, err := p.advance(); err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while advancing after rule check"), err)
				}
			}

			body, err := rule.Parse(p)

			if err != nil {
				return []Bytecode{}, errors.Join(fmt.Errorf("got error while parsing rule - %s", rule.Id), err)
			}

			if len(body) > 0 {
				for {
					applied := false

					for _, pRule := range p.PostFix {
						if pRule.Rule(p) {
							if pRule.AdvanceToken {
								if _, err := p.advance(); err != nil {
									return nil, errors.Join(errors.New("error advancing after postfix rule check"), err)
								}
							}

							body, err = pRule.Parse(p, body)

							if err != nil {
								return nil, errors.Join(fmt.Errorf("error parsing postfix rule - (%s:%s)", rule.Id, pRule.Id), err)
							}

							applied = true
							break
						}
					}

					if !applied {
						break
					}
				}

				return body, nil
			} else {
				return []Bytecode{}, nil
			}
		}
	}

	currentToken, err := p.peek()

	if err != nil {
		return nil, errors.Join(errors.New("error peeeking after not matching any tokens"), err)
	}

	return []Bytecode{}, fmt.Errorf("Unknown rule? (%d - %s)", currentToken.Type, string(currentToken.Value))
}

func (p *Parser) parseWithRule(id string) ([]Bytecode, error) {
	for _, rule := range p.Rules {
		if rule.Id == id {
			if rule.Rule(p) {
				if rule.AdvanceToken {
					if _, err := p.advance(); err != nil {
						return []Bytecode{}, errors.Join(errors.New("got error while advancing after rule check"), err)
					}
				}

				res, err := rule.Parse(p)

				if err != nil {
					return []Bytecode{}, errors.Join(fmt.Errorf("got error while parsing rule - %s", rule.Id), err)
				}

				return res, nil
			} else {
				return []Bytecode{}, fmt.Errorf("Rule %s, didn't pass the initial check", id)
			}
		}
	}

	return []Bytecode{}, fmt.Errorf("No rule with id %s", id)
}

func (p *Parser) match(tokenType TokenType, value string) bool {
	checkRes := p.check(tokenType, value)

	if checkRes {
		_, err := p.advance()

		if err != nil {
			panic(err)
		}
	}

	return checkRes
}

func (p *Parser) check(tokenType TokenType, value string) bool {
	currentToken, err := p.peek()

	if err != nil {
		panic(err)
	}

	if currentToken.Type != tokenType {
		return false
	}

	if string(currentToken.Value) == value {
		return true
	}

	return false
}

func (p *Parser) matchKeyword(value string) bool {
	return p.match(TokenKeyword, value)
}

func (p *Parser) matchOperator(value string) bool {
	return p.match(TokenOperator, value)
}

func (p *Parser) peek() (Token, error) {
	if p.LastToken.Type == TokenInvalid {
		token, err := p.Scanner.Next()

		if err != nil {
			return Token{}, err
		}

		p.LastToken = token
	}

	return p.LastToken, nil
}

func (p *Parser) advance() (Token, error) {
	lastToken := p.LastToken

	token, err := p.Scanner.Next()

	if err != nil {
		return Token{}, err
	}

	p.LastToken = token

	return lastToken, nil
}

type Bytecode byte

const (
	B_DECLARE Bytecode = iota
	B_SET
	B_LITERAL
	B_RETURN
	B_RAISE
	B_NEW_SCOPE
	B_END_SCOPE
	B_DOT
	B_CALL
	B_RESOLVE
	B_COND_JUMP
	B_BIN_OP
	B_LOOP
	B_CONTINUE
	B_BREAK
)

type BinOp Bytecode

const (
	B_OP_ADD Bytecode = iota
	B_OP_MIN
	B_OP_MUL
	B_OP_DIV
	B_OP_EQ
	B_OP_GT
	B_OP_LT
)

type ImportType Bytecode

const (
	B_IMPORT_STX Bytecode = iota
)

type ReferenceDeclaration struct {
	Reference string
	Dynamic   bool
}

type FunctionDeclaration struct {
	Params []string
	Body   []Bytecode
}

type ObjDefinition struct {
	Entries [][]Bytecode
}

type ListDefinition struct {
	Entries [][]Bytecode
}

func (p *Parser) AppendLiteral(literal Literal) ([]Bytecode, error) {
	existingIdx := -1

	if literal.LiteralType != ObjLiteral && literal.LiteralType != ListLiteral && literal.LiteralType != FunLiteral {
		for idx, existingLiteral := range p.Literals {
			if existingLiteral.LiteralType != literal.LiteralType {
				continue
			}

			if existingLiteral.Value == literal.Value {
				existingIdx = idx
			}
		}
	}

	if existingIdx == -1 {
		p.Literals = append(p.Literals, literal)

		encoded, err := encodeLen(len(p.Literals) - 1)

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding literal offset"), err)
		}

		return append([]Bytecode{B_LITERAL}, encoded...), nil
	} else {
		encoded, err := encodeLen(existingIdx)

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding literal offset"), err)
		}

		return append([]Bytecode{B_LITERAL}, encoded...), nil
	}
}

func encodeLen(num int) ([]Bytecode, error) {
	if num < 0 {
		return []Bytecode{}, errors.New("length can't be negative")
	}

	if num <= 125 {
		return []Bytecode{Bytecode(num)}, nil
	}

	if num <= 65535 {
		return []Bytecode{126, Bytecode(num >> 8), Bytecode(num)}, nil
	}

	return []Bytecode{
		127,
		Bytecode(num >> 56),
		Bytecode(num >> 48),
		Bytecode(num >> 40),
		Bytecode(num >> 32),
		Bytecode(num >> 24),
		Bytecode(num >> 16),
		Bytecode(num >> 8),
		Bytecode(num),
	}, nil
}

func mustEncodeLen(num int) []Bytecode {
	val, err := encodeLen(num)

	if err != nil {
		panic(err)
	}

	return val
}
