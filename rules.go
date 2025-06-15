package parts

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"
)

type ScannerRule struct {
	Result     TokenType
	BaseRule   func(r rune) bool
	Rule       func(runs []rune) bool
	Process    func(mappings map[string]string, runs []rune) ([]Token, error)
	Skip       bool
	Mappings   map[string]string
	ValidChars []rune
}

func GetScannerRules() []ScannerRule {
	return []ScannerRule{
		{
			Result: TokenOperator,
			Process: func(mappings map[string]string, runs []rune) ([]Token, error) {
				tokenValue := string(runs)
				name, ok := mappings[tokenValue]

				if ok {
					return []Token{{Type: TokenOperator, Value: []rune(name)}}, nil
				}

				offset := 0
				retTokens := make([]Token, 0)

				temp := tokenValue

				for offset != len(tokenValue) {
					if len(temp) == 0 {
						return []Token{}, fmt.Errorf("not valid operator ( %s )", tokenValue[offset:])
					}

					name, ok := mappings[temp]

					if ok {
						offset += len(temp)

						retTokens = append(retTokens, Token{Type: TokenOperator, Value: []rune(name)})

						temp = tokenValue[offset:]
					} else {
						temp = temp[:len(temp)-1]
					}
				}

				return retTokens, nil
			},
			ValidChars: []rune{'+', '-', '/', '*', ';', '[', ']', '(', ')', '{', '}', '.', ':', ',', '|', '&', '>', '<', '!', '#', '-', '=', '?'},
			Mappings: map[string]string{
				"+": "PLUS", "-": "MINUS", "/": "SLASH", "*": "STAR",
				";": "SEMICOLON", ":": "COLON",
				".": "DOT", ",": "COMMA",
				"(": "LEFT_PAREN", ")": "RIGHT_PAREN",
				"{": "LEFT_BRACE", "}": "RIGHT_BRACE",
				"[": "LEFT_BRACKET", "]": "RIGHT_BRACKET",
				"@": "AT", "=": "EQUALS",
				"|>": "OBJ_START", "<|": "OBJ_END",
				"#>": "META", "==": "EQUALITY",
				"<": "LESS_THAN", ">": "MORE_THAN",
				"<=": "LESS_EQ", ">=": "MORE_EQ",
			},
		},
		{
			Result:   TokenNumber,
			BaseRule: func(r rune) bool { return r >= '0' && r <= '9' },
		},
		{
			Result: TokenKeyword,
			BaseRule: func(r rune) bool {
				return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
			},
			Process: func(mappings map[string]string, runs []rune) ([]Token, error) {
				token := Token{
					Type:  TokenKeyword,
					Value: runs,
				}

				if _, has := mappings[string(token.Value)]; has {
					token.Value = []rune(strings.ToUpper(string(token.Value)))
				} else {
					token.Type = TokenIdentifier
				}

				return []Token{token}, nil
			},
			Mappings: map[string]string{
				"false": "", "if": "", "let": "", "true": "",
				"fun": "", "return": "", "else": "", "for": "",
				"import": "", "from": "", "as": "", "syntax": "",
				"use": "", "raise": "", "break": "",
			},
		},
		{
			Result: TokenSpace,
			BaseRule: func(r rune) bool {
				return unicode.IsSpace(r)
			},
			Skip: true,
		},
		{
			Result:   TokenString,
			BaseRule: func(r rune) bool { return true },
			Rule: func(runs []rune) bool {
				return len(runs) == 1 || runs[0] != runs[len(runs)-1]
			},
			Process: func(mappings map[string]string, runs []rune) ([]Token, error) {
				token := Token{Type: TokenString, Value: runs}

				leftSide := runs[0]

				if leftSide == '"' {
					if runs[len(runs)-1] != '"' {
						return []Token{}, errors.New("got unterminated string")
					}
				} else if leftSide == '`' {
					if runs[len(runs)-1] != '`' {
						return []Token{}, errors.New("got unterminated string")
					}
				} else {
					return []Token{}, errors.New("got unexpected token expected '\"' or '`' charater")
				}

				content := runs[1 : len(runs)-1]

				var unescaped []rune
				for i := 0; i < len(content); i++ {
					if content[i] == '\\' {
						if i+1 >= len(content) {
							return []Token{}, errors.New("got unterminated escape sequence")
						}
						switch content[i+1] {
						case '"':
							unescaped = append(unescaped, '"')
						case '\\':
							unescaped = append(unescaped, '\\')
						case 'n':
							unescaped = append(unescaped, '\n')
						case 't':
							unescaped = append(unescaped, '\t')
						case 'r':
							unescaped = append(unescaped, '\r')
						case 'b':
							unescaped = append(unescaped, '\b')
						case 'f':
							unescaped = append(unescaped, '\f')
						default:
							return []Token{}, fmt.Errorf("invalid escape sequence: \\%c", content[i+1])
						}

						i++
					} else {
						unescaped = append(unescaped, content[i])
					}
				}
				token.Value = unescaped

				return []Token{token}, nil
			},
		},
	}
}

