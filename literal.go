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
	PointerLiteral
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
		capturedVM := vm
		funcObj := l.Value.(PartsCallable)

		val := func(args ...any) (any, error) {
			values := make([]*Literal, len(args))

			for idx, val := range args {
				lit, err := LiteralFromGo(val)

				if err != nil {
					return nil, errors.Join(errors.New("got error while converting from go value to parts"), err)
				}

				resolvedExpr, err := capturedVM.simplifyLiteral(lit, true)

				if err != nil {
					return nil, errors.Join(errors.New("got error while converting from go value to parts"), err)
				}

				values[idx] = resolvedExpr
			}

			tempVM, res, err := capturedVM.callFunctionVM(funcObj, values)

			if err != nil {
				return nil, errors.Join(errors.New("got error while calling function in parts"), err)
			}

			if res == nil {
				gofied, err := res.ToGoTypes(tempVM)

				if err != nil {
					return nil, errors.Join(errors.New("got error while converting expression to go (processing function results)"), err)
				}

				return gofied, nil
			}

			return nil, nil
		}

		return val, nil
	case ObjLiteral:
		newVal, err := vm.simplifyLiteral(l, true)

		if err != nil {
			return nil, errors.Join(fmt.Errorf("got error while converting value, %s", l.Value), err)
		}

		entryMap := make(map[string]any)

		for key, entry := range newVal.Value.(PartsIndexable).GetAll() {
			val, err := entry.ToGoTypes(vm)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("got error while resolving value, %s", newVal.Value), err)
			}

			entryMap[key] = val
		}

		return entryMap, nil

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
		temp, err := vm.simplifyLiteral(l, true)

		if err != nil {
			return nil, errors.Join(errors.New("got error while simplifiyng list"), err)
		}

		return temp.ToGoTypes(vm)
	case ParsedListLiteral:
		entriesList := make([]any, len(l.Value.(PartsIndexable).GetAll()))

		for i := range len(l.Value.(PartsIndexable).GetAll()) {
			entry := l.Value.(PartsIndexable).GetByKey(fmt.Sprintf("IT%d", i))

			val, err := entry.ToGoTypes(vm)

			if err != nil {
				return nil, errors.Join(fmt.Errorf("got error while resolving value, %s", l.Value), err)
			}

			entriesList[i] = val
		}

		return entriesList, nil
	case PointerLiteral:
		return l.Value, nil
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
	case ListLiteral, ParsedListLiteral:
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

func (l *Literal) opGt(other *Literal) (*Literal, error) {
	if l.LiteralType != other.LiteralType {
		return &Literal{BoolLiteral, false}, nil
	}

	if l.LiteralType == RefLiteral {
		return nil, errors.New("expected value got reference")
	}

	switch l.LiteralType {
	case IntLiteral:
		return &Literal{BoolLiteral, l.Value.(int) > other.Value.(int)}, nil
	case DoubleLiteral:
		return &Literal{BoolLiteral, l.Value.(float64) > other.Value.(float64)}, nil
	case BoolLiteral:
		return &Literal{BoolLiteral, l.Value.(bool) && !other.Value.(bool)}, nil
	case StringLiteral:
		return &Literal{BoolLiteral, l.Value.(string) > other.Value.(string)}, nil
	case ObjLiteral, ParsedObjLiteral:
		return nil, errors.New("object check not implemented")
	case ListLiteral, ParsedListLiteral:
		return nil, errors.New("list check not implemented")
	default:
		return nil, errors.New("equality cannot be checked")
	}
}

func (l *Literal) opLt(other *Literal) (*Literal, error) {
	if l.LiteralType != other.LiteralType {
		return &Literal{BoolLiteral, false}, nil
	}

	if l.LiteralType == RefLiteral {
		return nil, errors.New("expected value got reference")
	}

	switch l.LiteralType {
	case IntLiteral:
		return &Literal{BoolLiteral, l.Value.(int) < other.Value.(int)}, nil
	case DoubleLiteral:
		return &Literal{BoolLiteral, l.Value.(float64) < other.Value.(float64)}, nil
	case BoolLiteral:
		return &Literal{BoolLiteral, !l.Value.(bool) && other.Value.(bool)}, nil
	case StringLiteral:
		return &Literal{BoolLiteral, l.Value.(string) < other.Value.(string)}, nil
	case ObjLiteral, ParsedObjLiteral:
		return nil, errors.New("object check not implemented")
	case ListLiteral, ParsedListLiteral:
		return nil, errors.New("list check not implemented")
	default:
		return nil, errors.New("equality cannot be checked")
	}
}

func (l *Literal) opMod(other *Literal) (*Literal, error) {
	if l.LiteralType == IntLiteral && other.LiteralType == l.LiteralType {
		return &Literal{IntLiteral, l.Value.(int) % other.Value.(int)}, nil
	} else {
		return nil, fmt.Errorf("operation not supported - mod (dbl|bool|str|ref|fun|obj|ptr, %d)", other.LiteralType)
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
		return fmt.Sprintf("<ref to '%s'>", l.Value)
	case FunLiteral:
		funcObj := l.Value.(PartsCallable)
		return fmt.Sprintf("func(%s)", strings.Join(funcObj.GetArguments(), ","))
	case PointerLiteral:
		return fmt.Sprintf("<pointer>")
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
		v := reflect.ValueOf(value)
		return &Literal{IntLiteral, int(v.Int())}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := reflect.ValueOf(value)
		return &Literal{IntLiteral, int(v.Uint())}, nil
	case reflect.Float32, reflect.Float64:
		return &Literal{DoubleLiteral, value.(float64)}, nil
	case reflect.String:
		return &Literal{StringLiteral, value}, nil
	case reflect.Func:
		return &Literal{FunLiteral, FFIFunction{value}}, nil
	case reflect.Array, reflect.Slice:

		reflectVal := reflect.ValueOf(value)

		tempArr := make([]any, reflectVal.Len())

		for i := range reflectVal.Len() {
			tempArr[i] = reflectVal.Index(i).Interface()
		}

		return &Literal{ParsedListLiteral, ConvertListToParts(tempArr)}, nil
	case reflect.Map:
		return &Literal{ParsedObjLiteral, NewFFIMap(value)}, nil
	case reflect.Pointer:
		return &Literal{PointerLiteral, value}, nil
	case reflect.Struct:
		return &Literal{PointerLiteral, value}, nil
	}

	return nil, errors.New("value type not supported for conversion")
}

