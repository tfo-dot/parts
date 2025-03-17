package parts

import (
	"errors"
	"fmt"
)

type VM struct {
	Enviroment *VMEnviroment

	Idx int

	ReturnValue *Literal

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
			return errors.Join(errors.New("got error while running variable value"), err)
		}

		if exprType == NoValue {
			return fmt.Errorf("got no value, expected value at '%s'", nameLiteral.(*Literal).Value.(ReferenceDeclaration).Reference)
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

		oldEnv := vm.Enviroment.Enclosing

		vm.Enviroment = oldEnv

		return ScopeChange, nil, nil
	case B_LITERAL:
		vm.Idx++

		nameIdx, err := vm.decodeLen()

		if err != nil {
			return TypeLiteral, nil, errors.Join(errors.New("got error while decoding offset"), err)
		}

		return TypeLiteral, vm.Literals[nameIdx], nil
	case B_DOT:
		vm.Idx++

		exprType, rawAccessor, err := vm.runExpr(unwindDot)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral && exprType != DotExpression {
			return UndefinedExpression, nil, fmt.Errorf("expected value got %d (running dot accessor)", exprType)
		}

		accessor := rawAccessor.(*Literal)

		if accessor.LiteralType == RefLiteral {
			valueHash, err := HashLiteral(*accessor)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while resolving reference"), err)
			}

			accessor, err = vm.Enviroment.resolve(valueHash)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while resolving reference"), err)
			}
		}

		switch accessor.LiteralType {
		case ListLiteral, ObjLiteral, ParsedListLiteral, ParsedObjLiteral:
		default:
			return UndefinedExpression, nil, fmt.Errorf("unexpected value type (%d)", accessor.LiteralType)
		}

		if accessor.LiteralType == ListLiteral || accessor.LiteralType == ObjLiteral {
			if accessor, err = vm.simplifyLiteral(accessor, true); err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
			}
		}

		exprType, rawKey, err := vm.runExpr(unwindDot)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral && exprType != DotExpression {
			return UndefinedExpression, nil, errors.Join(fmt.Errorf("expected value got %d (dot value)", exprType), err)
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

		if has := accessor.Value.(PartsIndexable).HasByKey(key); has {
			return TypeLiteral, accessor.Value.(PartsIndexable).GetByKey(key), nil
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
			return DotExpression, expr, nil
		}

		resolvedExpr, err := vm.simplifyLiteral(expr.(*Literal), true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

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
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running variable value"), err)
			}

			simpleValue, err := vm.simplifyLiteral(value.(*Literal), true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while simplyfing value"), err)
			}

			val, err := vm.Enviroment.assignDot(vm, nameLiteral.([]*Literal), simpleValue)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while assigning to a variable"), err)
			}

			return TypeLiteral, val, nil
		}

		envKey, err := HashLiteral(*nameLiteral.(*Literal))

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while defining name hash"), err)
		}

		exprType, value, err := vm.runExpr(true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running variable value"), err)
		}

		simpleValue, err := vm.simplifyLiteral(value.(*Literal), true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while simplyfing value"), err)
		}

		_, err = vm.Enviroment.assign(envKey, simpleValue)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while assigning to a variable"), err)
		}

		return TypeLiteral, simpleValue, nil
	case B_RETURN:
		vm.Idx++

		exprType, val, err := vm.runExpr(true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running variable value"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, errors.New("expected literal as variable name")
		}

		vm.ReturnValue = val.(*Literal)

		return NoValue, nil, err
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
			return UndefinedExpression, nil, fmt.Errorf("expected function value got %d", resolvedExpr.LiteralType)
		}

		newEnv := VMEnviroment{
			Values:    make(map[string]*Literal),
			Enclosing: vm.Enviroment,
		}

		vm.Enviroment = &newEnv

		funcObj := resolvedExpr.Value.(PartsCallable)

		values := make([]*Literal, vm.Code[vm.Idx])

		vm.Idx++

		for i := 0; i < len(values); i++ {
			exprType, expr, err := vm.runExpr(true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
			}

			if exprType != TypeLiteral {
				return UndefinedExpression, nil, fmt.Errorf("expected value got %d (resolve call arguments)", exprType)
			}

			resolvedExpr, err := vm.simplifyLiteral(expr.(*Literal), true)

			values[i] = resolvedExpr
		}

		tempVM := vm.copyVM()

		for idx, param := range funcObj.GetArguments() {
			vm.Enviroment.define(fmt.Sprintf("RT%s", string(param)), values[idx])
		}

		funcObj.Call(&tempVM)

		if err = tempVM.Run(); err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running function body"), err)
		}

		if tempVM.ReturnValue != nil {
			return TypeLiteral, tempVM.ReturnValue, nil
		} else {
			return NoValue, nil, nil
		}
	case B_COND_JUMP:
		vm.Idx++
		exprType, jumpVal, err := vm.runExpr(true)

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running jump condition"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected value got %d (resolve jump condition)", exprType)
		}

		lit := jumpVal.(*Literal)

		if lit.LiteralType != BoolLiteral {
			return UndefinedExpression, nil, fmt.Errorf("expected boolean value got %d (resolve jump condition)", lit.LiteralType)
		}

		conditionTrue := lit.Value.(bool)

		if conditionTrue {
			len, err := vm.decodeLen()

			if err != nil {
				return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (then branch)", lit.LiteralType)
			}

			tempVM := vm.newVM(vm.Code[vm.Idx : vm.Idx+len])

			vm.Idx += len

			err = tempVM.Run()

			if err = tempVM.Run(); err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running then branch"), err)
			}

			len, err = vm.decodeLen()

			if err != nil {
				return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (else branch)", lit.LiteralType)
			}

			vm.Idx += len + 1

			if tempVM.ReturnValue == nil {
				return NoValue, nil, nil
			} else {
				return TypeLiteral, tempVM.ReturnValue, nil
			}
		} else {
			len, err := vm.decodeLen()

			if err != nil {
				return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (then branch)", lit.LiteralType)
			}

			vm.Idx += len

			len, err = vm.decodeLen()

			if err != nil {
				return UndefinedExpression, nil, fmt.Errorf("got error while decoding length %d (else branch)", lit.LiteralType)
			}

			if len == 0 {
				//No else branch just exit
				return NoValue, nil, nil
			} else {
				tempVM := vm.newVM(vm.Code[vm.Idx : vm.Idx+len])

				vm.Idx += len

				if err := tempVM.Run(); err != nil {
					return UndefinedExpression, nil, errors.Join(errors.New("got error while running then branch"), err)
				}

				if tempVM.ReturnValue == nil {
					return NoValue, nil, nil
				} else {
					return TypeLiteral, tempVM.ReturnValue, nil
				}
			}
		}
	case B_OP_ADD, B_OP_MIN, B_OP_MUL, B_OP_DIV, B_OP_EQ:
		rVal, err := vm.runOp()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running operation"), err)
		}

		return TypeLiteral, rVal, nil
	default:
		return UndefinedExpression, nil, fmt.Errorf("unrecognized bytecode: %d", vm.Code[vm.Idx])
	}
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
		return nil, errors.Join(errors.New("got error while running (left operand)"), err)
	}

	if exprType != TypeLiteral {
		return nil, fmt.Errorf("expected value got %d (left operand)", exprType)
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
)