type ParserRule struct {
	Id           string
	AdvanceToken bool
	Rule         func(*Parser) bool
	Parse        func(*Parser) ([]Bytecode, error)
}

func GetParserRules() []ParserRule {
	return []ParserRule{
		{
			Id:           "SyntaxChange",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "SYNTAX") },
			Parse: func(p *Parser) ([]Bytecode, error) {
				if !p.matchOperator("LEFT_BRACE") {
					return []Bytecode{}, errors.New("expected '{' after syntax change keyword")
				}

				tempBytecode, err := p.parseWithRule("ParseStr")

				if err != nil {
					return []Bytecode{}, err
				}

				if tempBytecode[0] != B_LITERAL {
					return []Bytecode{}, errors.New("expected string literal")
				}

				tempBytecode = tempBytecode[1:]
				stringIdx := -1

				if tempBytecode[0] <= 125 {
					stringIdx = int(tempBytecode[0])
				}

				if tempBytecode[0] == 126 {
					stringIdx = int(tempBytecode[1])<<8 | int(tempBytecode[2])
				}

				if tempBytecode[0] == 127 {
					stringIdx = int(tempBytecode[1])<<56 |
						int(tempBytecode[2])<<48 |
						int(tempBytecode[3])<<40 |
						int(tempBytecode[4])<<32 |
						int(tempBytecode[5])<<24 |
						int(tempBytecode[6])<<16 |
						int(tempBytecode[7])<<8 |
						int(tempBytecode[8])
				}

				if stringIdx == -1 {
					return []Bytecode{}, errors.New("expected string literal")
				}

				stringLiteral := p.Literals[stringIdx]

				if stringLiteral.LiteralType != StringLiteral {
					return []Bytecode{}, errors.New("expected string literal")
				}

				if !p.check(TokenOperator, "RIGHT_BRACE") {
					return []Bytecode{}, errors.New("expected '}' after syntax body")
				}

				vm, err := GetVMWithSource(stringLiteral.Value.(string), p.ModulePath)

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error when parsing syntax block code"), err)
				}

				FillConsts(vm, p)

				err = vm.Run()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error when executing syntax block code"), err)
				}

				_, err = p.advance()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error when reading next block"), err)
				}

				return []Bytecode{}, nil
			},
		},
		{
			Id:           "ImportStmt",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "IMPORT") },
			Parse: func(p *Parser) ([]Bytecode, error) {
				if p.matchKeyword("SYNTAX") {
					if !p.matchKeyword("FROM") {
						return []Bytecode{}, errors.New("expected 'from' keyword after import syntax")
					}

					tempBytecode, err := p.parseWithRule("ParseStr")

					if err != nil {
						return []Bytecode{}, err
					}

					if tempBytecode[0] != B_LITERAL {
						return []Bytecode{}, errors.New("expected string literal")
					}

					tempBytecode = tempBytecode[1:]
					stringIdx := -1

					switch tempBytecode[0] {
					case 126:
						stringIdx = int(tempBytecode[1])<<8 | int(tempBytecode[2])
					case 127:
						stringIdx = int(tempBytecode[1])<<56 | int(tempBytecode[2])<<48 |
							int(tempBytecode[3])<<40 | int(tempBytecode[4])<<32 |
							int(tempBytecode[5])<<24 | int(tempBytecode[6])<<16 |
							int(tempBytecode[7])<<8 | int(tempBytecode[8])
					default:
						stringIdx = int(tempBytecode[0])
					}

					if stringIdx == -1 {
						return []Bytecode{}, errors.New("expected string literal")
					}

					stringLiteral := p.Literals[stringIdx]

					if stringLiteral.LiteralType != StringLiteral {
						return []Bytecode{}, errors.New("expected string literal")
					}

					source := stringLiteral.Value.(string)

					if !p.matchKeyword("AS") {
						return []Bytecode{}, errors.New("expected 'as' keyword after import path")
					}

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

					rawFile, err := os.ReadFile(path.Join(p.ModulePath, source))

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("got error while reading foreign module file"), err)
					}

					literalIdx, err := p.AppendLiteral(Literal{
						LiteralType: ParsedObjLiteral,
						Value: PartsSpecialObject{Internal: &PartsObject{
							Entries: map[string]*Literal{
								"RTUse": {FunLiteral, NativeMethod{
									Args: []string{"code"},
									Body: func(vm *VM, args []*Literal) error {

										if len(args) < 1 {
											return nil
										}

										if args[0].LiteralType != StringLiteral {
											return nil
										}

										newVm, err := RunStringWithSyntax(args[0].Value.(string), string(rawFile), p.ModulePath)

										if err != nil {
											return errors.Join(errors.New("got error while running syntax"), err)
										}

										vm.ReturnValue = newVm.LastExpr

										return nil
									},
								}},
							},
						}, Hash: "Parts.Syntax"},
					})

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("got error while encoding length"), err)
					}

					return append(append([]Bytecode{B_DECLARE}, literalCode...), literalIdx...), nil
				}

				return []Bytecode{}, nil
			},
		},
		{
			Id:           "UseExpr",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "USE") },
			Parse: func(p *Parser) ([]Bytecode, error) {
				tempBytecode, err := p.parseWithRule("ParseGroup")

				if err != nil {
					return []Bytecode{}, err
				}

				if tempBytecode[0] != B_LITERAL {
					return []Bytecode{}, errors.New("expected string literal")
				}

				objCode := tempBytecode

				if !p.matchOperator("LEFT_BRACE") {
					return []Bytecode{}, errors.New("expected '{' after use obj")
				}

				tempBytecode, err = p.parseWithRule("ParseStr")

				if err != nil {
					return []Bytecode{}, err
				}

				if tempBytecode[0] != B_LITERAL {
					return []Bytecode{}, errors.New("expected string literal")
				}

				stringRef := tempBytecode

				tempBytecode = tempBytecode[1:]
				stringIdx := -1

				if tempBytecode[0] <= 125 {
					stringIdx = int(tempBytecode[0])
				}

				if tempBytecode[0] == 126 {
					stringIdx = int(tempBytecode[1])<<8 | int(tempBytecode[2])
				}

				if tempBytecode[0] == 127 {
					stringIdx = int(tempBytecode[1])<<56 | int(tempBytecode[2])<<48 |
						int(tempBytecode[3])<<40 | int(tempBytecode[4])<<32 |
						int(tempBytecode[5])<<24 | int(tempBytecode[6])<<16 |
						int(tempBytecode[7])<<8 | int(tempBytecode[8])
				}

				if stringIdx == -1 {
					return []Bytecode{}, errors.New("expected string literal")
				}

				stringLiteral := p.Literals[stringIdx]

				if stringLiteral.LiteralType != StringLiteral {
					return []Bytecode{}, errors.New("expected string literal")
				}

				if !p.matchOperator("RIGHT_BRACE") {
					return []Bytecode{}, errors.New("expected '}' after use code")
				}

				useIdx, err := p.AppendLiteral(Literal{LiteralType: RefLiteral,
					Value: ReferenceDeclaration{
						Reference: "Use",
						Dynamic:   false,
					},
				})

				if err != nil {
					return []Bytecode{}, err
				}

				return append(append(append(append([]Bytecode{B_CALL, B_DOT}, objCode...), useIdx...), 1), stringRef...), nil
			},
		},
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
					token, err := p.peek()

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("got error while reading function params"), err)
					}

					declaration := FunctionDeclaration{Params: []string{}, Body: []Bytecode{}}

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
						expr, err := p.parse()

						if err != nil {
							return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), err)
						}

						declaration.Body = append(append(declaration.Body, B_RETURN), expr...)
					} else {
						body, err := p.parseWithRule("BlockExpr")

						if err != nil {
							return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), err)
						}

						declaration.Body = body
					}

					idx, err := p.AppendLiteral(Literal{FunLiteral, declaration})

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

				declaration := FunctionDeclaration{Params: []string{}, Body: []Bytecode{}}

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
					expr, err := p.parse()

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), err)
					}

					declaration.Body = append([]Bytecode{B_RETURN}, expr...)
				} else {
					body, err := p.parseWithRule("BlockExpr")

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("encountered err in function body"), err)
					}

					declaration.Body = body
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
			Id:           "ForExpr",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "FOR") },
			Parse: func(p *Parser) ([]Bytecode, error) {
				var loopCondition []Bytecode

				if p.check(TokenOperator, "LEFT_PAREN") {
					btc, err := p.parse()

					if err != nil {
						return []Bytecode{}, errors.Join(errors.New("got error while parsing loop condition"), err)
					}

					loopCondition = btc
				}

				if len(loopCondition) == 0 {
					loopCondition = []Bytecode{B_LITERAL, 1}
				}

				forBody, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing loop body"), err)
				}

				bodyLength, err := encodeLen(len(forBody))

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding for body)"), err)
				}

				tempReverseJump, err := encodeLen(len(loopCondition) + len(forBody) + len(bodyLength))

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding reverse jump length)"), err)
				}

				bodyLength, err = encodeLen(len(forBody) + len(tempReverseJump) + 1)

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding for body - 2 pass)"), err)
				}

				tempReverseJump, err = encodeLen(len(loopCondition) + len(forBody) + len(bodyLength))

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding reverse jump length - 2 pass)"), err)
				}

				bodyLength, err = encodeLen(len(forBody) + len(tempReverseJump) + 1)

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding for body - 3 pass)"), err)
				}

				tempReverseJump, err = encodeLen(len(loopCondition) + len(forBody) + len(bodyLength))

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while encoding length expression (encoding reverse jump length - 3 pass)"), err)
				}

				return append(append(append(append(append(append([]Bytecode{B_COND_JUMP}, loopCondition...), bodyLength...), forBody...), B_JUMP_REV), tempReverseJump...), 0), nil
			},
		},
		{
			Id: "Break",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "BREAK") },
			Parse: func(p *Parser) ([]Bytecode, error) {
				return []Bytecode{B_RETURN}, nil
			},
		},
		{
			Id:           "BlockExpr",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_BRACE") },
			Parse: func(p *Parser) ([]Bytecode, error) {
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
			Id:           "RaiseExpr",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "RAISE") },
			Parse: func(p *Parser) ([]Bytecode, error) {
				expr, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("got error while parsing return value"), err)
				}

				return append([]Bytecode{B_RAISE}, expr...), nil
			},
		},
		{
			Id:           "BoolFExpr",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "FALSE") },
			Parse:        func(p *Parser) ([]Bytecode, error) { return []Bytecode{B_LITERAL, 0}, nil },
		},
		{
			Id:           "BoolTExpr",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenKeyword, "TRUE") },
			Parse:        func(p *Parser) ([]Bytecode, error) { return []Bytecode{B_LITERAL, 1}, nil },
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

				if !p.matchOperator("RIGHT_BRACKET") {
					for {
						elt, err := p.parse()

						if err != nil {
							return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
						}

						elements = append(elements, elt)

						if !p.matchOperator("COMMA") {
							break
						}
					}

					if !p.matchOperator("RIGHT_BRACKET") {
						tok, _ := p.peek()

						return []Bytecode{}, fmt.Errorf("expected ']' after array elements, got %s", string(tok.Value))
					}
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
}

