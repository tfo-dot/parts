package parts

import (
	"errors"
	"fmt"
	"strconv"
)

type Parser struct {
	Scanner *Scanner

	LastToken Token
	Rules     []ParserRule
	PostFix   []PostFixRule

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

	return []Bytecode{}, errors.New("Unknown rule?")
}

type ParserRule struct {
	Id           string
	AdvanceToken bool
	Rule         func(*Parser) bool
	Parse        func(*Parser) ([]Bytecode, error)
}

var ParserRules = []ParserRule{
	{
		Id:           "LetStmt",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "LET") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			identifierToken, err := p.advance()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while advancing token"), err)
			}

			if identifierToken.Type != TokenIdentifier {
				return []Bytecode{}, fmt.Errorf("got invalid token instead of identifier ( %d )", identifierToken.Type)
			}

			literalCode, err := p.AppendLiteral(Literal{
				LiteralType: RefLiteral,
				Value:       ReferenceDeclaration{Reference: string(identifierToken.Value), Dynamic: false},
			})

			initialValue := []Bytecode{}

			if p.matchOperator("LEFT_PAREN") {
				funDeclaration, err := p.parseFunction(true)

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing function definition"), err)
				}

				idx, err := p.AppendLiteral(Literal{FunLiteral, funDeclaration})

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("encountered err in length encoding"), err)
				}

				initialValue = idx
			} else {
				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while writing literal offset"), err)
				}

				if !p.matchOperator("EQUALS") {
					return []Bytecode{}, errors.New("expected value after equals operator")
				}

				rawVal, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving expression)"), err)
				}

				initialValue = rawVal
			}

			return append(append([]Bytecode{B_DECLARE}, literalCode...), initialValue...), nil
		},
	},
	{
		Id:           "MetaStmt",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "META") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			metaKeyToken, err := p.advance()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading meta value"), err)
			}

			if !p.matchOperator("COLON") {
				return []Bytecode{}, errors.New("missing ':' after meta key")
			}

			metaValueToken, err := p.advance()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading meta value"), err)
			}

			p.Meta[string(metaKeyToken.Value)] = string(metaValueToken.Value)

			return []Bytecode{}, nil
		},
	},
	{
		Id:           "FunExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "FUN") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			declaration, err := p.parseFunction(false)

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered err in length encoding"), err)
			}

			idx, err := p.AppendLiteral(Literal{FunLiteral, declaration})

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered err in length encoding"), err)
			}

			return idx, nil
		},
	},
	{
		Id:           "IfExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "IF") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			condition, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving condition)"), err)
			}

			thenBranch, err := p.parse()

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
				elseBranch, err = p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing expression (resolving else branch)"), err)
				}

				elseLength, err = encodeLen(len(elseBranch))

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding else branch)"), err)
				}
			}

			return append(append(append(append(append([]Bytecode{B_COND_JUMP}, condition...), thenLength...), thenBranch...), elseLength...), elseBranch...), nil
		},
	},
	{
		Id:           "BodyExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_BRACE") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			rVal, rErr := p.parseBlock(false)

			if rErr != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), rErr)
			}

			return rVal, nil
		},
	},
	{
		Id:           "ReturnExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "RETURN") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			expr, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while parsing return value"), err)
			}

			return append([]Bytecode{B_RETURN}, expr...), nil
		},
	},
	{
		Id:           "BoolFExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "FALSE") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			return []Bytecode{B_LITERAL, 0}, nil
		},
	},
	{
		Id:           "BoolTExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "TRUE") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			return []Bytecode{B_LITERAL, 1}, nil
		},
	},
	{
		Id: "ParseNum",
		Rule: func(p *Parser) bool {
			currentToken, err := p.peek()

			if err != nil {
				return false
			}

			return currentToken.Type == TokenNumber
		},
		Parse: func(p *Parser) ([]Bytecode, error) {
			raw, err := p.advance()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered error while reading num value"), err)
			}

			val, err := strconv.Atoi(string(raw.Value))

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
		},
	},
	{
		Id: "ParseStr",
		Rule: func(p *Parser) bool {
			currentToken, err := p.peek()

			if err != nil {
				return false
			}

			return currentToken.Type == TokenString
		},
		Parse: func(p *Parser) ([]Bytecode, error) {
			raw, err := p.advance()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading string literal"), err)
			}

			literalIdx, err := p.AppendLiteral(Literal{
				LiteralType: StringLiteral,
				Value:       string(raw.Value),
			})

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
			}

			return literalIdx, nil
		},
	},
	{
		Id: "ParseVar",
		Rule: func(p *Parser) bool {
			currentToken, err := p.peek()

			if err != nil {
				return false
			}

			return currentToken.Type == TokenIdentifier
		},
		Parse: func(p *Parser) ([]Bytecode, error) {
			raw, err := p.advance()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading string literal"), err)
			}

			literalIdx, err := p.AppendLiteral(Literal{
				LiteralType: RefLiteral,
				Value:       ReferenceDeclaration{Reference: string(raw.Value), Dynamic: false},
			})

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
			}

			return literalIdx, nil
		},
	},
	{
		Id:           "ParseGroup",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_PAREN") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			val, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("encountered error while parsing parenthesis"), err)
			}

			if !p.matchOperator("RIGHT_PAREN") {
				return []Bytecode{}, errors.New("no closing parenthesis")
			}

			return val, nil
		},
	},
	{
		Id:           "ParseObj",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "OBJ_START") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			entries := make([][]Bytecode, 0)

			if !p.matchOperator("OBJ_END") {
				for {
					entry := make([]Bytecode, 0)

					objKey, err := p.parse()

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("encountered error while parsing obj key"), err)
					}

					entry = append(entry, objKey...)

					if !p.matchOperator("COLON") {
						return []Bytecode{}, errors.New("expected colon to separate key and value")
					}

					objVal, err := p.parse()

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
		},
	},
	{
		Id:           "ParseArr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_BRACKET") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			elements := make([][]Bytecode, 0)

			for cond := true; cond; cond = p.matchOperator("COMMA") {
				elt, err := p.parse()

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
		},
	},
	{
		Id:           "SemiSkip",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "SEMICOLON") },
		Parse: func(p *Parser) ([]Bytecode, error) {
			return []Bytecode{}, nil
		},
	},
}

