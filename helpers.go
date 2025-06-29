package parts

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
)

func GetScannerWithSource(source string) Scanner {
	return Scanner{Source: []rune(source), Rules: GetScannerRules()}
}

func GetParserWithSource(source, modulePath string) Parser {
	scanner := GetScannerWithSource(source)

	return Parser{
		Scanner:   &scanner,
		Literals:  InitialLiterals,
		LastToken: Token{Type: TokenInvalid},
		Meta:      make(map[string]string),
		Rules:     GetParserRules(),
		PostFix:   GetPostFixRules(),
		ModulePath: path.Dir(modulePath),
	}
}

func GetVMWithSource(source string, path string) (*VM, error) {
	parser := GetParserWithSource(source, path)

	code, err := parser.ParseAll()

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

func RunString(codeString, modulePath string) (*VM, error) {
	parser := GetParserWithSource(codeString, modulePath)

	code, err := parser.ParseAll()

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

func RunStringWithSyntax(codeString, syntax, modulePath string) (*VM, error) {
	syntaxVM, err := GetVMWithSource(syntax, modulePath)

	if err != nil {
		return nil, errors.Join(errors.New("got error when parsing syntax code"), err)
	}

	parser := GetParserWithSource(codeString, modulePath)

	FillConsts(syntaxVM, &parser)

	err = syntaxVM.Run()

	if err != nil {
		return nil, errors.Join(errors.New("got error while running syntax code"), err)
	}

	code, err := parser.ParseAll()

	if err != nil {
		return nil, errors.Join(errors.New("got error from within syntax parser"), err)
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
		fieldValue.Set(reflect.ValueOf(val.(bool)).Convert(field.Type))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.Set(reflect.ValueOf(val.(int)).Convert(field.Type))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldValue.Set(reflect.ValueOf(val.(int)).Convert(field.Type))
	case reflect.Float32, reflect.Float64:
		fieldValue.Set(reflect.ValueOf(val.(float64)).Convert(field.Type))
	case reflect.String:
		fieldValue.Set(reflect.ValueOf(val.(string)).Convert(field.Type))
	case reflect.Slice:
		if rawVal.LiteralType == ParsedListLiteral {
			FillSlice(fieldIdx, val.([]any), vm, out)
		} else {
			panic(errors.New("trying to assign element as list"))
		}
	case reflect.Struct:
		if rawVal.LiteralType != ParsedObjLiteral {
			panic(errors.New("wrong variable type"))
		}

		FillStruct(val.(map[string]any), vm, fieldValue.Addr().Interface())
	case reflect.Func:
		fieldValue.Set(reflect.ValueOf(val))
	case reflect.Map:
		if rawVal.LiteralType != ParsedObjLiteral {
			panic(errors.New("wrong variable type"))
		}

		obj := rawVal.Value.(PartsIndexable).GetAll()

		newMap := reflect.MakeMap(field.Type)

		for rawKey, rawVal := range obj {
			key := reflect.ValueOf(returnExpected(rawKey)).Convert(field.Type.Key())

			val, err := rawVal.ToGoTypes(vm)

			if err != nil {
				panic(err)
			}

			reflectVal := reflect.ValueOf(val).Convert(field.Type.Elem())

			newMap.SetMapIndex(key, reflectVal)
		}

		fieldValue.Set(newMap)
	default:
		fmt.Printf("%s type not supported - ignoring\n", field.Type.Kind().String())
	}
}

func FillStruct[T any](data map[string]any, vm *VM, out T) {
	outStruct := reflect.ValueOf(out).Elem()
	outType := outStruct.Type()

	for i := range outType.NumField() {
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
				fieldValue.Set(reflect.ValueOf(val.(bool)).Convert(field.Type))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldValue.Set(reflect.ValueOf(val.(int)).Convert(field.Type))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldValue.Set(reflect.ValueOf(val.(int)).Convert(field.Type))
			case reflect.Float32, reflect.Float64:
				fieldValue.Set(reflect.ValueOf(val.(float64)).Convert(field.Type))
			case reflect.String:
				fieldValue.Set(reflect.ValueOf(val.(string)).Convert(field.Type))
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
			case reflect.Map:

				newMap := reflect.MakeMap(field.Type)

				for rawKey, val := range val.(map[string]any) {
					key := reflect.ValueOf(returnExpected(rawKey)).Convert(field.Type.Key())

					reflectVal := reflect.ValueOf(val).Convert(field.Type.Elem())

					newMap.SetMapIndex(key, reflectVal)
				}

				fieldValue.Set(newMap)
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

		for i := range data {
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

	for i := range data {
		newSlice = reflect.Append(newSlice, reflect.ValueOf(data[i]).Convert(fieldValue.Type().Elem()))
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
