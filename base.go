package parts

import (
	"errors"
	"fmt"
	"strings"
)

func FillConsts(vm *VM, act *Parser) {
	vm.Enviroment.AppendValues(map[string]any{
		"TokenOperator":   int(TokenOperator),
		"TokenNumber":     int(TokenNumber),
		"TokenKeyword":    int(TokenKeyword),
		"TokenIdentifier": int(TokenIdentifier),
		"TokenString":     int(TokenString),
		"TokenSpace":      int(TokenSpace),
		"TokenInvalid":    int(TokenInvalid),
	})

	vm.Enviroment.AppendValues(map[string]any{
		"IntLiteral":        int(IntLiteral),
		"DoubleLiteral":     int(DoubleLiteral),
		"BoolLiteral":       int(BoolLiteral),
		"StringLiteral":     int(StringLiteral),
		"RefLiteral":        int(RefLiteral),
		"FunLiteral":        int(FunLiteral),
		"ObjLiteral":        int(ObjLiteral),
		"ParsedObjLiteral":  int(ParsedObjLiteral),
		"ListLiteral":       int(ListLiteral),
		"ParsedListLiteral": int(ParsedListLiteral),
		"PointerLiteral":    int(PointerLiteral),
	})

	vm.Enviroment.DefineFunction("ClearScanner", func() {
		act.Scanner.Rules = make([]ScannerRule, 0)
	})

	vm.Enviroment.DefineFunction("ClearParser", func() {
		act.Rules = make([]ParserRule, 0)
		act.PostFix = make([]PostFixRule, 0)
	})

	vm.Enviroment.DefineFunction("ParserAppendLiteral", func(p *Parser, obj any) []Bytecode {
		keyed := obj.(map[string]any)
		lit := Literal{
			LiteralType: LiteralType(keyed["RTLiteralType"].(int)),
		}

		switch lit.LiteralType {
		case IntLiteral:
			lit.Value = keyed["RTValue"].(int)
		case DoubleLiteral:
			lit.Value = keyed["RTValue"].(float32)
		case BoolLiteral:
			lit.Value = keyed["RTValue"].(bool)
		case StringLiteral:
			lit.Value = keyed["RTValue"].(string)
		case RefLiteral:
			tempObj := keyed["RTValue"].(map[string]any)

			lit.Value = ReferenceDeclaration{
				Reference: tempObj["RTReference"].(string),
				Dynamic:   tempObj["RTDynamic"].(bool),
			}
		case ListLiteral:
			tempList := keyed["RTValue"].([]any)

			for idx, elt := range tempList {
				fmt.Printf("%d - %t", idx, elt)
			}
		default:
			panic(fmt.Errorf("%d type not implemented", lit.LiteralType))
		}

		res, err := p.AppendLiteral(lit)

		if err != nil {
			panic(err)
		}

		return res
	})

	vm.Enviroment.DefineFunction("ParserCheck", func(p *Parser, tt TokenType, val string) bool {
		return p.check(tt, val)
	})

	vm.Enviroment.DefineFunction("ParserMatch", func(p *Parser, tt TokenType, val string) bool {
		return p.match(tt, val)
	})

	vm.Enviroment.DefineFunction("ParserPeek", func(p *Parser) (Token, error) {
		return p.peek()
	})

	vm.Enviroment.DefineFunction("TokenType", func(t Token) TokenType {
		return t.Type
	})

	vm.Enviroment.DefineFunction("TokenValue", func(t Token) string {
		return string(t.Value)
	})

	vm.Enviroment.DefineFunction("ParserAdvance", func(p *Parser) (Token, error) {
		return p.advance()
	})

	vm.Enviroment.DefineFunction("ParserParse", func(p *Parser) ([]Bytecode, error) {
		return p.parse()
	})

	vm.Enviroment.DefineFunction("AddScannerRule", func(obj any) {
		rule := ScannerRule{}

		for key, val := range obj.(map[string]any) {
			key, found := strings.CutPrefix(key, "RT")

			if !found {
				panic("expected refererence got something else instead")
			}

			switch key {
			case "Result":
				rule.Result = TokenType(val.(int))
			case "BaseRule":
				rule.BaseRule = func(r rune) bool {
					cast, ok := val.(func(...any) (any, error))

					if !ok {
						panic("invalid base rule")
					}

					res, err := cast(string(r))

					if err != nil {
						panic(err)
					}

					return res.(bool)
				}
			case "Rule":
				rule.Rule = func(runes []rune) bool {
					cast, ok := val.(func(...any) (any, error))

					if !ok {
						panic("invalid rule")
					}

					res, err := cast(string(runes))

					if err != nil {
						panic(err)
					}

					return res.(bool)
				}
			case "Skip":
				rule.Skip = val.(bool)
			case "Mappings":
				tempMap := val.(map[string]any)
				res := make(map[string]string, 0)

				for key, val := range tempMap {
					key, found := strings.CutPrefix(key, "ST")

					if !found {
						panic("expected string got something else instead")
					}

					res[key] = val.(string)
				}

				rule.Mappings = res
			case "ValidChars":
				tempList := val.([]any)
				res := make([]rune, 0)

				for _, val := range tempList {
					temp, _ := val.(string)
					for _, tmp := range temp {
						res = append(res, tmp)
					}
				}

				rule.ValidChars = res
			case "Process":
				rule.Process = func(mappings map[string]string, runs []rune) ([]Token, error) {
					cast, ok := val.(func(...any) (any, error))

					if !ok {
						return []Token{}, errors.New("invalid function")
					}

					res, err := cast(mappings, string(runs))

					if err != nil {
						return []Token{}, errors.Join(errors.New("got error in function execution"), err)
					}

					if out, ok := res.([]any); ok {
						arr := make([]Token, len(out))

						for _, elt := range out {
							eltMap := elt.(map[string]any)

							arr = append(arr, Token{
								Type:  TokenType(eltMap["RTType"].(int)),
								Value: []rune(eltMap["RTValue"].(string)),
							})
						}

						return arr, nil
					} else {
						eltMap := res.(map[string]any)

						return []Token{{
							Type:  TokenType(eltMap["RTType"].(int)),
							Value: []rune(eltMap["RTValue"].(string)),
						}}, nil
					}
				}
			}
		}

		act.Scanner.Rules = append(act.Scanner.Rules, rule)
	})

	vm.Enviroment.DefineFunction("AddParserRule", func(postfix bool, obj any) {
		if !postfix {
			rule := ParserRule{}

			for key, val := range obj.(map[string]any) {
				key, found := strings.CutPrefix(key, "RT")

				if !found {
					panic("expected refererence got something else instead")
				}

				switch key {
				case "Id":
					rule.Id = val.(string)
				case "AdvanceToken":
					rule.AdvanceToken = val.(bool)
				case "Rule":
					rule.Rule = func(p *Parser) bool {
						cast, ok := val.(func(...any) (any, error))

						if !ok {
							panic("invalid rule")
						}

						res, err := cast(p)

						if err != nil {
							panic(err)
						}

						return res.(bool)
					}
				case "Parse":
					rule.Parse = func(p *Parser) ([]Bytecode, error) {
						cast, ok := val.(func(...any) (any, error))

						if !ok {
							return nil, errors.New("expected function, got something else")
						}

						res, err := cast(p)

						if err != nil {
							return nil, errors.Join(errors.New("got error while executing parts function"), err)
						}

						bytecode, ok := res.([]any)

						if !ok {
							return nil, errors.New("expected array of ints, got something else")
						}

						temp := make([]Bytecode, 0, len(bytecode))

						for _, elt := range bytecode {
							if casted, ok := elt.(int); !ok {
								return nil, errors.New("expected int, got something else")
							} else {
								temp = append(temp, Bytecode(casted))
							}
						}

						return temp, nil
					}
				}
			}

			act.Rules = append(act.Rules, rule)
		} else {
			rule := PostFixRule{}

			for key, val := range obj.(map[string]any) {
				key, found := strings.CutPrefix(key, "RT")

				if !found {
					panic("expected refererence got something else instead")
				}

				switch key {
				case "Id":
					rule.Id = val.(string)
				case "AdvanceToken":
					rule.AdvanceToken = val.(bool)
				case "Rule":
					rule.Rule = func(p *Parser) bool {
						cast, ok := val.(func(...any) (any, error))

						if !ok {
							panic("invalid rule")
						}

						res, err := cast(p)

						if err != nil {
							panic(err)
						}

						return res.(bool)
					}
				case "Parse":
					rule.Parse = func(p *Parser, btc []Bytecode) ([]Bytecode, error) {
						cast, ok := val.(func(...any) (any, error))

						if !ok {
							return []Bytecode{}, errors.New("expected function got something else")
						}

						res, err := cast(p, btc)

						if err != nil {
							return []Bytecode{}, errors.Join(errors.New("got error while running parts function"), err)
						}

						bytecode, ok := res.([]any)

						if !ok {
							return nil, errors.New("expected array of ints, got something else")
						}

						temp := make([]Bytecode, 0, len(bytecode))

						for _, elt := range bytecode {
							if casted, ok := elt.(int); !ok {
								return nil, errors.New("expected int, got something else")
							} else {
								temp = append(temp, Bytecode(casted))
							}
						}

						return temp, nil
					}
				}
			}

			act.PostFix = append(act.PostFix, rule)
		}
	})
}
