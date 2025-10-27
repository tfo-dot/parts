package parts

import (
	"errors"
	"fmt"
	"slices"
)

type ExitCode = int

const (
	NormalCode ExitCode = iota
	BreakCode
	ContinueCode
	ReturnCode
)

type VM struct {
	Enviroment *VMEnviroment

	Idx int

	ReturnValue *Literal
	LastExpr    *Literal
	EarlyExit   bool
	ExitCode    ExitCode

	//Filled from parser
	Code     []Bytecode
	Literals []*Literal
	Meta     map[string]string
}

func (vm *VM) Run() error {
	for vm.Idx < len(vm.Code) {
		err := vm.Execute()

		if err != nil {
			return errors.Join(errors.New("got error while executing bytecode"), err)
		}

		if vm.EarlyExit {
			vm.Idx = len(vm.Code)
		}
	}

	return nil
}

func (vm *VM) Execute() error {
	switch vm.Code[vm.Idx] {
	case B_DECLARE:
		vm.Idx++

		exprType, nameLiteral, err := vm.runExpr(true)

		if err != nil {
			return errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral {
			return errors.New("expected literal as variable name")
		}

		envKey, err := HashLiteral(*nameLiteral.(*Literal))

		if err != nil {
			return errors.Join(errors.New("got error while defining name hash"), err)
		}

		exprType, value, err := vm.runExpr(true)

		if err != nil {
			return errors.Join(errors.New("got error while running variable value (B_DECLARE)"), err)
		}

		if exprType != TypeLiteral {
			return fmt.Errorf("expected value got %d (declare value)", exprType)
		}

		simpleValue, err := vm.simplifyLiteral(value.(*Literal), true)

		if err != nil {
			return errors.Join(errors.New("got error while simplyfing value"), err)
		}

		if _, err = vm.Enviroment.define(envKey, simpleValue); err != nil {
			return errors.Join(errors.New("got error while defining variable"), err)
		}
	default:
		if vm.Idx >= len(vm.Code) {
			return errors.New("tried running bytecode after the end")
		}

		if _, _, err := vm.runExpr(true); err != nil {
			return errors.Join(errors.New("got error while running bytecode"), err)
		}
	}

	return nil
}

func (vm *VM) handleNestedSet(accessor *Literal) (*Literal, error) {
	exprType, nameLiteral, err := vm.runExpr(false)

	if err != nil {
		return nil, errors.Join(errors.New("got error while running expression"), err)
	}

	if exprType != TypeLiteral && exprType != DotExpression {
		return nil, errors.New("expected literal as variable name")
	}

	typ, value, err := vm.runExpr(true)

	if err != nil {
		return nil, errors.Join(errors.New("got error while running variable value (B_DOT, B_SET)"), err)
	}

	if typ != TypeLiteral {
		return nil, errors.New("expected literal as value")
	}

	simpleValue, err := vm.simplifyLiteral(value.(*Literal), true)

	if err != nil {
		return nil, errors.Join(errors.New("got error while simplyfing value"), err)
	}

	var key []*Literal

	if exprType == DotExpression {
		key = append([]*Literal{accessor}, nameLiteral.([]*Literal)...)
	} else {
		key = []*Literal{accessor, nameLiteral.(*Literal)}
	}

	val, err := vm.Enviroment.assignDot(vm, key, simpleValue)

	if err != nil {
		return nil, errors.Join(errors.New("got error while assigning to a variable"), err)
	}

	return val, nil
}

func (vm *VM) handleNestedCall(accessor *Literal, argCount int) (*Literal, error) {
	values := make([]*Literal, argCount)

	for i := range values {
		exprType, expr, err := vm.runExpr(true)

		if err != nil {
			return nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral {
			return nil, fmt.Errorf("expected value got %d (resolve call arguments)", exprType)
		}

		resolvedExpr, err := vm.simplifyLiteral(expr.(*Literal), true)

		if err != nil {
			return nil, errors.Join(errors.New("got error while simplyfing expression"), err)
		}

		values[i] = resolvedExpr
	}

	return vm.callFunction(accessor.Value.(PartsCallable), values)
}

func (vm *VM) runExpr(unwindDot bool) (ExpressionType, any, error) {
	switch vm.Code[vm.Idx] {
	case B_NEW_SCOPE:
		vm.Idx++

		newEnv := VMEnviroment{
			Values:    make(map[string]*Literal),
			Enclosing: vm.Enviroment,
		}

		vm.Enviroment = &newEnv

		return ScopeChange, nil, nil
	case B_END_SCOPE:
		vm.Idx++

		if vm.Enviroment.Enclosing == nil {
			return ScopeChange, nil, errors.New("leaving scope but already at top level")
		}

		vm.Enviroment = vm.Enviroment.Enclosing

		return ScopeChange, nil, nil
	case B_BIN_OP:
		vm.Idx++

		rVal, err := vm.runOp()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running operation"), err)
		}

		vm.LastExpr = rVal

		return TypeLiteral, rVal, nil
	case B_LITERAL:
		vm.Idx++

		nameIdx, err := vm.decodeLen()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while decoding offset"), err)
		}

		vm.LastExpr = vm.Literals[nameIdx]

		return TypeLiteral, vm.Literals[nameIdx], nil
	case B_DOT:
		vm.Idx++

		exprType, rawAccessor, err := vm.runExpr(false)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral && exprType != DotExpression {
			return UndefinedExpression, nil, fmt.Errorf("expected value got %d (running dot accessor)", exprType)
		}

		accessor := rawAccessor.(*Literal)

		if accessor.LiteralType == RefLiteral {
			if acc, err := vm.simplifyLiteral(accessor, true); err == nil {
				accessor = acc
			} else {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while resolving reference (running dot accessor)"), err)
			}
		}

		switch accessor.LiteralType {
		case ListLiteral, ObjLiteral, ParsedListLiteral, ParsedObjLiteral:
		default:
			return UndefinedExpression, nil, fmt.Errorf("unexpected value type (%d) (B_DOT)", accessor.LiteralType)
		}

		if accessor.LiteralType == ListLiteral || accessor.LiteralType == ObjLiteral {
			accessor, err = vm.simplifyLiteral(accessor, true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
			}
		}

		if vm.Code[vm.Idx] == B_SET {
			vm.Idx++

			val, err := vm.handleNestedSet(rawAccessor.(*Literal))

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while setting field (B_DOT, B_SET)"), err)
			}

			vm.LastExpr = val

			return TypeLiteral, val, nil
		}

		fCall := false

		if vm.Code[vm.Idx] == B_CALL {
			fCall = true
			vm.Idx++
		}

		exprType, rawKey, err := vm.runExpr(unwindDot)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral && exprType != DotExpression {
			return UndefinedExpression, nil, errors.Join(fmt.Errorf("expected value got %d (dot value)", exprType), err)
		}

		argCount := 0

		if fCall {
			argCount = int(vm.Code[vm.Idx])
			vm.Idx++
		}

		if !unwindDot {
			resVal := make([]*Literal, 0)

			if val, ok := rawAccessor.([]*Literal); ok {
				resVal = append(resVal, val...)
			} else {
				resVal = append(resVal, rawAccessor.(*Literal))
			}

			if exprType == TypeLiteral {
				resVal = append(resVal, rawKey.(*Literal))
			} else {
				resVal = append(resVal, rawKey.([]*Literal)...)
			}

			return DotExpression, resVal, nil
		}

		key, err := HashLiteral(*rawKey.(*Literal))

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while hashing value"), err)
		}

		if has := accessor.Value.(PartsIndexable).HasByKey(key); has {
			rVal := accessor.Value.(PartsIndexable).GetByKey(key)

			if fCall {
				rx, err := vm.handleNestedCall(rVal, argCount)

				if err != nil {
					return UndefinedExpression, nil, errors.Join(errors.New("got error while calling function (B_DOT, B_CALL)"), err)
				}

				if rx == nil {
					vm.LastExpr = nil
					return NoValue, nil, nil
				}

				vm.LastExpr = rx
				return TypeLiteral, rx, nil
			}

			vm.LastExpr = rVal
			return TypeLiteral, rVal, nil
		} else {
			return UndefinedExpression, nil, fmt.Errorf("key not found: %s", key)
		}
	case B_RESOLVE:
		vm.Idx++
		exprType, expr, err := vm.runExpr(unwindDot)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral && exprType != DotExpression {
			return UndefinedExpression, nil, fmt.Errorf("expected value got %d (resolve)", exprType)
		}

		if exprType == DotExpression && unwindDot {
			vm.LastExpr = expr.(*Literal)
			return DotExpression, expr, nil
		}

		resolvedExpr, err := vm.simplifyLiteral(expr.(*Literal), true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		vm.LastExpr = resolvedExpr
		return TypeLiteral, resolvedExpr, nil
	case B_SET:
		vm.Idx++

		exprType, nameLiteral, err := vm.runExpr(false)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral && exprType != DotExpression {
			return UndefinedExpression, nil, errors.New("expected literal as variable name")
		}

		if exprType == DotExpression {
			_, value, err := vm.runExpr(true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running variable value (B_SET)"), err)
			}

			simpleValue, err := vm.simplifyLiteral(value.(*Literal), true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while simplyfing value"), err)
			}

			val, err := vm.Enviroment.assignDot(vm, nameLiteral.([]*Literal), simpleValue)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while assigning to a variable"), err)
			}

			vm.LastExpr = val
			return TypeLiteral, val, nil
		}

		envKey, err := HashLiteral(*nameLiteral.(*Literal))

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while defining name hash"), err)
		}

		exprType, value, err := vm.runExpr(true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running variable value (B_SET)"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected value type got %d (running set)", exprType)
		}

		simpleValue, err := vm.simplifyLiteral(value.(*Literal), true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while simplyfing value"), err)
		}

		_, err = vm.Enviroment.assign(envKey, simpleValue)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while assigning to a variable"), err)
		}

		vm.LastExpr = simpleValue

		return TypeLiteral, simpleValue, nil
	case B_CALL:
		vm.Idx++
		exprType, expr, err := vm.runExpr(true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected value got %d (call)", exprType)
		}

		resolvedExpr, err := vm.simplifyLiteral(expr.(*Literal), true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if resolvedExpr.LiteralType != FunLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected function value got %d (%s)", resolvedExpr.LiteralType, resolvedExpr.pretify())
		}

		values := make([]*Literal, vm.Code[vm.Idx])

		vm.Idx++

		for i := range values {
			exprType, expr, err := vm.runExpr(true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
			}

			if exprType != TypeLiteral {
				return UndefinedExpression, nil, fmt.Errorf("expected value got %d (resolve call arguments)", exprType)
			}

			resolvedExpr, err := vm.simplifyLiteral(expr.(*Literal), true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while simplyfing expression"), err)
			}

			values[i] = resolvedExpr
		}

		funResult, err := vm.callFunction(resolvedExpr.Value.(PartsCallable), values)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while executing function body"), err)
		}

		if funResult == nil {
			return NoValue, nil, nil
		}

		return TypeLiteral, funResult, nil
	case B_COND_JUMP:
		vm.Idx++
		exprType, jumpVal, err := vm.runExpr(true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running jump condition"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected value got %d (jump condition)", exprType)
		}

		lit, err := vm.simplifyLiteral(jumpVal.(*Literal), true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running jump condition"), err)
		}

		if lit.LiteralType != BoolLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected boolean value got %d (jump condition)", lit.LiteralType)
		}

		if lit.Value.(bool) {
			length, err := vm.decodeLen()

			if err != nil {
				return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (then branch)", lit.LiteralType)
			}

			tempVM := vm.newVM(vm.Code[vm.Idx : vm.Idx+length])

			vm.Idx += length

			if err = tempVM.Run(); err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running then branch"), err)
			}

			if tempVM.EarlyExit {
				switch tempVM.ExitCode {
				case BreakCode, ContinueCode:
					vm.ExitCode = tempVM.ExitCode
					vm.EarlyExit = true
					vm.Idx = len(vm.Code)

					return NoValue, nil, nil
				case ReturnCode:
					vm.EarlyExit = true
					vm.ExitCode = ReturnCode
					vm.Idx = len(vm.Code)

					if tempVM.ReturnValue != nil {
						vm.ReturnValue = tempVM.ReturnValue

						return TypeLiteral, tempVM.ReturnValue, nil
					} else {
						return NoValue, nil, nil
					}
				}
			}

			length, err = vm.decodeLen()

			if err != nil {
				return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (else branch)", lit.LiteralType)
			}

			vm.Idx += length
			vm.LastExpr = tempVM.LastExpr

			if tempVM.LastExpr == nil {
				vm.ExitCode = NormalCode
				return NoValue, nil, nil
			} else {
				vm.ExitCode = ReturnCode
				return TypeLiteral, tempVM.LastExpr, nil
			}
		}

		length, err := vm.decodeLen()

		if err != nil {
			return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (then branch)", lit.LiteralType)
		}

		vm.Idx += length

		length, err = vm.decodeLen()

		if err != nil {
			return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (else branch)", lit.LiteralType)
		}

		if length == 0 {
			return NoValue, nil, nil
		}

		tempVM := vm.newVM(vm.Code[vm.Idx : vm.Idx+length])

		vm.Idx += length

		if err := tempVM.Run(); err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running then branch"), err)
		}

		if tempVM.EarlyExit {
			vm.ExitCode = tempVM.ExitCode
			vm.EarlyExit = true
			vm.Idx = len(vm.Code)

			if tempVM.ExitCode == ReturnCode && tempVM.ReturnValue != nil {
				vm.ReturnValue = tempVM.ReturnValue

				return TypeLiteral, tempVM.ReturnValue, nil
			}

			return NoValue, nil, nil
		}

		if tempVM.LastExpr == nil {
			vm.LastExpr = nil
			vm.ExitCode = NormalCode
			return NoValue, nil, nil
		} else {
			vm.LastExpr = tempVM.LastExpr
			vm.ExitCode = ReturnCode
			return TypeLiteral, tempVM.LastExpr, nil
		}
	case B_LOOP:
		vm.Idx++

		condidionLen, err := vm.decodeLen()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while decoding condition length"), err)
		}

		condidion := vm.Code[vm.Idx : vm.Idx+condidionLen]

		vm.Idx += condidionLen

		bodyLen, err := vm.decodeLen()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while decoding body length"), err)
		}

		body := vm.Code[vm.Idx : vm.Idx+bodyLen]

		vm.Idx += bodyLen

		baseVM := vm.copyVM()

		condidionVM := baseVM.newVM(condidion)

		if err = condidionVM.Run(); err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running condidion"), err)
		}

		lastExpr, err := condidionVM.simplifyLiteral(condidionVM.LastExpr, true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running condidion (simplyfing)"), err)
		}

		if lastExpr.LiteralType == BoolLiteral {
			if lastExpr.Value.(bool) {
				for lastExpr.Value.(bool) {
					bodyVM := baseVM.newVM(body)

					err = bodyVM.Run()

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running body"), err)
					}

					if bodyVM.EarlyExit {
						switch bodyVM.ExitCode {
						case BreakCode:
							break
						case ReturnCode:
							vm.ExitCode = ReturnCode
							vm.EarlyExit = true
							vm.Idx = len(vm.Code)

							if bodyVM.ReturnValue != nil {
								vm.ReturnValue = bodyVM.ReturnValue

								return TypeLiteral, bodyVM.ReturnValue, nil
							} else {
								return NoValue, nil, nil
							}
						}

						if bodyVM.ExitCode == BreakCode {
							break
						}
					}

					condidionVM := baseVM.newVM(condidion)

					err = condidionVM.Run()

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running condidion"), err)
					}

					lastExpr, err = condidionVM.simplifyLiteral(condidionVM.LastExpr, true)

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running condidion (simplyfing)"), err)
					}

					if lastExpr.LiteralType != BoolLiteral {
						break
					}

					if bodyVM.EarlyExit && bodyVM.ExitCode == ContinueCode {
						continue
					}
				}
			}
		}

		if lastExpr.LiteralType == ParsedObjLiteral {
			obj := lastExpr

			th := obj.Value.(PartsIndexable).TypeHash()

			if th == "Parts.Iterator" {
				iter := obj.Value.(PartsSpecialObject)

				nextFunc := iter.Get(&Literal{
					LiteralType: RefLiteral,
					Value:       "next",
				})

				res, err := condidionVM.callFunction(nextFunc.Value.(PartsCallable), []*Literal{})

				if err != nil {
					return UndefinedExpression, nil, errors.Join(errors.New("got error while running loop condition iterator"), err)
				}

				for IsOptionSome(res) {
					bodyVM := baseVM.newVM(body)

					bodyVM.Enviroment.Define("it", res.Value.(PartsSpecialObject).GetByKey("RTValue"))

					err = bodyVM.Run()

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running body"), err)
					}

					if bodyVM.EarlyExit {
						switch bodyVM.ExitCode {
						case BreakCode:
							break
						case ReturnCode:
							vm.ExitCode = ReturnCode
							vm.EarlyExit = true
							vm.Idx = len(vm.Code)

							if bodyVM.ReturnValue != nil {
								vm.ReturnValue = bodyVM.ReturnValue

								return TypeLiteral, bodyVM.ReturnValue, nil
							} else {
								return NoValue, nil, nil
							}
						}

						if bodyVM.ExitCode == BreakCode {
							break
						}
					}

					res, err = condidionVM.callFunction(nextFunc.Value.(PartsCallable), []*Literal{})

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running loop condidion iterator"), err)
					}
				}
			} else {
				iter := obj.Value.(PartsIndexable)

				nextFunc := iter.Get(&Literal{
					LiteralType: RefLiteral,
					Value:       "next",
				})

				if nextFunc.LiteralType != FunLiteral {
					return UndefinedExpression, nil, errors.New("expected function type in object field")
				}

				res, err := condidionVM.callFunction(nextFunc.Value.(PartsCallable), []*Literal{})

				for IsOptionSome(res) {
					bodyVM := baseVM.newVM(body)

					bodyVM.Enviroment.Define("it", res.Value.(PartsSpecialObject).GetByKey("RTValue"))

					err = bodyVM.Run()

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running body"), err)
					}

					if bodyVM.EarlyExit {
						switch bodyVM.ExitCode {
						case BreakCode:
							break
						case ReturnCode:
							vm.ExitCode = ReturnCode
							vm.EarlyExit = true
							vm.Idx = len(vm.Code)

							if bodyVM.ReturnValue != nil {
								vm.ReturnValue = bodyVM.ReturnValue

								return TypeLiteral, bodyVM.ReturnValue, nil
							} else {
								return NoValue, nil, nil
							}
						}

						if bodyVM.ExitCode == BreakCode {
							break
						}
					}

					res, err = condidionVM.callFunction(nextFunc.Value.(PartsCallable), []*Literal{})

					if err != nil {
						return UndefinedExpression, nil, errors.Join(errors.New("got error while running loop condidion iterator"), err)
					}
				}
			}
		}

		return NoValue, nil, nil
	case B_CONTINUE, B_BREAK, B_RAISE, B_RETURN:
		code := vm.Code[vm.Idx]

		vm.Idx++
		vm.EarlyExit = true

		switch code {
		case B_RAISE, B_RETURN:
			vm.ExitCode = ReturnCode
		case B_CONTINUE:
			vm.ExitCode = ContinueCode
		case B_BREAK:
			vm.ExitCode = BreakCode
		}

		if code == B_RAISE || code == B_RETURN {
			exprType, val, err := vm.runExpr(true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running variable value (B_RAISE)"), err)
			}

			if exprType != TypeLiteral {
				return UndefinedExpression, nil, errors.New("expected literal as variable name")
			}

			simplifed, err := vm.simplifyLiteral(val.(*Literal), true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while simplyfing raise value"), err)
			}

			if code == B_RAISE {
				if IsResult(simplifed) {
					vm.ReturnValue = simplifed
				} else {
					vm.ReturnValue = NewResultError(simplifed)
				}
			}
		}

		return NoValue, nil, nil
	default:
		return UndefinedExpression, nil, fmt.Errorf("unrecognized bytecode: %d", vm.Code[vm.Idx])
	}
}

