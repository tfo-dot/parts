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
	Meta     map[string]string
}

func (p *Parser) parseAll() ([]Bytecode, error) {
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
	if p.matchKeyword("LET") {
		identifierToken, err := p.advance()

		if err != nil {
			return []Bytecode{}, err
		}

		if identifierToken.Type != TokenIdentifier {
			return []Bytecode{}, errors.New("got invalid token instead of identifier")
		}

		literalCode, err := p.AppendLiteral(Literal{
			LiteralType: RefLiteral,
			Value:       ReferenceDeclaration{Reference: string(identifierToken.Value), Dynamic: false},
		})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while writing literal offset"), err)
		}

		initialValue := []Bytecode{}

		if p.matchOperator("EQUALS") {
			value, err := p.parseInScope(Expression)

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving expression)"), err)
			}

			initialValue = value
		}

		return append(append([]Bytecode{B_DECLARE}, literalCode...), initialValue...), nil
	}

	if p.matchOperator("META") {
		metaKeyToken, err := p.advance()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while reading meta value"), err)
		}

		if metaKeyToken.Type != TokenString {
			return []Bytecode{}, errors.New("only strings are valid meta keys")
		}

		if !p.matchOperator("COLON") {
			return []Bytecode{}, errors.New("missing ':' after meta key")
		}

		metaValueToken, err := p.peek()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while reading meta value"), err)
		}

		if metaValueToken.Type != TokenString {
			return []Bytecode{}, errors.New("only strings are valid meta values")
		}

		p.Meta[string(metaKeyToken.Value)] = string(metaValueToken.Value)

		return []Bytecode{}, nil
	}

	return p.parseExpression()
}

func (p *Parser) parseExpression() ([]Bytecode, error) {
	//TODO all the cases

	if p.matchKeyword("FUN") {
		functionName := []Bytecode{}
		{
			currentToken, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing identifier name"), err)
			}

			if currentToken.Type == TokenIdentifier {
				functionNameToken, err := p.advance()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing identifier name"), err)
				}

				functionName, err = p.AppendLiteral(Literal{
					LiteralType: RefLiteral,
					Value:       ReferenceDeclaration{Reference: string(functionNameToken.Value), Dynamic: false},
				})

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding identifier literal"), err)
				}
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
			Params: []string{},
			Body:   []Bytecode{},
		}

		if token.Type != TokenOperator && string(token.Value) != "RIGHT_PAREN" {
			for cond := true; cond; cond = p.matchOperator("COMMA") {
				identifierToken, err := p.advance()

				if identifierToken.Type != TokenIdentifier {
					return []Bytecode{}, errors.Join(errors.New("encountered unexpected token in function params"), err)
				}

				declaration.Params = append(declaration.Params, string(identifierToken.Value))
			}
		}

		if !p.matchOperator("RIGHT_PAREN") {
			token, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading function params"), err)
			}

			return []Bytecode{}, fmt.Errorf("expected ')' after function params got '%s'", string(token.Value))
		}

		if p.matchOperator("EQUALS") {
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

		idx, err := p.AppendLiteral(Literal{FunLiteral, declaration})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("encountered err in length encoding"), err)
		}

		if len(functionName) == 0 {
			return idx, nil
		} else {
			return append(append([]Bytecode{B_DECLARE}, functionName...), idx...), nil
		}
	}

	if p.matchKeyword("IF") {
		condition, err := p.parseInScope(Expression)

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving condition)"), err)
		}

		thenBranch, err := p.parseInScope(Expression)

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving then branch)"), err)
		}

		thenLength, err := encodeLen(len(thenBranch))

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding then branch)"), err)
		}

		elseBranch := []Bytecode{}
		elseLength := []Bytecode{0}

		if p.matchKeyword("ELSE") {
			elseBranch, err = p.parseInScope(Expression)

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving else branch)"), err)
			}

			elseLength, err = encodeLen(len(elseBranch))

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding else branch)"), err)
			}
		}

		return append(append(append(append(append([]Bytecode{B_COND_JUMP}, condition...), thenLength...), thenBranch...), elseLength...), elseBranch...), nil
	}

	if p.matchOperator("LEFT_BRACE") {
		rVal, rErr := p.parseBlock(false)

		if rErr != nil {
			return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), rErr)
		}

		return rVal, nil
	}

	if p.matchKeyword("RETURN") {
		expr, err := p.parseExpression()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while parsing return value"), err)
		}

		return append([]Bytecode{B_RETURN}, expr...), nil
	}

	rVal, rErr := p.parsePrimary()

	if rErr != nil {
		_, err := p.peek()

		if err != nil {
			panic(err)
		}

		return []Bytecode{}, errors.Join(errors.New("got error while parsing primary expression"), rErr)
	}

	for {
		if p.matchOperator("DOT") {
			accessor, rErr := p.parsePrimary()

			if rErr != nil {
				return []Bytecode{}, rErr
			}

			rVal = append(append([]Bytecode{B_DOT}, rVal...), accessor...)

			continue
		}

		break
	}

	for {
		if p.matchOperator("LEFT_BRACKET") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			if !p.matchOperator("RIGHT_BRACKET") {
				return []Bytecode{}, errors.New("expected ']' after index operator")
			}

			rVal = append(append(append([]Bytecode{B_DOT}, rVal...), B_RESOLVE), elt...)
		}

		break
	}

	for {
		if p.matchOperator("PLUS") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			rVal = append(append([]Bytecode{B_OP_ADD}, rVal...), elt...)

			continue
		}

		if p.matchOperator("MINUS") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			rVal = append(append([]Bytecode{B_OP_MIN}, rVal...), elt...)

			continue
		}

		if p.matchOperator("STAR") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			rVal = append(append([]Bytecode{B_OP_MUL}, rVal...), elt...)

			continue
		}

		if p.matchOperator("SLASH") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			rVal = append(append([]Bytecode{B_OP_DIV}, rVal...), elt...)

			continue
		}

		if p.matchOperator("EQUALITY") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			rVal = append(append([]Bytecode{B_OP_EQ}, rVal...), elt...)

			continue
		}

		break
	}

	if p.matchOperator("LEFT_PAREN") {
		argsCount := 0
		arguments := make([]Bytecode, 0)
		token, err := p.peek()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while reading call operation arguments"), err)
		}

		if token.Type != TokenOperator && string(token.Value) != "RIGHT_PAREN" {
			for cond := true; cond; cond = p.matchOperator("COMMA") {
				arg, err := p.parseExpression()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while reading call operation arguments"), err)
				}

				arguments = append(arguments, arg...)
				argsCount++
			}
		}

		if !p.matchOperator("RIGHT_PAREN") {
			token, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error wihle reading call operation arguments"), err)
			}

			return []Bytecode{}, fmt.Errorf("expected ')' after call arguments got '%s'", string(token.Value))
		}

		rVal = append(append(append([]Bytecode{B_CALL}, rVal...), Bytecode(argsCount)), arguments...)
	}

	if p.matchOperator("EQUALS") {
		expr, err := p.parseExpression()

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while resolving assign expression"), err)
		}

		return append(append([]Bytecode{B_SET}, rVal...), expr...), nil
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
		return []Bytecode{}, errors.New("closing brace not found")
	}

	return append(append([]Bytecode{B_NEW_SCOPE}, body...), B_END_SCOPE), nil
}

