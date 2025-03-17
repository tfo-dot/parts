package parts

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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
		return nil, errors.Join(errors.New("got error from within parser"), err)
	}

	literals := make([]*Literal, len(parser.Literals))

	for idx, literal := range parser.Literals {
		literals[idx] = &literal
	}

	vmEnv := VMEnviroment{
		Enclosing: nil,
		Values:    StandardLibrary,
	}

	return &VM{
		Enviroment: &VMEnviroment{
			Enclosing: &vmEnv,
			Values:    make(map[string]*Literal),
		},
		Idx:      0,
		Code:     code,
		Literals: literals,
		Meta:     parser.Meta,
	}, nil
}

func RunString(codeString string) (*VM, error) {
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
		return nil, errors.Join(errors.New("got error from within parser"), err)
	}

	literals := make([]*Literal, len(parser.Literals))

	for idx, literal := range parser.Literals {
		literals[idx] = &literal
	}

	vmEnv := VMEnviroment{
		Enclosing: nil,
		Values:    StandardLibrary,
	}

	vm := VM{
		Enviroment: &VMEnviroment{
			Enclosing: &vmEnv,
			Values:    make(map[string]*Literal),
		},
		Idx:      0,
		Code:     code,
		Literals: literals,
		Meta:     parser.Meta,
	}

	err = vm.Run()

	if err != nil {
		return nil, err
	}

	return &vm, nil
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

	if !fieldValue.CanSet() {
		return
	}

	key := field.Name

	ignoreEmpty := false

	if tag, has := field.Tag.Lookup("parts"); has {
		rawKey, options := parseTag(tag)

		key = rawKey

		if options.Contains("ignoreEmpty") {
			ignoreEmpty = true
		}
	}

	key = fmt.Sprintf("RT%s", key)

	if !vm.Enviroment.Has(key) {
		if !fieldValue.IsZero() {
			return
		} else {
			if !field.IsExported() {
				return
			}

			if ignoreEmpty {
				return
			}

			panic(fmt.Errorf("field has no default value and was expected (%s)", field.Name))
		}
	}

	rawVal, err := vm.Enviroment.resolve(key)

	if err != nil {
		panic(err)
	}

	val, err := rawVal.ToGoTypes(vm)

	if err != nil {
		panic(err)
	}

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
	case reflect.Func:
		fieldValue.Set(reflect.ValueOf(val))
	default:
		fmt.Printf("%s type not supported - ignoring\n", field.Type.Kind().String())
	}
}

func FillStruct[T any](data map[string]any, vm *VM, out T) {
	outStruct := reflect.ValueOf(out).Elem()
	outType := outStruct.Type()

	for i := 0; i < outType.NumField(); i++ {
		field := outType.Field(i)
		fieldValue := outStruct.Field(i)

		key := field.Name

		ignoreEmpty := false

		if tag, has := field.Tag.Lookup("parts"); has {
			rawKey, options := parseTag(tag)

			key = rawKey

			if options.Contains("ignoreEmpty") {
				ignoreEmpty = true
			}
		}

		key = fmt.Sprintf("RT%s", key)

		if val, ok := data[key]; ok {
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
			case reflect.Func:
				fieldValue.Set(reflect.ValueOf(val))
			default:
				fmt.Printf("%s type not supported - ignoring\n", field.Type.Kind().String())
			}
		} else {
			if !fieldValue.IsZero() {
				return
			} else {
				if !field.IsExported() {
					return
				}

				if ignoreEmpty {
					return
				}

				panic(fmt.Errorf("field has no default value and was expected (%s)", field.Name))
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

/* From: https://cs.opensource.google/go/go/+/master:src/encoding/json/tags.go;l=1 */

type tagOptions string

func parseTag(tag string) (string, tagOptions) {
	tag, opt, _ := strings.Cut(tag, ",")
	return tag, tagOptions(opt)
}

func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}

	s := string(o)

	for s != "" {
		var name string
		name, s, _ = strings.Cut(s, ",")
		if name == optionName {
			return true
		}
	}
	return false
}