func (vm *VM) callFunction(fun PartsCallable, args []*Literal) (*Literal, error) {
	_, val, err := vm.callFunctionVM(fun, args)

	return val, err
}

func (vm *VM) callFunctionVM(fun PartsCallable, args []*Literal) (*VM, *Literal, error) {
	funArgs := fun.GetArguments()

	if len(funArgs) > len(args) {
		return nil, nil, errors.New("got less arguments than expected")
	}

	tempVM := vm.copyVM()

	for idx, key := range funArgs {
		tempVM.Enviroment.define(fmt.Sprintf("RT%s", key), args[idx])
	}

	if err := fun.Call(&tempVM); err != nil {
		return &tempVM, nil, errors.Join(errors.New("got error while running function body"), err)
	}

	if tempVM.EarlyExit {
		if tempVM.ExitCode == ReturnCode {
			if tempVM.ReturnValue != nil {
				return &tempVM, tempVM.ReturnValue, nil
			}

			return &tempVM, nil, nil
		}

		if !slices.Contains([]ExitCode{ReturnCode, NormalCode}, tempVM.ExitCode) {
			return nil, nil, fmt.Errorf("unexpected exit code in function call %d", tempVM.ExitCode)
		}
	}

	if tempVM.LastExpr != nil {
		simplified, err := tempVM.simplifyLiteral(tempVM.LastExpr, true)

		if err != nil {
			return nil, nil, errors.Join(errors.New("got error while simplyfing expression (processing function result)"), err)
		}

		return &tempVM, simplified, nil
	}

	return &tempVM, nil, nil
}

