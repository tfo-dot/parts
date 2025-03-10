package parts

import (
	"errors"
	"fmt"
)

type VMEnviroment struct {
	Enclosing *VMEnviroment

	Values map[string]*Literal
}

func (env *VMEnviroment) define(key string, value *Literal) (*Literal, error) {
	_, exists := env.Values[key]

	if exists {
		return nil, errors.New("redefining variable in the same scope")
	}

	env.Values[key] = value

	return value, nil
}

func (env *VMEnviroment) resolve(key string) (*Literal, error) {
	if value, exists := env.Values[key]; exists {
		return value, nil
	}

	if env.Enclosing != nil {
		return env.Enclosing.resolve(key)
	}

	return nil, errors.New("undefined variable resolve")
}

func (env *VMEnviroment) assign(key string, value *Literal) (*Literal, error) {
	_, exists := env.Values[key]

	if !exists {
		if env.Enclosing == nil {
			return nil, errors.New("setting to a variable that doesn't exist")
		} else {
			return env.Enclosing.assign(key, value)
		}
	}

	env.Values[key] = value

	return value, nil
}

func (env *VMEnviroment) assignDot(vm *VM, key []*Literal, value *Literal) (*Literal, error) {
	hash, err := HashLiteral(*key[0])

	if err != nil {
		return nil, errors.Join(errors.New("got errror while unwinding set expr"), err)
	}

	accessor, exists := env.Values[hash]

	if !exists {
		if env.Enclosing == nil {
			return nil, errors.New("setting to a variable that doesn't exist (dot operation)")
		} else {
			return env.Enclosing.assignDot(vm, key, value)
		}
	}

	for i := 1; i < len(key); i++ {
		switch accessor.LiteralType {
		case ListLiteral, ObjLiteral, ParsedListLiteral, ParsedObjLiteral:
		default:
			return nil, fmt.Errorf("unexpected value type at %d", i)
		}

		if accessor.LiteralType == ListLiteral || accessor.LiteralType == ObjLiteral {
			if accessor, err = vm.simplifyLiteral(accessor, true); err != nil {
				return nil, errors.Join(errors.New("got error while running expression"), err)
			}
		}

		localKey := key[i]

		keyHash, err := HashLiteral(*localKey)

		if err != nil {
			return nil, errors.Join(errors.New("got error while hashing key"), err)
		}

		if _, has := accessor.Value.(PartsObject).Entries[keyHash]; has {
			accessor.Value.(PartsObject).Entries[keyHash] = value
		} else {
			return nil, fmt.Errorf("key not found: %s", keyHash)
		}
	}

	return value, nil
}
