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
	FunLiteral
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

	if p.matchKeyword("fun") {
		functionName := -1
		{
			currentToken, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing identifier name"))
			}

			if currentToken.Type == TokenIdentifier {
				functionNameToken, err := p.advance()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing identifier name"))
				}

				p.Literals = append(p.Literals, Literal{
					LiteralType: RefLiteral,
					Value:       functionNameToken.Value,
				})

				functionName = len(p.Literals) - 1
			}
		}

		if !p.matchOperator("LEFT_PAREN") {
			token, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading function params"), err)
			}

			return []Bytecode{}, fmt.Errorf("expected '(' after function declaration got '%s'", string(token.Value))
		}

		token, err := p.peek()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while reading function params"), err)
		}

		declaration := FunctionDeclaration{
			Params: [][]rune{},
			Body:   []Bytecode{},
		}

		if token.Type != TokenOperator && string(token.Value) != "RIGHT_PAREN" {
			for {
				identifierToken, err := p.peek()

				if identifierToken.Type != TokenIdentifier {
					return []Bytecode{}, errors.Join(errors.New("encountered unexpected token in function params"), err)
				}

				declaration.Params = append(declaration.Params, identifierToken.Value)
			}
		}

		if !p.matchOperator("RIGHT_PAREN") {
			token, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading function params"), err)
			}

			return []Bytecode{}, fmt.Errorf("expected ')' after function params got '%s'", string(token.Value))
		}

		if p.matchOperator("EQUAL") {
			expr, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), err)
			}

			declaration.Body = append(append(declaration.Body, B_RETURN), expr...)
		} else {
			body, err := p.parseBlock(true)

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), err)
			}

			declaration.Body = body
		}

		p.Literals = append(p.Literals, Literal{FunLiteral, declaration})
		idx := len(p.Literals) - 1

		if functionName == -1 {
			return []Bytecode{Bytecode(idx)}, nil
		} else {
			return []Bytecode{B_DECLARE, DECLARE_FUN, Bytecode(functionName), Bytecode(idx)}, nil
		}
	}

	if p.matchOperator("LEFT_BRACE") {
		rVal, rErr := p.parseBlock(false)

		if rErr != nil {
			return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), rErr)
		}

		return rVal, nil
	}

	rVal, rErr := p.parsePrimary()

	if rErr != nil {
		return []Bytecode{}, rErr
	}

	p.matchOperator("SEMICOLON")

	return rVal, nil
}

func (p *Parser) parseBlock(checkBrace bool) ([]Bytecode, error) {
	if checkBrace {
		if !p.matchOperator("LEFT_BRACE") {
			return []Bytecode{}, errors.New("opening brace not found")
		}
	}

	body := make([]Bytecode, 0)

	for {
		currentToken, err := p.peek()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while parsing block body"), err)
		}

		if currentToken.Type == TokenOperator && string(currentToken.Value) == "RIGHT_BRACE" {
			break
		}

		statement, err := p.parseInScope(TopLevel)

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while parsing block body"), err)
		}

		body = append(body, statement...)
	}

	if !p.matchOperator("RIGHT_BRACE") {
		return []Bytecode{}, errors.New("opening brace not found")
	}

	return append(append([]Bytecode{B_NEW_SCOPE}, body...), B_END_SCOPE), nil
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
	B_RETURN
	B_NEW_SCOPE
	B_END_SCOPE
)

const (
	DECLARE_LET Bytecode = iota
	DECLARE_FUN
)

type FunctionDeclaration struct {
	Params [][]rune
	Body   []Bytecode
}