func (vm *VM) runOp() (*Literal, error) {
	opcode := vm.Code[vm.Idx]

	vm.Idx++

	exprType, left, err := vm.runExpr(true)

	if err != nil {
		return nil, errors.Join(errors.New("got error while running (left operand)"), err)
	}

	if exprType != TypeLiteral {
		return nil, fmt.Errorf("expected value got %d (left operand)", exprType)
	}

	simpleLeft, err := vm.simplifyLiteral(left.(*Literal), true)

	if err != nil {
		return nil, errors.Join(errors.New("got error while simplyfing left operand"), err)
	}

	exprType, right, err := vm.runExpr(true)

	if err != nil {
		return nil, errors.Join(errors.New("got error while running (right operand)"), err)
	}

	if exprType != TypeLiteral {
		return nil, fmt.Errorf("expected value got %d (right operand)", exprType)
	}

	simpleRight, err := vm.simplifyLiteral(right.(*Literal), true)

	if err != nil {
		return nil, errors.Join(errors.New("got error while simplyfing right operand"), err)
	}

	switch opcode {
	case B_OP_ADD:
		return simpleLeft.opAdd(simpleRight)
	case B_OP_MIN:
		return simpleLeft.opSub(simpleRight)
	case B_OP_DIV:
		return simpleLeft.opDiv(simpleRight)
	case B_OP_MUL:
		return simpleLeft.opMul(simpleRight)
	case B_OP_EQ:
		return simpleLeft.opEq(simpleRight)
	case B_OP_GT:
		return simpleLeft.opGt(simpleRight)
	case B_OP_LT:
		return simpleLeft.opLt(simpleRight)
	case B_OP_MOD:
		return simpleLeft.opMod(simpleRight)

	default:
		return nil, fmt.Errorf("unrecognized operation: %d", vm.Code[vm.Idx])
	}
}

