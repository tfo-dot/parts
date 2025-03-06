package parts

import (
	"errors"
	"fmt"
)

type VM struct {
	Enviroment *VMEnviroment

	Idx int

	//Filled from parser

	Code     []Bytecode
	Literals []Literal
	Meta     map[string]string
}

func (vm *VM) Run() error {

	for i, l := range vm.Literals {
		println(i, l.LiteralType, l.Value)
	}

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

		exprType, nameLiteral, err := vm.runExpr()

		if err != nil {
			return errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral {
			return errors.New("expected literal as variable name")
		}

		envKey, err := HashLiteral(nameLiteral.(Literal))

		if err != nil {
			return errors.Join(errors.New("got error while defining name hash"), err)
		}

		exprType, value, err := vm.runExpr()

		if err != nil {
			return errors.Join(errors.New("got error while running variable value"), err)
		}

		simpleValue, err := vm.simplifyLiteral(value.(Literal), true)

		if err != nil {
			return errors.Join(errors.New("got error while simplyfing value"), err)
		}

		_, err = vm.Enviroment.define(envKey, simpleValue)

		if err != nil {
			return errors.Join(errors.New("got error while defining variable"), err)
		}
	case B_NEW_SCOPE:
		vm.Idx++

		newEnv := VMEnviroment{
			Values:    make(map[string]Literal),
			Enclosing: vm.Enviroment,
		}

		vm.Enviroment = &newEnv
	case B_END_SCOPE:
		vm.Idx++

		if vm.Enviroment.Enclosing == nil {
			return errors.New("leaving scope but already at top level")
		}

		oldEnv := vm.Enviroment.Enclosing

		vm.Enviroment = oldEnv
	default:
		if vm.Idx >= len(vm.Code) {
			return errors.New("tried running bytecode after the end")
		}

		if _, _, err := vm.runExpr(); err != nil {
			return errors.Join(errors.New("got error while running bytecode"), err)
		}
	}

	return nil
}

func (vm *VM) runExpr() (ExpressionType, any, error) {
	switch vm.Code[vm.Idx] {
	case B_NEW_SCOPE:
		vm.Idx++

		newEnv := VMEnviroment{
			Values:    make(map[string]Literal),
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
	case B_LITERAL:
		vm.Idx++

		nameIdx, err := vm.decodeLen()

		if err != nil {
			return TypeLiteral, nil, errors.Join(errors.New("got error while decoding offset"), err)
		}

		return TypeLiteral, vm.Literals[nameIdx], nil
	case B_DOT:
		vm.Idx++
		exprType, rawAccessor, err := vm.runExpr()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, errors.Join(fmt.Errorf("expected value got %d", exprType), err)
		}

		accessor := rawAccessor.(Literal)

		if accessor.LiteralType == RefLiteral {
			valueHash, err := HashLiteral(accessor)

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
			return UndefinedExpression, nil, errors.New("unexpected value type")
		}

		if accessor.LiteralType == ListLiteral || accessor.LiteralType == ObjLiteral {
			accessor, err = vm.simplifyLiteral(accessor, true)

			if err != nil {
				return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
			}
		}

		exprType, rawKey, err := vm.runExpr()

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while running expression"), err)
		}

		if exprType != TypeLiteral {
			return UndefinedExpression, nil, errors.Join(fmt.Errorf("expected value got %d", exprType), err)
		}

		key, err := HashLiteral(rawKey.(Literal))

		if err != nil {
			return UndefinedExpression, nil, errors.Join(errors.New("got error while hashing key"), err)
		}

		if rVal, has := accessor.Value.(PartsObject).Entries[key]; has {
			return TypeLiteral, rVal, nil
		} else {
			return UndefinedExpression, nil, fmt.Errorf("key not found: %s", key)
		}
	}

	return UndefinedExpression, nil, fmt.Errorf("unrecognized bytecode: %d", vm.Code[vm.Idx])
}

type ExpressionType = int

const (
	UndefinedExpression ExpressionType = iota
	TypeLiteral
	ScopeChange
)

