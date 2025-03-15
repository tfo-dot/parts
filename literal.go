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

func (l *Literal) opAdd(other *Literal) (*Literal, error) {
	if l.LiteralType == RefLiteral || other.LiteralType == RefLiteral {
		return nil, errors.New("got reference, expected value")
	}

	switch l.LiteralType {
	case IntLiteral, DoubleLiteral:
		switch other.LiteralType {
		case IntLiteral:

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) + other.Value.(int)}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) + float64(other.Value.(int))}, nil
			}

		case DoubleLiteral:

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) + int(other.Value.(float64))}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) + other.Value.(float64)}, nil
			}

		case BoolLiteral:
			val := 0

			if other.Value.(bool) {
				val = 1
			}

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) + val}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) + float64(val)}, nil
			}
		case StringLiteral, FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral:
			return nil, fmt.Errorf("operation not supported - add (number, %d)", other.LiteralType)
		}
	case BoolLiteral:
		switch other.LiteralType {
		case IntLiteral, DoubleLiteral:
			return other.opAdd(l)
		case BoolLiteral:
			lhs, _ := l.Value.(bool)
			rhs, _ := other.Value.(bool)

			return &Literal{BoolLiteral, lhs || rhs}, nil
		case StringLiteral, FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral:
			return nil, fmt.Errorf("operation not supported - add (bool, %d)", other.LiteralType)
		}
	case StringLiteral:
		switch other.LiteralType {
		case IntLiteral:
			return &Literal{StringLiteral, fmt.Sprintf("%s%d", l.Value.(string), other.Value.(int))}, nil
		case DoubleLiteral:
			return &Literal{StringLiteral, fmt.Sprintf("%s%f", l.Value.(string), other.Value.(float64))}, nil
		case BoolLiteral:
			return &Literal{StringLiteral, fmt.Sprintf("%s%t", l.Value.(string), other.Value.(bool))}, nil
		case StringLiteral:
			return &Literal{StringLiteral, fmt.Sprintf("%s%s", l.Value.(string), other.Value.(string))}, nil
		case FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral:
			return nil, fmt.Errorf("operation not supported - add (string, %d)", other.LiteralType)
		}
	case ListLiteral:
		idx := len(l.Value.(PartsObject).Entries)

		l.Value.(PartsObject).Entries[fmt.Sprintf("IT%d", idx)] = other

		return l, nil
	case FunLiteral, ObjLiteral, ParsedObjLiteral:
		return nil, fmt.Errorf("operation not supported - add (fun|obj, %d)", other.LiteralType)
	}

	return nil, fmt.Errorf("unexpected value type %d + %d", l.LiteralType, other.LiteralType)
}

func (l *Literal) opSub(other *Literal) (*Literal, error) {
	if l.LiteralType == RefLiteral || other.LiteralType == RefLiteral {
		return nil, errors.New("got reference, expected value")
	}

	switch l.LiteralType {
	case IntLiteral, DoubleLiteral:
		switch other.LiteralType {
		case IntLiteral:

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) - other.Value.(int)}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) - float64(other.Value.(int))}, nil
			}

		case DoubleLiteral:

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) - int(other.Value.(float64))}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) - other.Value.(float64)}, nil
			}

		case BoolLiteral:
			val := 0

			if other.Value.(bool) {
				val = 1
			}

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) - val}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) - float64(val)}, nil
			}
		case StringLiteral, FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral:
			return nil, fmt.Errorf("operation not supported - subtract (number, %d)", other.LiteralType)
		}
	case FunLiteral, ObjLiteral, ParsedObjLiteral, StringLiteral, ListLiteral, BoolLiteral:
		return nil, fmt.Errorf("operation not supported - subtract (fun|obj|string|list|bool, %d)", other.LiteralType)
	}

	return nil, fmt.Errorf("unexpected value type %d - %d", l.LiteralType, other.LiteralType)
}