type ExpressionType = int

const (
	UndefinedExpression ExpressionType = iota
	NoValue
	TypeLiteral
	ScopeChange
	DotExpression
	JumpStmt
)

func (vm *VM) simplifyLiteral(literal *Literal, resolveRef bool) (*Literal, error) {
	if literal.LiteralType == PointerLiteral {
		if val, ok := literal.Value.(Literal); ok {
			return vm.simplifyLiteral(&val, resolveRef)
		} else {
			return literal, nil
		}
	}

	if literal.LiteralType == RefLiteral && resolveRef {
		hash, err := HashLiteral(*literal)

		if err != nil {
			return nil, errors.Join(errors.New("error resolivng reference"), err)
		}

		rVal, err := vm.Enviroment.resolve(hash)

		if err != nil {
			return nil, errors.Join(errors.New("error resolivng reference"), err)
		}

		return vm.simplifyLiteral(rVal, resolveRef)
	}

	if literal.LiteralType == ObjLiteral {
		objectData := PartsObject{Entries: make(map[string]*Literal)}

		for i, entry := range literal.Value.(ObjDefinition).Entries {
			tempVM := vm.newVM(entry)

			_, keyValue, err := tempVM.runExpr(true)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("error resolivng length object key idx: %d", i), err)
			}

			simplifiedKeyValue, err := tempVM.simplifyLiteral(keyValue.(*Literal), false)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("error resolivng simplyfing object key idx: %d", i), err)
			}

			entryKey, err := HashLiteral(*simplifiedKeyValue)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("error resolivng hashing object key idx: %d", i), err)
			}

			_, actualValue, err := tempVM.runExpr(true)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("error resolivng object value idx: %d", i), err)
			}

			simplifiedValue, err := tempVM.simplifyLiteral(actualValue.(*Literal), true)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("error simplyfing object value idx: %d", i), err)
			}

			objectData.Entries[entryKey] = simplifiedValue
		}

		return &Literal{LiteralType: ParsedObjLiteral, Value: &objectData}, nil
	}

	if literal.LiteralType == ListLiteral {
		objectData := PartsObject{Entries: make(map[string]*Literal)}

		for i, entry := range literal.Value.(ListDefinition).Entries {
			entryKey, err := HashLiteral(Literal{
				LiteralType: IntLiteral,
				Value:       i,
			})

			if err != nil {
				return nil, errors.Join(fmt.Errorf("error encoding index, idx: %d", i), err)
			}

			tempVM := vm.newVM(entry)

			exprType, resolvedValue, err := tempVM.runExpr(true)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("got error while parsing array element, idx: %d", i), err)
			}

			if exprType != TypeLiteral {
				return nil, errors.Join(fmt.Errorf("expected value got %d", exprType), err)
			}

			simplifiedValue, err := vm.simplifyLiteral(resolvedValue.(*Literal), true)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("got error while simplyfing array element, idx: %d", i), err)
			}

			objectData.Entries[entryKey] = simplifiedValue
		}

		return &Literal{LiteralType: ParsedListLiteral, Value: &objectData}, nil
	}

	return literal, nil
}