func (p *Parser) parsePrimary() ([]Bytecode, error) {
	if p.matchKeyword("FALSE") {
		return []Bytecode{B_LITERAL, 0}, nil
	}

	if p.matchKeyword("TRUE") {
		return []Bytecode{B_LITERAL, 1}, nil
	}

	currentToken, err := p.peek()

	if err != nil {
		return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
	}

	if currentToken.Type == TokenNumber {
		p.advance()

		val, err := strconv.Atoi(string(currentToken.Value))

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("encountered wrong format for number"), err)
		}

		literalIdx, err := p.AppendLiteral(Literal{
			LiteralType: IntLiteral,
			Value:       val,
		})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
		}

		return literalIdx, nil
	}

	if currentToken.Type == TokenString {
		p.advance()

		literalIdx, err := p.AppendLiteral(Literal{
			LiteralType: StringLiteral,
			Value:       string(currentToken.Value),
		})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
		}

		return literalIdx, nil
	}

	if currentToken.Type == TokenIdentifier {
		p.advance()

		literalIdx, err := p.AppendLiteral(Literal{
			LiteralType: RefLiteral,
			Value:       ReferenceDeclaration{Reference: string(currentToken.Value), Dynamic: false},
		})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
		}

		return literalIdx, nil
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

	if p.matchOperator("OBJ_START") {
		entries := make([][]Bytecode, 0)

		if !p.matchOperator("OBJ_END") {
			for {
				entry := make([]Bytecode, 0)

				objKey, err := p.parseExpression()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("encountered error while parsing obj key"), err)
				}

				entry = append(entry, objKey...)

				if !p.matchOperator("COLON") {
					return []Bytecode{}, errors.New("expected colon to separate key and value")
				}

				objVal, err := p.parseExpression()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("encountered error while parsing obj value"), err)
				}

				entries = append(entries, append(entry, objVal...))

				if !p.matchOperator("COMMA") {
					break
				}
			}

			if !p.matchOperator("OBJ_END") {
				return []Bytecode{}, errors.New("expected closing operator for object")
			}
		}

		literalIdx, err := p.AppendLiteral(Literal{
			LiteralType: ObjLiteral,
			Value:       ObjDefinition{Entries: entries},
		})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
		}

		return literalIdx, nil
	}

	if p.matchOperator("LEFT_BRACKET") {
		elements := make([][]Bytecode, 0)

		for cond := true; cond; cond = p.matchOperator("COMMA") {
			elt, err := p.parseExpression()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			elements = append(elements, elt)
		}

		if !p.matchOperator("RIGHT_BRACKET") {
			return []Bytecode{}, errors.New("expected ']' after array elements")
		}

		literalIdx, err := p.AppendLiteral(Literal{
			LiteralType: ListLiteral,
			Value:       ListDefinition{Entries: elements},
		})

		if err != nil {
			return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
		}

		return literalIdx, nil
	}

	return []Bytecode{}, fmt.Errorf("expected expression, got %s", string(currentToken.Value))
}

func (p *Parser) match(tokenType TokenType, value string) bool {
	currentToken, err := p.peek()

	if err != nil || currentToken.Type != tokenType {
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
	B_SET
	B_LITERAL
	B_RETURN
	B_NEW_SCOPE
	B_END_SCOPE
	B_DOT
	B_CALL
	B_RESOLVE
	B_COND_JUMP
	B_JUMP
	B_OP_ADD
	B_OP_MIN
	B_OP_MUL
	B_OP_DIV
	B_OP_EQ
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

	if literal.LiteralType != ObjLiteral && literal.LiteralType != ListLiteral {
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