func (l *Literal) opMul(other *Literal) (*Literal, error) {
	if l.LiteralType == RefLiteral || other.LiteralType == RefLiteral {
		return nil, errors.New("got reference, expected value")
	}

	switch l.LiteralType {
	case IntLiteral, DoubleLiteral:
		switch other.LiteralType {
		case IntLiteral:

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) * other.Value.(int)}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) * float64(other.Value.(int))}, nil
			}

		case DoubleLiteral:

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) * int(other.Value.(float64))}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) * other.Value.(float64)}, nil
			}

		case BoolLiteral:
			val := 0

			if other.Value.(bool) {
				val = 1
			}

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) * val}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) * float64(val)}, nil
			}
		case StringLiteral, FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral:
			return nil, fmt.Errorf("operation not supported - times (number, %d)", other.LiteralType)
		}
	case BoolLiteral:
		switch other.LiteralType {
		case IntLiteral, DoubleLiteral:
			return other.opAdd(l)
		case BoolLiteral:
			lhs, _ := l.Value.(bool)
			rhs, _ := other.Value.(bool)

			return &Literal{BoolLiteral, lhs && rhs}, nil
		case StringLiteral, FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral:
			return nil, fmt.Errorf("operation not supported - times (bool, %d)", other.LiteralType)
		}
	case FunLiteral, ObjLiteral, ParsedObjLiteral, ListLiteral, StringLiteral:
		return nil, fmt.Errorf("operation not supported - times (fun|obj|list|str, %d)", other.LiteralType)
	}

	return nil, fmt.Errorf("unexpected value type %d * %d", l.LiteralType, other.LiteralType)
}

func (l *Literal) opDiv(other *Literal) (*Literal, error) {
	if l.LiteralType == RefLiteral || other.LiteralType == RefLiteral {
		return nil, errors.New("got reference, expected value")
	}

	switch l.LiteralType {
	case IntLiteral, DoubleLiteral:
		switch other.LiteralType {
		case IntLiteral:
			if other.Value == 0 {
				return nil, errors.New("dividing by zero")
			}

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) / other.Value.(int)}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) / float64(other.Value.(int))}, nil
			}

		case DoubleLiteral:
			if other.Value == 0 {
				return nil, errors.New("dividing by zero")
			}

			if l.LiteralType == IntLiteral {
				return &Literal{IntLiteral, l.Value.(int) * int(other.Value.(float64))}, nil
			} else {
				return &Literal{DoubleLiteral, l.Value.(float64) * other.Value.(float64)}, nil
			}
		case StringLiteral, FunLiteral, ObjLiteral, ListLiteral, ParsedListLiteral, ParsedObjLiteral, BoolLiteral:
			return nil, fmt.Errorf("operation not supported - div (number, %d)", other.LiteralType)
		}
	case FunLiteral, ObjLiteral, ParsedObjLiteral, ListLiteral, StringLiteral, BoolLiteral:
		return nil, fmt.Errorf("operation not supported - div (fun|obj|list|str|bool, %d)", other.LiteralType)
	}

	return nil, fmt.Errorf("unexpected value type %d / %d", l.LiteralType, other.LiteralType)
}

func (l *Literal) opEq(other *Literal) (*Literal, error) {
	if l.LiteralType != other.LiteralType {
		return &Literal{BoolLiteral, false}, nil
	}

	if l.LiteralType == RefLiteral {
		return nil, errors.New("expected value got reference")
	}

	switch l.LiteralType {
	case IntLiteral:
		return &Literal{BoolLiteral, l.Value.(int) == other.Value.(int)}, nil
	case DoubleLiteral:
		return &Literal{BoolLiteral, l.Value.(float64) == other.Value.(float64)}, nil
	case BoolLiteral:
		return &Literal{BoolLiteral, l.Value.(bool) == other.Value.(bool)}, nil
	case StringLiteral:
		return &Literal{BoolLiteral, l.Value.(string) == other.Value.(string)}, nil
	case ObjLiteral, ParsedObjLiteral:
		return nil, errors.New("object check not implemented")
	case ListLiteral, ParsedListLiteral:
		return nil, errors.New("list check not implemented")
	default:
		return nil, errors.New("equality cannot be checked")
	}
}