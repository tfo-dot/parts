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

	return &VM{
		Enviroment: &VMEnviroment{
			Enclosing: nil,
			Values:    make(map[string]any),
		},
		Idx:      0,
		Code:     code,
		Literals: parser.Literals,
		Meta:     parser.Meta,
	}, nil
}

func ReadFromParts[T any](vm *VM, out *T) {
	outStruct := reflect.TypeOf(*out)

	for i := range outStruct.NumField() {
		field := outStruct.Field(i)

		key, has := field.Tag.Lookup("parts")

		if !has {
			key = field.Name
		}

		val, err := vm.Enviroment.resolve(fmt.Sprintf("RT%s", key))

		if err != nil {
			panic(err)
		}

		valueType := val.(Literal).LiteralType

		switch valueType {
		case IntLiteral:
			reflect.ValueOf(out).Elem().Field(i).SetInt(int64(val.(Literal).Value.(int)))
		case DoubleLiteral:
			reflect.ValueOf(out).Elem().Field(i).SetFloat(float64(val.(Literal).Value.(float32)))
		case BoolLiteral:
			reflect.ValueOf(out).Elem().Field(i).SetBool(val.(Literal).Value.(bool))
		case StringLiteral:
			reflect.ValueOf(out).Elem().Field(i).SetString(val.(Literal).Value.(string))
		//TODO handle rest of the objects
		default:
			reflect.ValueOf(out).Elem().Field(i).SetZero()
		}
	}
}