func (vm *VM) simplifyLiteral(literal *Literal, resolveRef bool) (*Literal, error) {
	if literal.LiteralType == RefLiteral && resolveRef {
		if literal.Value.(ReferenceDeclaration).Dynamic {
			return literal, nil
		}

		rVal, rErr := vm.Enviroment.resolve(fmt.Sprintf("RT%s", literal.Value.(ReferenceDeclaration).Reference))

		if rErr != nil {
			return nil, errors.Join(errors.New("error resolivng reference"), rErr)
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

			_, resolvedValue, err := tempVM.runExpr(true)

			if err != nil {
				return nil, errors.Join(errors.New("got error while parsing array element"), err)
			}

			simplifiedValue, err := vm.simplifyLiteral(resolvedValue.(*Literal), true)

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
		value := int(vm.Code[vm.Idx+1])<<56 | int(vm.Code[vm.Idx+2])<<48 | int(vm.Code[vm.Idx+3])<<40 | int(vm.Code[vm.Idx+4])<<32 | int(vm.Code[vm.Idx+5])<<24 | int(vm.Code[vm.Idx+6])<<16 | int(vm.Code[vm.Idx+7])<<8 | int(vm.Code[vm.Idx+8])

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
		Code:        []Bytecode{},
		Literals:    vm.Literals,
		Meta:        vm.Meta,
	}
}

type PartsCallable interface {
	Call(vm *VM)
	GetArguments() []string
}

func (f FunctionDeclaration) Call(vm *VM) {
	vm.Code = f.Body

	if err := vm.Run(); err != nil {
		panic(err)
	}
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