func (vm *VM) decodeLen() (int, error) {
	if vm.Code[vm.Idx] <= 125 {
		value := vm.Code[vm.Idx]
		vm.Idx++
		return int(value), nil
	}

	if vm.Code[vm.Idx] == 126 {
		value := int(vm.Code[vm.Idx+1])<<8 | int(vm.Code[vm.Idx+2])

		vm.Idx += 3

		return value, nil
	}

	if vm.Code[vm.Idx] == 127 {
		value := int(vm.Code[vm.Idx+1])<<56 |
			int(vm.Code[vm.Idx+2])<<48 |
			int(vm.Code[vm.Idx+3])<<40 |
			int(vm.Code[vm.Idx+4])<<32 |
			int(vm.Code[vm.Idx+5])<<24 |
			int(vm.Code[vm.Idx+6])<<16 |
			int(vm.Code[vm.Idx+7])<<8 |
			int(vm.Code[vm.Idx+8])

		vm.Idx += 9

		return value, nil
	}

	return 0, errors.New("something went wrong while deocding length")
}

func (vm *VM) newVM(code []Bytecode) VM {
	v := vm.copyVM()

	v.Code = code

	return v
}

func (vm *VM) copyVM() VM {
	env := VMEnviroment{
		Enclosing: vm.Enviroment,
		Values:    make(map[string]*Literal),
	}

	return VM{
		Enviroment:  &env,
		Idx:         0,
		ReturnValue: nil,
		EarlyExit:   false,
		Code:        []Bytecode{},
		Literals:    vm.Literals,
		Meta:        vm.Meta,
	}
}

