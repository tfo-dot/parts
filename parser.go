package parts

import (
	"errors"
	"fmt"
	"strconv"
)

type Scope int

const (
	TopLevel Scope = iota
	Expression
)

type Parser struct {
	Scanner *Scanner
	Scope   Scope

	LastToken Token

	Literals []Literal
}

type LiteralType int

const (
	IntLiteral LiteralType = iota
	DoubleLiteral
	BoolLiteral
	StringLiteral
	RefLiteral
)

var InitialLiterals = []Literal{
	{BoolLiteral, false},
	{BoolLiteral, true},
}

type Literal struct {
	LiteralType LiteralType
	Value       any
}

func (p *Parser) next() ([]Bytecode, error) {
	return p.parse()
}

func (p *Parser) parseInScope(wantedScope Scope) ([]Bytecode, error) {
	lastScope := p.Scope

	p.Scope = wantedScope

	rValue, rError := p.parse()

	p.Scope = lastScope

	return rValue, rError
}

func (p *Parser) parse() ([]Bytecode, error) {
	switch p.Scope {
	case TopLevel:
		return p.parseTopLevel()
	case Expression:
		return p.parseExpression()
	}

	return []Bytecode{}, errors.New("invalid scope")
}

func (p *Parser) parseTopLevel() ([]Bytecode, error) {
	if p.matchKeyword("let") {
		identifierToken, err := p.advance()

		if err != nil {
			return []Bytecode{}, err
		}

		if identifierToken.Type != TokenIdentifier {
			return []Bytecode{}, errors.New("got invalid token instead of identifier")
		}

		p.Literals = append(p.Literals, Literal{
			LiteralType: RefLiteral,
			Value:       identifierToken.Value,
		})

		literalIdx := len(p.Literals) - 1

		initialValue := []Bytecode{}

		if p.matchOperator("EQUALS") {
			value, err := p.parseInScope(Expression)

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving expression)"), err)
			}

			initialValue = value
		}

		return append([]Bytecode{B_DECLARE, DECLARE_LET, Bytecode(literalIdx)}, initialValue...), nil
	}

	return p.parseExpression()
}

func (p *Parser) parseExpression() ([]Bytecode, error) {
	//TODO all the cases

	rVal, rErr := p.parsePrimary()

	if rErr != nil {
		return []Bytecode{}, rErr
	}

	p.matchOperator("SEMICOLON")

	return rVal, nil
}

func (p *Parser) parsePrimary() ([]Bytecode, error) {
	if p.matchKeyword("false") {
		return []Bytecode{0}, nil
	}

	if p.matchKeyword("true") {
		return []Bytecode{1}, nil
	}

	currentToken, err := p.peek()

	if currentToken.Type == TokenNumber {
		p.advance()

		val, err := strconv.Atoi(string(currentToken.Value))

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("encountered wrong format for number"), err)
		}

		p.Literals = append(p.Literals, Literal{
			LiteralType: IntLiteral,
			Value:       val,
		})

		literalIdx := len(p.Literals) - 1

		return []Bytecode{Bytecode(literalIdx)}, nil
	}

	if currentToken.Type == TokenString {
		p.advance()

		p.Literals = append(p.Literals, Literal{
			LiteralType: IntLiteral,
			Value:       currentToken.Value,
		})

		literalIdx := len(p.Literals) - 1

		return []Bytecode{Bytecode(literalIdx)}, nil
	}

	if p.matchOperator("LEFT_PAREN") {
		val, err := p.parseExpression()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("encountered error while parsing parenthesis"), err)
		}

		if !p.matchOperator("RIGHT_PAREN") {
			return []Bytecode{}, errors.New("no closing parenthesis")
		}

		return val, nil
	}

	if err != nil {
		return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
	}

	return []Bytecode{}, fmt.Errorf("expected expression, got %s", string(currentToken.Value))
}

func (p *Parser) match(tokenType TokenType, value string) bool {
	currentToken, err := p.peek()

	if err != nil {
		return false
	}

	if currentToken.Type != tokenType {
		return false
	}

	if string(currentToken.Value) == value {
		p.advance()

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
)

const (
	DECLARE_LET Bytecode = iota
	DECLARE_CONST
)