type PostFixRule struct {
	Id           string
	AdvanceToken bool
	Rule         func(*Parser) bool
	Parse        func(*Parser, []Bytecode) ([]Bytecode, error)
}

var PostFixRules = []PostFixRule{
	{
		Id:           "DotExpr",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "DOT") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			accessor, rErr := p.parse()

			if rErr != nil {
				return []Bytecode{}, rErr
			}

			return append(append([]Bytecode{B_DOT}, code...), accessor...), nil
		},
	},
	{
		Id:           "ArrIndex",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_BRACKET") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			elt, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			if !p.matchOperator("RIGHT_BRACKET") {
				return []Bytecode{}, errors.New("expected ']' after index operator")
			}

			return append(append(append([]Bytecode{B_DOT}, code...), B_RESOLVE), elt...), nil
		},
	},
	{
		Id:           "PlusOp",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "PLUS") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			elt, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			return append(append([]Bytecode{B_OP_ADD}, code...), elt...), nil
		},
	},
	{
		Id:           "MinusOp",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "MINUS") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			elt, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			return append(append([]Bytecode{B_OP_MIN}, code...), elt...), nil
		},
	},
	{
		Id:           "MulOp",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "STAR") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			elt, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			return append(append([]Bytecode{B_OP_MUL}, code...), elt...), nil
		},
	},
	{
		Id:           "DivOp",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "SLASH") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			elt, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			return append(append([]Bytecode{B_OP_DIV}, code...), elt...), nil
		},
	},
	{
		Id:           "EqOp",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "EQUALITY") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			elt, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
			}

			return append(append([]Bytecode{B_OP_EQ}, code...), elt...), nil
		},
	},
	{
		Id:           "FunCall",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_PAREN") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			argsCount := 0
			arguments := make([]Bytecode, 0)
			token, err := p.peek()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while reading call operation arguments"), err)
			}

			if token.Type != TokenOperator && string(token.Value) != "RIGHT_PAREN" {
				for cond := true; cond; cond = p.matchOperator("COMMA") {
					arg, err := p.parse()

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

			return append(append(append([]Bytecode{B_CALL}, code...), Bytecode(argsCount)), arguments...), nil
		},
	},
	{
		Id:           "SetOp",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "EQUALS") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			expr, err := p.parse()

			if err != nil {
				return []Bytecode{}, errors.Join(errors.New("got error while resolving assign expression"), err)
			}

			return append(append([]Bytecode{B_SET}, code...), expr...), nil
		},
	},
	{
		Id:           "SemiSkip",
		AdvanceToken: true,
		Rule:         func(p *Parser) bool { return p.check(TokenOperator, "SEMICOLON") },
		Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
			return code, nil
		},
	},
}

func (p *Parser) parseFunction(ignoreLeftParen bool) (FunctionDeclaration, error) {
	if !p.matchOperator("LEFT_PAREN") && !ignoreLeftParen {
		token, err := p.peek()

		if err != nil {
			return FunctionDeclaration{}, errors.Join(errors.New("got error while reading function params"), err)
		}

		return FunctionDeclaration{}, fmt.Errorf("expected '(' after function declaration got '%s'", string(token.Value))
	}

	token, err := p.peek()

	if err != nil {
		return FunctionDeclaration{}, errors.Join(errors.New("got error while reading function params"), err)
	}

	declaration := FunctionDeclaration{
		Params: []string{},
		Body:   []Bytecode{},
	}

	if token.Type != TokenOperator && string(token.Value) != "RIGHT_PAREN" {
		for cond := true; cond; cond = p.matchOperator("COMMA") {
			identifierToken, err := p.advance()

			if identifierToken.Type != TokenIdentifier {
				return FunctionDeclaration{}, errors.Join(errors.New("encountered unexpected token in function params"), err)
			}

			declaration.Params = append(declaration.Params, string(identifierToken.Value))
		}
	}

	if !p.matchOperator("RIGHT_PAREN") {
		token, err := p.peek()

		if err != nil {
			return FunctionDeclaration{}, errors.Join(errors.New("got error while reading function params"), err)
		}

		return FunctionDeclaration{}, fmt.Errorf("expected ')' after function params got '%s'", string(token.Value))
	}

	if p.matchOperator("EQUALS") {
		expr, err := p.parse()

		if err != nil {
			return FunctionDeclaration{}, errors.Join(errors.New("encountered err in function body"), err)
		}

		declaration.Body = append(append(declaration.Body, B_RETURN), expr...)
	} else {
		body, err := p.parseBlock(true)

		if err != nil {
			return FunctionDeclaration{}, errors.Join(errors.New("encountered err in function body"), err)
		}

		declaration.Body = body
	}

	return declaration, nil
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

		statement, err := p.parse()

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

func (p *Parser) match(tokenType TokenType, value string) bool {
	checkRes := p.check(tokenType, value)

	if checkRes {
		p.advance()
	}

	return checkRes
}

func (p *Parser) check(tokenType TokenType, value string) bool {
	currentToken, err := p.peek()

	if err != nil || currentToken.Type != tokenType {
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
	B_RULE_CHANGE
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
