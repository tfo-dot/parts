package parts

import (
	"errors"
	"fmt"
)

type LiteralType int

const (
	IntLiteral LiteralType = iota
	DoubleLiteral
	BoolLiteral
	StringLiteral
	RefLiteral
	FunLiteral
	ObjLiteral
	ParsedObjLiteral
	ListLiteral
	ParsedListLiteral
)

type Literal struct {
	LiteralType LiteralType
	Value       any
}

var InitialLiterals = []Literal{
	{BoolLiteral, false},
	{BoolLiteral, true},
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

func (l *Literal) ToGoTypes(vm *VM) (any, error) {
	switch l.LiteralType {
	case IntLiteral:
		return l.Value.(int), nil
	case DoubleLiteral:
		return l.Value.(float32), nil
	case BoolLiteral:
		return l.Value.(bool), nil
	case StringLiteral:
		return l.Value.(string), nil
	case RefLiteral:
		literalHash, err := HashLiteral(*l)

		if err != nil {
			return nil, errors.Join(fmt.Errorf("got error while resolving variable, %s", l.Value), err)
		}

		value, err := vm.Enviroment.resolve(literalHash)

		if err != nil {
			return nil, errors.Join(fmt.Errorf("got error while resolving variable, %s", l.Value), err)
		}

		return value.ToGoTypes(vm)
	case FunLiteral:
		//TODO add functions
		return nil, errors.New("functions are not supported (yet)")
	case ObjLiteral:
		return nil, errors.New("use simplyfied ParsedObjLiteral")
	case ParsedObjLiteral:
		entryMap := make(map[string]any)

		for key, entry := range l.Value.(PartsObject).Entries {
			val, err := entry.ToGoTypes(vm)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("got error while resolving value, %s", l.Value), err)
			}

			entryMap[key] = val
		}

		return entryMap, nil
	case ListLiteral:
		return nil, errors.New("use simplyfied ParsedListLiteral")
	case ParsedListLiteral:
		entriesList := make([]any, len(l.Value.(PartsObject).Entries))

		for i := 0; i < len(l.Value.(PartsObject).Entries); i++ {

			entry := l.Value.(PartsObject).Entries[fmt.Sprintf("IT%d", i)]

			val, err := entry.ToGoTypes(vm)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("got error while resolving value, %s", l.Value), err)
			}

			entriesList[i] = val
		}

		return entriesList, nil
	}

	return nil, errors.New("invalid type to convert")
}
