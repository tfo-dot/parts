package parts

import (
	"errors"
	"fmt"
)

type VMEnviroment struct {
	Enclosing *VMEnviroment

	Values map[string]*Literal
}

func (env *VMEnviroment) Define(key string, value *Literal) error {
	key = fmt.Sprintf("RT%s", key)

	_, exists := env.Values[key]

	if exists {
		return fmt.Errorf("redefining variable in the same scope ('%s')", key)
	}

	env.Values[key] = value

	return nil
}

func (env *VMEnviroment) Resolve(key string) (*Literal, error) {
	if value, exists := env.Values[fmt.Sprintf("RT%s", key)]; exists {
		return value, nil
	}

	if env.Enclosing != nil {
		return env.Enclosing.Resolve(key)
	}

	return nil, fmt.Errorf("undefined variable resolve '%s'", key)
}

func (env *VMEnviroment) DefineFunction(key string, val any) error {
	return env.Define(key, &Literal{FunLiteral, FFIFunction{val}})
}

func (env *VMEnviroment) AppendValues(values map[string]any) error {
	for key, rawVal := range values {
		val, err := LiteralFromGo(rawVal)

		if err != nil {
			return errors.Join(fmt.Errorf("error converting value to parts at %s", key), err)
		}

		env.Define(key, val)
	}

	return nil
}

func (env *VMEnviroment) define(key string, value *Literal) (*Literal, error) {
	_, exists := env.Values[key]

	if exists {
		return nil, fmt.Errorf("redefining variable in the same scope ('%s')", key)
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

	return nil, fmt.Errorf("undefined variable resolve '%s'", key)
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

		if has := accessor.Value.(PartsIndexable).HasByKey(keyHash); has {
			return accessor.Value.(PartsIndexable).SetByKey(keyHash, value), nil
		} else {
			return nil, fmt.Errorf("key not found: %s", keyHash)
		}
	}

	return value, nil
}

func (env *VMEnviroment) Has(key string) bool {
	_, exists := env.Values[key]

	if !exists && env.Enclosing != nil {
		return env.Enclosing.Has(key)
	}

	return exists
}

func (env *VMEnviroment) Append(other *VMEnviroment) {
	deepestEnv := other
	for deepestEnv.Enclosing != nil {
		deepestEnv = deepestEnv.Enclosing
	}

	deepestEnv.Enclosing = env.Enclosing
	env.Enclosing = other
}
