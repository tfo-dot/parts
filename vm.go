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

		//variable type
		switch vm.Code[vm.Idx] {
		case DECLARE_LET:
			vm.Idx++

			nameIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			envKey, err := HashLiteral(vm.Literals[nameIdx])

			if err != nil {
				return errors.Join(errors.New("got error while defining name hash"), err)
			}

			valueIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			simpleLiteral, err := vm.simplifyLiteral(vm.Literals[valueIdx], true)

			if err != nil {
				return errors.Join(errors.New("got error while simplyfing value"), err)
			}

			_, err = vm.Enviroment.define(envKey, simpleLiteral)

			if err != nil {
				return errors.Join(errors.New("got error while defining variable"), err)
			}
		case DECLARE_FUN:
			vm.Idx++

			nameIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			envKey, err := HashLiteral(vm.Literals[nameIdx])

			if err != nil {
				return errors.Join(errors.New("got error while defining name hash"), err)
			}

			valueIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			simpleLiteral, err := vm.simplifyLiteral(vm.Literals[valueIdx], true)

			if err != nil {
				return errors.Join(errors.New("got error while simplyfing value"), err)
			}

			_, err = vm.Enviroment.define(envKey, simpleLiteral)

			if err != nil {
				return errors.Join(errors.New("got error while defining variable"), err)
			}
		}
	case B_NEW_SCOPE:
		newEnv := VMEnviroment{
			Values:    make(map[string]Literal),
			Enclosing: vm.Enviroment,
		}

		vm.Enviroment = &newEnv
	case B_END_SCOPE:

		if vm.Enviroment.Enclosing == nil {
			return errors.New("leaving scope but already at top level")
		}

		oldEnv := vm.Enviroment.Enclosing

		vm.Enviroment = oldEnv
	}

	return nil
}

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
			keyIdx, offset, keyErr := decodeLen(entry, 0)

			if keyErr != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng length object key idx: %d", i), keyErr)
			}

			simplifiedKeyValue, err := vm.simplifyLiteral(vm.Literals[keyIdx], false)

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng simplyfing object key idx: %d", i), err)
			}

			entryKey, err := HashLiteral(simplifiedKeyValue)

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng hashing object key idx: %d", i), err)
			}
			
			valueIdx, _, keyErr := decodeLen(entry, offset)

			if keyErr != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng length object value idx: %d", i), keyErr)
			}

			simplifiedValue, err := vm.simplifyLiteral(vm.Literals[valueIdx], true)

			objectData.Entries[entryKey] = simplifiedValue
		}

		return Literal{LiteralType: ParsedObjLiteral, Value: objectData}, nil
	}

	if literal.LiteralType == ListLiteral {
		objectData := PartsObject{Entries: make(map[string]Literal)}

		for i, entry := range literal.Value.(ListDefinition).Entries {

			entryKey, err :=  HashLiteral(Literal{
				LiteralType: IntLiteral,
				Value: i,
			})

			if err != nil {
				return Literal{}, errors.Join(fmt.Errorf("error encoding index, idx: %d", i), err)
			}
					
			valueIdx, _, keyErr := decodeLen(entry, 0)

			if keyErr != nil {
				return Literal{}, errors.Join(fmt.Errorf("error resolivng list value idx: %d", i), keyErr)
			}

			simplifiedValue, err := vm.simplifyLiteral(vm.Literals[valueIdx], true)

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