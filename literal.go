package parts

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
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
		return func(args ...any) (any, error) {
			funcObj := l.Value.(PartsCallable)

			values := make([]*Literal, len(args))

			for idx, val := range args {
				lit, err := LiteralFromGo(val)

				if err != nil {
					return nil, errors.Join(errors.New("got error while converting from go value to parts"), err)
				}

				resolvedExpr, err := vm.simplifyLiteral(lit, true)

				if err != nil {
					return nil, errors.Join(errors.New("got error while converting from go value to parts"), err)
				}

				values[idx] = resolvedExpr
			}

			tempVM := vm.copyVM()

			for idx, param := range funcObj.GetArguments() {
				vm.Enviroment.define(fmt.Sprintf("RT%s", string(param)), values[idx])
			}

			funcObj.Call(&tempVM)

			if err := tempVM.Run(); err != nil {
				return nil, errors.Join(errors.New("got error while running function body"), err)
			}

			if tempVM.ReturnValue != nil {
				converted, err := tempVM.ReturnValue.ToGoTypes(&tempVM)

				if err != nil {
					return nil, errors.Join(errors.New("got error while converting from parts value to go"), err)
				}

				return converted, nil
			} else {
				return nil, nil
			}

		}, nil
	case ObjLiteral:
		return nil, errors.New("use simplyfied ParsedObjLiteral")
	case ParsedObjLiteral:
		entryMap := make(map[string]any)

		for key, entry := range l.Value.(PartsIndexable).GetAll() {
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
		entriesList := make([]any, len(l.Value.(PartsIndexable).GetAll()))

		for i := 0; i < len(l.Value.(PartsIndexable).GetAll()); i++ {
			entry := l.Value.(PartsIndexable).GetByKey(fmt.Sprintf("IT%d", i))

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
		l.Value.(PartsIndexable).SetByKey(fmt.Sprintf("IT%d", l.Value.(PartsIndexable).Length()), other)

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

func (l *Literal) pretify() string {
	switch l.LiteralType {
	case IntLiteral:
		return fmt.Sprintf("%d", l.Value.(int))
	case DoubleLiteral:
		return fmt.Sprintf("%f", l.Value.(float64))
	case BoolLiteral:
		return fmt.Sprintf("%t", l.Value.(bool))
	case StringLiteral:
		return l.Value.(string)
	case ParsedObjLiteral:
		var parts []string

		for key, value := range l.Value.(PartsIndexable).GetAll() {
			parts = append(parts, fmt.Sprintf("%q: %s", key, value.pretify()))
		}

		return "|>" + strings.Join(parts, ", ") + "<|"
	case ParsedListLiteral:
		var parts []string

		for _, value := range l.Value.(PartsIndexable).GetAll() {
			parts = append(parts, value.pretify())
		}

		return "[" + strings.Join(parts, ", ") + "]"
	case RefLiteral:
		ref := l.Value.(ReferenceDeclaration)
		return fmt.Sprintf("|>key: %s, dyn: %t<|", ref.Reference, ref.Dynamic)
	case FunLiteral:
		funcObj := l.Value.(PartsCallable)
		return fmt.Sprintf("func(%s)", strings.Join(funcObj.GetArguments(), ","))
	default:
		panic(fmt.Errorf("Cant pretify that (%d)", l.LiteralType))
	}
}

func LiteralFromGo(value any) (*Literal, error) {
	typeOf := reflect.TypeOf(value)

	switch typeOf.Kind() {
	case reflect.Bool:
		return &Literal{BoolLiteral, value}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Literal{IntLiteral, value.(int)}, nil
		//Different case in case sometimes uint types are added to parts
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Literal{IntLiteral, value.(int)}, nil
	case reflect.Float32, reflect.Float64:
		return &Literal{DoubleLiteral, value.(float64)}, nil
	case reflect.String:
		return &Literal{StringLiteral, value}, nil
	case reflect.Func:
		return &Literal{FunLiteral, FFIFunction{value}}, nil
	case reflect.Array:
		return &Literal{ParsedListLiteral, ConvertListToParts(value.([]any))}, nil
	case reflect.Slice:
		return &Literal{ParsedListLiteral, ConvertListToParts(value.([]any))}, nil
	case reflect.Map:
		return &Literal{ParsedObjLiteral, &FFIMap[any, any]{value.(map[any]any)}}, nil
	}

	//TODO types: Struct, (for references and no copies) Pointer

	return nil, errors.New("value type not supported for conversion")
}

type FFIFunction struct {
	Function any
}

func (ffi FFIFunction) Call(vm *VM) {
	funcVal := reflect.ValueOf(ffi.Function)
	funcType := funcVal.Type()

	if funcType.Kind() != reflect.Func {
		panic("Not a function in FFIFunction (GetArguments)")
	}

	resCount := funcType.NumOut()

	if resCount > 1 {
		panic("Function should return one or less values")
	}

	args := ffi.GetArguments()
	values := make([]reflect.Value, len(args))

	for idx, key := range args {
		val, err := vm.Enviroment.resolve(fmt.Sprintf("RT%s", key))

		if err != nil {
			panic(err)
		}

		converted, err := val.ToGoTypes(vm)

		if err != nil {
			panic(err)
		}

		values[idx] = reflect.ValueOf(converted)
	}

	funcOut := funcVal.Call(values)

	if resCount == 1 {
		resConverted, err := LiteralFromGo(funcOut[0].Interface())

		if err != nil {
			panic(err)
		}

		vm.ReturnValue = resConverted
	}

}

func (ffi FFIFunction) GetArguments() []string {
	funcVal := reflect.ValueOf(ffi.Function)
	funcType := funcVal.Type()

	if funcType.Kind() != reflect.Func {
		panic("Not a function in FFIFunction (GetArguments)")
	}

	numIn := funcType.NumIn()
	args := make([]string, numIn)

	for i := 0; i < numIn; i++ {
		args[i] = funcType.In(i).Name()
	}

	return args
}

func ConvertListToParts(list []any) *PartsObject {
	values := PartsObject{Entries: make(map[string]*Literal)}

	for idx, val := range list {
		converted, err := LiteralFromGo(val)

		if err != nil {
			panic(err)
		}

		values.SetByKey(fmt.Sprintf("IT%d", idx), converted)
	}

	return &values
}

type FFIMap[K comparable, V any] struct {
	Entries map[K]V
}

func (ffi *FFIMap[K, V]) Get(key *Literal) *Literal {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return ffi.GetByKey(hash)
}

func (ffi *FFIMap[K, V]) Set(key *Literal, value *Literal) *Literal {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return ffi.SetByKey(hash, value)
}

func (ffi *FFIMap[K, V]) Has(key *Literal) bool {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return ffi.HasByKey(hash)
}

func (ffi *FFIMap[K, V]) Length() int {
	return len(ffi.Entries)
}

func (ffi *FFIMap[K, V]) GetAll() map[string]*Literal {
	temp := make(map[string]*Literal)

	for key, val := range ffi.Entries {
		keyLit, err := LiteralFromGo(key)

		if err != nil {
			panic(err)
		}

		hash, err := HashLiteral(*keyLit)

		if err != nil {
			panic(err)
		}

		valLit, err := LiteralFromGo(val)

		if err != nil {
			panic(err)
		}

		temp[hash] = valLit
	}

	return temp
}

func (ffi *FFIMap[K, V]) GetByKey(key string) *Literal {
	val := returnExpected(key)

	lit, err := LiteralFromGo(ffi.Entries[val.(K)])

	if err != nil {
		panic(err)
	}

	return lit
}

func (ffi *FFIMap[K, V]) SetByKey(key string, value *Literal) *Literal {
	val, err := value.ToGoTypes(nil)

	keyParsed := returnExpected(key)

	if err != nil {
		panic(err)
	}

	ffi.Entries[keyParsed.(K)] = val.(V)

	return value
}

func (ffi *FFIMap[K, V]) HasByKey(key string) bool {
	keyParsed := returnExpected(key)

	_, has := ffi.Entries[keyParsed.(K)]
	return has
}

func returnExpected(key string) any {
	switch key[0:2] {
	case "IT":
		n, err := strconv.Atoi(key[2:])

		if err != nil {
			panic(err)
		}

		return n
	case "DT":
		n, err := strconv.ParseFloat(key[2:], 64)

		if err != nil {
			panic(err)
		}

		return n
	case "BT":
		return key[2] == '1'
	case "ST":
		return key[2:]
	case "RT":
		return key[2:]
	default:
		panic("Invalid hash")
	}
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
