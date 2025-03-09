package parts

import (
	"errors"
	"fmt"
	"reflect"
)

func GetScannerWithSource(source string) Scanner {
	return Scanner{Source: []rune(source), Rules: ScannerRules}
}

func GetParserWithSource(source string) Parser {
	scanner := GetScannerWithSource(source)

	return Parser{
		Scanner:   &scanner,
		Scope:     TopLevel,
		Literals:  InitialLiterals,
		LastToken: Token{Type: TokenInvalid},
		Meta:      make(map[string]string),
	}
}

func GetVMWithSource(source string) (*VM, error) {
	scanner := GetScannerWithSource(source)

	parser := Parser{
		Scanner:   &scanner,
		Scope:     TopLevel,
		Literals:  InitialLiterals,
		LastToken: Token{Type: TokenInvalid},
		Meta:      make(map[string]string),
	}

	code, err := parser.parseAll()

	if err != nil {
		return &VM{}, errors.Join(errors.New("got error from within parser"), err)
	}

	literals := make([]*Literal, len(parser.Literals))

	for idx, literal := range parser.Literals {
		literals[idx] = &literal 
	}

	return &VM{
		Enviroment: &VMEnviroment{
			Enclosing: nil,
			Values:    make(map[string]*Literal),
		},
		Idx:      0,
		Code:     code,
		Literals: literals,
		Meta:     parser.Meta,
	}, nil
}

func RunString(codeString string) (VM, error) {
	scanner := GetScannerWithSource(codeString)

	parser := Parser{
		Scanner:   &scanner,
		Scope:     TopLevel,
		Literals:  InitialLiterals,
		LastToken: Token{Type: TokenInvalid},
		Meta:      make(map[string]string),
	}

	code, err := parser.parseAll()

	if err != nil {
		return VM{}, errors.Join(errors.New("got error from within parser"), err)
	}

	literals := make([]*Literal, len(parser.Literals))

	for _, literal := range parser.Literals {
		literals = append(literals, &literal)
	}

	vm := VM{
		Enviroment: &VMEnviroment{
			Enclosing: nil,
			Values:    make(map[string]*Literal),
		},
		Idx:      0,
		Code:     code,
		Literals: literals,
		Meta:     parser.Meta,
	}

	err = vm.Run()

	if err != nil {
		return VM{}, err
	}

	return vm, nil
}

func ReadFromParts[T any](vm *VM, out *T) {
	for i := range reflect.TypeOf(*out).NumField() {
		FillField(i, vm, out)
	}
}

func FillField[T any](fieldIdx int, vm *VM, out *T) {
	outStruct := reflect.TypeOf(*out)

	field := outStruct.Field(fieldIdx)

	fieldValue := reflect.ValueOf(out).Elem().Field(fieldIdx)

	key := field.Name

	if tag, has := field.Tag.Lookup("parts"); has {
		key = tag
	}

	rawVal, err := vm.Enviroment.resolve(fmt.Sprintf("RT%s", key))

	if err != nil {
		panic(err)
	}

	val, err := rawVal.ToGoTypes(vm)

	switch field.Type.Kind() {
	case reflect.Bool:
		fieldValue.SetBool(val.(bool))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(int64(val.(int)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldValue.SetUint(uint64(val.(int)))
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(val.(float64))
	case reflect.String:
		fieldValue.SetString(val.(string))
	case reflect.Slice:
		if rawVal.LiteralType == ParsedListLiteral {
			FillSlice(fieldIdx, val.([]any), vm, out)
		} else {
			panic(errors.New("trying to assign element as list"))
		}
	case reflect.Struct:
		if rawVal.LiteralType == ParsedObjLiteral {
			FillStruct(val.(map[string]any), vm, fieldValue.Addr().Interface())
		} else {
			panic(errors.New("wrong variable type"))
		}

	default:
		//TODO handle function type
		fmt.Printf("%s type not supported - ignoring\n", field.Type.Kind().String())
	}
}

func FillStruct[T any](data map[string]any, vm *VM, out T) {
	outStruct := reflect.ValueOf(out).Elem()
	outType := outStruct.Type()

	for i := 0; i < outType.NumField(); i++ {
		field := outType.Field(i)
		key := field.Name

		if tag, has := field.Tag.Lookup("parts"); has {
			key = tag
		}

		if val, ok := data[fmt.Sprintf("RT%s", key)]; ok {
			fieldValue := reflect.ValueOf(out).Elem().Field(i)

			switch field.Type.Kind() {
			case reflect.Bool:
				fieldValue.SetBool(val.(bool))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldValue.SetInt(int64(val.(int)))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldValue.SetUint(uint64(val.(int)))
			case reflect.Float32, reflect.Float64:
				fieldValue.SetFloat(val.(float64))
			case reflect.String:
				fieldValue.SetString(val.(string))
			case reflect.Slice:
				if val, ok := val.([]any); ok {
					FillSlice(i, val, vm, out)
				} else {
					panic(errors.New("trying to assign element as list"))
				}

			case reflect.Struct:
				if val, ok := val.(map[string]any); ok {
					FillStruct(val, vm, fieldValue.Addr().Interface())
				} else {
					panic(errors.New("wrong variable type"))
				}

			default:
				//TODO handle function type
				fmt.Printf("%s type not supported - ignoring\n", field.Type.Kind().String())
			}
		}
	}
}

func FillSlice[T any](fieldIdx int, data []any, vm *VM, out T) {
	fieldValue := reflect.ValueOf(out).Elem().Field(fieldIdx)

	if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Struct {
		newSlice := reflect.MakeSlice(fieldValue.Type(), 0, len(data))

		for i := 0; i < len(data); i++ {
			newStruct := reflect.New(fieldValue.Type().Elem()).Elem()

			if val, ok := data[i].(map[string]any); ok {
				FillStruct(val, vm, newStruct.Addr().Interface())
			} else {
				panic(errors.New("wrong variable type"))
			}

			newSlice = reflect.Append(newSlice, newStruct)
		}

		return
	}

	newSlice := reflect.MakeSlice(fieldValue.Type(), 0, len(data))

	for i := 0; i < len(data); i++ {
		newSlice = reflect.Append(newSlice, reflect.ValueOf(data[i]))
	}

	fieldValue.Set(newSlice)
}