func (vm *VM) simplifyLiteral(literal Literal, resolveRef bool) (Literal, error) {
	if literal.LiteralType == RefLiteral && resolveRef {

		if literal.Value.(ReferenceDeclaration).Dynamic {
			return literal, nil
		}

		rVal, rErr := vm.Enviroment.resolve(fmt.Sprintf("RT%s", literal.Value.(ReferenceDeclaration).Reference))

		if rErr != nil {
			return Literal{}, errors.Join(errors.New("error resolivng reference"), rErr)
		}

		return rVal, nil
	}

	if literal.LiteralType == ObjLiteral {
		objectData := PartsObject{Entries: make(map[string]Literal)}

		for i, entry := range literal.Value.(ObjDefinition).Entries {
			//TODO fix this

			oldCode := vm.Code
			oldIndex := vm.Idx

			vm.Code = entry
			vm.Idx = 0

			_, keyValue, err := vm.runExpr()

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng length object key idx: %d", i), err)
			}

			simplifiedKeyValue, err := vm.simplifyLiteral(keyValue.(Literal), false)

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng simplyfing object key idx: %d", i), err)
			}

			entryKey, err := HashLiteral(simplifiedKeyValue)

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng hashing object key idx: %d", i), err)
			}

			_, actualValue, err := vm.runExpr()

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng object value idx: %d", i), err)
			}

			simplifiedValue, err := vm.simplifyLiteral(actualValue.(Literal), true)

			objectData.Entries[entryKey] = simplifiedValue

			vm.Code = oldCode
			vm.Idx = oldIndex
		}

		return Literal{LiteralType: ParsedObjLiteral, Value: objectData}, nil
	}

	if literal.LiteralType == ListLiteral {
		objectData := PartsObject{Entries: make(map[string]Literal)}

		for i, entry := range literal.Value.(ListDefinition).Entries {

			entryKey, err := HashLiteral(Literal{
				LiteralType: IntLiteral,
				Value:       i,
			})

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error encoding index, idx: %d", i), err)
			}

			//TODO fix this

			oldCode := vm.Code
			oldIndex := vm.Idx

			vm.Code = entry
			vm.Idx = 0

			_, resolvedValue, err := vm.runExpr()

			vm.Code = oldCode
			vm.Idx = oldIndex

			if err != nil {
				return Literal{}, errors.Join(errors.New("got error while parsing array element"), err)
			}

			simplifiedValue, err := vm.simplifyLiteral(resolvedValue.(Literal), true)

			objectData.Entries[entryKey] = simplifiedValue
		}

		return Literal{LiteralType: ParsedListLiteral, Value: objectData}, nil
	}

	return literal, nil
}

type PartsObject struct {
	Entries map[string]Literal
}

type VMEnviroment struct {
	Enclosing *VMEnviroment

	Values map[string]Literal
}

func (env *VMEnviroment) define(key string, value Literal) (Literal, error) {
	_, exists := env.Values[key]

	if exists {
		return Literal{}, errors.New("redefining variable in the same scope")
	}

	env.Values[key] = value

	return value, nil
}

func (env *VMEnviroment) resolve(key string) (Literal, error) {
	_, exists := env.Values[key]

	if exists {
		return env.Values[key], nil
	}

	if env.Enclosing != nil {
		return env.Enclosing.resolve(key)
	}

	return Literal{}, errors.New("undefined variable resolve")
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

func decodeLen(code []Bytecode, idx int) (int, int, error) {
	if code[idx] <= 125 {
		value := code[idx]
		return int(value), 1, nil
	}

	if code[idx] == 126 {
		value := int(code[idx+1])<<8 | int(code[idx+2])

		return value, 3, nil
	}

	if code[idx] == 127 {
		value := int(code[idx+1])<<56 | int(code[idx+2])<<48 | int(code[idx+3])<<40 | int(code[idx+4])<<32 | int(code[idx+5])<<24 | int(code[idx+6])<<16 | int(code[idx+7])<<8 | int(code[idx+8])

		return value, 9, nil
	}

	return 0, 0, errors.New("something went wrong while deocding length")
}

func HashLiteral(literal Literal) (string, error) {
	switch literal.LiteralType {
	case IntLiteral:
		return fmt.Sprintf("IT%d", literal.Value.(int)), nil
	case DoubleLiteral:
		return fmt.Sprintf("DT%f", literal.Value.(float64)), nil
	case BoolLiteral:
		if literal.Value.(bool) {
			return "BT1", nil
		} else {
			return "BT0", nil
		}
	case StringLiteral:
		return fmt.Sprintf("ST%s", literal.Value.(string)), nil
	case RefLiteral:
		return fmt.Sprintf("RT%s", literal.Value.(ReferenceDeclaration).Reference), nil
	}

	return "", errors.New("literal not hashable")
}
