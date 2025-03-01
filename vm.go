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

			envKey, err := vm.HashLiteral(vm.Literals[nameIdx])

			println(envKey)

			if err != nil {
				return errors.Join(errors.New("got error while defining name hash"), err)
			}

			valueIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			_, err = vm.Enviroment.define(envKey, vm.Literals[valueIdx])

			if err != nil {
				return errors.Join(errors.New("got error while defining variable"), err)
			}
		case DECLARE_FUN:
			vm.Idx++

			nameIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			envKey, err := vm.HashLiteral(vm.Literals[nameIdx])

			if err != nil {
				return errors.Join(errors.New("got error while defining name hash"), err)
			}

			valueIdx, err := vm.decodeLen()

			if err != nil {
				return errors.Join(errors.New("got error while decoding offset"), err)
			}

			_, err = vm.Enviroment.define(envKey, vm.Literals[valueIdx])

			if err != nil {
				return errors.Join(errors.New("got error while defining variable"), err)
			}
		}
	case B_NEW_SCOPE:
		newEnv := VMEnviroment{
			Values: make(map[string]any),
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

func (vm *VM) simplifyLiteral(literal Literal) (any, error) {
	if literal.LiteralType == RefLiteral {
		rVal, rErr := vm.Enviroment.resolve(string(literal.Value.([]rune)))

		if rErr != nil {
			return nil, errors.Join(errors.New("error resolivng reference"), rErr)
		}

		return rVal, nil
	}

	if literal.LiteralType == ObjLiteral {
		
	}

	if literal.LiteralType == ListLiteral {

	}

	return literal, nil
}

type VMEnviroment struct {
	Enclosing *VMEnviroment

	Values map[string]any
}

func (env *VMEnviroment) define(key string, value any) (any, error) {
	_, exists := env.Values[key]

	if exists {
		return nil, errors.New("redefining variable in the same scope")
	}

	env.Values[key] = value

	return value, nil
}

func (env *VMEnviroment) resolve(key string) (any, error) {
	_, exists := env.Values[key]

	if exists {
		return env.Values[key], nil
	}

	if env.Enclosing != nil {
		return env.Enclosing.resolve(key)
	}

	return nil, errors.New("undefined variable resolve")
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

func (vm *VM) HashLiteral(literal Literal) (string, error) {
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
		return fmt.Sprintf("RT%s", string(literal.Value.([]rune))), nil
	}

	return "", errors.New("literal not hashable")
}

type WrappingValue[T any] struct {
	Value T
}