type PartsCallable interface {
	Call(vm *VM) error
	GetArguments() []string
}

func (f FunctionDeclaration) Call(vm *VM) error {
	vm.Code = f.Body

	if err := vm.Run(); err != nil {
		return err
	}

	return nil
}

func (f FunctionDeclaration) GetArguments() []string {
	return f.Params
}

type PartsIndexable interface {
	Get(key *Literal) *Literal
	Set(key *Literal, value *Literal) *Literal
	Has(key *Literal) bool

	GetByKey(key string) *Literal
	SetByKey(key string, value *Literal) *Literal
	HasByKey(key string) bool

	GetAll() map[string]*Literal
	Length() int
	TypeHash() string
}

type PartsObject struct {
	Entries map[string]*Literal
}

func (o *PartsObject) Get(key *Literal) *Literal {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return o.GetByKey(hash)
}

func (o *PartsObject) Set(key *Literal, value *Literal) *Literal {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return o.SetByKey(hash, value)
}

func (o *PartsObject) Has(key *Literal) bool {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return o.HasByKey(hash)
}

func (o *PartsObject) Length() int {
	return len(o.Entries)
}

func (o *PartsObject) GetAll() map[string]*Literal {
	return o.Entries
}

func (o *PartsObject) GetByKey(key string) *Literal {
	return o.Entries[key]
}

func (o *PartsObject) SetByKey(key string, value *Literal) *Literal {
	o.Entries[key] = value

	return value
}

func (o *PartsObject) HasByKey(key string) bool {
	_, has := o.Entries[key]
	return has
}

func (o *PartsObject) TypeHash() string {
	return ""
}