type PostFixRule struct {
	Id           string
	AdvanceToken bool
	Rule         func(*Parser) bool
	Parse        func(*Parser, []Bytecode) ([]Bytecode, error)
}

func GetPostFixRules() []PostFixRule {
	return []PostFixRule{
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

				return append(append([]Bytecode{B_BIN_OP, B_OP_ADD}, code...), elt...), nil
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

				return append(append([]Bytecode{B_BIN_OP, B_OP_MIN}, code...), elt...), nil
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

				return append(append([]Bytecode{B_BIN_OP, B_OP_MUL}, code...), elt...), nil
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

				return append(append([]Bytecode{B_BIN_OP, B_OP_DIV}, code...), elt...), nil
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

				return append(append([]Bytecode{B_BIN_OP, B_OP_EQ}, code...), elt...), nil
			},
		},
		{
			Id:           "LtOp",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LESS_THAN") },
			Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
				elt, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
				}

				return append(append([]Bytecode{B_BIN_OP, B_OP_LT}, code...), elt...), nil
			},
		},
		{
			Id:           "GtOp",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenOperator, "MORE_THAN") },
			Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
				elt, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
				}

				return append(append([]Bytecode{B_BIN_OP, B_OP_GT}, code...), elt...), nil
			},
		},
		{
			Id:           "GtEqOp",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenOperator, "MORE_EQ") },
			Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
				elt, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
				}

				returnArr := make([]Bytecode, 0, 6+len(code)*2+len(elt)*2)

				returnArr = append(returnArr, B_BIN_OP, B_OP_ADD, B_BIN_OP, B_OP_GT)
				returnArr = append(returnArr, code...)
				returnArr = append(returnArr, elt...)
				returnArr = append(returnArr, B_BIN_OP, B_OP_EQ)
				returnArr = append(returnArr, code...)
				returnArr = append(returnArr, elt...)

				return returnArr, nil
			},
		},
		{
			Id:           "LtEqOp",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LESS_EQ") },
			Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
				elt, err := p.parse()

				if err != nil {
					return []Bytecode{}, errors.Join(errors.New("expected expression, got error"), err)
				}

				returnArr := make([]Bytecode, 0, 6+len(code)*2+len(elt)*2)

				returnArr = append(returnArr, B_BIN_OP, B_OP_ADD, B_BIN_OP, B_OP_LT)
				returnArr = append(returnArr, code...)
				returnArr = append(returnArr, elt...)
				returnArr = append(returnArr, B_BIN_OP, B_OP_EQ)
				returnArr = append(returnArr, code...)
				returnArr = append(returnArr, elt...)

				return returnArr, nil
			},
		},
		{
			Id:           "FunCall",
			AdvanceToken: true,
			Rule:         func(p *Parser) bool { return p.check(TokenOperator, "LEFT_PAREN") },
			Parse: func(p *Parser, code []Bytecode) ([]Bytecode, error) {
				argsCount := 0
				arguments := make([]Bytecode, 0)

				if !p.check(TokenOperator, "RIGHT_PAREN") {
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
}