type FFIFunction struct {
	Function any
}

func (ffi FFIFunction) Call(vm *VM) error {
	funcVal := reflect.ValueOf(ffi.Function)
	funcType := funcVal.Type()

	if funcType.Kind() != reflect.Func {
		return errors.New("Not a function in FFIFunction (GetArguments)")
	}

	switch funcType.NumOut() {
	case 0:
	case 1:
	case 2:
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if funcType.Out(1) != errorType {
			return errors.New("Function with two return values must have error as the second type")
		}
	default:
		return errors.New("Function should return one, two (with error), or zero values")
	}

	args := ffi.GetArguments()
	values := make([]reflect.Value, len(args))

	for idx, key := range args {
		val, err := vm.Enviroment.resolve(fmt.Sprintf("RT%s", key))

		if err != nil {
			return err
		}

		converted, err := val.ToGoTypes(vm)

		if err != nil {
			return err
		}

		reflectNew := reflect.New(funcType.In(idx)).Elem()

		reflectNew.Set(reflect.ValueOf(converted))

		values[idx] = reflectNew
	}

	funcOut := funcVal.Call(values)

	if len(funcOut) == 0 {
		vm.ReturnValue = &Literal{ParsedObjLiteral, &PartsSpecialObject{
			Internal: &PartsObject{},
			Hash:     "Option.None",
		}}

		return nil
	}

	if len(funcOut) == 2 {
		if err, ok := funcOut[1].Interface().(error); ok && err != nil {
			vm.ReturnValue = &Literal{ParsedObjLiteral, &PartsSpecialObject{
				Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": {LiteralType: StringLiteral, Value: err.Error()}}},
				Hash:     "Result.Error",
			}}

			vm.ExitCode = ReturnCode
			vm.EarlyExit = true

			return nil
		}
	}

	resConverted, err := LiteralFromGo(funcOut[0].Interface())

	if err != nil {
		return err
	}

	vm.ReturnValue = resConverted
	vm.ExitCode = ReturnCode
	vm.EarlyExit = true

	return nil
}

func (ffi FFIFunction) GetArguments() []string {
	funcVal := reflect.ValueOf(ffi.Function)
	funcType := funcVal.Type()

	if funcType.Kind() != reflect.Func {
		panic("Not a function in FFIFunction (GetArguments)")
	}

	numIn := funcType.NumIn()
	args := make([]string, numIn)

	for i := range numIn {
		args[i] = fmt.Sprintf("val_%d", i)
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

type FFIMap struct {
	Entries reflect.Value
}

func (ffi *FFIMap) Get(key *Literal) *Literal {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return ffi.GetByKey(hash)
}

func (ffi *FFIMap) Set(key *Literal, value *Literal) *Literal {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return ffi.SetByKey(hash, value)
}

func (ffi *FFIMap) Has(key *Literal) bool {
	hash, err := HashLiteral(*key)

	if err != nil {
		panic(err)
	}

	return ffi.HasByKey(hash)
}

func (ffi *FFIMap) Length() int {
	return ffi.Entries.Len()
}

func (ffi *FFIMap) GetAll() map[string]*Literal {
	temp := make(map[string]*Literal)

	iter := ffi.Entries.MapRange()

	for iter.Next() {
		key := iter.Key()
		val := iter.Value()

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

func (ffi *FFIMap) GetByKey(key string) *Literal {
	val := returnExpected(key)

	lit, err := LiteralFromGo(ffi.Entries.MapIndex(reflect.ValueOf(val)).Interface())

	if err != nil {
		panic(err)
	}

	return lit
}

func (ffi *FFIMap) SetByKey(key string, value *Literal) *Literal {
	val, err := value.ToGoTypes(nil)

	keyParsed := returnExpected(key)

	if err != nil {
		panic(err)
	}

	ffi.Entries.SetMapIndex(reflect.ValueOf(keyParsed), reflect.ValueOf(val))

	return value
}

func (ffi *FFIMap) HasByKey(key string) bool {
	keyParsed := returnExpected(key)

	return ffi.Entries.MapIndex(reflect.ValueOf(keyParsed)).IsValid()
}

func (ffi *FFIMap) TypeHash() string {
	return ""
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
		return fmt.Sprintf("RT%s", literal.Value), nil
	case PointerLiteral:
		return fmt.Sprintf("PT%s", reflect.ValueOf(literal.Value).Type().String()), nil
	}

	return "", fmt.Errorf("literal not hashable %d", literal.LiteralType)
}

func NewFFIMap(val any) *FFIMap {
	return &FFIMap{reflect.ValueOf(val)}
}
